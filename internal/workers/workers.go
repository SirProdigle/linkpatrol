package workers

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	"github.com/sirprodigle/linkpatrol/internal/logger"
	"github.com/sirprodigle/linkpatrol/internal/scanner"
	"github.com/sirprodigle/linkpatrol/internal/tester"
	"github.com/sirprodigle/linkpatrol/internal/walker"
)

type WorkerPool struct {
	logger            *logger.Logger
	cache             *cache.Cache
	walkerConcurrency int
	testerConcurrency int
	timeout           time.Duration
	rateLimitValue    int
	domainLimiters    map[string]*domainLimiter
	limiterMutex      sync.RWMutex

	filesToWalk chan scanner.FileInfo
	results     chan walker.WalkerResult

	activeWalkers                 *int32
	activeTesters                 *int32
	workCompletedSinceLastResults *int32

	wgWalkers sync.WaitGroup
	wgTesters sync.WaitGroup
}

type domainLimiter struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

func NewWorkerPool(cache *cache.Cache, walkerConcurrency, testerConcurrency int, timeout time.Duration, rateLimit int, log *logger.Logger) *WorkerPool {
	return &WorkerPool{
		logger:                        log,
		cache:                         cache,
		walkerConcurrency:             walkerConcurrency,
		testerConcurrency:             testerConcurrency,
		timeout:                       timeout,
		rateLimitValue:                rateLimit,
		domainLimiters:                make(map[string]*domainLimiter),
		filesToWalk:                   make(chan scanner.FileInfo, 500),
		results:                       make(chan walker.WalkerResult, 500),
		activeWalkers:                 new(int32),
		activeTesters:                 new(int32),
		workCompletedSinceLastResults: new(int32),
	}
}

func (wp *WorkerPool) GetFileChannel() chan<- scanner.FileInfo {
	return wp.filesToWalk
}

func (wp *WorkerPool) Start(ctx context.Context) {
	wp.startWalkers(ctx)
	wp.startTesters(ctx)
}

func (wp *WorkerPool) startWalkers(ctx context.Context) {
	sendResults := (chan<- walker.WalkerResult)(wp.results)

	for i := 0; i < wp.walkerConcurrency; i++ {
		wp.wgWalkers.Add(1)
		go func() {
			defer wp.wgWalkers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case file, ok := <-wp.filesToWalk:
					if !ok {
						return
					}
					atomic.AddInt32(wp.activeWalkers, 1)
					switch file.FileType {
					case scanner.FileTypeMarkdown:
						wp.logger.Debug("Walking markdown file: %s", file.FilePath)
						walker := walker.NewMarkdownWalker(wp.cache, sendResults)
						walker.Walk(ctx, file.FilePath)
					case scanner.FileTypeHTML:
						wp.logger.Debug("Walking HTML file: %s", file.FilePath)
						walker := walker.NewHtmlWalker(wp.cache, wp.timeout, sendResults)
						walker.Walk(ctx, file.FilePath)
					}
					atomic.AddInt32(wp.activeWalkers, -1)
					atomic.AddInt32(wp.workCompletedSinceLastResults, 1)
				}
			}
		}()
	}
}

func (wp *WorkerPool) startTesters(ctx context.Context) {
	receiveResults := (<-chan walker.WalkerResult)(wp.results)

	for i := 0; i < wp.testerConcurrency; i++ {
		wp.wgTesters.Add(1)
		go func() {
			defer wp.wgTesters.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case result, ok := <-wp.results:
					if !ok {
						return
					}
					atomic.AddInt32(wp.activeTesters, 1)
					wp.logger.Trace("Testing result: %+v", result)
					tester := tester.NewTester(wp.cache, receiveResults, wp, wp.logger.IsVerbose())
					tester.Test(ctx, result)
					wp.logger.Trace("Finished testing result: %+v", result)
					atomic.AddInt32(wp.activeTesters, -1)
				}
			}
		}()
	}
}

func (wp *WorkerPool) SendFiles(ctx context.Context, markdownFiles, htmlFiles []string) {
	for _, filePath := range markdownFiles {
		select {
		case <-ctx.Done():
			return
		case wp.filesToWalk <- scanner.FileInfo{FilePath: filePath, FileType: scanner.FileTypeMarkdown}:
		}
	}
	for _, filePath := range htmlFiles {
		select {
		case <-ctx.Done():
			return
		case wp.filesToWalk <- scanner.FileInfo{FilePath: filePath, FileType: scanner.FileTypeHTML}:
		}
	}
}

func (wp *WorkerPool) IsIdle() bool {
	walkers := atomic.LoadInt32(wp.activeWalkers)
	testers := atomic.LoadInt32(wp.activeTesters)
	queueEmpty := len(wp.filesToWalk) == 0 && len(wp.results) == 0
	return walkers == 0 && testers == 0 && queueEmpty
}

func (wp *WorkerPool) GetWorkCompleted() int32 {
	return atomic.LoadInt32(wp.workCompletedSinceLastResults)
}

func (wp *WorkerPool) Close() {
	close(wp.filesToWalk)
	wp.wgWalkers.Wait()
	close(wp.results)
	wp.wgTesters.Wait()
}

func (wp *WorkerPool) Wait() {
	wp.wgWalkers.Wait()
	wp.wgTesters.Wait()
}

func (wp *WorkerPool) GetDomainLimiter(domain string) *rate.Limiter {
	wp.limiterMutex.RLock()
	domainLim, exists := wp.domainLimiters[domain]
	wp.limiterMutex.RUnlock()

	if !exists {
		wp.limiterMutex.Lock()
		// Double-check in case another goroutine created it
		if domainLim, exists = wp.domainLimiters[domain]; !exists {
			var limiter *rate.Limiter
			if wp.rateLimitValue <= 0 {
				// No rate limiting - use unlimited limiter
				limiter = rate.NewLimiter(rate.Inf, 1)
			} else {
				// Create rate limiter with specified requests per second
				limiter = rate.NewLimiter(rate.Limit(wp.rateLimitValue), 1)
			}
			domainLim = &domainLimiter{
				limiter:  limiter,
				lastUsed: time.Now(),
			}
			wp.domainLimiters[domain] = domainLim
			wp.logger.RateLimit(domain, wp.rateLimitValue, "Created rate limiter")
		}
		wp.limiterMutex.Unlock()
	}

	// Update last used time
	wp.limiterMutex.Lock()
	domainLim.lastUsed = time.Now()
	wp.limiterMutex.Unlock()

	return domainLim.limiter
}

func (wp *WorkerPool) CleanupInactiveLimiters(maxAge time.Duration) {
	wp.limiterMutex.Lock()
	defer wp.limiterMutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for domain, limiter := range wp.domainLimiters {
		if limiter.lastUsed.Before(cutoff) {
			delete(wp.domainLimiters, domain)
			wp.logger.Debug("Cleaned up inactive rate limiter for domain: %s", domain)
		}
	}
}

func (wp *WorkerPool) GetDomainCount() int {
	wp.limiterMutex.RLock()
	defer wp.limiterMutex.RUnlock()
	return len(wp.domainLimiters)
}

func (wp *WorkerPool) GetStats() WorkerPoolStats {
	return WorkerPoolStats{
		ActiveWalkers:   atomic.LoadInt32(wp.activeWalkers),
		ActiveTesters:   atomic.LoadInt32(wp.activeTesters),
		DomainCount:     int32(wp.GetDomainCount()),
		TotalGoroutines: int32(runtime.NumGoroutine()),
		FilesQueued:     int32(len(wp.filesToWalk)),
		ResultsQueued:   int32(len(wp.results)),
	}
}

type WorkerPoolStats struct {
	ActiveWalkers   int32
	ActiveTesters   int32
	DomainCount     int32
	TotalGoroutines int32
	FilesQueued     int32
	ResultsQueued   int32
}
