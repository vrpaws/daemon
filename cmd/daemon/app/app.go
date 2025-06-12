package app

import (
	"fmt"
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
)

type Model struct {
	config   *settings.Config
	window   screen
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

func NewModel(path string) Model {
	config := &settings.Config{
		Username:  "Unknown",
		Path:      "~/Pictures/VRChat",
		Server:    "Unset",
		LastWorld: "Unknown",
	}

	model := Model{
		config: config,
		tabs: tabs.New([]string{
			"Logger",
			"Upload",
			"Settings",
		}, config.Username),
		logger: logger.NewLogger(),
		uploader: upload.NewModel(lib.NewWatcher(
			config.Path,
			time.NewTicker(30*time.Second),
			5*time.Second,
			nil,
		)),
		settings: settings.New(config),
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
