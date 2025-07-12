package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/sirprodigle/linkpatrol/internal/logger"
	"github.com/sirprodigle/linkpatrol/internal/scanner"
)

type FileEvent = scanner.FileInfo

type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	fileChannel chan<- FileEvent
	logger      *logger.Logger
}

func New(fileChannel chan<- FileEvent, log *logger.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem watcher: %w", err)
	}

	return &Watcher{
		fsWatcher:   fsWatcher,
		fileChannel: fileChannel,
		logger:      log,
	}, nil
}

func (w *Watcher) AddDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return w.fsWatcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.fsWatcher.Events:
				if !ok {
					return
				}
				w.handleEvent(ctx, event)
			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				w.logger.WatchError(err)
			}
		}
	}()
}

func (w *Watcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	// Handle file changes
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		if scanner.IsRelevantFile(event.Name) {
			w.logger.FileChange(event.Name)
			
			fileType := scanner.GetFileType(event.Name)
			select {
			case <-ctx.Done():
				return
			case w.fileChannel <- scanner.FileInfo{
				FilePath: event.Name,
				FileType: fileType,
			}:
			}
		}
	}

	// Handle new directories
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			w.fsWatcher.Add(event.Name)
		}
	}
}

func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}