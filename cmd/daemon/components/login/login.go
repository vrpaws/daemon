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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/pkg/browser"

	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

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

//go:embed all:login/out/*
var login embed.FS

//go:embed all:success/out/*
var success embed.FS

type Model struct {
	config    *settings.Config
	server    api.Server[*vrpaws.Me, *vrpaws.UploadResponse]
	program   *tea.Program
	local     *http.Server
	loginFS   http.Handler
	successFS http.Handler
	me        *vrpaws.Me

	button lipgloss.Style

	width  int
	height int
}

func New(config *settings.Config, server *vrpaws.Server) *Model {
	loginFS, err := fs.Sub(login, "login/out")
	if err != nil {
		log.Fatal(err)
	}
	loginHandler := http.FileServer(http.FS(loginFS))

	successFS, err := fs.Sub(success, "success/out")
	if err != nil {
		log.Fatal(err)
	}
	successHandler := http.FileServer(http.FS(successFS))

	return &Model{config: config, server: server, loginFS: loginHandler, successFS: successHandler}
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		if m.me == nil {
			if user, err := m.server.ValidToken(m.config.Token); err == nil {
				return user
			}

			if m.config.Token == "" || m.config.Token == "Unset" {
				return m.login()
			}
		}
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case message.LoginRequest:
		return m, m.login
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
					return m, m.login
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
	return lipgloss.PlaceVertical(m.height-12, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			"Welcome to VRPaws Client!",
			loginButton,
		))
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

func (m *Model) login() tea.Msg {
	var hostname string
	var listener net.Listener
	var err error

	if isLocalhostAccessible() {
		listener, err = net.Listen("tcp", "localhost:0")
		if err == nil {
			hostname = "localhost"
		} else {
			listener, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return err
			}
			hostname = "127.0.0.1"
		}
	} else {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
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
				m.program.Send(user)
			} else {
				m.program.Send(err)
			}

			m.successFS.ServeHTTP(w, r)
		} else if m.me != nil {
			m.successFS.ServeHTTP(w, r)
		} else {
			m.loginFS.ServeHTTP(w, r)
		}
	})

	m.local = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := m.local.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http.Serve: %v", err)
		}
	}()

	connectURL := fmt.Sprintf(
		"http://vrpa.ws/client/connect?redirect_url=%s&service_name=%s",
		url.QueryEscape(redirectURL),
		"vrpaws-client",
	)

	return browser.OpenURL(connectURL)
}
