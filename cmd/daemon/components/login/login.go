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
	"github.com/pkg/browser"

	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
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
		if m.me == nil && m.config.Token == "" || m.config.Token == "Unset" {
			return m.login()
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
	}
	return m, nil
}

func (m *Model) View() string {
	return "Launching login page...\n"
}

func (m *Model) login() tea.Msg {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	addr := listener.Addr().String()
	redirectURL := fmt.Sprintf("http://%s", addr)

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
