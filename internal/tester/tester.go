package tester

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"golang.org/x/time/rate"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	"github.com/sirprodigle/linkpatrol/internal/logger"
	"github.com/sirprodigle/linkpatrol/internal/walker"
)

type Tester struct {
	logger      *logger.Logger
	cache       *cache.ResultsCache
	toTestChan  <-chan walker.WalkerRequest
	resultsChan chan<- cache.CacheEntry
	workerPool  DomainLimiterProvider
	activeCount *atomic.Int32
	client      *http.Client
}

type DomainLimiterProvider interface {
	GetDomainLimiter(domain string) *rate.Limiter
}

func NewTester(cache *cache.ResultsCache, results <-chan walker.WalkerRequest, workerPool DomainLimiterProvider, verbose bool, activeCount *atomic.Int32, client *http.Client, resultsChan chan<- cache.CacheEntry) *Tester {
	return &Tester{
		logger:      logger.New(verbose),
		cache:       cache,
		toTestChan:  results,
		workerPool:  workerPool,
		activeCount: activeCount,
		client:      client,
		resultsChan: resultsChan,
	}
}

func (t *Tester) Test(ctx context.Context, requestData walker.WalkerRequest) {
	t.activeCount.Add(1)
	defer t.activeCount.Add(-1)

	// Check if the url is in the cache first
	if t.cache.HasResult(requestData.Path) {
		t.logger.Debug("🟡 Cache hit for %s (status: %v)", requestData.Path, t.cache.GetResult(requestData.Path).Status)
		return
	}

	// Handle fragment URLs (like #section) - check if they exist on the original page
	if strings.HasPrefix(requestData.Path, "#") {
		if requestData.BasePath != "" {
			t.checkFragmentOnPage(ctx, requestData.Path, requestData.BasePath)
		} else {
			t.resultsChan <- cache.CacheEntry{
				URL:    requestData.Path,
				Status: cache.Dead,
				Error:  "Fragment URL with no base page to check against",
			}
			t.logger.Debug("❌ %s -> DEAD (no base page)", requestData.Path)
		}
		return
	}

	// Resolve relative URLs using BasePath
	resolvedURL := requestData.Path
	if parsed, err := url.Parse(requestData.Path); err == nil && parsed.Host == "" && requestData.BasePath != "" {
		base, err := url.Parse(requestData.BasePath)
		if err == nil {
			resolvedURL = base.ResolveReference(parsed).String()
		}
	}

	// Add HTTPS default for URLs without protocol
	if parsed, err := url.Parse(resolvedURL); err == nil && parsed.Scheme == "" {
		resolvedURL = "https://" + resolvedURL
	}

	t.logger.Debug("🟦 Testing %s", resolvedURL)

	// Check if the url is valid
	if _, err := url.Parse(resolvedURL); err != nil {
		t.resultsChan <- cache.CacheEntry{
			URL:    resolvedURL,
			Status: cache.Dead,
			Error:  err.Error(),
		}
		t.logger.Debug("❌ %s -> DEAD (invalid URL: %v)", resolvedURL, err)
		return
	}
	// Check if the URL is live
	finalURL, err := t.PingUrlWithFallback(ctx, resolvedURL)
	if err != nil {
		// check if http timeout error
		if isTimeout, err := isTimeoutError(err); isTimeout {
			t.resultsChan <- cache.CacheEntry{
				URL:    finalURL,
				Status: cache.Timeout,
				Error:  err.Error(),
			}
			t.logger.Debug("⏰ %s -> TIMEOUT (%v)", finalURL, err)
			return
		}
		t.resultsChan <- cache.CacheEntry{
			URL:    finalURL,
			Status: cache.Dead,
			Error:  err.Error(),
		}
		t.logger.Debug("❌ %s -> DEAD (%v)", finalURL, err)
		return
	}
	t.resultsChan <- cache.CacheEntry{
		URL:    finalURL,
		Status: cache.Live,
		Error:  "",
	}
	t.logger.Debug("✅ %s -> LIVE", finalURL)

}

func (t *Tester) PingUrlWithFallback(ctx context.Context, path string) (string, error) {
	// First try the URL as-is (likely HTTPS)
	err := t.PingUrl(ctx, path)
	if err == nil {
		return path, nil
	}

	// If it's an HTTPS URL and failed, try HTTP fallback
	if parsed, parseErr := url.Parse(path); parseErr == nil && parsed.Scheme == "https" {
		httpURL := strings.Replace(path, "https://", "http://", 1)
		t.logger.Debug("🔄 HTTPS failed, trying HTTP fallback: %s", httpURL)

		httpErr := t.PingUrl(ctx, httpURL)
		if httpErr == nil {
			return httpURL, nil
		}

		// Return the original HTTPS error since HTTP also failed
		return path, err
	}

	// Not an HTTPS URL or some other issue, return original error
	return path, err
}

func (t *Tester) PingUrl(ctx context.Context, path string) error {
	// Extract domain for rate limiting
	u, err := url.Parse(path)
	if err != nil {
		return err
	}

	// Get domain-specific rate limiter
	domainLimiter := t.workerPool.GetDomainLimiter(u.Host)

	// Wait for rate limiter permit
	if !domainLimiter.Allow() {
		t.logger.Progress("Waiting for rate limit permit for domain: %s", u.Host)
		if err := domainLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return err
	}

	// Fake a real browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &url.Error{
			Op:  "GET",
			URL: path,
			Err: fmt.Errorf("HTTP %d", resp.StatusCode),
		}
	}

	return nil
}

func (t *Tester) TestEmail(ctx context.Context, path string) error {
	// Do MX lookup
	mx, err := net.LookupMX(path)
	if err != nil {
		return err
	}

	// If there are no MX records, return an error
	if len(mx) == 0 {
		return fmt.Errorf("no MX records found for %s", path)
	}

	return nil
}

// checkFragmentOnPage checks if a fragment (like #section) exists on the given page
func (t *Tester) checkFragmentOnPage(ctx context.Context, fragment, basePage string) {
	// Remove the # from fragment
	targetId := strings.TrimPrefix(fragment, "#")

	// If it's just "#", it's always valid (top of page)
	if targetId == "" {
		t.resultsChan <- cache.CacheEntry{
			URL:    fragment,
			Status: cache.Live,
			Error:  "",
		}
		t.logger.Debug("✅ %s -> LIVE (top of page)", fragment)
		return
	}

	// Fetch the page content
	resp, err := t.client.Get(basePage)
	if err != nil {
		t.resultsChan <- cache.CacheEntry{
			URL:    fragment,
			Status: cache.Dead,
			Error:  fmt.Sprintf("Could not fetch base page to check fragment: %v", err),
		}
		t.logger.Debug("❌ %s -> DEAD (could not fetch base page)", fragment)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.resultsChan <- cache.CacheEntry{
			URL:    fragment,
			Status: cache.Dead,
			Error:  fmt.Sprintf("Base page returned HTTP %d", resp.StatusCode),
		}
		t.logger.Debug("❌ %s -> DEAD (base page HTTP %d)", fragment, resp.StatusCode)
		return
	}

	// Read page content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.resultsChan <- cache.CacheEntry{
			URL:    fragment,
			Status: cache.Dead,
			Error:  fmt.Sprintf("Could not read base page: %v", err),
		}
		t.logger.Debug("❌ %s -> DEAD (could not read base page)", fragment)
		return
	}

	// Check if the target ID exists in the page
	pageContent := string(body)
	idPatterns := []string{
		fmt.Sprintf(`id="%s"`, targetId),
		fmt.Sprintf(`id='%s'`, targetId),
		fmt.Sprintf(`id=%s`, targetId),
	}

	found := false
	for _, pattern := range idPatterns {
		if strings.Contains(pageContent, pattern) {
			found = true
			break
		}
	}

	if found {
		t.resultsChan <- cache.CacheEntry{
			URL:    fragment,
			Status: cache.Live,
			Error:  "",
		}
		t.logger.Debug("✅ %s -> LIVE (element found)", fragment)
	} else {
		t.resultsChan <- cache.CacheEntry{
			URL:    fragment,
			Status: cache.Dead,
			Error:  fmt.Sprintf("Element with id='%s' not found on page", targetId),
		}
		t.logger.Debug("❌ %s -> DEAD (element not found)", fragment)
	}
}

// isTimeoutError checks if the error is a timeout error
func isTimeoutError(err error) (bool, error) {
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Timeout(), urlErr.Err
	}
	return false, nil
}
