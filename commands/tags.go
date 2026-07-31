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

type tagsModel struct {
	tags      []store.Tag
	cursor    int
	height    int
	phase     tagsPhase
	descInput textinput.Model
	saveFn    func(name, description string) error
	saveErr   error
}

func initialTagsModel(tags []store.Tag, saveFn func(name, description string) error) tagsModel {
	return tagsModel{
		tags:      tags,
		descInput: textinput.New(),
		saveFn:    saveFn,
	}
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
	switch key.Type {
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyRunes:
		if key.String() == "q" {
			return m, tea.Quit
		}
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.tags)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if len(m.tags) > 0 {
			m.phase = tagsPhaseEntity
			m.descInput.SetValue(m.tags[m.cursor].Description)
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
	desc := m.descInput.Value()
	if err := m.saveFn(m.tags[m.cursor].Name, desc); err != nil {
		m.saveErr = err
		return m, nil
	}
	m.saveErr = nil
	m.tags[m.cursor].Description = desc
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

	start, end := m.visibleTagRange(header, footer)
	for i := start; i < end; i++ {
		tag := m.tags[i]
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

	b.WriteString(footer)
	return b.String()
}

// visibleTagRange returns the slice of m.tags to render, scrolled so the
// cursor row always stays on screen once the terminal height is known.
// Before the first tea.WindowSizeMsg arrives, m.height is 0 and the whole
// list is shown.
func (m tagsModel) visibleTagRange(header, footer string) (int, int) {
	if m.height <= 0 {
		return 0, len(m.tags)
	}

	chrome := styles.View().Render(styles.Banner() + header + footer)
	reserved := len(strings.Split(chrome, "\n"))
	visible := max(m.height-reserved, 1)
	if visible >= len(m.tags) {
		return 0, len(m.tags)
	}

	start := max(m.cursor-visible/2, 0)
	start = min(start, len(m.tags)-visible)
	return start, start + visible
}

func (m tagsModel) entityView() string {
	var b strings.Builder

	tag := m.tags[m.cursor]
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
