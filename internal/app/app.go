package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirprodigle/linkpatrol/internal/cache"
	"github.com/sirprodigle/linkpatrol/internal/config"
	"github.com/sirprodigle/linkpatrol/internal/logger"
	"github.com/sirprodigle/linkpatrol/internal/scanner"
	"github.com/sirprodigle/linkpatrol/internal/watcher"
	"github.com/sirprodigle/linkpatrol/internal/workers"
)

type App struct {
	config     *config.Config
	cache      *cache.Cache
	workerPool *workers.WorkerPool
	watcher    *watcher.Watcher
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

	log := logger.New(cfg.Verbose, loggerOpts...)
	cacheInstance := cache.NewCache()
	workerPool := workers.NewWorkerPool(cacheInstance, cfg.Concurrency, cfg.TesterConcurrency, cfg.Timeout, cfg.Rate, log)

	return &App{
		config:     cfg,
		cache:      cacheInstance,
		workerPool: workerPool,
		logger:     log,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.StartSection("LinkPatrol Starting")
	a.logger.Config(a.config.Dir, a.config.Watch, a.config.Concurrency, a.config.TesterConcurrency, a.config.Timeout, a.config.Rate)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Start worker pool
	a.logger.Debug("Starting worker pool with %d walkers and %d testers", a.config.Concurrency, a.config.TesterConcurrency)
	a.workerPool.Start(ctx)

	// Scan directory for initial files
	a.logger.StartSection("Scanning Files")
	markdownFiles, htmlFiles, err := scanner.ScanDirectory(a.config.Dir)
	if err != nil {
		a.logger.Error("Failed to scan directory: %v", err)
		return err
	}

	a.logger.FilesFound(len(markdownFiles), len(htmlFiles))

	// Log individual files in verbose mode
	for _, file := range markdownFiles {
		a.logger.FileWalk("markdown file", file)
	}
	for _, file := range htmlFiles {
		a.logger.FileWalk("HTML file", file)
	}

	// Send initial files to workers
	a.logger.StartSection("Testing Links")
	a.workerPool.SendFiles(ctx, markdownFiles, htmlFiles)

	if a.config.Watch {
		return a.runWatchMode(ctx, cancel, sigChan)
	}

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

func (a *App) runWatchMode(ctx context.Context, cancel context.CancelFunc, sigChan chan os.Signal) error {
	// Set up filesystem watcher
	a.logger.StartSection("Watch Mode")
	workerFileChannel := a.workerPool.GetFileChannel()
	var err error
	a.watcher, err = watcher.New(workerFileChannel, a.logger)
	if err != nil {
		a.logger.Error("Failed to create watcher: %v", err)
		return err
	}
	defer a.watcher.Close()

	if err := a.watcher.AddDirectory(a.config.Dir); err != nil {
		a.logger.Error("Failed to add directory to watcher: %v", err)
		return fmt.Errorf("failed to add directories to watcher: %w", err)
	}

	a.logger.Watch(a.config.Dir)

	// Start filesystem watcher
	a.watcher.Start(ctx)

	// Monitor work completion and display results
	go a.monitorResults(ctx)

	// Wait for shutdown signal
	<-sigChan
	a.logger.Shutdown()
	cancel()

	// Wait for workers to finish
	a.workerPool.WaitAndClose()
	a.logger.Success("All workers stopped")

	return nil
}

func (a *App) monitorResults(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var lastDisplayedWorkCount int32

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if a.workerPool.IsIdle() {
				workCompleted := a.workerPool.GetWorkCompleted()
				if workCompleted > lastDisplayedWorkCount {
					a.logger.StartSection(fmt.Sprintf("Results (work completed: %d)", workCompleted))
					a.cache.PrettyPrint(a.logger)
					a.logger.Waiting()
					lastDisplayedWorkCount = workCompleted
				}
			}
		}
	}
}
