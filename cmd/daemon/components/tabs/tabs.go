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
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"vrc-moments/cmd/daemon/components/message"
	"vrc-moments/pkg/gradient"
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

	normalTab = lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(highlight).
			Padding(0, 1)

	activeTab = normalTab.Border(activeTabBorder, true)

	tabGap = normalTab.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
)

type Tabs struct {
	items []*tabItem

	prefix string
	height int
	width  int

	activeIndex uint8
	out         []string
	extra       string
	spinner     spinner.Model
}

type tabItem struct {
	prefix  string
	content string
	style   lipgloss.Style
}

func New(items []string, username string) Tabs {
	const prefix = "tab"

	tabs := Tabs{
		prefix:  prefix,
		out:     make([]string, len(items)+1),
		items:   make([]*tabItem, len(items)),
		extra:   username,
		spinner: spinner.New(spinner.WithSpinner(spinner.Moon)),
	}

	gradient.Global.New(
		username,
		gradient.StepsFromDuration(lipgloss.Width(username), time.Second, 60),
		gradient.BlueGreenYellow...,
	)
	for i, content := range items {
		tabs.items[i] = &tabItem{
			prefix:  prefix + content,
			content: content,
			style:   normalTab,
		}
	}

	return tabs
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
		switch msg.Action {
		case tea.MouseActionPress:
			for i, item := range m.items {
				if zone.Get(item.prefix).InBounds(msg) {
					m.activeIndex = uint8(i)
					break
				}
			}
		default:
			for _, item := range m.items {
				if zone.Get(item.prefix).InBounds(msg) {
					item.style = item.style.Foreground(lipgloss.Color("#ffb3e3")).Bold(true)
				} else {
					item.style = item.style.UnsetForeground().UnsetBold()
				}
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
	case message.UsernameSet:
		s := string(msg)
		if m.extra != s {
			gradient.Global.New(
				s,
				gradient.StepsFromDuration(lipgloss.Width(s), time.Second, 60),
				gradient.BlueGreenYellow...,
			)
		}
		m.extra = s
		return m, nil
	}
	return m, nil
}

func (m Tabs) Next() Tabs {
	m.activeIndex = (m.activeIndex + 1) % uint8(len(m.items))
	return m
}

func (m Tabs) Previous() Tabs {
	a := m.activeIndex - 1
	b := uint8(len(m.items))
	m.activeIndex = (a%b + b) % b
	return m
}

func (m Tabs) Active() string {
	return m.items[m.activeIndex].content
}

func (m Tabs) Index() uint8 {
	return m.activeIndex
}

func (m Tabs) View() string {
	if m.out == nil {
		m.out = make([]string, len(m.items)+1)
	}

	m.out[0] = normalTab.Render(m.spinner.View())
	for i, item := range m.items {
		// Make sure to mark each tab when rendering.
		if m.activeIndex == uint8(i) {
			m.out[i+1] = zone.Mark(item.prefix, item.style.Border(activeTabBorder, true).Render(item.content))
		} else {
			m.out[i+1] = zone.Mark(item.prefix, item.style.Border(tabBorder).Render(item.content))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, m.out...)
	username := activeTab.Render(gradient.Global.RenderCurrent(m.extra))
	gap := tabGap.Render(strings.Repeat(" ", max(0, m.width-calculateWidths(row, username))))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap, username)
	return row
}

func (m Tabs) Login() string {
	if m.out == nil {
		m.out = make([]string, len(m.items)+1)
	}

	m.out[0] = normalTab.Render(m.spinner.View())
	for i, item := range m.items {
		m.out[i+1] = zone.Mark(item.prefix, item.style.Border(activeTabBorder, true).Render("Login"))
		break
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, m.out...)
	username := activeTab.Render(gradient.Global.RenderCurrent(m.extra))
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
