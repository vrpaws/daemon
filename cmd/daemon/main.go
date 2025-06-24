package main

import (
	"cmp"
	_ "embed"
	"errors"
	"fmt"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
	_ "github.com/joho/godotenv/autoload"
	"github.com/sqweek/dialog"

	"vrc-moments/cmd/daemon/app"
	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/cmd/daemon/components/settings"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/gradient"
	"vrc-moments/pkg/vrc"
)

func main() {
	config, saveError := lib.DecodeFromFile[*settings.Config](lib.ConfigPath)
	if saveError != nil {
		config = new(settings.Config)
	}

	config.Username = cmp.Or(os.Getenv("VRPAWS_USERNAME"), config.Username, "Unknown")
	config.Token = cmp.Or(os.Getenv("VRPAWS_TOKEN"), config.Token)
	config.Path = cmp.Or(os.Getenv("VRPAWS_PATH"), config.Path, "Unset")
	config.LastWorld = cmp.Or(config.LastWorld, "Unknown")

	remote := getRemote(config)
	usernameErr := getUsername(config)
	roomErr := getRoom(config)
	patterns, patternErr := getPatterns(config)

	if errors.Is(saveError, os.ErrNotExist) {
		saveError = config.Save()
	}

	model := app.NewModel(remote, config)
	program := model.Run()
	logFile, done := lib.LogOutput(&model)
	defer done()
	go systray.Run(onReady(program), func() {})

	program.Send(logFile)
	program.Send(program)
	program.Send(saveError)
	program.Send(usernameErr)
	program.Send(roomErr)
	program.Send(patternErr)

	program.Send(lib.NewWatcher(
		patterns,
		5*time.Second,
		message.Invoke[*fsnotify.Event](program.Send),
	))

	program.Send(logger.NewMessageTime("Started up!"))
	if config.Username != "Unknown" {
		program.Send(logger.Concat{
			Items: []logger.Renderable{
				logger.NewMessageTime("Hello "),
				logger.NewGradientString(config.Username, gradient.PastelColors...),
				logger.Message("!"),
			},
			Separator: "",
			Save:      true,
		})
	}

	program.Wait()
}

func getRemote(config *settings.Config) *url.URL {
	server := os.Getenv("VRPAWS_SERVER")
	if server != "" {
		parsed, err := url.Parse(server)
		if err == nil {
			config.Server = parsed.String()
			return parsed
		}
	}

	if config.Server != "" {
		parsed, err := url.Parse(config.Server)
		if err == nil {
			config.Server = parsed.String()
			return parsed
		}
	}

	remote := &url.URL{
		Scheme: "https",
		Host:   "vrpa.ws",
	}
	config.Server = remote.String()
	return remote
}

func getUsername(config *settings.Config) error {
	if config.Username != "Unknown" {
		return nil
	}
	usernames, err := vrc.GetUsername(vrc.DefaultLogPath)
	if err != nil {
		return err
	} else if len(usernames) > 0 {
		config.Username = usernames[0]
	}

	return nil
}

func getRoom(config *settings.Config) error {
	if config.LastWorld != "Unknown" {
		return nil
	}
	roomName, err := vrc.ExtractCurrentRoomName(vrc.DefaultLogPath)
	if err != nil {
		return fmt.Errorf("error getting room name: %w", err)
	} else {
		config.SetRoom(roomName)
	}

	return nil
}

func getPatterns(config *settings.Config) ([]string, error) {
	if config.Path == "Unset" {
		startDir := "."
		homedir, err := os.UserHomeDir()
		if err == nil {
			startDir = homedir
			if vrcDefault := filepath.Join(startDir, "Pictures", "VRChat"); lib.FileExists(vrcDefault) {
				startDir = vrcDefault
			}
		}
		directory, err := dialog.Directory().SetStartDir(startDir).Title("Choose your VRChat Photos folder").Browse()
		if err != nil {
			return nil, fmt.Errorf("error getting directory: %w", err)
		}

		if strings.HasPrefix(directory, homedir) {
			directory = filepath.Join("~", strings.TrimPrefix(directory, homedir))
		}

		config.Path = directory
	}

	if strings.HasSuffix(config.Path, "***") {
		config.Path = strings.TrimRight(config.Path, `*\/`+string(filepath.Separator))
	}

	prints := filepath.Join("!"+config.Path, "Prints", "***")
	stickers := filepath.Join("!"+config.Path, "Stickers", "***")
	config.Path = filepath.Join(config.Path, "***")
	config.Path = strings.ReplaceAll(config.Path, `\`, "/")

	patterns, err := lib.ExpandPatterns(true, false, config.Path, prints, stickers)
	if err != nil {
		return nil, err
	}

	return patterns, config.Save()
}

//go:embed src/icon.ico
var trayIcon []byte

func onReady(program *tea.Program) func() {
	return func() {
		systray.SetIcon(trayIcon)
		systray.SetTitle("VRPaws Client")
		systray.SetTooltip("VRPaws Client")

		showItem := systray.AddMenuItem("Show Window", "Restore the TUI window")
		showItem.Disable()
		loginItem := systray.AddMenuItem("Login", "Login to VRPaws client")
		systray.AddSeparator()
		pauseItem := systray.AddMenuItem("Pause", "Pause the app")
		browseItem := systray.AddMenuItem("Set VRChat Folder", "Set the VRChat Pictures Folder")

		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Exit", "Exit the app")
		program.Send(message.SetPause(func(paused bool) {
			if paused {
				pauseItem.Checked()
			} else {
				pauseItem.Uncheck()
			}
		}))
		program.Send(message.SetUsername(func(username string) {
			loginItem.SetTitle(fmt.Sprintf("Logged in (%s)", username))
			loginItem.Check()
		}))

		go func() {
			for {
				select {
				case <-browseItem.ClickedCh:
					program.Send(message.BrowseRequest{})
				case <-loginItem.ClickedCh:
					program.Send(message.LoginRequest{})
				case <-showItem.ClickedCh:
					continue
				case <-pauseItem.ClickedCh:
					if pauseItem.Checked() {
						pauseItem.Uncheck()
					} else {
						pauseItem.Check()
					}
					program.Send(message.Pause(pauseItem.Checked()))
				case <-exitItem.ClickedCh:
					program.Send(tea.Quit())
					return
				}
			}
		}()
	}
}
