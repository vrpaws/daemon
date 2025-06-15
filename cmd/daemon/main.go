package main

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"vrc-moments/cmd/daemon/app"
	"vrc-moments/cmd/daemon/components/logger"
	"vrc-moments/cmd/daemon/components/settings"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/vrc"
)

func main() {
	config, err := lib.DecodeFromFile[*settings.Config](lib.ConfigPath)
	if err != nil {
		config = new(settings.Config)
	}

	config.Username = cmp.Or(os.Getenv("USERNAME"), config.Username, "Unknown")
	config.Token = cmp.Or(os.Getenv("TOKEN"), config.Token)
	config.Path = cmp.Or(os.Getenv("PATH"), config.Path, "~/Pictures/VRChat")
	config.LastWorld = cmp.Or(config.LastWorld, "Unknown")

	if errors.Is(err, os.ErrNotExist) {
		err = config.Save()
	}

	remote := getRemote(config)
	usernameErr := getUsername(config)
	roomErr := getRoom(config)

	model := app.NewModel(remote, config)
	program := model.Run()
	defer lib.LogOutput(&model)()

	program.Send(program)
	program.Send(err)
	program.Send(usernameErr)
	program.Send(roomErr)
	program.Send(logger.MessageTime{Message: "Started up!"})
	if config.Username != "Unknown" {
		username := lib.GradientText(config.Username, "#00E2FD", "#6D90FA", "#FF22EE", "#FF8D7A", "#FFC851")
		program.Send(logger.MessageTime{Message: "Hello " + username})
	}

	program.Wait()
}

func getRemote(config *settings.Config) *url.URL {
	server := os.Getenv("SERVER")
	if server != "" {
		parsed, err := url.Parse(server)
		if err == nil {
			config.Server = parsed.String()
			return parsed
		}
	}

	if config.Server != "" {
		parsed, err := url.Parse(server)
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
