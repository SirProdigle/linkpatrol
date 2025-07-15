package workers

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	. "github.com/sirprodigle/linkpatrol/internal/logger"
	. "github.com/sirprodigle/linkpatrol/internal/tester"
	"github.com/sirprodigle/linkpatrol/internal/walker"
)

type WorkerPool struct {
	logger         *Logger
	resultsCache   *cache.ResultsCache
	concurrency    int
	rateLimitValue int
	domainLimiters map[string]*domainLimiter
	limiterMutex   sync.RWMutex
	resultsChan    chan<- cache.CacheEntry
	toTestChan     chan walker.WalkerRequest
	toWalkChan     chan walker.WalkerRequest
	timeout        time.Duration
	client         *http.Client
	baseUrl        string

	activeWalkers atomic.Int32
	activeTesters atomic.Int32

	defaultRateLimiter *rate.Limiter
}

type domainLimiter struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

func NewWorkerPool(cache *cache.ResultsCache, concurrency int, timeout time.Duration, rateLimit int, resultsChan chan<- cache.CacheEntry, toWalkChan chan walker.WalkerRequest, toTestChan chan walker.WalkerRequest, log *Logger, baseUrl string) *WorkerPool {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        2000,
			MaxIdleConnsPerHost: 1000,
			MaxConnsPerHost:     1000,
			IdleConnTimeout:     120 * time.Second,
			ForceAttemptHTTP2:   true,

			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				CurvePreferences: []tls.CurveID{
					tls.X25519,
					tls.CurveP256,
				},
				ClientSessionCache: tls.NewLRUClientSessionCache(1025),
			},

			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,

			DisableCompression: true,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						d := net.Dialer{
							Timeout: time.Millisecond * 200,
						}
						return d.DialContext(ctx, network, "1.1.1.1:53")
					},
				},
			}).DialContext,
		},
	}
	return &WorkerPool{
		logger:             log,
		resultsCache:       cache,
		concurrency:        concurrency,
		timeout:            timeout,
		rateLimitValue:     rateLimit,
		domainLimiters:     make(map[string]*domainLimiter, 100),
		resultsChan:        resultsChan,
		client:             client,
		baseUrl:            baseUrl,
		defaultRateLimiter: rate.NewLimiter(rate.Inf, 0),
		toWalkChan:         toWalkChan,
		toTestChan:         toTestChan,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	wp.startWalkers(ctx)
	wp.startTesters(ctx)
}

func (wp *WorkerPool) startWalkers(ctx context.Context) {
	for i := 0; i < wp.concurrency; i++ {
		walker := walker.NewWalker(wp.client, wp.resultsCache, wp.toWalkChan, wp.toTestChan, &wp.activeWalkers, wp.logger, wp.baseUrl, wp, wp.resultsChan)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case toTest, ok := <-wp.toWalkChan:
					if !ok {
						return
					}
					walker.Walk(ctx, toTest)
				}
			}
		}()
	}
}

func (wp *WorkerPool) startTesters(ctx context.Context) {

	for i := 0; i < wp.concurrency; i++ {
		go func(workerID int) {
			tester := NewTester(wp.resultsCache, wp.toTestChan, wp, wp.logger.IsVerbose(), &wp.activeTesters, wp.client, wp.resultsChan)
			for {
				select {
				case <-ctx.Done():
					return
				case toTest, ok := <-wp.toTestChan:
					if !ok {
						return
					}
					tester.Test(ctx, toTest)
				}
			}
		}(i)
	}
}

func (wp *WorkerPool) IsIdle() bool {
	walkers := wp.activeWalkers.Load()
	testers := wp.activeTesters.Load()
	queueEmpty := len(wp.toTestChan) == 0 && len(wp.toWalkChan) == 0 && len(wp.resultsChan) == 0
	return walkers == 0 && testers == 0 && queueEmpty
}

func (wp *WorkerPool) WaitAndClose() {
	for {
		if !wp.logger.IsVerbose() {
			wp.logger.PrettyPrintStats(wp)
		}

		if wp.IsIdle() {
			// Require 2 consecutive idle checks to close
			time.Sleep(100 * time.Millisecond)
			if wp.IsIdle() {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	close(wp.toTestChan)
	close(wp.toWalkChan)
	close(wp.resultsChan)
}

func (wp *WorkerPool) SendURLs(ctx context.Context, urls ...string) {
	for _, url := range urls {
		wp.logger.Debug("Sending url to walker: %s", url)
		wp.toWalkChan <- walker.WalkerRequest{
			Path:     url,
			BasePath: wp.baseUrl,
		}
	}
}

func (wp *WorkerPool) GetDomainLimiter(domain string) *rate.Limiter {
	if wp.rateLimitValue == 0 {
		return wp.defaultRateLimiter
	}
	wp.limiterMutex.RLock()
	domainLim, exists := wp.domainLimiters[domain]
	wp.limiterMutex.RUnlock()

	if !exists {
		wp.limiterMutex.Lock()
		limiter := rate.NewLimiter(rate.Limit(wp.rateLimitValue), 5)
		domainLim = &domainLimiter{
			limiter:  limiter,
			lastUsed: time.Now(),
		}
		wp.domainLimiters[domain] = domainLim
		wp.limiterMutex.Unlock()
		return limiter
	}

	// Update last used time
	wp.limiterMutex.Lock()
	domainLim.lastUsed = time.Now()
	wp.limiterMutex.Unlock()

	return domainLim.limiter
}

func (wp *WorkerPool) GetDomainCount() int {
	wp.limiterMutex.RLock()
	defer wp.limiterMutex.RUnlock()
	return len(wp.domainLimiters)
}
