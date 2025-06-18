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

	var renderers = []Renderer{
		m.logger.View,
		m.uploader.View,
		m.settings.View,
	}

	row := top.NewRow()
	if m.me == nil {
		row.AddCells(stick.NewCell(1, 1).SetContent(m.tabs.(tabs.Tabs).Login()))
		renderers = []Renderer{m.login.View}
	} else {
		row.AddCells(stick.NewCell(1, 1).SetContent(m.tabs.View()))
	}
	top.SetRows([]*stick.Row{row})

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
