package app

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"vrc-moments/cmd/daemon/components/footer"
	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/login"
	"vrc-moments/cmd/daemon/components/settings"
	"vrc-moments/cmd/daemon/components/tabs"
	"vrc-moments/cmd/daemon/components/upload"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api/vrpaws"
)

type Model struct {
	window screen

	config  *settings.Config
	watcher *lib.Watcher

	ctx    context.Context
	server *vrpaws.Server
	me     *vrpaws.Me

	login    tea.Model
	tabs     tea.Model
	logger   tea.Model
	uploader tea.Model
	settings tea.Model
	footer   tea.Model
	logFile  io.Writer

	setPause func(bool)
	paused   bool

	setUsername func(string)
}

type screen struct {
	Width  int
	Height int
}

func NewModel(u *url.URL, config *settings.Config) Model {
	ctx := context.Background()
	server := vrpaws.NewVRPaws(u, ctx)

	model := Model{
		config: config,
		server: server,
		login:  login.New(config, server),
		tabs: tabs.New([]string{
			"Logger",
			"Upload",
			"Settings",
		}, config.Username),
		logger:   logger.NewLogger(),
		uploader: upload.NewModel(ctx, config, server),
		settings: settings.New(config, server),
		footer: footer.New([]*string{
			&config.LastWorld,
			&config.Server,
		}),
		logFile: io.Discard,
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
