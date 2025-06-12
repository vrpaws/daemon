package lib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type contextWithWatcher struct {
	ctx  context.Context
	done func()
}

type Watcher struct {
	Path     string
	Ticker   *time.Ticker // should not be used after
	Work     func()
	Debounce time.Duration

	context  *contextWithWatcher
	mu       sync.Mutex
	debounce *time.Timer
}

func (w *Watcher) Stop() error {
	if w.context == nil {
		return errors.New("watcher is nil")
	}
	w.context.done()
	return nil
}

func (w *Watcher) Watch() error {
	if w.Work == nil {
		return errors.New("watcher: no work function")
	}

	if w.Path == "" {
		return errors.New("watcher: no path")
	}

	var ticker time.Ticker
	if w.Ticker != nil {
		ticker = *w.Ticker
	} else {
		ticker = *time.NewTicker(30 * time.Second)
	}

	if w.Debounce == 0 {
		w.Debounce = 5 * time.Second
	}

	if w.context != nil {
		w.context.done()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	err = watcher.Add(w.Path)
	if err != nil {
		return fmt.Errorf("failed to add watcher: %w", err)
	}

	ctx, done := context.WithCancel(context.Background())
	w.context = &contextWithWatcher{ctx, done}
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Write) {
					continue
				}
				w.scheduleDebouncedWork()
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			case <-ticker.C:
				w.mu.Lock()
				w.Work()
				w.mu.Unlock()
			}
		}
	}()

	return nil
}

func (w *Watcher) scheduleDebouncedWork() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Debounce <= 0 {
		w.Work()
		return
	}

	// if there's already a timer pending, stop it
	if w.debounce != nil {
		w.debounce.Stop()
	}

	// start a new one
	w.debounce = time.AfterFunc(w.Debounce, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.Work()
	})
}
