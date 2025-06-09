package footer

import (
	"fmt"
	"strings"

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

type Model struct {
	prefix string
	height int
	width  int

	Items []*string
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		for i, item := range m.Items {
			if item == nil {
				continue
			}
			if zone.Get(fmt.Sprintf("%s-%d", m.prefix, i)).InBounds(msg) {
				return m, nil
			}
		}

		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	left := lipgloss.JoinHorizontal(lipgloss.Top, activeTab.Render(strings.TrimSpace(*m.Items[0])))
	right := lipgloss.JoinHorizontal(lipgloss.Top, activeTab.Render(strings.TrimSpace(*m.Items[1])))
	gap := tabGap.Render(strings.Repeat(" ", min(8, max(0, m.width-lipgloss.Width(left)-2-lipgloss.Width(right)-8))))
	leftGap := tabGap.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-2-lipgloss.Width(right)-2-lipgloss.Width(gap))))
	return lipgloss.JoinHorizontal(lipgloss.Bottom, leftGap, left, gap, right)
}

func New(items []*string) Model {
	if len(items) != 2 {
		panic("footer.New: items must have exactly two items")
	}
	return Model{
		prefix: "tab",
		Items:  items,
	}
}
