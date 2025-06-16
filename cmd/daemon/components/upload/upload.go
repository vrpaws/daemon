package upload

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
	"vrc-moments/pkg/flight"
	"vrc-moments/pkg/gradient"
	"vrc-moments/pkg/worker"
)

type Uploader struct {
	watcher *lib.Watcher
	ctx     context.Context
	config  *settings.Config
	program *tea.Program

	uploadFlight flight.Cache[*fsnotify.Event, string]
	queue        worker.Pool[*fsnotify.Event, error]

	server api.Server[*vrpaws.Me]
}

func NewModel(ctx context.Context, config *settings.Config, server api.Server[*vrpaws.Me]) *Uploader {
	uploader := &Uploader{
		ctx:    ctx,
		config: config,
		server: server,
	}
	uploader.uploadFlight = flight.NewCache(uploader.upload)
	uploader.queue = worker.NewPool(runtime.NumCPU(), func(event *fsnotify.Event) error {
		_, err := uploader.uploadFlight.Get(event)
		return err
	})

	return uploader
}

func (m *Uploader) Init() tea.Cmd {
	return nil
}

func (m *Uploader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *fsnotify.Event:
		return m, m.async(msg)
	case *tea.Program:
		m.program = msg
		return m, nil
	case *lib.Watcher:
		m.watcher = msg
		err := m.watcher.Watch()
		if err != nil {
			log.Printf("failed to start watcher: %v", err)
			panic(err)
		}
		paths := m.watcher.Paths()
		log.Printf("Watcher started with %d directories:", len(paths))
		for i, path := range paths {
			log.Printf("%s", path)
			if i == 10 {
				log.Printf("and %d more...", len(paths)-i)
				break
			}
		}
		m.queue.Work()
		return m, nil
	default:
		return m, nil
	}
}

func (m *Uploader) View() string {
	return "empty"
}

// async function to prepare and call relevant [lib.Watcher.AddPath] or upload
func (m *Uploader) async(event *fsnotify.Event) func() tea.Msg {
	return func() tea.Msg {
		if m.watcher == nil {
			return errors.New("upload: program not yet initialized")
		}

		if fi, err := os.Stat(event.Name); err == nil {
			if fi.IsDir() {
				if event.Op.Has(fsnotify.Remove) {
					err := m.watcher.RemovePath(event.Name)
					if err != nil {
						return fmt.Errorf("could not remove %s to watcher: %w", event.Name, err)
					} else {
						return nil
					}
				} else {
					err := m.watcher.AddPath(event.Name)
					if err != nil {
						return fmt.Errorf("could not add %s to watcher: %w", event.Name, err)
					} else {
						return nil
					}
				}
			}
		}

		// TODO: try to match with settings.Config.Path
		if !strings.HasSuffix(event.Name, ".png") {
			return nil
		}

		if event.Op.Has(fsnotify.Create) {
			dir, file := filepath.Split(event.Name)
			folder := filepath.Base(dir)
			return message.Cmd(logger.NewMessageTimef("A new photo was taken at %s", filepath.Join(folder, file)))
		}

		switch event.Op {
		case fsnotify.Create:
			dir, file := filepath.Split(event.Name)
			folder := filepath.Base(dir)
			return logger.NewMessageTimef("A new photo was taken at %s", filepath.Join(folder, file))
		case fsnotify.Write:
			return <-m.queue.Promise(event)
		default:
			return nil
		}
	}
}

// the actual upload function
func (m *Uploader) upload(event *fsnotify.Event) (string, error) {
	if m.program == nil {
		return "", errors.New("upload: program not yet initialized")
	}

	f, err := api.OpenFile(event.Name)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", event.Name, err)
	}

	payload := api.UploadPayload{
		Username: m.config.Username,
		UserID:   m.config.UserID,
		File:     f,
	}

	if f.Metadata != nil && f.Metadata.Author.ID != "" {
		m.config.UserID = f.Metadata.Author.ID
		err := m.config.Save()
		if err != nil {
			return "", fmt.Errorf("saving config: %w", err)
		}
	}

	m.program.Send(logger.NewMessageTimef("Trying to upload %s...", payload.File.Filename))
	err = m.server.Upload(m.ctx, payload)
	if err != nil {
		return "", fmt.Errorf("uploading %s: %w", payload.File.Filename, err)
	} else {
		m.program.Send(logger.Concat{
			Items: []logger.Renderable{
				logger.NewMessageTime("Successfully uploaded "),
				logger.NewGradientString(payload.File.Filename, time.Second, gradient.PastelColors...),
				logger.Message("!"),
			},
			Separator: "",
			Save:      true,
		})
		return payload.File.SHA256, nil
	}
}
