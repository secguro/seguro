package fix

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/secguro/secguro-cli/pkg/functional"
	"github.com/secguro/secguro-cli/pkg/output"
	"github.com/secguro/secguro-cli/pkg/scan"
	"github.com/secguro/secguro-cli/pkg/types"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

type item struct {
	title          string
	description    string
	unifiedFinding types.UnifiedFinding
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type listKeyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleSpinner: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle spinner"),
		),
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
	}
}

type model struct {
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
}

func newModel(directoryToScan string, unifiedFindingsNotIgnored []types.UnifiedFinding) model {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	// Make initial list of items
	items := functional.MapWithIndex(unifiedFindingsNotIgnored,
		func(unifiedFinding types.UnifiedFinding, i int) list.Item {
			return item{
				title:          output.GetFindingTitle(i),
				description:    output.GetFindingBody(false, unifiedFinding),
				unifiedFinding: unifiedFinding,
			}
		})

	// Setup list
	delegate := newItemDelegate(directoryToScan, delegateKeys)
	findingsList := list.New(items, delegate, 0, 0)
	findingsList.Title = "Findings"
	findingsList.Styles.Title = titleStyle
	findingsList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleSpinner,
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
		}
	}

	return model{
		list:         findingsList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint: ireturn
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleSpinner):
			cmd := m.list.ToggleSpinner()
			return m, cmd

		case key.Matches(msg, m.keys.toggleTitleBar):
			v := !m.list.ShowTitle()
			m.list.SetShowTitle(v)
			m.list.SetShowFilter(v)
			m.list.SetFilteringEnabled(v)

			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.list.SetShowStatusBar(!m.list.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return appStyle.Render(m.list.View())
}

var actionPastFixSelection func() error = nil

var showProblemsList func() error = nil

func CommandFix(directoryToScan string, gitMode bool, disabledDetectors []string) error {
	unifiedFindingsNotIgnored, _, err := scan.PerformScan(directoryToScan, gitMode, disabledDetectors)
	if err != nil {
		return err
	}

	showProblemsList = func() error {
		model := newModel(directoryToScan, unifiedFindingsNotIgnored)
		if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
			return err
		}

		if actionPastFixSelection != nil {
			actionToExecute := actionPastFixSelection
			actionPastFixSelection = nil
			return actionToExecute()
		}

		return nil
	}

	return showProblemsList()
}

func fixUnifiedFinding(directoryToScan string,
	previousStep func() error, unifiedFinding types.UnifiedFinding) error {
	if scan.IsSecretDetectionRule(unifiedFinding.Rule) {
		return fixSecret(directoryToScan, previousStep, unifiedFinding)
	}

	return fixProblemViaAi(directoryToScan, previousStep, unifiedFinding)
}
