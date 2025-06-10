package settings

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vrc-moments/pkg/vrc"
)

type Model struct {
	config  *Config
	inputs  []textinput.Model
	focused int
	err     error
}

type Config struct {
	Username  string
	Path      string
	Server    string
	LastWorld string

	server Server // some server
}

type Server interface {
	ValidUser(string) error
}

// TODO: Implement server
type todoServer struct{}

func (s todoServer) ValidUser(username string) error {
	return errors.New("server not yet implemented")
}

type (
	UsernameSet string
	RoomSet     string
)

const inputs = 3

const (
	username = iota
	path
	serverURL
)

const submit = -1

const (
	hotPink     = lipgloss.Color("#FF06B7")
	redError    = lipgloss.Color("#EB4034")
	pastelGreen = lipgloss.Color("#6A994E")
	darkGray    = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	errorStyle    = lipgloss.NewStyle().Foreground(redError)
	successStyle  = lipgloss.NewStyle().Foreground(pastelGreen)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)

	normalStyle = lipgloss.NewStyle().Italic(true)
)

func New(c *Config) *Model {
	inputs := make([]textinput.Model, inputs)

	for i := range inputs {
		switch i {
		case username:
			inputs[i] = textinput.New()
			inputs[i].Placeholder = c.Username
			inputs[i].SetValue(c.Username)
			inputs[i].CharLimit = 64
			inputs[i].Width = 64
			inputs[i].Prompt = ""
			inputs[i].Focus()

		case path:
			inputs[i] = textinput.New()
			inputs[i].Placeholder = c.Path
			inputs[i].SetValue(c.Path)
			inputs[i].CharLimit = 64
			inputs[i].Width = 64
			inputs[i].Prompt = ""

		case serverURL:
			inputs[i] = textinput.New()
			inputs[i].Placeholder = c.Server
			inputs[i].SetValue(c.Server)
			inputs[i].CharLimit = 64
			inputs[i].Width = 64
			inputs[i].Prompt = ""
			inputs[i].Validate = urlValidator
		}
	}

	// TODO: Implement server
	c.server = todoServer{}

	return &Model{
		config: c,
		inputs: inputs,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))

	switch msg := msg.(type) {
	case UsernameSet:
		m.inputs[username].SetValue(string(msg))
		return m, m.save()
	case RoomSet:
		m.config.SetRoom(string(msg))
		return m, m.Poll()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focused == len(m.inputs) {
				return m, m.save()
			}
			m.nextInput()
		case tea.KeyShiftTab, tea.KeyCtrlP, tea.KeyUp:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN, tea.KeyDown:
			m.nextInput()
		default:
		}
		for i := range m.inputs {
			if i == m.focused {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
	case error:
		m.err = msg
		return m, nil
	case []error:
		m.err = errors.Join(msg...)
		return m, nil
	}

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return fmt.Sprintf("%s Settings\n\n %s\n %s\n\n %s\n %s\n\n %s\n %s\n\n %s\n",
		m.errorMessage(),
		inputStyle.Width(64).Render("Username"),
		m.render(username),
		inputStyle.Width(64).Render("Pictures Directory"),
		m.render(path),
		inputStyle.Width(64).Render("Server URL"),
		m.render(serverURL),
		m.render(submit),
	)
}

func (m *Model) errorMessage() string {
	if m.err == nil {
		return ""
	}

	return fmt.Sprintf("%s: %s!\n\n", errorStyle.Bold(true).Render("Error"), m.err.Error())
}

func (m *Model) render(i int) string {
	set := func(b bool) {
		if b {
			m.inputs[i].TextStyle = successStyle
		} else {
			m.inputs[i].TextStyle = normalStyle
		}
	}

	switch i {
	case username:
		if m.config.Username == m.inputs[i].Value() {
			if m.config.server.ValidUser(m.config.Username) != nil {
				m.inputs[i].TextStyle = errorStyle.Italic(true)
			} else {
				set(true)
			}
		} else {
			set(false)
		}
	case path:
		set(m.config.Path == m.inputs[i].Value())
	case serverURL:
		if m.config.Server == m.inputs[i].Value() {
			if urlValidator(m.config.Server) != nil {
				m.inputs[i].TextStyle = errorStyle.Italic(true)
			} else {
				set(true)
			}
		} else {
			set(false)
		}
	case submit:
		if m.focused == len(m.inputs) {
			return inputStyle.Underline(true).Bold(true).Render("Continue ->")
		} else {
			return continueStyle.Render("Continue ->")
		}
	}

	return m.inputs[i].View()
}

func urlValidator(s string) error {
	_, err := url.Parse(s)
	return err
}

func (m *Model) save() tea.Cmd {
	return func() tea.Msg {
		var errors []error
		for i, input := range m.inputs {
			if input.Value() == "" {
				continue
			}
			switch i {
			case username:
				if m.config.Username == input.Value() && m.err == nil {
					continue
				}
				err := m.config.SetUsername(input.Value())
				if err != nil {
					errors = append(errors, err)
				}
			case path:
				if m.config.Path == input.Value() && m.err == nil {
					continue
				}
				m.config.SetPath(input.Value())
			case serverURL:
				if m.config.Server == input.Value() && m.err == nil {
					continue
				}
				err := m.config.SetServer(input.Value())
				if err != nil {
					errors = append(errors, err)
				}
			}
		}

		if len(errors) > 0 {
			return errors
		}

		return errors
	}
}

// nextInput focuses the next input field
func (m *Model) nextInput() {
	m.focused = (m.focused + 1) % (len(m.inputs) + 1)
}

// prevInput focuses the previous input field
func (m *Model) prevInput() {
	a := m.focused - 1
	b := len(m.inputs) + 1
	m.focused = (a%b + b) % b
}

func (c *Config) SetPath(path string) {
	c.Path = path
}

func (c *Config) SetUsername(username string) error {
	c.Username = username
	if err := c.server.ValidUser(username); err != nil {
		return fmt.Errorf("username %q not found in remote userlist: %w", username, err)
	}

	if err := os.WriteFile("username.txt", []byte(username), 0644); err != nil {
		return err
	}

	return nil
}

func (c *Config) SetServer(link string) error {
	parsed, err := url.Parse(link)
	if err != nil {
		return err
	}
	c.Server = parsed.String()

	return nil
}

func (c *Config) SetRoom(room string) {
	c.LastWorld = room
}

func (m *Model) Poll() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		roomName, err := vrc.ExtractCurrentRoomName(vrc.DefaultLogPath)
		if err != nil {
			return err
		}

		return RoomSet(roomName)
	})
}
