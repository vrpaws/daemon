package lib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type contextWithWatcher struct {
	ctx  context.Context
	done func()
}

type Watcher struct {
	paths    []string
	ticker   *time.Ticker // should not be used after
	work     func()
	cooldown time.Duration

	context  *contextWithWatcher
	mu       sync.Mutex
	debounce *time.Timer
	watcher  *fsnotify.Watcher
}

func NewWatcher(paths []string, ticker *time.Ticker, debounce time.Duration, work func()) *Watcher {
	return &Watcher{paths: paths, ticker: ticker, cooldown: debounce, work: work}
}

func (w *Watcher) SetPaths(paths []string) error {
	if w.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create watcher: %w", err)
		}
		w.watcher = watcher
	} else {
		for _, path := range w.watcher.WatchList() {
			if !slices.Contains(paths, path) {
				err := w.watcher.Remove(path)
				if err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
			}
		}
	}

	for _, path := range paths {
		err := w.watcher.Add(path)
		if err != nil {
			return fmt.Errorf("failed to add %s: %w", path, err)
		}
	}

	return nil
}

func (w *Watcher) AddPath(path string) error {
	if w.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create watcher: %w", err)
		}
		w.watcher = watcher
	}

	w.paths = append(w.paths, path)
	err := w.watcher.Add(path)
	if err != nil {
		return fmt.Errorf("failed to add watcher: %w", err)
	}

	return nil
}

func (w *Watcher) SetWork(work func()) {
	w.mu.Lock()
	w.work = work
	w.mu.Unlock()
}

func (w *Watcher) Stop() error {
	if w.context == nil {
		return errors.New("watcher is nil")
	}
	w.context.done()
	return nil
}

func (w *Watcher) Watch() error {
	if w.work == nil {
		return errors.New("watcher: no work function")
	}

	if len(w.paths) == 0 {
		return errors.New("watcher: no path")
	}

	var ticker time.Ticker
	if w.ticker != nil {
		ticker = *w.ticker
	} else {
		ticker = *time.NewTicker(30 * time.Second)
	}

	if w.cooldown == 0 {
		w.cooldown = 5 * time.Second
	}

	if w.context != nil {
		w.context.done()
	}

	if w.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create watcher: %w", err)
		}
		w.watcher = watcher
	}

	for _, path := range w.paths {
		err := w.watcher.Add(path)
		if err != nil {
			return fmt.Errorf("failed to add %s: %w", path, err)
		}
	}

	ctx, done := context.WithCancel(context.Background())
	w.context = &contextWithWatcher{ctx, done}
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Write) {
					continue
				}
				w.scheduleDebouncedWork()
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			case <-ticker.C:
				w.mu.Lock()
				w.work()
				w.mu.Unlock()
			}
		}
	}()

	return nil
}

func (w *Watcher) scheduleDebouncedWork() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cooldown <= 0 {
		w.work()
		return
	}

	// if there's already a timer pending, stop it
	if w.debounce != nil {
		w.debounce.Stop()
	}

	// start a new one
	w.debounce = time.AfterFunc(w.cooldown, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.work()
	})
}
