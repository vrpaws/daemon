package settings

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/sqweek/dialog"

	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/message"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
	"vrc-moments/pkg/gradient"
	"vrc-moments/pkg/vrc"
)

type Model struct {
	config  *Config
	inputs  []textinput.Model
	focused int
	err     error
	buttons *ButtonManager
}

type Config struct {
	Username  string `json:"vrchat_username"`
	UserID    string `json:"user_id,omitempty"`
	Token     string `json:"token"`
	Path      string `json:"path"`
	Server    string `json:"server"`
	LastWorld string `json:"last_world,omitempty"`

	server api.Server[*vrpaws.Me, *vrpaws.UploadPayload, *vrpaws.UploadResponse]
	me     *vrpaws.Me
}

const (
	username = iota
	token
	path
	serverURL
)

const submit = -1

const (
	hotPink     = lipgloss.Color("#FF06B7")
	redError    = lipgloss.Color("#EB4034")
	lightGrey   = lipgloss.Color("#D3D3D3")
	pastelGreen = lipgloss.Color("#6A994E")
	darkGray    = lipgloss.Color("#767676")
	highlight   = lipgloss.Color("#874BFD")
	special     = lipgloss.Color("#43BF6D")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	errorStyle    = lipgloss.NewStyle().Foreground(redError)
	disabledStyle = lipgloss.NewStyle().Foreground(darkGray).Strikethrough(true)
	submitStyle   = lipgloss.NewStyle().Underline(true).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(pastelGreen)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)

	normalStyle = lipgloss.NewStyle().Italic(true)
)

const (
	loginButton  = "relogin-button"
	browseButton = "browse-button"
)

// ButtonManager handles button styles independently
type ButtonManager struct {
	styles map[string]*lipgloss.Style
}

func NewManager() *ButtonManager {
	baseStyle := lipgloss.NewStyle().
		Foreground(highlight).
		Padding(0, 1).
		Margin(0, 0, 0, 1)

	browseStyle := lipgloss.NewStyle().Inherit(baseStyle).SetString(" Browse")
	loginStyle := lipgloss.NewStyle().Inherit(baseStyle).SetString(" Login")

	return &ButtonManager{
		styles: map[string]*lipgloss.Style{
			browseButton: &browseStyle,
			loginButton:  &loginStyle,
		},
	}
}

func (b *ButtonManager) GetStyle(buttonID string) lipgloss.Style {
	if style, exists := b.styles[buttonID]; exists {
		return *style
	}
	return lipgloss.NewStyle()
}

func (b *ButtonManager) SetHover(buttonID string) {
	if style, exists := b.styles[buttonID]; exists {
		*style = style.Foreground(special)
	}
}

func (b *ButtonManager) SetClick(buttonID string) {
	if style, exists := b.styles[buttonID]; exists {
		*style = style.Foreground(highlight)
	}
}

func (b *ButtonManager) Reset(buttonID string) {
	switch buttonID {
	case browseButton:
		b.AddButton(browseButton, " Browse")
	case loginButton:
		b.AddButton(loginButton, " Login")
	}
}

// AddButton adds a new button to the style manager
func (b *ButtonManager) AddButton(buttonID, text string) {
	baseStyle := lipgloss.NewStyle().
		Foreground(highlight).
		Padding(0, 1).
		Margin(0, 0, 0, 1)

	style := lipgloss.NewStyle().Inherit(baseStyle).SetString(text)
	b.styles[buttonID] = &style
}

func New(config *Config, server *vrpaws.Server) *Model {
	var inputs []textinput.Model

models:
	for i := range math.MaxInt {
		switch i {
		case username:
			input := textinput.New()
			input.Placeholder = config.Username
			input.SetValue(config.Username)
			input.CharLimit = 64
			input.Width = 64
			input.Prompt = ""
			input.Blur()
			inputs = append(inputs, input)

		case token:
			input := textinput.New()
			input.Placeholder = config.Token
			input.SetValue(config.Token)
			input.CharLimit = 64
			input.Width = 64
			input.Prompt = ""
			input.Focus()
			inputs = append(inputs, input)

		case path:
			input := textinput.New()
			input.Placeholder = config.Path
			input.SetValue(config.Path)
			input.CharLimit = 64
			input.Width = 64
			input.Prompt = ""
			inputs = append(inputs, input)

		case serverURL:
			input := textinput.New()
			input.Placeholder = config.Server
			input.SetValue(config.Server)
			input.CharLimit = 64
			input.Width = 64
			input.Prompt = ""
			input.Validate = urlValidator
			inputs = append(inputs, input)

		default:
			break models
		}
	}

	config.server = server
	me, err := config.server.ValidToken(config.Token)
	if err == nil {
		config.me = me
	}

	return &Model{
		config:  config,
		inputs:  inputs,
		focused: 1,
		buttons: NewManager(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.save(), textinput.Blink, m.Poll())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))

	switch msg := msg.(type) {
	case message.BrowseRequest:
		return m, m.browseDirectory
	case *vrpaws.Me:
		m.config.me = msg
		m.config.Token = msg.User.AccessToken
		m.inputs[token].SetValue(m.config.Token)
		return m, tea.Batch(
			m.save(),
			message.Callback(m.config.Save),
			message.Cmd(
				logger.Concat{
					Separator: " ",
					Save:      true,
					Items: []logger.Renderable{
						logger.NewMessageTime("Sucessfully logged in to vrpaws as"),
						logger.NewStaticString(msg.User.Username, gradient.BlueGreenYellow...),
					},
				},
			),
		)
	case message.UsernameSet:
		m.inputs[username].SetValue(string(msg))
		return m, m.save()
	case message.RoomSet:
		m.config.SetRoom(string(msg))
		return m, m.Poll()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlS:
			return m, m.save()
		case tea.KeyEnter:
			if m.focused == len(m.inputs) {
				return m, m.save()
			}
			m.nextInput()
			for m.focused == username {
				m.nextInput()
			}
		case tea.KeyShiftTab, tea.KeyCtrlP, tea.KeyUp:
			m.prevInput()
			for m.focused == username {
				m.prevInput()
			}
		case tea.KeyTab, tea.KeyCtrlN, tea.KeyDown:
			m.nextInput()
			for m.focused == username {
				m.nextInput()
			}
		case tea.KeyEsc:
			return m, m.discard()
		default:
		}
		for i := range m.inputs {
			if i == m.focused {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			if msg.Button != tea.MouseButtonLeft {
				return m, nil
			}
			if zone.Get(browseButton).InBounds(msg) {
				bsm := m.buttons
				bsm.SetHover(browseButton)
				return m, nil
			}
		case tea.MouseActionRelease:
			if zone.Get(browseButton).InBounds(msg) {
				bsm := m.buttons
				bsm.SetClick(browseButton)
				if msg.Button == tea.MouseButtonLeft {
					return m, m.browseDirectory
				}
			} else {
				bsm := m.buttons
				bsm.Reset(browseButton)
			}
			if zone.Get(loginButton).InBounds(msg) {
				bsm := m.buttons
				bsm.SetHover(loginButton)
				if msg.Button == tea.MouseButtonLeft {
					return m, message.Msg[message.ManualRequest]()
				}
			} else {
				bsm := m.buttons
				bsm.Reset(loginButton)
			}
		case tea.MouseActionMotion:
			for id := range m.buttons.styles {
				if zone.Get(id).InBounds(msg) {
					bsm := m.buttons
					bsm.SetHover(id)
				} else {
					bsm := m.buttons
					bsm.Reset(id)
				}
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
	return fmt.Sprintf("%sSettings\n\n %s\n",
		m.errorMessage(),
		m.renderAll(),
	)
}

func (m *Model) errorMessage() string {
	if m.err == nil {
		return ""
	}

	return fmt.Sprintf("%s: %s!\n\n", errorStyle.Bold(true).Render("Error"), m.err.Error())
}

func (m *Model) renderAll() string {
	var b strings.Builder

	for i := range len(m.inputs) {
		r := m.render(i)
		if r == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(r)
	}
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(m.render(submit))

	return b.String()
}

func (m *Model) render(i int) string {
	var (
		prefix string = "  "
		suffix string
		title  string
	)
	changed := func(b bool) {
		if b {
			m.inputs[i].TextStyle = successStyle
		} else {
			prefix = disabledStyle.Strikethrough(false).Render("※ ")
			m.inputs[i].TextStyle = normalStyle
		}
	}

	switch i {
	case serverURL:
		title = "Server URL"
		if m.config.Server == m.inputs[i].Value() {
			if urlValidator(m.config.Server) != nil {
				m.inputs[i].TextStyle = errorStyle.Italic(true)
			} else {
				changed(true)
			}
		} else {
			changed(false)
		}
	case token:
		title = "Token"
		changed(m.config.Token == m.inputs[i].Value())
		if m.config.me != nil {
			suffix = fmt.Sprintf(" (%s)", m.config.me.User.Username)
		}
		loginButton := zone.Mark(loginButton, m.buttons.GetStyle(loginButton).Render())
		suffix = suffix + loginButton
	case username:
		title = "Username"
		m.inputs[i].TextStyle = disabledStyle
	case path:
		title = "Path"
		changed(m.config.Path == m.inputs[i].Value())
		browseButton := zone.Mark(browseButton, m.buttons.GetStyle(browseButton).Render())
		suffix = browseButton
	case submit:
		if m.focused == len(m.inputs) {
			return submitStyle.Render("Press Enter to save ->")
		} else {
			return continueStyle.Render("Continue ->")
		}
	default:
		return errorStyle.Bold(true).Render("Unknown input")
	}

	title = lipgloss.NewStyle().Width(64).Render(inputStyle.Render(title) + suffix)
	return prefix + title + "\n   " + m.inputs[i].View()
}

func urlValidator(s string) error {
	_, err := url.Parse(s)
	return err
}

func (m *Model) discard() tea.Cmd {
	for i := range m.inputs {
		m.inputs[i].SetValue(m.inputs[i].Placeholder)
	}
	return nil
}

func (m *Model) save() tea.Cmd {
	return func() tea.Msg {
		var cmds []tea.Cmd
		for i, input := range m.inputs {
			value := input.Value()
			switch i {
			case username:
				if m.config.Username != value {
					if err := m.config.SetUsername(value); err != nil {
						m.inputs[i].TextStyle = errorStyle.Italic(true)
						cmds = append(cmds, message.Cmd(fmt.Errorf("could not set username: %w", err)))
						continue
					}
					m.inputs[i].Placeholder = value
					cmds = append(cmds, message.Cmd(message.UsernameSet(value)))
				}
			case token:
				if m.config.Token != value {
					if err := m.config.SetToken(value); err != nil {
						cmds = append(cmds, message.Cmd(fmt.Errorf("could not set token: %w", err)))
						continue
					}
					m.inputs[i].Placeholder = value
					cmds = append(cmds, message.Cmd(message.TokenSet(value)))
				}
			case path:
				if m.config.Path != value {
					patterns, err := lib.ExpandPatterns(true, false, value)
					if err != nil {
						m.inputs[i].TextStyle = errorStyle.Italic(true)
						cmds = append(cmds, message.Cmd(fmt.Errorf("could not set path: %w", err)))
						continue
					}
					if err := m.config.SetPath(value); err != nil {
						cmds = append(cmds, message.Cmd(fmt.Errorf("could not set path: %w", err)))
						continue
					}
					m.inputs[i].Placeholder = value
					cmds = append(cmds, message.Cmd(message.PathSet(value)), message.Cmd(message.PatternsSet(patterns)))
				}
			case serverURL:
				if m.config.Server != value {
					if err := m.config.SetServer(value); err != nil {
						cmds = append(cmds, message.Cmd(fmt.Errorf("could not set server: %w", err)))
						continue
					}
					m.inputs[i].Placeholder = value
					cmds = append(cmds, message.Cmd(message.ServerSet(value)))
				}
			}
		}

		if len(cmds) == 0 {
			return nil
		}

		return tea.Batch(cmds...)
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

func (c *Config) Save() error {
	return lib.EncodeToFile(lib.ConfigPath, c)
}

func (c *Config) SetPath(path string) error {
	c.Path = path

	if err := c.Save(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

func (c *Config) SetUsername(username string) error {
	c.Username = username

	if err := c.Save(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

func (c *Config) SetToken(token string) error {
	c.Token = token
	me, err := c.server.ValidToken(token)
	if err != nil {
		return fmt.Errorf("token is not valid: %w", err)
	}
	c.me = me

	if err := c.Save(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

func (c *Config) SetServer(link string) error {
	parsed, err := url.Parse(link)
	if err != nil {
		return err
	}
	c.Server = parsed.String()

	return c.server.SetRemote(c.Server)
}

func (c *Config) SetRoom(room string) {
	if c.LastWorld != "Unknown" && c.LastWorld != room {
		log.Printf("World changed to %s", room)
	}
	c.LastWorld = room
}

func (m *Model) Poll() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		roomName, err := vrc.ExtractCurrentRoomName(vrc.DefaultLogPath)
		if err != nil {
			return err
		}

		return message.RoomSet(roomName)
	})
}

func (m *Model) browseDirectory() tea.Msg {
	dir, patterns, err := lib.SelectVRChatDirectory(m.config.Path)
	if err != nil && !errors.Is(err, dialog.ErrCancelled) {
		return err
	}

	log.Printf("Changed directory to %s with %d directories:", dir, len(patterns))
	for i, path := range patterns {
		log.Printf("%s", path)
		if i == 10 {
			log.Printf("and %d more...", len(patterns)-i)
			break
		}
	}

	m.inputs[path].SetValue(dir)
	m.inputs[path].Placeholder = dir
	m.config.Path = dir
	return tea.Batch(
		message.Cmd(message.PathSet(dir)),
		message.Cmd(message.PatternsSet(patterns)),
		m.save(),
		message.Callback(m.config.Save),
	)
}
