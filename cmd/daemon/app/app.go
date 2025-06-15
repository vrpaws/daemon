package app

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"vrc-moments/cmd/daemon/components/footer"
	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/settings"
	"vrc-moments/cmd/daemon/components/tabs"
	"vrc-moments/cmd/daemon/components/upload"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
)

type Model struct {
	window screen

	config  *settings.Config
	watcher *lib.Watcher

	ctx    context.Context
	server api.Server

	tabs     tea.Model
	logger   tea.Model
	uploader tea.Model
	settings tea.Model
	footer   tea.Model
}

type screen struct {
	Width  int
	Height int
}

func NewModel(u *url.URL, config *settings.Config) Model {
	ctx := context.Background()
	server := vrpaws.NewVRPaws(u, ctx, config.Token)

	patterns, err := lib.ExpandPatterns(true, false, config.Path)
	if err != nil || len(patterns) == 0 {
		log.Fatalf("failed to expand patterns for %s: %v", config.Path, err)
	}

	watcher := lib.NewWatcher(
		patterns,
		10*time.Second,
		nil,
	)

	model := Model{
		config:  config,
		watcher: watcher,
		server:  server,
		tabs: tabs.New([]string{
			"Logger",
			"Upload",
			"Settings",
		}, config.Username),
		logger:   logger.NewLogger(),
		uploader: upload.NewModel(watcher, ctx, config, server),
		settings: settings.New(config, server),
		footer: footer.New([]*string{
			&config.LastWorld,
			&config.Server,
		}),
	}

	return model
}

// Run runs the program but does not block.
func (m *Model) Run() *tea.Program {
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithoutCatchPanics())
	go Run(p)
	return p
}

func Run(program *tea.Program) {
	zone.NewGlobal()
	defer zone.Close()
	if _, err := program.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
