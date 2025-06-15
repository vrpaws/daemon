package logger

// A simple example that shows how to send messages to a Bubble Tea program
// from outside the program using Program.Send(Msg).

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

type Renderable interface {
	String(width int) (text string, height int)
	Len() int
}

type Concat struct {
	Items     []Renderable
	Separator string
}

func (c Concat) String(maxWidth int) (string, int) {
	itemCount := len(c.Items)
	// non-nil, empty slice of rows; pre-allocate capacity = number of items
	lines := make([][]string, 0, itemCount)
	maxHeight := 0

	for _, renderable := range c.Items {
		text, height := renderable.String(maxWidth)
		if height > maxHeight {
			maxHeight = height
		}

		// split into lines, trim trailing newline
		textLines := strings.Split(strings.TrimRight(text, "\n"), "\n")

		// pad textLines up to height with empty strings
		for len(textLines) < height {
			textLines = append(textLines, "")
		}

		// ensure we have enough rows in lines
		for len(lines) < height {
			// each row starts as an empty slice of parts, capacity = itemCount
			lines = append(lines, make([]string, 0, itemCount))
		}

		// append this renderable's i-th line into row i
		for i := 0; i < height; i++ {
			lines[i] = append(lines[i], textLines[i])
		}
	}

	// in case some items were shorter than maxHeight,
	// ensure we have exactly maxHeight rows
	for len(lines) < maxHeight {
		lines = append(lines, make([]string, 0, itemCount))
	}

	// build final string, joining each row with Separator
	var sb strings.Builder
	for _, parts := range lines {
		sb.WriteString(strings.Join(parts, c.Separator))
		sb.WriteByte('\n')
	}

	return sb.String(), maxHeight
}

func (c Concat) Len() int {
	for _, renderable := range c.Items {
		if renderable == nil {
			continue
		}
		if renderable.Len() > 0 {
			return renderable.Len()
		}
	}

	return 0
}

type Message string // sending Message will only append to the logger but not the log file

// MessageTime is like Message, but with an optional Time
type MessageTime struct {
	Message string
	Width   int
	Time    time.Time
}

func NewMessageTime(message string) *MessageTime {
	return &MessageTime{
		Message: message,
		Width:   lipgloss.Width(message),
		Time:    time.Now(),
	}
}

func NewMessageTimef(format string, a ...any) *MessageTime {
	message := fmt.Sprintf(format, a...)
	return &MessageTime{
		Message: message,
		Width:   lipgloss.Width(message),
		Time:    time.Now(),
	}
}

func (r *MessageTime) String(maxWidth int) (string, int) {
	return render(greyStyle.Render(r.Time.Format("2006/01/02 15:04:05 "))+r.Message, maxWidth)
}

func (r *MessageTime) Len() int {
	return r.Width
}

func (r Message) String(maxWidth int) (string, int) {
	return render(string(r), maxWidth)
}

func (r Message) Len() int {
	return lipgloss.Width(string(r))
}

type GradientString struct {
	Message string
	Width   int
	Colors  []string
}

const fps = float64(60)

func NewGradientString(message string, duration time.Duration, colors ...string) *GradientString {
	str := &GradientString{
		Message: message,
		Width:   lipgloss.Width(message),
		Colors:  colors,
	}
	frames := int(duration.Nanoseconds() * int64(fps) / int64(time.Second))
	steps := min(max(frames, 5), 120)
	gradient.Global.New(str.Message, steps, str.Colors...)
	return str
}

func (r *GradientString) String(maxWidth int) (string, int) {
	return render(gradient.Global.RenderCurrent(r.Message), maxWidth)
}

func (r *GradientString) Advance() {
	gradient.Global.Advance(r.Message)
}

func (r *GradientString) Len() int {
	return r.Width
}

func render(text string, maxWidth int) (string, int) {
	if lipgloss.Width(text) <= maxWidth {
		return text, 1
	}

	var s strings.Builder
	words := strings.Fields(text)
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
	messages []Renderable
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
			messages: make([]Renderable, numLastResults),
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
	case Renderable:
		m.messages = append(m.messages[1:], msg)
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
