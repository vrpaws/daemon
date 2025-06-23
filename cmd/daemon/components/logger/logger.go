package logger

// A simple example that shows how to send messages to a Bubble Tea program
// from outside the program using Program.Send(Msg).

import (
	"fmt"
	"io"
	"log"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/pkg/gradient"
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
	mu        sync.Mutex
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
	case Delete:
		return m, nil
	case Renderable:
		m.messages = append(m.messages[1:], msg)
		if msg.ShouldSave() {
			go m.writeToLog(msg.Raw())
			return m, nil
		}

		return m, message.CallbackValue(m.register, msg)
	case spinner.TickMsg:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		_, cmd = m.propagate(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		gradient.Global.AdvanceAll()
		return m, tea.Batch(cmds...)
	default:
		return m.propagate(msg)
	}
}

func (m *Logger) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var containsDelete func(Renderable) bool
	containsDelete = func(r Renderable) bool {
		switch v := r.(type) {
		case Delete, *Delete:
			return true
		case Concat:
			if len(v.Items) == 0 {
				return true
			}
			for _, item := range v.Items {
				if containsDelete(item) {
					return true
				}
			}
		case *Concat:
			if len(v.Items) == 0 {
				return true
			}
			for _, item := range v.Items {
				if containsDelete(item) {
					return true
				}
			}
		case Anchor:
			return containsDelete(v.Message)
		case *Anchor:
			return containsDelete(v.Message)
		}
		return false
	}

	var propagate func(Renderable) Renderable
	propagate = func(r Renderable) Renderable {
		switch v := r.(type) {
		case *Spinner:
			model, cmd := v.Model.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			v.Model = &model
			return v
		case *Progress:
			model, cmd := v.Update(msg)
			if p, ok := model.(progress.Model); cmd != nil && ok {
				v.Model = &p
				cmds = append(cmds, cmd)
			}
			return v
		case Concat:
			for i, item := range v.Items {
				v.Items[i] = propagate(item)
			}
			return v
		case *Concat:
			for i, item := range v.Items {
				v.Items[i] = propagate(item)
			}
			return v
		case Anchor:
			v.Message = propagate(v.Message)
			return v
		case *Anchor:
			v.Message = propagate(v.Message)
			return v
		default:
			return r
		}
	}

	m.messages = slices.DeleteFunc(m.messages, func(r Renderable) bool {
		if r == nil {
			return false
		}
		return containsDelete(r)
	})

	for i, r := range m.messages {
		m.messages[i] = propagate(r)
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m *Logger) register(r Renderable) tea.Msg {
	callbacks := make(map[string]tea.Cmd)
	var cmds []tea.Cmd
	collectCallbacks(r, callbacks, &cmds)

	m.mu.Lock()
	maps.Copy(m.callbacks, callbacks)
	m.mu.Unlock()

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}

	return nil
}

func collectCallbacks(r Renderable, out map[string]tea.Cmd, cmds *[]tea.Cmd) {
	switch v := r.(type) {
	case tea.Model:
		cmd := v.Init()
		if cmd != nil {
			*cmds = append(*cmds, cmd)
		}
	case Anchor:
		if v.OnClick != nil {
			out[v.Prefix] = v.OnClick
		}
	case *Anchor:
		if v.OnClick != nil {
			out[v.Prefix] = v.OnClick
		}
	case Concat:
		for _, item := range v.Items {
			collectCallbacks(item, out, cmds)
		}
	case *Concat:
		for _, item := range v.Items {
			collectCallbacks(item, out, cmds)
		}
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
	renderable := slices.Clone(m.messages)
	slices.Reverse(renderable)
	for i, r := range renderable {
		if r == nil || r.Len() == 0 {
			m.last = i
			break
		}
	}
	if m.last != -1 {
		renderable = renderable[:m.last]
	}
	for i, res := range renderable {
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
