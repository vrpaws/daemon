package logger

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"vrc-moments/pkg/gradient"
)

type Renderable interface {
	String(width int) (text string, height int)
	Len() int
	ShouldSave() bool
	Raw() string
}

type Anchor struct {
	Prefix  string
	OnClick tea.Cmd
	Message Renderable
}

func NewAnchor(message Renderable, callback tea.Cmd, prefix string) Anchor {
	if prefix == "" {
		prefix = zone.NewPrefix() + message.Raw()
	}
	return Anchor{
		Prefix:  prefix,
		OnClick: callback,
		Message: message,
	}
}

func (a Anchor) String(width int) (string, int) {
	text, height := a.Message.String(width)
	return zone.Mark(a.Prefix, text), height
}

func (a Anchor) Len() int {
	return a.Message.Len()
}

func (a Anchor) ShouldSave() bool {
	return a.Message.ShouldSave()
}

func (a Anchor) Raw() string {
	return a.Message.Raw()
}

type Concat struct {
	Separator string
	Save      bool // saves every Renderable if set to true, otherwise will use each's Renderable.ShouldSave
	Items     []Renderable
}

func (c Concat) Raw() string {
	var sb strings.Builder

	for _, renderable := range c.Items {
		if !c.ShouldSave() && (renderable == nil || !renderable.ShouldSave()) {
			continue
		}
		if c.Separator != "" && sb.Len() > 0 {
			sb.WriteString(c.Separator)
		}
		sb.WriteString(strings.TrimRight(renderable.Raw(), "\r\n"))
	}

	return sb.String()
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

func (c Concat) ShouldSave() bool {
	return c.Save
}

type Message string // sending Message will only append to the logger but not the log file

func Messagef(format string, args ...any) Message {
	return Message(fmt.Sprintf(format, args...))
}

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

func (r *MessageTime) Raw() string {
	return r.Time.Format("2006/01/02 15:04:05 ") + r.Message
}

func (r *MessageTime) ShouldSave() bool {
	return true
}

func (r Message) String(maxWidth int) (string, int) {
	return render(string(r), maxWidth)
}

func (r Message) Len() int {
	return lipgloss.Width(string(r))
}

func (r Message) ShouldSave() bool {
	return false
}

func (r Message) Raw() string {
	return string(r)
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

func (r *GradientString) ShouldSave() bool {
	return false
}

func (r *GradientString) Raw() string {
	return r.Message
}

type StaticString struct {
	Message string
	Width   int
	Colors  []string
	Save    bool

	pre string
}

func NewStaticString(message string, colors ...string) *StaticString {
	str := &StaticString{
		Message: message,
		Width:   lipgloss.Width(message),
		Colors:  colors,
		pre:     gradient.Static(message, colors...),
	}

	return str
}

func (r *StaticString) String(maxWidth int) (string, int) {
	if r.pre == "" && lipgloss.Width(r.Message) > 0 {
		r.pre = gradient.Static(r.Message, r.Colors...)
	}
	return render(r.pre, maxWidth)
}

func (r *StaticString) Len() int {
	return r.Width
}

func (r *StaticString) ShouldSave() bool {
	return false
}

func (r *StaticString) Raw() string {
	return r.Message
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
