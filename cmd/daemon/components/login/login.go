package login

import (
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/pkg/browser"

	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	"vrc-moments/pkg/api/vrpaws"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	redError  = lipgloss.Color("#EB4034")

	errorStyle = lipgloss.NewStyle().Foreground(redError)

	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(0, 1).
			Margin(1, 0).
			SetString("Login to VRPaws")

	buttonHoverStyle = buttonStyle.
				BorderForeground(special).
				Foreground(special).
				SetString("Login to VRPaws")

	buttonClickStyle = buttonStyle.
				BorderForeground(highlight).
				Foreground(highlight).
				SetString("Login to VRPaws")
)

//go:embed all:next/out/*
var success embed.FS

type Model struct {
	config  *settings.Config
	server  *vrpaws.Server
	program *tea.Program
	local   *http.Server
	loginFS http.Handler
	me      *vrpaws.Me
	once    *sync.Once

	button lipgloss.Style

	url    string
	width  int
	height int
	err    error
}

func New(config *settings.Config, server *vrpaws.Server) *Model {
	loginFS, err := fs.Sub(success, "next/out")
	if err != nil {
		log.Fatal(err)
	}
	loginHandler := http.FileServer(http.FS(loginFS))

	return &Model{
		config:  config,
		server:  server,
		loginFS: loginHandler,
		button:  buttonStyle,
		once:    new(sync.Once),
	}
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		if m.me == nil {
			if user, err := m.server.ValidToken(m.config.Token); err == nil {
				return user
			}

			if m.config.Token == "" || m.config.Token == "Unset" {
				return message.CallbackValue(m.login, true)
			}
		}
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		m.err = msg
		return m, nil
	case message.ManualRequest:
		return m, message.CallbackValue(m.login, false)
	case message.LoginRequest:
		return m, message.CallbackValue(m.login, true)
	case *tea.Program:
		m.program = msg
		return m, nil
	case *vrpaws.Me:
		m.me = msg
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			if msg.Button != tea.MouseButtonLeft {
				return m, nil
			}
			if zone.Get("login-button").InBounds(msg) {
				m.button = buttonClickStyle
				return m, nil
			}
		case tea.MouseActionRelease:
			if zone.Get("login-button").InBounds(msg) {
				m.button = buttonHoverStyle
			} else {
				m.button = buttonStyle
			}
			if msg.Button == tea.MouseButtonLeft {
				if zone.Get("login-button").InBounds(msg) {
					m.button = buttonClickStyle
					return m, message.CallbackValue(m.login, true)
				}
			}
		case tea.MouseActionMotion:
			if zone.Get("login-button").InBounds(msg) {
				m.button = buttonHoverStyle
			} else {
				m.button = buttonStyle
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.me != nil {
		return "Logged in as " + m.me.User.Username + "\n"
	}

	loginButton := zone.Mark("login-button", m.button.Render())
	strs := []string{"Welcome to VRPaws Client!", loginButton}
	if m.err != nil {
		strs = append(strs, errorStyle.Render("error while logging in\n"), m.err.Error())
	}

	return lipgloss.PlaceVertical(m.height-8, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, strs...))
}

func isLocalhostAccessible() bool {
	addrs, err := net.LookupHost("localhost")
	if err != nil {
		return false
	}

	// Check if localhost resolves to 127.0.0.1
	for _, addr := range addrs {
		if addr == "127.0.0.1" || addr == "::1" {
			return true
		}
	}
	return false
}

func (m *Model) login(direct bool) tea.Msg {
	var err error
	m.once.Do(func() {
		var hostname string
		var listener net.Listener

		if isLocalhostAccessible() {
			listener, err = net.Listen("tcp", "localhost:0")
			if err == nil {
				hostname = "localhost"
			} else {
				listener, err = net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return
				}
				hostname = "127.0.0.1"
			}
		} else {
			listener, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return
			}
			hostname = "127.0.0.1"
		}

		_, port, _ := net.SplitHostPort(listener.Addr().String())
		redirectURL := fmt.Sprintf("http://%s:%s", hostname, port)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("access_token")
			if token != "" {
				if user, err := m.server.ValidToken(token); err == nil {
					m.err = nil
					m.program.Send(user)
				} else {
					m.program.Send(fmt.Errorf("got token %q but it was not valid: %w", token, err))
				}
			}

			m.loginFS.ServeHTTP(w, r)
		})

		m.local = &http.Server{
			Handler: mux,
		}

		go func() {
			if err := m.local.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("http.Serve: %v", err)
			}
		}()

		m.url = redirectURL
	})
	if err != nil {
		return err
	}

	if direct {
		connectURL := fmt.Sprintf(
			"http://vrpa.ws/client/connect?redirect_url=%s&service_name=%s",
			url.QueryEscape(m.url),
			"vrpaws-client",
		)
		return browser.OpenURL(connectURL)
	} else {
		return browser.OpenURL(m.url)
	}
}
