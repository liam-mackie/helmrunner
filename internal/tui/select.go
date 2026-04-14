package tui

import (
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type selectModel struct {
	definitions []config.Definition
	cursor      int
	selected    map[int]bool
}

func newSelectModel(defs []config.Definition) selectModel {
	return selectModel{
		definitions: defs,
		selected:    make(map[int]bool),
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (selectModel, tea.Cmd) {
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
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			var selected []config.Definition
			for i, def := range m.definitions {
				if m.selected[i] {
					selected = append(selected, def)
				}
			}
			return m, func() tea.Msg {
				return selectDoneMsg{selected: selected}
			}
		case "q", "esc":
			return m, func() tea.Msg {
				return selectDoneMsg{selected: nil}
			}
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select definitions to deploy"))
	b.WriteString("\n\n")

	for i, def := range m.definitions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checked := "[ ]"
		if m.selected[i] {
			checked = "[x]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, checked, def.Name)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("space: toggle • enter: confirm • q: quit"))

	return b.String()
}
