package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"github.com/liam-mackie/helmrunner/internal/helm"
	tea "github.com/charmbracelet/bubbletea"
)

type deployStatus int

const (
	statusPending deployStatus = iota
	statusRunning
	statusSuccess
	statusError
)

type deployResult struct {
	index int
	err   error
}

type executeModel struct {
	definitions []config.ResolvedDefinition
	statuses    []deployStatus
	errors      []string
	current     int
	allDone     bool
}

func newExecuteModel(defs []config.ResolvedDefinition) executeModel {
	return executeModel{
		definitions: defs,
		statuses:    make([]deployStatus, len(defs)),
		errors:      make([]string, len(defs)),
		current:     0,
	}
}

func (m executeModel) Init() tea.Cmd {
	if len(m.definitions) == 0 {
		return func() tea.Msg { return executeDoneMsg{} }
	}
	return m.runNext(0)
}

func (m executeModel) runNext(index int) tea.Cmd {
	def := m.definitions[index]
	return func() tea.Msg {
		err := helm.Install(context.Background(), def)
		return deployResult{index: index, err: err}
	}
}

func (m executeModel) Update(msg tea.Msg) (executeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case deployResult:
		if msg.err != nil {
			m.statuses[msg.index] = statusError
			m.errors[msg.index] = msg.err.Error()
		} else {
			m.statuses[msg.index] = statusSuccess
		}

		next := msg.index + 1
		if next < len(m.definitions) {
			m.current = next
			m.statuses[next] = statusRunning
			return m, m.runNext(next)
		}

		m.allDone = true
		return m, nil

	case tea.KeyMsg:
		if m.allDone && (msg.String() == "enter" || msg.String() == "q") {
			return m, func() tea.Msg { return executeDoneMsg{} }
		}
	}

	return m, nil
}

func (m executeModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Deploying..."))
	b.WriteString("\n\n")

	for i, def := range m.definitions {
		var icon string
		switch m.statuses[i] {
		case statusPending:
			icon = dimStyle.Render("○")
		case statusRunning:
			icon = selectedStyle.Render("●")
		case statusSuccess:
			icon = successStyle.Render("✓")
		case statusError:
			icon = errorStyle.Render("✗")
		}

		line := fmt.Sprintf(" %s %s → %s/%s", icon, def.Name, def.Namespace, def.Release)
		b.WriteString(line + "\n")

		if m.statuses[i] == statusError {
			b.WriteString("   " + errorStyle.Render(m.errors[i]) + "\n")
		}
	}

	if m.allDone {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("enter/q: exit"))
	}

	return b.String()
}
