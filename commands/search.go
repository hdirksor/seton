package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hdicksonjr/seton/store"
	"github.com/spf13/cobra"
)

type searchFocus int

const (
	searchFocusInput searchFocus = iota
	searchFocusList
)

type searchModel struct {
	input    textinput.Model
	allTags  []string
	filtered []string
	selected map[string]bool
	cursor   int
	focus    searchFocus
	done     bool
}

func initialSearchModel(tags []string) searchModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter tags..."
	ti.Focus()

	filtered := make([]string, len(tags))
	copy(filtered, tags)

	return searchModel{
		input:    ti,
		allTags:  tags,
		filtered: filtered,
		selected: map[string]bool{},
	}
}

// filterTags returns tags whose names contain query as a substring (case-insensitive).
// The # prefix is stripped from both sides before comparison.
func filterTags(all []string, query string) []string {
	if query == "" {
		return all
	}
	q := strings.ToLower(strings.TrimPrefix(query, "#"))
	var out []string
	for _, tag := range all {
		name := strings.ToLower(strings.TrimPrefix(tag, "#"))
		if strings.Contains(name, q) {
			out = append(out, tag)
		}
	}
	return out
}

func (m searchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit — always handled.
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		if m.focus == searchFocusInput {
			switch msg.Type {
			case tea.KeyEsc:
				return m, tea.Quit
			case tea.KeyEnter:
				m.done = true
				return m, tea.Quit
			case tea.KeyDown:
				if len(m.filtered) > 0 {
					m.focus = searchFocusList
					m.cursor = 0
					m.input.Blur()
				}
				return m, nil
			}
			// All other keys are forwarded to the text input.
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.filtered = filterTags(m.allTags, m.input.Value())
			if m.cursor >= len(m.filtered) {
				m.cursor = max(0, len(m.filtered)-1)
			}
			return m, cmd
		}

		if m.focus == searchFocusList {
			switch msg.Type {
			case tea.KeyEsc:
				return m, tea.Quit
			case tea.KeyEnter:
				m.done = true
				return m, tea.Quit
			case tea.KeyUp:
				if m.cursor == 0 {
					m.focus = searchFocusInput
					m.input.Focus()
				} else {
					m.cursor--
				}
			case tea.KeyDown:
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
			case tea.KeySpace:
				if len(m.filtered) > 0 {
					tag := m.filtered[m.cursor]
					m.selected[tag] = !m.selected[tag]
				}
			case tea.KeyRunes:
				if msg.String() == "q" {
					return m, tea.Quit
				}
			}
			return m, nil
		}
	}

	return m, nil
}

var (
	selectedTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	cursorTagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	dimTagStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headerTagStyle   = lipgloss.NewStyle().Bold(true)
)

func (m searchModel) View() string {
	var b strings.Builder

	b.WriteString(headerTagStyle.Render("Search tags") + "\n\n")
	b.WriteString(m.input.View() + "\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(dimTagStyle.Render("no matching tags") + "\n")
	} else {
		for i, tag := range m.filtered {
			cursor := "  "
			if m.focus == searchFocusList && i == m.cursor {
				cursor = cursorTagStyle.Render("> ")
			}

			checkbox := "[ ]"
			if m.selected[tag] {
				checkbox = selectedTagStyle.Render("[x]")
			}

			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, tag))
		}
	}

	selectedCount := 0
	for _, v := range m.selected {
		if v {
			selectedCount++
		}
	}
	hint := fmt.Sprintf("\n%d selected · ↑/↓ navigate · space toggle · enter confirm · q quit",
		selectedCount)
	b.WriteString(dimTagStyle.Render(hint))

	return b.String()
}

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search",
		Short: "Interactively search tags and query matching notes",
		Args:  cobra.NoArgs,
		RunE:  runSearch,
	}
}

func runSearch(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	allTags, err := store.ListTags(db)
	if err != nil {
		return err
	}

	if len(allTags) == 0 {
		fmt.Println("No tags found. Add some notes first.")
		return nil
	}

	p := tea.NewProgram(initialSearchModel(allTags))
	result, err := p.Run()
	if err != nil {
		return err
	}

	final := result.(searchModel)
	if !final.done {
		return nil
	}

	var selected []string
	for tag, ok := range final.selected {
		if ok {
			selected = append(selected, tag)
		}
	}

	if len(selected) == 0 {
		return nil
	}

	notes, err := store.QueryNotes(db, selected)
	if err != nil {
		return err
	}

	fmt.Println()
	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return nil
	}

	for i, n := range notes {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 40))
		}
		fmt.Printf("[%d] %s  tags: %s\n\n%s\n", n.ID, n.CreatedAt, strings.Join(n.Tags, " "), n.Text)
	}

	return nil
}
