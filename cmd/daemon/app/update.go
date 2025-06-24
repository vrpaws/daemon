package app

import (
	"io"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/pkg"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.login.Init(), m.logger.Init(), m.tabs.Init(), m.settings.Init(), m.uploader.Init(), tea.SetWindowTitle("VRPaws Client"))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case message.Pause:
		m.paused = bool(msg)
		if m.setPause != nil {
			m.setPause(m.paused)
		}
		return m.propagate(msg, &m.logger, &m.tabs, &m.uploader)
	case message.SetPause:
		m.setPause = msg
		m.setPause(m.paused)
		return m, message.Cmd(logger.NewAutoDelete(logger.NewMessageTime("Tray icon ready!"), 5*time.Second))
	case message.SetUsername:
		m.setUsername = msg
		if m.me != nil {
			m.setUsername(m.me.User.Username)
		}
		return m, nil
	case tea.Cmd:
		return m, msg
	case io.Writer:
		m.logFile = msg
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if m.window.Width != msg.Width || m.window.Height != msg.Height {
			m.window.Width = msg.Width
			m.window.Height = msg.Height
			if m.window.Width == 0 && m.window.Height == 0 {
				go lib.HideConsole()
			} else {
				go lib.ShowConsole()
			}
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
	case api.UploadPayload, *fsnotify.Event:
		return m.propagate(msg, &m.uploader)
	case logger.Renderable:
		return m.propagate(msg, &m.logger)
	case *vrpaws.Me:
		m.me = msg
		if m.setUsername != nil {
			m.setUsername(m.me.User.Username)
		}
		return m.propagate(msg)
	case []error:
		return m, tea.Sequence(message.Cmds(msg...)...)
	}

	return m.propagate(msg)
}

func (m *Model) Write(p []byte) (n int, err error) {
	m.logger, _ = m.logger.Update(logger.Message(p))
	return len(p), nil
}

func (m *Model) propagate(msg tea.Msg, models ...*tea.Model) (tea.Model, tea.Cmd) {
	if len(models) == 0 {
		models = []*tea.Model{&m.logger, &m.tabs, &m.settings, &m.footer, &m.uploader, &m.login}
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
