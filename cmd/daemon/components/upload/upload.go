package upload

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	zone "github.com/lrstanley/bubblezone"
	"github.com/pkg/browser"

	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/api/vrpaws"
	"vrc-moments/pkg/flight"
	"vrc-moments/pkg/gradient"
	"vrc-moments/pkg/worker"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	redError  = lipgloss.Color("#EB4034")

	pauseButtonStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlight).
				Padding(0, 1).
				Margin(1, 0).
				SetString("Pause Uploads")

	pauseButtonHoverStyle = pauseButtonStyle.
				BorderForeground(special).
				Foreground(special).
				SetString("Pause Uploads")

	pauseButtonClickStyle = pauseButtonStyle.
				BorderForeground(highlight).
				Foreground(highlight).
				SetString("Pause Uploads")

	resumeButtonStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(special).
				Padding(0, 1).
				Margin(1, 0).
				SetString("Resume Uploads")

	resumeButtonHoverStyle = resumeButtonStyle.
				BorderForeground(highlight).
				Foreground(highlight).
				SetString("Resume Uploads")

	resumeButtonClickStyle = resumeButtonStyle.
				BorderForeground(special).
				Foreground(special).
				SetString("Resume Uploads")
)

type Uploader struct {
	watcher *lib.Watcher
	ctx     context.Context
	config  *settings.Config
	program *tea.Program

	uploadFlight flight.Cache[string, *vrpaws.UploadResponse]
	queue        worker.Pool[string, error]

	server api.Server[*vrpaws.Me, *vrpaws.UploadPayload, *vrpaws.UploadResponse]

	button lipgloss.Style
	paused bool
	width  int
	height int
}

func NewModel(ctx context.Context, config *settings.Config, server *vrpaws.Server) *Uploader {
	uploader := &Uploader{
		ctx:    ctx,
		config: config,
		server: server,
		button: pauseButtonStyle,
	}
	uploader.uploadFlight = flight.NewCache(uploader.upload)
	uploader.queue = worker.NewPool(runtime.NumCPU(), func(path string) error {
		_, err := uploader.uploadFlight.Get(path)
		return err
	})

	return uploader
}

func (m *Uploader) Init() tea.Cmd {
	return nil
}

func (m *Uploader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *fsnotify.Event:
		return m, m.async(msg)
	case *tea.Program:
		m.program = msg
		return m, nil
	case *message.PatternsSet:
		return m, message.Cmd(m.watcher.SetPaths(*msg))
	case *lib.Watcher:
		m.watcher = msg
		err := m.watcher.Watch()
		if err != nil {
			log.Printf("failed to start watcher: %v", err)
			panic(err)
		}
		paths := m.watcher.Paths()
		log.Printf("Watcher started with %d directories:", len(paths))
		for i, path := range paths {
			log.Printf("%s", path)
			if i == 10 {
				log.Printf("and %d more...", len(paths)-i)
				break
			}
		}
		m.queue.Work()
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
			if zone.Get("pause-resume-button").InBounds(msg) {
				if m.paused {
					m.button = resumeButtonClickStyle
				} else {
					m.button = pauseButtonClickStyle
				}
				return m, nil
			}
		case tea.MouseActionRelease:
			if zone.Get("pause-resume-button").InBounds(msg) {
				if m.paused {
					m.button = resumeButtonHoverStyle
				} else {
					m.button = pauseButtonHoverStyle
				}
			} else {
				if m.paused {
					m.button = resumeButtonStyle
				} else {
					m.button = pauseButtonStyle
				}
			}
			if msg.Button == tea.MouseButtonLeft {
				if zone.Get("pause-resume-button").InBounds(msg) {
					m.paused = !m.paused
					if m.watcher == nil {
						return m, nil
					}
					if m.paused {
						m.button = resumeButtonStyle
						return m, tea.Batch(message.Callback(m.watcher.Stop), message.Cmd(message.Pause(true)))
					} else {
						m.button = pauseButtonStyle
						return m, tea.Batch(message.Callback(m.watcher.Watch), message.Cmd(message.Pause(false)))
					}
				}
			}
		case tea.MouseActionMotion:
			if zone.Get("pause-resume-button").InBounds(msg) {
				if m.paused {
					m.button = resumeButtonHoverStyle
				} else {
					m.button = pauseButtonHoverStyle
				}
			} else {
				if m.paused {
					m.button = resumeButtonStyle
				} else {
					m.button = pauseButtonStyle
				}
			}
		}
	default:
		return m, nil
	}
	return m, nil
}

func (m *Uploader) View() string {
	statusText := "Uploads Active"
	if m.paused {
		statusText = "Uploads Paused"
	}

	button := zone.Mark("pause-resume-button", m.button.Render())

	strs := []string{
		statusText,
		button,
	}

	return lipgloss.PlaceVertical(m.height-8, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, strs...))
}

// async function to prepare and call relevant [lib.Watcher.AddPath] or upload
func (m *Uploader) async(event *fsnotify.Event) func() tea.Msg {
	if m.paused {
		return nil
	}

	return func() tea.Msg {
		if m.watcher == nil {
			return errors.New("upload: program not yet initialized")
		}

		if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
			if event.Op.Has(fsnotify.Remove) {
				err := m.watcher.RemovePath(event.Name)
				if err != nil {
					return fmt.Errorf("could not remove %s to watcher: %w", event.Name, err)
				}
			} else {
				err := m.watcher.AddPath(event.Name)
				if err != nil {
					return fmt.Errorf("could not add %s to watcher: %w", event.Name, err)
				}
			}
			return nil
		}

		// TODO: try to match with settings.Config.Path
		if !strings.HasPrefix(filepath.Base(event.Name), "VRChat") || !strings.HasSuffix(event.Name, ".png") {
			return nil
		}

		dir, file := filepath.Split(event.Name)
		folder := filepath.Base(dir)
		switch {
		case event.Op.Has(fsnotify.Create):
			return logger.NewMessageTimef("A new photo was taken at %s", filepath.Join(folder, file))
		case event.Op.Has(fsnotify.Rename):
			return logger.NewMessageTimef("A new photo was moved to %s", filepath.Join(folder, file))
		case event.Op.Has(fsnotify.Write):
			return <-m.queue.Promise(event.Name)
		default:
			return nil
		}
	}
}

// the actual upload function
func (m *Uploader) upload(path string) (*vrpaws.UploadResponse, error) {
	if m.program == nil {
		return nil, errors.New("upload: program not yet initialized")
	}

	if m.config.Token == "" {
		return nil, errors.New("upload: token not yet initialized")
	}

	f, err := api.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	progress := logger.NewProgress()
	var uploadingMessage = &logger.Concat{
		Save:      false,
		Separator: " ",
		Items: []logger.Renderable{
			logger.NewMessageTime(""),
			logger.NewSpinner(),
			logger.Messagef("Uploading %s", f.Filename),
			progress,
		},
	}
	m.program.Send(uploadingMessage)
	payload := &vrpaws.UploadPayload{
		SetProgress: func(r logger.Renderable, f float64) {
			if len(uploadingMessage.Items) > 2 {
				uploadingMessage.Items[2] = r
			}
			m.program.Send(progress.SetPercent(f))
		},
		UploadPayload: &api.UploadPayload{
			Username: m.config.Username,
			UserID:   m.config.UserID,
			Token:    m.config.Token,
			File:     f,
		},
	}

	if f.Metadata != nil && f.Metadata.Author.ID != "" {
		m.config.UserID = f.Metadata.Author.ID
		err := m.config.Save()
		if err != nil {
			return nil, fmt.Errorf("saving config: %w", err)
		}
	}

	response, err := m.server.Upload(m.ctx, payload)
	if err != nil {
		uploadingMessage.Items = []logger.Renderable{
			logger.NewMessageTimef("Failed to upload %s: %v", payload.File.Filename, err),
		}
		time.Sleep(5 * time.Second)
		uploadingMessage.Items = nil
		return nil, fmt.Errorf("uploading %s: %w", payload.File.Filename, err)
	} else {
		go func() {
			time.Sleep(1 * time.Second)
			uploadingMessage.Items = []logger.Renderable{
				logger.NewMessageTime("Done!"),
				progress,
			}
			time.Sleep(3 * time.Second)
			uploadingMessage.Items = nil
		}()
		m.program.Send(logger.Concat{
			Save: true,
			Items: []logger.Renderable{
				logger.NewMessageTime("Successfully uploaded "),
				logger.NewGradientString(payload.File.Filename, gradient.PastelColors...),
				logger.Message("!"),
			},
		})
		m.program.Send(logger.Concat{
			Save: true,
			Items: []logger.Renderable{
				logger.NewMessageTime(""),
				logger.Anchor{
					Prefix: response.Image,
					OnClick: func() tea.Msg {
						return browser.OpenURL("https://vrpa.ws/photo/" + response.Image)
					},
					Message: logger.Concat{
						Save: true,
						Items: []logger.Renderable{
							logger.Message("https://vrpa.ws/photo/"),
							logger.NewGradientString(response.Image,
								lib.Random(
									gradient.BlueGreenYellow,
									gradient.PastelRainbow,
									gradient.PastelGreenBlue,
									gradient.GreenPinkBlue,
								)...,
							),
						},
					},
				},
			},
		})
		return response, nil
	}
}
