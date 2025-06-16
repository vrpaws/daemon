package logger

// A simple example that shows how to send messages to a Bubble Tea program
// from outside the program using Program.Send(Msg).

import (
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#EB4034")).Bold(true).SetString("ERROR")
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	greyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dotStyle      = helpStyle.UnsetMargins()
	durationStyle = dotStyle
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	scrollStyle   = helpStyle.Margin(0, 0, 1)
)

func (m *Logger) Write(p []byte) (n int, err error) {
	_, _ = m.Update(Message(p))
	return len(p), nil
}

type Logger struct {
	spinner  spinner.Model
	messages []Renderable
	offset   int
	quitting bool

	maxHeight int
	width     int
	height    int
	last      int

	callbacks map[string]tea.Cmd
	logWriter io.Writer
}

var globalLogger *Logger

func NewLogger() *Logger {
	if globalLogger == nil {
		const numLastResults = 256
		s := spinner.New()
		s.Style = spinnerStyle
		globalLogger = &Logger{
			spinner:  s,
			messages: make([]Renderable, numLastResults),
			last:     -1,

			callbacks: make(map[string]tea.Cmd),
			logWriter: io.Discard,
		}
	}

	return globalLogger
}

func (m *Logger) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Logger) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case io.Writer:
		m.logWriter = msg
		return m, nil
	case error:
		m.messages = append(m.messages[1:], NewMessageTimef("%s: %v", errorStyle, msg))
		m.writeToLog(fmt.Sprintf("ERROR: %v", msg))
		return m, nil
	case tea.WindowSizeMsg:
		m.maxHeight = max(0, msg.Height-14)
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft {
			for prefix, callback := range m.callbacks {
				if zone.Get(prefix).InBounds(msg) {
					return m, callback
				}
			}
		}
		if !tea.MouseEvent(msg).IsWheel() {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.last != -1 {
				m.offset = min(m.offset+1, m.last-1)
			} else {
				m.offset = min(len(m.messages)-1, m.offset+1)
			}
		case tea.MouseButtonWheelDown:
			m.offset = max(0, m.offset-1)
		default:
			return m, nil
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, nil
		default:
			return m, nil
		}
	case Renderable:
		if concat, ok := msg.(Concat); ok {
			for _, item := range concat.Items {
				if anchor, ok := item.(Anchor); ok && anchor.OnClick != nil {
					m.callbacks[anchor.Prefix] = anchor.OnClick
				}
			}
		}
		if anchor, ok := msg.(Anchor); ok && anchor.OnClick != nil {
			m.callbacks[anchor.Prefix] = anchor.OnClick
		}
		m.messages = append(m.messages[1:], msg)
		if msg.ShouldSave() {
			go m.writeToLog(msg.Raw())
			return m, nil
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		for _, renderable := range m.messages {
			switch renderable := renderable.(type) {
			case Concat:
				for _, item := range renderable.Items {
					if g, ok := item.(*GradientString); ok {
						g.Advance()
					}
				}
			case *GradientString:
				renderable.Advance()
			}
		}
		return m, cmd
	default:
		return m, nil
	}
}

func (m *Logger) writeToLog(v string) {
	if m.logWriter == nil || m.logWriter == io.Discard {
		msg := errorStyle.String() + ": trying to write to logWriter but program is not initialized!"
		m.messages = append(m.messages, Message(msg))
		log.Println("ERROR: trying to write to logWriter but program is not initialized!")
		return
	}

	v = strings.TrimRight(v, "\r\n")
	_, err := m.logWriter.Write([]byte(time.Now().Format("2006/01/02 15:04:05 ") + v + "\n"))
	if err != nil {
		log.Printf("ERROR: error writing to logWriter: %v", err)
		return
	}
}

func (m *Logger) writeToLogf(pattern string, a ...any) {
	m.writeToLog(fmt.Sprintf(pattern, a...))
}

func (m *Logger) View() string {
	var s strings.Builder

	if m.quitting {
		s.WriteString("Cleaning up...")
	} else {
		s.WriteString(m.spinner.View())
		s.WriteString(" VRPaws Client working... 🐇🐕")
	}

	s.WriteString("\n\n")

	for _, line := range m.getVisibleLogs() {
		s.WriteString(line)
		if len(line) > 0 && line[len(line)-1] != '\n' {
			s.WriteString("\n")
		}
	}

	if !m.quitting {
		s.WriteString(helpStyle.Render("Press Ctrl + C to exit"))
	}

	if m.quitting {
		s.WriteString("\nQuitting...\n")
	}

	var scroll strings.Builder
	if m.offset > 0 {
		for i := 0; i < m.maxHeight-1; i++ {
			if i >= m.offset {
				break
			}
			if i == 0 {
				scroll.WriteString("┌\n")
			} else {
				scroll.WriteString("│\n")
			}
		}
		scroll.WriteString("↓")
	} else {
		scroll.WriteString(" ")
	}

	return appStyle.Render(lipgloss.JoinHorizontal(lipgloss.Bottom, scrollStyle.Render(scroll.String()), " ", s.String()))
}

func (m *Logger) getVisibleLogs() []string {
	if m.maxHeight <= 0 {
		return nil
	}
	var height int
	logs := make([]string, 0, m.maxHeight)
	messages := slices.Clone(m.messages)
	slices.Reverse(messages)
	for i, message := range messages {
		if message == nil || message.Len() == 0 {
			m.last = i
			break
		}
	}
	if m.last != -1 {
		messages = messages[:m.last]
	}
	for i, res := range messages {
		if i < m.offset {
			continue
		}
		out, h := res.String(m.width - 24)
		height += h
		if m.offset > 0 && height+2 > m.maxHeight {
			break
		}
		if height > m.maxHeight {
			break
		}
		logs = append(logs, out)
	}

	slices.Reverse(logs)
	return logs
}
