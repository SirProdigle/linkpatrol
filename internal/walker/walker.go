package walker

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
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

var bannedPaths = []string{
	"/cdn-cgi/",
	"/wp-admin/",
	"/wp-login.php",
}

func (w *Walker) Walk(ctx context.Context, toTest WalkerRequest) {

	w.activeWalkers.Add(1)
	defer w.activeWalkers.Add(-1)

	// Try to claim this URL atomically - if we can't, another worker is handling it
	if !w.cache.TryClaim(toTest.Path) {
		w.logger.Trace("No need to walk url: %s, it's already tested or being processed", toTest.Path)
		return
	}

	// Skip if any banned path appears in the string
	for _, banned := range bannedPaths {
		if strings.Contains(toTest.Path, banned) {
			w.logger.Debug("Skipping url: %s, it's a banned url", toTest.Path)
			return
		}
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

	w.logger.Progress("Reading entire body from url %s", toTest.Path)

	// Read entire response body into memory
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

	// Mark as live since we successfully read the body
	w.logger.Debug("Sending result to resultsChan for url %s", toTest.Path)
	w.resultsChan <- cache.CacheEntry{
		URL:    toTest.Path,
		Status: cache.Live,
		Error:  "",
	}

	// Process entire body with all regexes
	bodyText := string(body)
	regexes := GetRegexes()
	seenUrls := make(map[string]bool)

	for regexId, regex := range regexes {
		matches := regex.FindAllStringSubmatch(bodyText, -1)
		for _, match := range matches {
			if len(match) == 0 {
				continue
			}

			var matchedUrl string
			if len(match) > 1 {
				// Use capture group (match[1]) for HTML patterns that extract URLs from attributes
				matchedUrl = match[1]
			} else {
				// Use full match (match[0]) for patterns that match the URL directly
				matchedUrl = match[0]
			}

			// Special handling for srcset - extract individual URLs
			if regexId == ImgSrcsetRegexIdentifier {
				w.processSrcsetUrls(matchedUrl, toTest, seenUrls, bodyText)
				continue
			}

			// Skip duplicates
			if seenUrls[matchedUrl] {
				continue
			}
			seenUrls[matchedUrl] = true

			w.logger.Trace("Found match: %s on url %s", match[0], toTest.Path)
			w.processFoundUrl(matchedUrl, toTest, bodyText)
		}
	}
}

// processSrcsetUrls extracts individual URLs from srcset attribute values
func (w *Walker) processSrcsetUrls(srcsetValue string, toTest WalkerRequest, seenUrls map[string]bool, bodyText string) {
	// srcset format: "url1 descriptor1, url2 descriptor2, ..."
	// Extract URLs (everything before whitespace or comma)
	urls := strings.Split(srcsetValue, ",")
	for _, urlEntry := range urls {
		urlEntry = strings.TrimSpace(urlEntry)
		if urlEntry == "" {
			continue
		}

		// Split by whitespace to get just the URL part (before descriptor like "330w" or "2x")
		parts := strings.Fields(urlEntry)
		if len(parts) > 0 {
			url := strings.TrimSpace(parts[0])
			if url != "" && !seenUrls[url] {
				seenUrls[url] = true
				w.logger.Trace("Found srcset URL: %s on url %s", url, toTest.Path)
				w.processFoundUrl(url, toTest, bodyText)
			}
		}
	}
}

// processFoundUrl handles a discovered URL
func (w *Walker) processFoundUrl(matchedUrl string, toTest WalkerRequest, bodyText string) {
	if w.IsSameDomain(matchedUrl, w.targetBaseUrl) {
		w.logger.Debug("Sending same domain url to walker: %s", matchedUrl)
		// Resolve relative URLs using BasePath
		resolvedURL := matchedUrl
		if parsed, err := url.Parse(matchedUrl); err == nil && parsed.Host == "" && toTest.BasePath != "" {
			base, err := url.Parse(toTest.BasePath)
			if err == nil {
				resolvedURL = base.ResolveReference(parsed).String()
				w.logger.Debug("🟦 Resolved URL: %s", resolvedURL)
			}
		}
		w.toWalkChan <- WalkerRequest{
			Path:     resolvedURL,
			BasePath: toTest.BasePath,
		}
	} else {
		w.logger.Debug("Sending url to tester: %s", matchedUrl)
		if strings.HasPrefix(matchedUrl, "#") {
			// Check directly for an id tag in the body
			if matched, _ := regexp.Match("id=\""+matchedUrl+"\"", []byte(bodyText)); matched {
				matchedUrl = toTest.Path + matchedUrl
				// Store directly in resultsChan
				w.resultsChan <- cache.CacheEntry{
					URL:    matchedUrl,
					Status: cache.Live,
					Error:  "",
				}
				return
			}
		}
		w.toTestChan <- WalkerRequest{
			Path:     matchedUrl,
			BasePath: toTest.BasePath,
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
		// Check for a dot followed by at least one character (file extension)
		if idx := strings.LastIndex(last, "."); idx != -1 && idx < len(last)-1 {
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
