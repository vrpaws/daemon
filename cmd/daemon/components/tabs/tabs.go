// Package tabs is a component for rendering a submissions of tabs using [bubblezone].
// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this [source code] is governed by the [MIT license] that can be found in
// the [LICENSE] file.
//
// [bubblezone]: https://github.com/lrstanley/bubblezone
// [source code]: https://github.com/lrstanley/bubblezone/blob/master/examples/full-lipgloss/tabs.go
// [MIT license]: https://opensource.org/license/mit
// [LICENSE]: https://github.com/lrstanley/bubblezone/blob/master/LICENSE
package tabs

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()
)

var (
	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)

	activeTab = tab.Copy().Border(activeTabBorder, true)

	tabGap = tab.Copy().
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
)

type Tabs struct {
	prefix string
	height int
	width  int

	activeIndex uint8
	Items       []string
	out         []string
	extra       *string
	spinner     spinner.Model
}

func (m Tabs) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Tabs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		for i, item := range m.Items {
			// Check each item to see if it's in bounds.
			if zone.Get(m.prefix + item).InBounds(msg) {
				m.activeIndex = uint8(i)
				break
			}
		}

		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+shift+tab":
			return m.Previous(), nil
		case "ctrl+tab":
			return m.Next(), nil
		default:
			return m, nil
		}
	}
	return m, nil
}

func (m Tabs) Next() Tabs {
	m.activeIndex = (m.activeIndex + 1) % uint8(len(m.Items))
	return m
}

func (m Tabs) Previous() Tabs {
	m.activeIndex = (m.activeIndex - 1) % uint8(len(m.Items))
	return m
}

func (m Tabs) Active() string {
	return m.Items[m.activeIndex]
}

func (m Tabs) Index() uint8 {
	return m.activeIndex
}

func (m Tabs) View() string {
	if m.out == nil {
		m.out = make([]string, len(m.Items)+1)
	}
	m.out[0] = tab.Render(m.spinner.View())
	for i, item := range m.Items {
		// Make sure to mark each tab when rendering.
		if uint8(i) == m.activeIndex {
			m.out[i+1] = zone.Mark(m.prefix+item, activeTab.Render(item))
		} else {
			m.out[i+1] = zone.Mark(m.prefix+item, tab.Render(item))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, m.out...)
	var username string
	if m.extra != nil {
		username = activeTab.Render(*m.extra)
	}
	gap := tabGap.Render(strings.Repeat(" ", max(0, m.width-calculateWidths(row, username))))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap, username)
	return row
}

func calculateWidths(items ...string) int {
	var total int
	for _, item := range items {
		total += lipgloss.Width(item) + 2
	}
	return total
}

func New(items []string, username *string) Tabs {
	return Tabs{
		prefix:  "tab",
		Items:   items,
		out:     make([]string, len(items)+1),
		extra:   username,
		spinner: spinner.New(spinner.WithSpinner(spinner.Moon)),
	}
}
