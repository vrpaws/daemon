//go:build windows

package lib

import (
	"log"
	"os"

	"golang.org/x/sys/windows"
)

func DisableQuickEdit() {
	winConsole := windows.Handle(os.Stdin.Fd())

	var mode uint32
	err := windows.GetConsoleMode(winConsole, &mode)
	if err != nil {
		log.Println(err)
	}
	log.Printf("%d", mode)

	// Disable this mode
	mode &^= windows.ENABLE_QUICK_EDIT_MODE

	// Enable this mode
	mode |= windows.ENABLE_EXTENDED_FLAGS

	log.Printf("%d", mode)
	err = windows.SetConsoleMode(winConsole, mode)
	if err != nil {
		log.Println(err)
	}
}
