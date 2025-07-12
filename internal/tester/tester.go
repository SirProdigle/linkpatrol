package tester

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	"github.com/sirprodigle/linkpatrol/internal/logger"
	"github.com/sirprodigle/linkpatrol/internal/walker"
)

type Tester struct {
	logger     *logger.Logger
	cache      *cache.Cache
	results    <-chan walker.WalkerResult
	workerPool DomainLimiterProvider
}

type DomainLimiterProvider interface {
	GetDomainLimiter(domain string) *rate.Limiter
}

func NewTester(cache *cache.Cache, results <-chan walker.WalkerResult, workerPool DomainLimiterProvider, verbose bool) *Tester {
	return &Tester{
		logger:     logger.New(verbose),
		cache:      cache,
		results:    results,
		workerPool: workerPool,
	}
}

func (t *Tester) Test(ctx context.Context, result walker.WalkerResult) {
	// Check if the url is in the cache first
	if cachedEntry := t.cache.Get(result.Path); cachedEntry != nil {
		t.logger.Debug("🟡 Cache hit for %s (status: %v)", result.Path, cachedEntry.Status)
		return
	}

	// Check if already being tested by another worker
	if t.cache.IsBeingTested(result.Path) {
		t.logger.Debug("🟠 Already being tested: %s", result.Path)
		return
	}

	// Mark as being tested
	if !t.cache.StartTesting(result.Path) {
		t.logger.Debug("🟠 Race condition avoided for: %s", result.Path)
		return
	}

	// Ensure we clean up the testing flag
	defer t.cache.FinishTesting(result.Path)

	t.logger.Debug("🟦 Testing %s", result.Path)

	switch result.Type {
	case walker.PathTypeUrl:
		// Check if the url is valid
		if _, err := url.Parse(result.Path); err != nil {
			t.cache.Add(result.Path, cache.Dead, err.Error())
			t.logger.Debug("❌ %s -> DEAD (invalid URL: %v)", result.Path, err)
			return
		}
		// Check if the URL is live
		if err := t.PingUrl(ctx, result.Path); err != nil {
			// check if http timeout error
			if isTimeout, err := isTimeoutError(err); isTimeout {
				t.cache.Add(result.Path, cache.Timeout, err.Error())
				t.logger.Debug("⏰ %s -> TIMEOUT (%v)", result.Path, err)
				return
			}
			t.cache.Add(result.Path, cache.Dead, err.Error())
			t.logger.Debug("❌ %s -> DEAD (%v)", result.Path, err)
			return
		}
		t.cache.Add(result.Path, cache.Live)
		t.logger.Debug("✅ %s -> LIVE", result.Path)

	case walker.PathTypeFile:
		if err := t.TestFile(ctx, path.Join(result.BasePath, result.Path)); err != nil {
			t.cache.Add(result.Path, cache.Dead, err.Error())
			t.logger.Debug("❌ %s -> DEAD (file: %v)", result.Path, err)
			return
		}
		t.cache.Add(result.Path, cache.Live)
		t.logger.Debug("✅ %s -> LIVE (file)", result.Path)

	case walker.PathTypeEmail:
		// Extract email from mailto: prefix
		email := strings.TrimPrefix(result.Path, "mailto:")
		if err := t.TestEmail(ctx, email); err != nil {
			t.cache.Add(result.Path, cache.Dead, err.Error())
			t.logger.Debug("❌ %s -> DEAD (email: %v)", result.Path, err)
			return
		}
		t.cache.Add(result.Path, cache.Live)
		t.logger.Debug("✅ %s -> LIVE (email)", result.Path)

	case walker.PathTypeTel:
		// Telephone links are always considered valid (they're handled by the device)
		t.cache.Add(result.Path, cache.Live)
		t.logger.Debug("✅ %s -> LIVE (tel)", result.Path)

	case walker.PathTypeAnchor:
		// Anchor links are always considered valid (they're internal to the page)
		t.cache.Add(result.Path, cache.Live)
		t.logger.Debug("✅ %s -> LIVE (anchor)", result.Path)

	case walker.PathTypeRoot:
		// Root path is always considered valid (it's the home page)
		t.cache.Add(result.Path, cache.Live)
		t.logger.Debug("✅ %s -> LIVE (root)", result.Path)

	case walker.PathTypeFtp:
		// FTP URLs are not currently tested
		t.cache.Add(result.Path, cache.Dead, "FTP testing not implemented")
		t.logger.Debug("❌ %s -> DEAD (FTP not implemented)", result.Path)

	case walker.PathTypeGit:
		// Git URLs are not currently tested
		t.cache.Add(result.Path, cache.Dead, "Git testing not implemented")
		t.logger.Debug("❌ %s -> DEAD (Git not implemented)", result.Path)

	case walker.PathTypeRelativeFile:
		// Try as a local file first
		if err := t.TestFile(ctx, path.Join(result.BasePath, result.Path)); err == nil {
			t.cache.Add(result.Path, cache.Live)
			t.logger.Debug("✅ %s -> LIVE (relative file)", result.Path)
			return
		}
		// If the file doesn't exist, mark it as dead
		t.cache.Add(result.Path, cache.Dead, "File not found")
		t.logger.Debug("❌ %s -> DEAD (relative file not found)", result.Path)

	case walker.PathTypeRelativeUrl:
		// Try as a URL
		if _, err := url.Parse(path.Join(result.BasePath, result.Path)); err == nil {
			if err := t.PingUrl(ctx, path.Join(result.BasePath, result.Path)); err != nil {
				t.cache.Add(result.Path, cache.Dead, err.Error())
				t.logger.Debug("❌ %s -> DEAD (relative URL: %v)", result.Path, err)
				return
			}
			t.cache.Add(result.Path, cache.Live)
			t.logger.Debug("✅ %s -> LIVE (relative URL)", result.Path)
			return
		}
		t.cache.Add(result.Path, cache.Dead, "Invalid relative URL")
		t.logger.Debug("❌ %s -> DEAD (invalid relative URL)", result.Path)

	case walker.PathTypeUnknown:
		t.cache.Add(result.Path, cache.Dead, "Unknown path type")
		t.logger.Debug("❌ %s -> DEAD (unknown type)", result.Path)

	default:
		t.cache.Add(result.Path, cache.Dead, "Unhandled path type")
		t.logger.Debug("❌ %s -> DEAD (unhandled type)", result.Path)
	}
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
	t.logger.Progress("Waiting for rate limit permit for domain: %s", u.Host)
	if err := domainLimiter.Wait(ctx); err != nil {
		return err
	}
	t.logger.Debug("Got rate limit permit for domain: %s", u.Host)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return err
	}

	// Fake a real browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
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

func (t *Tester) TestFile(ctx context.Context, path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
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

// isTimeoutError checks if the error is a timeout error
func isTimeoutError(err error) (bool, error) {
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Timeout(), urlErr.Err
	}
	return false, nil
}
