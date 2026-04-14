package tui

import (
	"github.com/liam-mackie/helmrunner/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateSelecting state = iota
	stateInputting
	stateReviewing
	stateExecuting
	stateDone
)

type Result struct {
	Definitions []config.ResolvedDefinition
	Aborted     bool
}

type Model struct {
	state        state
	definitions  []config.Definition
	templateMode bool

	selectModel  selectModel
	inputModel   inputModel
	reviewModel  reviewModel
	executeModel executeModel

	result Result
	width  int
	height int
}

func New(defs []config.Definition, templateMode bool) Model {
	m := Model{
		state:        stateSelecting,
		definitions:  defs,
		templateMode: templateMode,
	}
	m.selectModel = newSelectModel(defs)
	return m
}

func (m Model) Init() tea.Cmd {
	return m.selectModel.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.result.Aborted = true
			return m, tea.Quit
		}
	}

	switch m.state {
	case stateSelecting:
		return m.updateSelecting(msg)
	case stateInputting:
		return m.updateInputting(msg)
	case stateReviewing:
		return m.updateReviewing(msg)
	case stateExecuting:
		return m.updateExecuting(msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateSelecting:
		return m.selectModel.View()
	case stateInputting:
		return m.inputModel.View()
	case stateReviewing:
		return m.reviewModel.View()
	case stateExecuting:
		return m.executeModel.View()
	case stateDone:
		return ""
	}
	return ""
}

func (m Model) GetResult() Result {
	return m.result
}

// Messages for state transitions
type selectDoneMsg struct {
	selected []config.Definition
}

type inputDoneMsg struct {
	varSets []map[string]string
}

type reviewDoneMsg struct {
	definitions []config.ResolvedDefinition
}

type executeDoneMsg struct{}

func (m Model) updateSelecting(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.selectModel, cmd = m.selectModel.Update(msg)

	if done, ok := msg.(selectDoneMsg); ok {
		if len(done.selected) == 0 {
			m.result.Aborted = true
			return m, tea.Quit
		}
		m.inputModel = newInputModel(done.selected)
		m.state = stateInputting
		return m, m.inputModel.Init()
	}

	return m, cmd
}

func (m Model) updateInputting(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.inputModel, cmd = m.inputModel.Update(msg)

	if done, ok := msg.(inputDoneMsg); ok {
		selected := m.inputModel.definitions
		resolved := make([]config.ResolvedDefinition, len(selected))
		for i, def := range selected {
			r, err := config.Resolve(def, done.varSets[i])
			if err != nil {
				resolved[i] = config.ResolvedDefinition{
					Name:      def.Name,
					Release:   def.Release,
					Namespace: def.Namespace,
					Chart:     def.Chart,
					Values:    def.Values,
				}
				continue
			}
			resolved[i] = r
		}
		m.reviewModel = newReviewModel(resolved, m.templateMode)
		m.state = stateReviewing
		return m, nil
	}

	return m, cmd
}

func (m Model) updateReviewing(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.reviewModel, cmd = m.reviewModel.Update(msg)

	if done, ok := msg.(reviewDoneMsg); ok {
		if m.templateMode {
			m.result.Definitions = done.definitions
			m.state = stateDone
			return m, tea.Quit
		}
		m.executeModel = newExecuteModel(done.definitions)
		m.state = stateExecuting
		return m, m.executeModel.Init()
	}

	return m, cmd
}

func (m Model) updateExecuting(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.executeModel, cmd = m.executeModel.Update(msg)

	if _, ok := msg.(executeDoneMsg); ok {
		m.state = stateDone
		return m, tea.Quit
	}

	return m, cmd
}
