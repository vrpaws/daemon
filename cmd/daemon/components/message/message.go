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

func Cmd[T any](v T) tea.Cmd {
	return func() tea.Msg {
		return v
	}
}

func Invoke[T any](f func(tea.Msg)) func(T) {
	return func(v T) {
		f(v)
	}
}

func Cmds(v ...any) []tea.Cmd {
	if len(v) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, len(v))
	for i := range v {
		cmds[i] = Cmd(v[i])
	}

	return cmds
}
