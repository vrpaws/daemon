package upload

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
)

type Uploader struct {
	watcher *lib.Watcher
	program *tea.Program

	server api.Server
}

func NewModel(watcher *lib.Watcher, server api.Server) *Uploader {
	uploader := &Uploader{
		watcher: watcher,
		server:  server,
	}
	uploader.watcher.SetWork(uploader.work)
	return uploader
}

func (m *Uploader) Init() tea.Cmd {
	return nil
}

func (m *Uploader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case *tea.Program:
		m.program = msg
		return m, nil
	case api.UploadPayload:
		return m, m.upload(msg)
	default:
	}

	return m, nil
}

func (m *Uploader) View() string {
	return "empty"
}

func (m *Uploader) work(event *fsnotify.Event) {

}

func (m *Uploader) upload(payload api.UploadPayload) func() tea.Msg {
	return func() tea.Msg {
		if m.program == nil {
			return nil
		}

		return m.server.Upload(context.Background(), payload)
	}
}
