package app

import (
	"log"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/pkg/vrc"
)

func (m *Model) Init() tea.Cmd {
	usernames, err := vrc.GetUsername(vrc.DefaultLogPath)
	if err != nil {
		m.config.Username = lipgloss.NewStyle().Foreground(lipgloss.Color("#EB4034")).Render(err.Error())
	} else if len(usernames) > 0 {
		m.config.Username = usernames[0]
	}

	roomName, err := vrc.ExtractCurrentRoomName(vrc.DefaultLogPath)
	if err != nil {
		log.Printf("Error extracting room name: %v", err)
	} else {
		m.config.SetRoom(roomName)
	}

	return tea.Batch(m.logger.Init(), m.tabs.Init(), m.settings.Init(), tea.SetWindowTitle("VRC Moments"))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if m.window.Width != msg.Width || m.window.Height != msg.Height {
			m.window.Width = msg.Width
			m.window.Height = msg.Height
			return m.propagate(msg)
		}
	case spinner.TickMsg:
		return m.propagate(msg, &m.logger, &m.tabs)
	case cursor.BlinkMsg:
		return m.propagate(msg, &m.settings)
	case message.RoomSet:
		return m.propagate(msg, &m.settings)
	case message.UsernameSet:
		return m.propagate(msg, &m.settings, &m.tabs)
	}

	return m.propagate(msg)
}

func (m *Model) Write(p []byte) (n int, err error) {
	m.logger, _ = m.logger.Update(logger.Message{
		Message: string(p),
	})
	return len(p), nil
}

func (m *Model) propagate(msg tea.Msg, models ...*tea.Model) (tea.Model, tea.Cmd) {
	if len(models) == 0 {
		models = []*tea.Model{&m.logger, &m.tabs, &m.settings, &m.footer}
	}

	cmds := make([]tea.Cmd, 0, len(models))

	for _, model := range models {
		var cmd tea.Cmd
		*model, cmd = (*model).Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
