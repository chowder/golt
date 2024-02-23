package golt

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"strings"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

func (i Account) FilterValue() string { return i.DisplayName }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Account)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.DisplayName)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type tuiModel struct {
	list     list.Model
	Choice   *Account
	quitting bool
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(Account)
			if ok {
				m.Choice = &i
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	if m.Choice != nil {
		return quitTextStyle.Render(fmt.Sprintf("Selected account '%s'", m.Choice.DisplayName))
	}
	if m.quitting {
		return quitTextStyle.Render("No account was selected.")
	}
	return "\n" + m.list.View()
}

type NoAccountChosenError struct{}

func (e *NoAccountChosenError) Error() string {
	return "no account was chosen"
}

func GetChosenAccount(user *UserDetails, accounts []Account) (*Account, error) {
	const defaultWidth = 20

	items := make([]list.Item, 0)
	for _, acc := range accounts {
		items = append(items, acc)
	}

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)

	l.Title = fmt.Sprintf("You are logged in as: %s#%s", user.DisplayName, user.Suffix)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := tuiModel{list: l}

	model, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, err
	}

	choice := model.(tuiModel).Choice
	if choice == nil {
		return nil, &NoAccountChosenError{}
	}

	return choice, nil
}
