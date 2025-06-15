package logger

// A simple example that shows how to send messages to a Bubble Tea program
// from outside the program using Program.Send(Msg).

import (
	"log"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type Message string // sending Message will only append to the logger but not the log file

// MessageTime is like Message, but with an optional Time
type MessageTime struct {
	Message string
	Time    time.Time
}

func (r Message) String(maxWidth int) (string, int) {
	if lipgloss.Width(string(r)) <= maxWidth {
		return string(r), 1
	}

	var s strings.Builder
	words := strings.Fields(string(r))
	currentLineLength := 0

	widths := make([]int, len(words))
	for i, w := range words {
		widths[i] = lipgloss.Width(w)
	}

	var height int
	var writtenNewline bool
	for i, word := range words {
		if currentLineLength+widths[i] > maxWidth {
			s.WriteString("\n")
			currentLineLength = 0
			height++
		}

		// Add space between words, but not before the first word
		if currentLineLength > 0 {
			s.WriteString(" ")
			currentLineLength++
		}

		// Write the word
		s.WriteString(word)
		writtenNewline = false
		currentLineLength += widths[i]

		// If we're not at the last word, and the next word will fit in the same line, continue
		if i < len(widths)-1 && currentLineLength+widths[i+1] > maxWidth {
			s.WriteString("\n")
			writtenNewline = true
			currentLineLength = 0
			height++
		}
	}

	if !writtenNewline {
		s.WriteString("\n")
		height++
	}

	return s.String(), height
}

func (m *Logger) Write(p []byte) (n int, err error) {
	_, _ = m.Update(Message(p))
	return len(p), nil
}

type Logger struct {
	spinner  spinner.Model
	messages []Message
	offset   int
	quitting bool

	maxHeight int
	width     int
	height    int
	last      int
}

var globalLogger *Logger

func NewLogger() *Logger {
	if globalLogger == nil {
		const numLastResults = 256
		s := spinner.New()
		s.Style = spinnerStyle
		globalLogger = &Logger{
			spinner:  s,
			messages: make([]Message, numLastResults),
			last:     -1,
		}
	}

	return globalLogger
}

func (m *Logger) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Logger) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		go log.Printf("%s: %v", errorStyle, msg)
		return m, nil
	case []error:
		go func() {
			for _, msg := range msg {
				log.Printf("%s: %v", errorStyle, msg)
			}
		}()
		return m, nil
	case tea.WindowSizeMsg:
		m.maxHeight = max(0, msg.Height-14)
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseMsg:
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
	case Message:
		m.messages = append(m.messages[1:], msg)
		return m, nil
	case MessageTime:
		if msg.Time.IsZero() {
			msg.Time = time.Now()
		}
		m.messages = append(m.messages[1:], Message(greyStyle.Render(msg.Time.Format("2006/01/02 15:04:05 "))+msg.Message))
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

// resizeToHeight resizes the results slice to the given height, keeping old messages if they fit
// Deprecated: use *Logger.getVisibleLogs
func resizeToHeight(results []Message, i int) []Message {
	if i < 0 {
		return results
	}

	if i < len(results) {
		return results[len(results)-i:]
	}

	newResults := make([]Message, i)
	copy(newResults[i-len(results):], results)
	return newResults
}

func (m *Logger) View() string {
	var s strings.Builder

	if m.quitting {
		s.WriteString("Cleaning up...")
	} else {
		s.WriteString(m.spinner.View())
		s.WriteString(" VRC Moments Daemon working... 🐇🐕")
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
	m.last = slices.Index(messages, "")
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
