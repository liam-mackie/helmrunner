package tui

import (
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type inputModel struct {
	definitions []config.Definition
	defIndex    int
	varIndex    int
	varSets     []map[string]string
	textInput   textinput.Model
	done        bool
}

func newInputModel(defs []config.Definition) inputModel {
	varSets := make([]map[string]string, len(defs))
	for i := range varSets {
		varSets[i] = make(map[string]string)
	}

	m := inputModel{
		definitions: defs,
		varSets:     varSets,
	}
	m.advanceToNextVariable()
	return m
}

func (m *inputModel) advanceToNextVariable() {
	for m.defIndex < len(m.definitions) {
		def := m.definitions[m.defIndex]
		if m.varIndex < len(def.Variables) {
			v := def.Variables[m.varIndex]
			ti := textinput.New()
			ti.Placeholder = v.Default
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 40
			m.textInput = ti
			return
		}
		m.defIndex++
		m.varIndex = 0
	}
	m.done = true
}

func (m inputModel) currentVariable() config.Variable {
	return m.definitions[m.defIndex].Variables[m.varIndex]
}

func (m inputModel) Init() tea.Cmd {
	if m.done {
		return func() tea.Msg {
			return inputDoneMsg{varSets: m.varSets}
		}
	}
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	if m.done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			v := m.currentVariable()
			value := m.textInput.Value()
			if value == "" {
				value = v.Default
			}
			m.varSets[m.defIndex][v.Name] = value
			m.varIndex++
			m.advanceToNextVariable()
			if m.done {
				return m, func() tea.Msg {
					return inputDoneMsg{varSets: m.varSets}
				}
			}
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	def := m.definitions[m.defIndex]
	v := m.currentVariable()

	b.WriteString(titleStyle.Render(fmt.Sprintf("Variables for: %s", def.Name)))
	b.WriteString("\n\n")

	b.WriteString(promptStyle.Render(v.Description))
	b.WriteString("\n")
	if v.Default != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("default: %s", v.Default)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("enter: accept • ctrl+c: abort"))

	return b.String()
}
