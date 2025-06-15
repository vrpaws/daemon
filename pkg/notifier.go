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
	work     func(*fsnotify.Event)
	cooldown time.Duration

	context *contextWithWatcher
	mu      sync.Mutex
	timers  map[string]*time.Timer
	watcher *fsnotify.Watcher
}

func NewWatcher(paths []string, debounce time.Duration, work func(*fsnotify.Event)) *Watcher {
	return &Watcher{paths: paths, cooldown: debounce, work: work, timers: make(map[string]*time.Timer)}
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

func (w *Watcher) SetWork(work func(*fsnotify.Event)) {
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
				w.scheduleDebouncedWork(&event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	return nil
}

func (w *Watcher) Paths() []string {
	return w.paths
}

func (w *Watcher) scheduleDebouncedWork(event *fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cooldown <= 0 {
		w.work(event)
		return
	}

	// if there's already a timer pending, stop it
	if t, ok := w.timers[event.Name]; ok {
		t.Stop()
	}

	// start a new one
	w.timers[event.Name] = time.AfterFunc(w.cooldown, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.work(event)
		delete(w.timers, event.Name)
	})
}
