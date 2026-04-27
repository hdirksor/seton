package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdirksor/seton/config"
	"github.com/hdirksor/seton/store"
	"github.com/hdirksor/seton/styles"
	"github.com/spf13/cobra"
)

type searchFocus int

const (
	searchFocusInput searchFocus = iota
	searchFocusList
)

type searchPhase int

const (
	searchPhaseSelect  searchPhase = iota
	searchPhaseResults
)

type searchModel struct {
	// selection phase
	input    textinput.Model
	allTags  []string
	filtered []string
	selected map[string]bool
	cursor   int
	focus    searchFocus

	// phase
	phase   searchPhase
	queryFn func([]string) ([]store.Note, error)

	// results phase
	notes    []store.Note
	queryErr error

	// outcome
	export bool
}

func initialSearchModel(tags []string, queryFn func([]string) ([]store.Note, error)) searchModel {
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
		queryFn:  queryFn,
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

// executeQuery runs the query for the currently selected tags and transitions
// to the results phase. Does nothing if no tags are selected.
func (m searchModel) executeQuery() (tea.Model, tea.Cmd) {
	var selected []string
	for tag, ok := range m.selected {
		if ok {
			selected = append(selected, tag)
		}
	}
	if len(selected) == 0 {
		return m, nil
	}
	notes, err := m.queryFn(selected)
	m.notes = notes
	m.queryErr = err
	m.phase = searchPhaseResults
	return m, nil
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

		if m.phase == searchPhaseResults {
			switch msg.Type {
			case tea.KeyCtrlE:
				m.export = true
				return m, tea.Quit
			case tea.KeyEsc, tea.KeyEnter:
				return m, tea.Quit
			case tea.KeyRunes:
				if msg.String() == "q" {
					return m, tea.Quit
				}
			}
			return m, nil
		}

		// searchPhaseSelect
		if m.focus == searchFocusInput {
			switch msg.Type {
			case tea.KeyEsc:
				return m, tea.Quit
			case tea.KeyEnter:
				return m.executeQuery()
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
				return m.executeQuery()
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


func (m searchModel) View() string {
	if m.phase == searchPhaseResults {
		return styles.View().Render(styles.Banner()+ m.resultsView())
	}
	return styles.View().Render(styles.Banner()+ m.selectView())
}

func (m searchModel) selectView() string {
	var b strings.Builder

	b.WriteString(styles.Header().Render("Search tags") + "\n\n")
	b.WriteString(m.input.View() + "\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(styles.Dim().Render("no matching tags") + "\n")
	} else {
		for i, tag := range m.filtered {
			focused := m.focus == searchFocusList && i == m.cursor
			sel := m.selected[tag]

			checkbox := "[ ]"
			if sel {
				checkbox = "[x]"
			}

			row := "  " + checkbox + " " + tag
			if focused {
				b.WriteString(styles.FocusedRow().Render(row) + "\n")
			} else if sel {
				b.WriteString("  " + styles.Selected().Render("[x]") + " " + tag + "\n")
			} else {
				b.WriteString(row + "\n")
			}
		}
	}

	selectedCount := 0
	for _, v := range m.selected {
		if v {
			selectedCount++
		}
	}
	hint := fmt.Sprintf("\n%d selected · ↑/↓ navigate · space toggle · enter search · q quit",
		selectedCount)
	b.WriteString(styles.Dim().Render(hint))

	return b.String()
}

func (m searchModel) resultsView() string {
	var b strings.Builder

	if m.queryErr != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.queryErr))
	} else if len(m.notes) == 0 {
		b.WriteString("No notes found.\n")
	} else {
		for _, n := range m.notes {
			meta := styles.Dim().Render(fmt.Sprintf("#%d · %s · %s", n.ID, n.CreatedAt, strings.Join(n.Tags, " ")))
			b.WriteString(styles.Card().Render(meta+"\n\n"+n.Text) + "\n")
		}
	}

	b.WriteString(styles.Dim().Render("\nctrl+e export · q quit"))
	return b.String()
}

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search",
		Short: "Interactively search tags and query matching notes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSearch(func(m tea.Model) (tea.Model, error) {
				return tea.NewProgram(m).Run()
			})
		},
	}
}

func runSearch(runProg func(tea.Model) (tea.Model, error)) error {
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

	m := initialSearchModel(allTags, func(tags []string) ([]store.Note, error) {
		return store.QueryNotes(db, tags)
	})

	result, err := runProg(m)
	if err != nil {
		return err
	}

	final := result.(searchModel)
	if final.phase != searchPhaseResults || !final.export || len(final.notes) == 0 {
		return nil
	}

	var selected []string
	for tag, ok := range final.selected {
		if ok {
			selected = append(selected, tag)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	path, err := exportNotesFile(final.notes, selected, cfg.Paths.Exports(), timestamp)
	if err != nil {
		fmt.Println(styles.Err().Render(fmt.Sprintf("✗  Export failed: %s", err.Error())))
		return err
	}
	fmt.Println(styles.Success().Render(fmt.Sprintf("✓  %d note(s) exported · %s", len(final.notes), path)))
	return nil
}
