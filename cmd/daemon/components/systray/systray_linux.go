//go:build !windows

package systray

import (
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vrc-moments/cmd/daemon/components/logger"
)

// Deprecated: Tray icon not available on this platform
func Run(program *tea.Program) {
	program.Send(logger.NewAutoDelete(logger.NewMessageTimef("Tray icon not available on this platform (%s)", runtime.GOOS), 5*time.Second))
}
