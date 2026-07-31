package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdirksor/seton/store"
	"github.com/hdirksor/seton/styles"
	"github.com/spf13/cobra"
)

type tagsPhase int

const (
	tagsPhaseList tagsPhase = iota
	tagsPhaseEntity
)

type tagsFocus int

const (
	tagsFocusInput tagsFocus = iota
	tagsFocusList
)

type tagsModel struct {
	tags        []store.Tag
	filtered    []store.Tag
	filterInput textinput.Model
	focus       tagsFocus
	cursor      int
	height      int
	phase       tagsPhase
	descInput   textinput.Model
	saveFn      func(name, description string) error
	saveErr     error
}

func initialTagsModel(tags []store.Tag, saveFn func(name, description string) error) tagsModel {
	fi := textinput.New()
	fi.Placeholder = "type to filter tags..."
	fi.Width = 40
	fi.Focus()

	filtered := make([]store.Tag, len(tags))
	copy(filtered, tags)

	return tagsModel{
		tags:        tags,
		filtered:    filtered,
		filterInput: fi,
		descInput:   textinput.New(),
		saveFn:      saveFn,
	}
}

// filterStoreTags returns tags whose names contain query as a substring
// (case-insensitive). The "#" prefix is stripped from both sides before
// comparison.
func filterStoreTags(tags []store.Tag, query string) []store.Tag {
	if query == "" {
		out := make([]store.Tag, len(tags))
		copy(out, tags)
		return out
	}
	q := strings.ToLower(strings.TrimPrefix(query, "#"))
	var out []store.Tag
	for _, tag := range tags {
		name := strings.ToLower(strings.TrimPrefix(tag.Name, "#"))
		if strings.Contains(name, q) {
			out = append(out, tag)
		}
	}
	return out
}

func (m tagsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tagsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = size.Height
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if m.phase == tagsPhaseList {
		return m.updateList(key)
	}
	return m.updateEntity(key)
}

func (m tagsModel) updateList(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == tagsFocusInput {
		return m.updateFilterInput(key)
	}
	return m.updateTagList(key)
}

func (m tagsModel) updateFilterInput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyDown:
		if len(m.filtered) > 0 {
			m.focus = tagsFocusList
			m.cursor = 0
			m.filterInput.Blur()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(key)
	m.filtered = filterStoreTags(m.tags, m.filterInput.Value())
	return m, cmd
}

func (m tagsModel) updateTagList(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyRunes:
		if key.String() == "q" {
			return m, tea.Quit
		}
	case tea.KeyUp:
		if m.cursor == 0 {
			m.focus = tagsFocusInput
			m.filterInput.Focus()
		} else {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if len(m.filtered) > 0 {
			m.phase = tagsPhaseEntity
			m.descInput.SetValue(m.filtered[m.cursor].Description)
			m.descInput.Focus()
		}
	}
	return m, nil
}

func (m tagsModel) updateEntity(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.phase = tagsPhaseList
		return m, nil
	case tea.KeyCtrlS:
		return m.saveDescription()
	}

	var cmd tea.Cmd
	m.descInput, cmd = m.descInput.Update(key)
	return m, cmd
}

func (m tagsModel) saveDescription() (tea.Model, tea.Cmd) {
	tag := m.filtered[m.cursor]
	desc := m.descInput.Value()
	if err := m.saveFn(tag.Name, desc); err != nil {
		m.saveErr = err
		return m, nil
	}
	m.saveErr = nil
	for i := range m.tags {
		if m.tags[i].Name == tag.Name {
			m.tags[i].Description = desc
		}
	}
	m.filtered = filterStoreTags(m.tags, m.filterInput.Value())
	m.phase = tagsPhaseList
	return m, nil
}

func (m tagsModel) View() string {
	if m.phase == tagsPhaseEntity {
		return styles.View().Render(styles.Banner() + m.entityView())
	}
	return styles.View().Render(styles.Banner() + m.listView())
}

func (m tagsModel) listView() string {
	header := styles.Header().Render("Tags") + "\n\n"
	footer := styles.Dim().Render("\n↑/↓ navigate · enter view · q quit")

	var b strings.Builder
	b.WriteString(header)
	b.WriteString(m.filterInput.View() + "\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(styles.Dim().Render("no matching tags") + "\n")
	} else {
		start, end := m.visibleTagRange(header, footer)
		for i := start; i < end; i++ {
			tag := m.filtered[i]
			noteWord := "notes"
			if tag.NoteCount == 1 {
				noteWord = "note"
			}
			cursor := "  "
			if i == m.cursor {
				cursor = "> "
			}
			row := fmt.Sprintf("%s%s  (%d %s)", cursor, tag.Name, tag.NoteCount, noteWord)
			if i == m.cursor {
				b.WriteString(styles.FocusedRow().Render(row) + "\n")
			} else {
				b.WriteString(row + "\n")
			}
		}
	}

	b.WriteString(footer)
	return b.String()
}

// visibleTagRange returns the slice of m.filtered to render, scrolled so the
// cursor row always stays on screen once the terminal height is known.
// Before the first tea.WindowSizeMsg arrives, m.height is 0 and the whole
// list is shown.
func (m tagsModel) visibleTagRange(header, footer string) (int, int) {
	if m.height <= 0 {
		return 0, len(m.filtered)
	}

	chrome := styles.View().Render(styles.Banner() + header + m.filterInput.View() + "\n\n" + footer)
	reserved := len(strings.Split(chrome, "\n"))
	visible := max(m.height-reserved, 1)
	if visible >= len(m.filtered) {
		return 0, len(m.filtered)
	}

	start := max(m.cursor-visible/2, 0)
	start = min(start, len(m.filtered)-visible)
	return start, start + visible
}

func (m tagsModel) entityView() string {
	var b strings.Builder

	tag := m.filtered[m.cursor]
	noteWord := "notes"
	if tag.NoteCount == 1 {
		noteWord = "note"
	}

	b.WriteString(styles.Header().Render(tag.Name) + "\n\n")
	b.WriteString(styles.Dim().Render(fmt.Sprintf("%d %s", tag.NoteCount, noteWord)) + "\n\n")
	b.WriteString("Description: " + m.descInput.View() + "\n")
	if m.saveErr != nil {
		b.WriteString(styles.Err().Render("Error: "+m.saveErr.Error()) + "\n")
	}
	b.WriteString(styles.Dim().Render("\nctrl+s save · esc back · ctrl+c quit"))
	return b.String()
}

func tagsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "Interactively browse tags and edit their descriptions",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTags(func(m tea.Model) (tea.Model, error) {
				return tea.NewProgram(m).Run()
			})
		},
	}
}

func runTags(runProg func(tea.Model) (tea.Model, error)) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	tags, err := store.ListTagsWithCounts(db)
	if err != nil {
		return err
	}

	if len(tags) == 0 {
		fmt.Println("No tags found. Add some notes first.")
		return nil
	}

	m := initialTagsModel(tags, func(name, description string) error {
		return store.UpdateTagDescription(db, name, description)
	})

	_, err = runProg(m)
	return err
}
