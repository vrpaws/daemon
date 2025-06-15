package message

import (
	tea "github.com/charmbracelet/bubbletea"
)

type (
	UsernameSet string
	TokenSet    string
	PathSet     string
	PatternsSet []string
	ServerSet   string
	RoomSet     string
)

func Cmd[T any](v T) func() tea.Msg {
	return func() tea.Msg {
		return v
	}
}
