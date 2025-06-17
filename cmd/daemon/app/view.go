package app

import (
	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"vrc-moments/cmd/daemon/components/tabs"
)

func (m *Model) View() string {
	return m.Render()
}

func (m *Model) Render() string {
	top := stick.New(m.window.Width, 3)
	top.SetRows(
		[]*stick.Row{
			top.NewRow().AddCells(
				stick.NewCell(1, 1).SetContent(m.tabs.View()),
			),
		})

	var renderers = []Renderer{
		m.logger.View,
		m.uploader.View,
		m.settings.View,
	}

	if m.me == nil {
		renderers = []Renderer{m.login.View}
	}

	return zone.Scan(lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.PlaceHorizontal(m.window.Width, lipgloss.Left, top.Render()),
		lipgloss.PlaceHorizontal(
			m.window.Width, lipgloss.Center,
			lipgloss.PlaceVertical(
				m.window.Height-6, lipgloss.Top,
				renderers[int(m.tabs.(tabs.Tabs).Index())%len(renderers)](),
			)),
		lipgloss.PlaceHorizontal(m.window.Width, lipgloss.Right, m.footer.View()),
	))
}

type Renderer = func() string

func empty() string {
	return "empty"
}
