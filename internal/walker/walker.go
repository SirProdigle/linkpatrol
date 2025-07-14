package walker

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"golang.org/x/time/rate"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	"github.com/sirprodigle/linkpatrol/internal/logger"
)

type DomainLimiterProvider interface {
	GetDomainLimiter(domain string) *rate.Limiter
}

type Walker struct {
	client        *http.Client
	toWalkChan    chan WalkerRequest
	toTestChan    chan WalkerRequest
	resultsChan   chan<- cache.CacheEntry
	activeWalkers *atomic.Int32
	cache         *cache.ResultsCache
	logger        *logger.Logger
	targetBaseUrl string
	workerPool    DomainLimiterProvider
}

func NewWalker(client *http.Client, resultsCache *cache.ResultsCache, toWalkChan chan WalkerRequest, toTestChan chan WalkerRequest, activeWalkers *atomic.Int32, logger *logger.Logger, targetBaseUrl string, workerPool DomainLimiterProvider, resultsChan chan<- cache.CacheEntry) *Walker {
	return &Walker{
		client:        client,
		toWalkChan:    toWalkChan,
		toTestChan:    toTestChan,
		cache:         resultsCache,
		activeWalkers: activeWalkers,
		logger:        logger,
		targetBaseUrl: targetBaseUrl,
		workerPool:    workerPool,
		resultsChan:   resultsChan,
	}
}

func (w *Walker) Walk(ctx context.Context, toTest WalkerRequest) {

	w.activeWalkers.Add(1)
	defer w.activeWalkers.Add(-1)

	if w.cache.HasResult(toTest.Path) {
		w.logger.Trace("No need to walk url: %s, it's already tested", toTest.Path)
		return
	}

	// Expand to be a more dedicated "ignore list" later on
	if strings.Contains(toTest.Path, "/cdn-cgi/") {
		w.logger.Debug("Skipping url: %s, it's a cdn-cgi url", toTest.Path)
		return
	}

	w.walkUrl(ctx, toTest)
}

func (w *Walker) walkUrl(ctx context.Context, toTest WalkerRequest) {
	// Get domain-specific rate limiter
	// Convert a null basepath to our target base url
	domain := toTest.BasePath
	if domain == "" {
		domain = w.targetBaseUrl
	}
	domainLimiter := w.workerPool.GetDomainLimiter(domain)

	// Wait for rate limiter permit
	if !domainLimiter.Allow() {
		w.logger.Progress("Waiting for rate limit permit for domain: %s", toTest.BasePath)
		if err := domainLimiter.Wait(ctx); err != nil {
			w.logger.Error("Error waiting for rate limit permit for domain: %s", toTest.BasePath)
			return
		}
	}

	// Make a HTTP request to the url
	w.logger.Debug("Making HTTP request to url %s", toTest.Path)
	resp, err := w.client.Get(toTest.Path)
	if err != nil {
		w.logger.Error("Error making HTTP request to url %s: %s", toTest.Path, err)
		w.resultsChan <- cache.CacheEntry{
			URL:    toTest.Path,
			Status: cache.Dead,
			Error:  err.Error(),
		}
		return
	}
	defer resp.Body.Close()

	w.logger.Debug("Reading body from url %s", toTest.Path)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("Error reading body from url %s: %s", toTest.Path, err)
		w.resultsChan <- cache.CacheEntry{
			URL:    toTest.Path,
			Status: cache.Dead,
			Error:  err.Error(),
		}
		return
	}

	// If we get body correctly, count as tested
	w.logger.Debug("Sending result to resultsChan for url %s", toTest.Path)
	w.resultsChan <- cache.CacheEntry{
		URL:    toTest.Path,
		Status: cache.Live,
		Error:  "",
	}

	w.logger.Debug("Finding matches in body of url %s", toTest.Path)
	regexes := GetRegexes()

	// Do line by line parsing of the body
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		lastMatch := ""
		//w.logger.Trace("Processing line: %s", line)
		for _, regex := range regexes {
			matches := regex.FindStringSubmatch(line)
			if len(matches) > 0 {
				w.logger.Trace("Found match: %s", matches[0])
			} else {
				continue
			}
			var matchedUrl string
			if len(matches) > 1 {
				// Use capture group (matches[1]) for HTML patterns that extract URLs from attributes
				matchedUrl = matches[1]
			} else {
				// Use full match (matches[0]) for patterns that match the URL directly
				matchedUrl = matches[0]
			}

			if lastMatch == matchedUrl {
				w.logger.Debug("Skipping duplicate url: %s", matchedUrl)
				continue
			}

			if w.IsSameDomain(matchedUrl, w.targetBaseUrl) {
				w.logger.Debug("Sending same domain url to walker: %s", matchedUrl)
				w.toWalkChan <- WalkerRequest{
					Path:     matchedUrl,
					BasePath: toTest.BasePath,
				}
				lastMatch = matchedUrl
			} else {
				w.logger.Debug("Sending url to tester: %s", matchedUrl)
				w.toTestChan <- WalkerRequest{
					Path:     matchedUrl,
					BasePath: toTest.BasePath,
				}
				lastMatch = matchedUrl
			}
		}
	}
}

func (w *Walker) IsSameDomain(target string, baseUrl string) bool {
	// Fragment URLs (like #section) should always go to testers, not walkers
	if strings.HasPrefix(target, "#") {
		return false
	}

	parsedTarget, err := url.Parse(target)
	if err != nil {
		w.logger.Error("Error parsing target url: %s", err)
		return false
	}

	// Only allow URLs ending with .html, .htm, or with no extension at all
	allowed := func(u *url.URL) bool {
		path := u.Path
		// If path is empty or ends with /, treat as allowed (directory or root)
		if path == "" || strings.HasSuffix(path, "/") {
			return true
		}
		// Get the last segment
		segments := strings.Split(path, "/")
		last := segments[len(segments)-1]
		// If last segment has no dot, treat as allowed (no extension)
		if !strings.Contains(last, ".") {
			return true
		}
		// Check for .html or .htm (case-insensitive)
		lower := strings.ToLower(last)
		if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
			return true
		}
		return false
	}

	if !allowed(parsedTarget) {
		w.logger.Debug("Skipping non-html url: %s", target)
		return false
	}

	// If the target is relative (no host), it's considered same domain
	if parsedTarget.Host == "" {
		return true
	}

	parsedBaseUrl, err := url.Parse(baseUrl)
	if err != nil {
		w.logger.Error("Error parsing base url: %s", err)
		return false
	}

	return parsedTarget.Host == parsedBaseUrl.Host
}
