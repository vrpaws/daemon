//go:build windows

package systray

import (
	_ "embed"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/getlantern/systray"

	"vrc-moments/cmd/daemon/components/message"
)

//go:embed icon.ico
var trayIcon []byte

func Run(program *tea.Program) {
	systray.Run(onReady(program), func() {})
}

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
