package main

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
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
	config, err := lib.DecodeFromFile[*settings.Config](lib.ConfigPath)
	if err != nil {
		config = new(settings.Config)
	}

	config.Username = cmp.Or(os.Getenv("VRPAWS_USERNAME"), config.Username, "Unknown")
	config.Token = cmp.Or(os.Getenv("VRPAWS_TOKEN"), config.Token)
	config.Path = cmp.Or(os.Getenv("VRPAWS_PATH"), config.Path, "Unset")
	config.LastWorld = cmp.Or(config.LastWorld, "Unknown")

	remote := getRemote(config)
	usernameErr := getUsername(config)
	roomErr := getRoom(config)

	if errors.Is(err, os.ErrNotExist) {
		err = config.Save()
	}

	model := app.NewModel(remote, config)
	program := model.Run()
	logFile, done := lib.LogOutput(&model)
	defer done()

	program.Send(logFile)
	program.Send(program)
	program.Send(err)
	program.Send(usernameErr)
	program.Send(roomErr)

	patterns, patternErr := getPatterns(config)
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
				logger.NewGradientString(config.Username, 250*time.Millisecond, gradient.PastelColors...),
				logger.Message("!"),
			},
			Separator: "",
			Save:      true,
		})
	}
	if config.Token == "" {
		program.Send(logger.NewMessageTime("Token is unset, please go to the settings tab to change it"))
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
		Path:   "vrpa.ws",
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

		config.Path = filepath.Join("~", strings.TrimPrefix(directory, homedir), "***")
	}

	return lib.ExpandPatterns(true, false, strings.TrimRight(config.Path, "/*")+"/***")
}
