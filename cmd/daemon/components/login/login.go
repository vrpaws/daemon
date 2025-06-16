package login

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"

	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
)

//go:embed login.html
var login []byte

//go:embed success.html
var success []byte

type Model struct {
	config  *settings.Config
	server  api.Server[*vrpaws.Me, *vrpaws.UploadResponse]
	program *tea.Program
	local   *http.Server
	me      *vrpaws.Me
}

func New(config *settings.Config, server *vrpaws.Server) *Model {
	return &Model{config: config, server: server}
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
		return m, message.Callback(func() tea.Msg {
			if m.local == nil {
				return nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return m.local.Shutdown(ctx)
		})
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

	page := bytes.ReplaceAll(login,
		[]byte("{CALLBACK_URL}"),
		[]byte(redirectURL),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		token := r.URL.Query().Get("access_token")
		if token != "" {
			w.Write(success)
			go func() {
				if user, err := m.server.ValidToken(token); err == nil {
					m.program.Send(user)
				} else {
					m.program.Send(err)
				}
				m.program.Send(m.local.Close())
			}()
			return
		}

		if _, err := w.Write(page); err != nil {
			log.Printf("Could not serve: %v", err)
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
