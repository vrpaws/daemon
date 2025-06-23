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

type (
	Pause bool
)

type (
	LoginRequest  struct{}
	ManualRequest struct{}
)

func Cmd[T any](v T) tea.Cmd {
	return func() tea.Msg {
		return v
	}
}

func Msg[T any]() tea.Cmd {
	return func() tea.Msg {
		var t T
		return t
	}
}

func Callback[T any](f func() T) tea.Cmd {
	return func() tea.Msg {
		return f()
	}
}

func CallbackValue[V any, T any](f func(V) T, v V) tea.Cmd {
	return func() tea.Msg {
		return f(v)
	}
}

func Invoke[T any](f func(tea.Msg)) func(T) {
	return func(v T) {
		f(v)
	}
}

func Cmds[T any](v ...T) []tea.Cmd {
	if len(v) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, len(v))
	for i := range v {
		cmds[i] = Cmd(v[i])
	}

	return cmds
}
