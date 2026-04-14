package tui

import (
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type editField int

const (
	editNone editField = iota
	editRelease
	editNamespace
)

type reviewModel struct {
	definitions  []config.ResolvedDefinition
	cursor       int
	editing      editField
	textInput    textinput.Model
	templateMode bool
}

func newReviewModel(defs []config.ResolvedDefinition, templateMode bool) reviewModel {
	return reviewModel{
		definitions:  defs,
		templateMode: templateMode,
	}
}

func (m reviewModel) Update(msg tea.Msg) (reviewModel, tea.Cmd) {
	if m.editing != editNone {
		return m.updateEditing(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.definitions)-1 {
				m.cursor++
			}
		case "r":
			ti := textinput.New()
			ti.SetValue(m.definitions[m.cursor].Release)
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 40
			m.textInput = ti
			m.editing = editRelease
			return m, textinput.Blink
		case "n":
			ti := textinput.New()
			ti.SetValue(m.definitions[m.cursor].Namespace)
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 40
			m.textInput = ti
			m.editing = editNamespace
			return m, textinput.Blink
		case "enter":
			return m, func() tea.Msg {
				return reviewDoneMsg{definitions: m.definitions}
			}
		case "q", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m reviewModel) updateEditing(msg tea.Msg) (reviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := m.textInput.Value()
			if m.editing == editRelease {
				m.definitions[m.cursor].Release = value
			} else {
				m.definitions[m.cursor].Namespace = value
			}
			m.editing = editNone
			return m, nil
		case "esc":
			m.editing = editNone
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m reviewModel) chartString(chart config.Chart) string {
	if chart.Source != "" {
		return chart.Source
	}
	s := chart.Name
	if chart.Version != "" {
		s += "@" + chart.Version
	}
	return s
}

func (m reviewModel) View() string {
	var b strings.Builder

	action := "deploy"
	if m.templateMode {
		action = "template"
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("Review — %s the following:", action)))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "  %-20s %-25s %-20s %s\n",
		dimStyle.Render("NAME"),
		dimStyle.Render("RELEASE"),
		dimStyle.Render("NAMESPACE"),
		dimStyle.Render("CHART"))
	b.WriteString("\n")

	for i, def := range m.definitions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%-20s %-25s %-20s %s",
			cursor, def.Name, def.Release, def.Namespace, m.chartString(def.Chart))

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	if m.editing != editNone {
		b.WriteString("\n")
		label := "Release"
		if m.editing == editNamespace {
			label = "Namespace"
		}
		b.WriteString(promptStyle.Render(fmt.Sprintf("Edit %s: ", label)))
		b.WriteString(m.textInput.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("r: edit release • n: edit namespace • enter: confirm • q: quit"))

	return b.String()
}
