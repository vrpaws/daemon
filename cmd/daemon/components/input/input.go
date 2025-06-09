package input

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func Ask(question string, placeholder ...string) (string, error) {
	p := tea.NewProgram(initialModel(question, placeholder...))
	output, err := p.Run()
	if err != nil {
		return "", err
	}

	return output.(model).textInput.Value(), nil
}

type (
	errMsg error
)

type model struct {
	question  string
	textInput textinput.Model
	err       error
}

func initialModel(question string, placeholder ...string) model {
	ti := textinput.New()
	if len(placeholder) > 0 {
		ti.Placeholder = placeholder[0]
	}
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		question:  question,
		textInput: ti,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEsc:
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.question,
		m.textInput.View(),
	) + "\n"
}
