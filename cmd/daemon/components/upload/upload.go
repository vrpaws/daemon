package upload

import (
	tea "github.com/charmbracelet/bubbletea"

	lib "vrc-moments/pkg"
)

type Uploader struct {
	watcher *lib.Watcher
}

func NewModel(watcher *lib.Watcher) *Uploader {
	return &Uploader{
		watcher: watcher,
	}
}

func (m *Uploader) Init() tea.Cmd {
	return nil
}

func (m *Uploader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg.(type) {
	default:
	}

	return m, nil
}

func (m *Uploader) View() string {
	return "empty"
}

func Work() {

}
