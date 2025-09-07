package main

import (
	"cmp"
	_ "embed"
	"errors"
	"fmt"
	"log"
	_ "net/http/pprof"
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
	"vrc-moments/cmd/daemon/components/systray"
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
	patterns, patternErr := getPatterns(config, "")

	if errors.Is(saveError, os.ErrNotExist) {
		saveError = config.Save()
	}

	model := app.NewModel(remote, config)
	program := model.Run()
	logFile, done := lib.LogOutput(&model)
	defer done()
	go systray.Run(program)

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
	roomName, err := vrc.ExtractCurrentRoomName(vrc.DefaultLogPath)
	if err != nil {
		return fmt.Errorf("error getting room name: %w", err)
	} else {
		config.SetRoom(roomName)
	}

	return nil
}

var retries = 0

func getPatterns(config *settings.Config, extra string) ([]string, error) {
	if config.Path == "Unset" {
		startDir := "."
		homedir, err := os.UserHomeDir()
		if err == nil {
			startDir = homedir
			if vrcDefault := filepath.Join(startDir, "Pictures", "VRChat"); lib.FileExists(vrcDefault) {
				startDir = vrcDefault
			}
		}

		if extra == "" && config.Path != "." {
			extra = fmt.Sprintf("\nDetected directory: %s", startDir)
		}

		directory, err := dialog.Directory().SetStartDir(startDir).Title("Choose your VRChat Photos folder" + extra).Browse()
		if err != nil {
			if errors.Is(err, dialog.ErrCancelled) && retries < 3 {
				retries = 3
				return getPatterns(config, "\nDid you mean to cancel? Cancel again to abort.")
			}
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

	prints := "!" + filepath.Join(config.Path, "Prints", "***")
	stickers := "!" + filepath.Join(config.Path, "Stickers", "***")
	emojis := "!" + filepath.Join(config.Path, "Emoji", "***")
	config.Path = filepath.Join(config.Path, "***")
	config.Path = strings.ReplaceAll(config.Path, `\`, "/")

	patterns, err := lib.ExpandPatterns(true, false, 64, config.Path, prints, stickers, emojis)
	if err != nil {
		if retries < 3 {
			retries++

			log.Printf(`Could not expand folder "%s", trying again: %v`, config.Path, err)
			err := strings.Split(err.Error(), ": ")
			sprintf := fmt.Sprintf("\nInvalid folder %s: %s", config.Path, err[len(err)-1])

			config.Path = "Unset"
			return getPatterns(config, sprintf)
		}
		return nil, err
	}

	if len(patterns) == 0 {
		log.Printf(`Invalid folder, trying again: "%v"`, config.Path)
		if retries < 3 {
			retries++
			return getPatterns(config, fmt.Sprintf("\nInvalid folder: %s", config.Path))
		}
	}

	return patterns, config.Save()
}
