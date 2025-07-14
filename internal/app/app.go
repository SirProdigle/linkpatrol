package app

import (
	"context"
	"fmt"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	"github.com/sirprodigle/linkpatrol/internal/config"
	"github.com/sirprodigle/linkpatrol/internal/logger"
	"github.com/sirprodigle/linkpatrol/internal/walker"
	"github.com/sirprodigle/linkpatrol/internal/workers"
)

type App struct {
	config     *config.Config
	cache      *cache.ResultsCache
	workerPool *workers.WorkerPool
	logger     *logger.Logger
}

func New(cfg *config.Config) *App {
	var loggerOpts []logger.Option
	if cfg.TermWidth > 0 {
		loggerOpts = append(loggerOpts, logger.WithTerminalWidth(cfg.TermWidth))
	}
	if cfg.NoTruncate {
		loggerOpts = append(loggerOpts, logger.WithNoTruncate(cfg.NoTruncate))
	}

	// Generate results Channel early so that the worker pool and cache can use it
	resultsChan := make(chan cache.CacheEntry, 100)
	toWalkChan := make(chan walker.WalkerRequest, 100)
	toTestChan := make(chan walker.WalkerRequest, 100)
	log := logger.New(cfg.Verbose, loggerOpts...)
	cacheInstance := cache.NewResultsCache(resultsChan)
	workerPool := workers.NewWorkerPool(
		cacheInstance,
		cfg.Concurrency,
		cfg.Timeout,
		cfg.Rate,
		resultsChan,
		toWalkChan,
		toTestChan,
		log,
		cfg.Target,
	)

	return &App{
		config:     cfg,
		cache:      cacheInstance,
		workerPool: workerPool,
		logger:     log,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.StartSection("LinkPatrol Starting")
	a.logger.Config(a.config.Target, false, a.config.Concurrency, a.config.Timeout, a.config.Rate)

	// Start worker pool
	a.logger.Debug("Starting worker pool with %d crawlers and testers", a.config.Concurrency)
	a.workerPool.Start(ctx)
	a.cache.DoLoop()

	// Get target URL from config
	if a.config.Target == "" {
		a.logger.Error("No target URL specified. Provide URL as first argument or use --target flag.")
		return fmt.Errorf("no target URL specified")
	}

	// Send initial URL to walker
	a.logger.StartSection("Testing Links")
	a.workerPool.SendURLs(ctx, a.config.Target)

	return a.runNormalMode()
}

func (a *App) runNormalMode() error {
	a.workerPool.WaitAndClose()
	a.logger.StartSection("Results")
	a.cache.PrettyPrint(a.logger)

	// Check for failures and exit with appropriate code
	if a.cache.HasFailures() {
		deadCount, timeoutCount := a.cache.GetFailureCount()
		a.logger.TestResults(deadCount, timeoutCount)
		return fmt.Errorf("link check failed: found %d dead and %d timeout links", deadCount, timeoutCount)
	}

	a.logger.TestResults(0, 0)
	return nil
}
