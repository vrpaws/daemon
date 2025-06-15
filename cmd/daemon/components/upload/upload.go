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
	"vrc-moments/cmd/daemon/components/settings"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
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

	server api.Server
}

func NewModel(watcher *lib.Watcher, ctx context.Context, config *settings.Config, server api.Server) *Uploader {
	uploader := &Uploader{
		watcher: watcher,
		ctx:     ctx,
		config:  config,
		server:  server,
	}
	uploader.watcher.SetWork(uploader.receive)
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
		err := m.watcher.Watch()
		if err != nil {
			log.Printf("failed to start watcher: %v", err)
			panic(err)
		}
		paths := m.watcher.Paths()
		log.Printf("Watcher started with %d directories: %s...", len(paths), paths[0])
		m.queue.Work()
		return m, nil
	default:
	}

	return m, nil
}

func (m *Uploader) View() string {
	return "empty"
}

// external callback to notify the *tea.Program
func (m *Uploader) receive(event *fsnotify.Event) {
	if m.program == nil {
		log.Println("Cannot receive: program not yet initialized")
		return
	}

	m.program.Send(event)
}

// async function to prepare and call relevant [lib.Watcher.AddPath] or upload
func (m *Uploader) async(event *fsnotify.Event) func() tea.Msg {
	return func() tea.Msg {
		if m.program == nil {
			return errors.New("upload: program not yet initialized")
		}

		if fi, err := os.Stat(event.Name); err == nil {
			if fi.IsDir() {
				err := m.watcher.AddPath(event.Name)
				if err != nil {
					return fmt.Errorf("could not add new directory to watcher: %w", err)
				} else {
					return nil
				}
			}
		}

		// TODO: try to match with settings.Config.Path
		if !strings.HasSuffix(event.Name, ".png") {
			return nil
		}

		dir, file := filepath.Split(event.Name)
		folder := filepath.Base(dir)
		m.program.Send(logger.NewMessageTimef("A new photo was taken at %s", filepath.Join(folder, file)))
		return <-m.queue.Promise(event)
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
		})
		return payload.File.SHA256, nil
	}
}
