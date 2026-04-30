package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/hdirksor/seton/config"
	"github.com/hdirksor/seton/store"
	"github.com/hdirksor/seton/styles"
	"github.com/spf13/cobra"
)

// jotModel wraps a huh form and intercepts ctrl+e to open the note in an
// external editor instead of saving it to the database.
type jotModel struct {
	form     *huh.Form
	noteText *string
	tags     *string
	toEditor bool
}

func newJotModel() jotModel {
	noteText := ""
	tags := ""
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Note").
				Placeholder("Write your note here...").
				Value(&noteText).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("note cannot be empty")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Tags").
				Description("Space-separated, e.g. #todo #refactor  ·  ctrl+e to open in editor").
				Value(&tags),
		),
	).WithTheme(huh.ThemeCharm())
	return jotModel{
		form:     form,
		noteText: &noteText,
		tags:     &tags,
	}
}

func (m jotModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m jotModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlE:
			m.toEditor = true
			return m, tea.Quit
		}
	}

	updated, cmd := m.form.Update(msg)
	if f, ok := updated.(*huh.Form); ok {
		m.form = f
	}
	if m.form.State == huh.StateCompleted {
		return m, tea.Quit
	}
	return m, cmd
}

func (m jotModel) View() string {
	return styles.View().Render(styles.Banner() + m.form.View())
}

// writeNoteFile writes text wrapped in delimiters to a timestamped .md file in
// notesDir and returns the path.
func writeNoteFile(text, openDelim, closeDelim, notesDir, timestamp string) (string, error) {
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return "", fmt.Errorf("creating notes dir: %w", err)
	}
	path := filepath.Join(notesDir, timestamp+".md")
	content := openDelim + "\n" + strings.TrimSpace(text) + "\n" + closeDelim + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing note file: %w", err)
	}
	return path, nil
}

func jotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jot",
		Short: "Interactively write and save a note",
		Args:  cobra.NoArgs,
		RunE:  runJot,
	}
}

func runJot(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	m := newJotModel()
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return err
	}

	final := result.(jotModel)

	if final.toEditor {
		timestamp := time.Now().Format("2006-01-02T15-04-05")
		path, err := writeNoteFile(
			*final.noteText,
			cfg.Delimiters.Open,
			cfg.Delimiters.Close,
			cfg.Paths.Notes(),
			timestamp,
		)
		if err != nil {
			return err
		}
		editor := cfg.ResolveEditor()
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if final.form.State != huh.StateCompleted {
		return nil
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := store.SaveNote(db, *final.noteText, *final.tags); err != nil {
		fmt.Println(styles.Err().Render("✗  Failed to save note: " + err.Error()))
		return err
	}

	tags := collectTags(*final.noteText, *final.tags)
	fmt.Println(styles.Success().Render("✓  " + noteSummary(*final.noteText, tags)))
	return nil
}

// collectTags merges tags extracted from the note body with the user-provided
// tag string, deduplicating while preserving order.
func collectTags(noteText, userTagsStr string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range store.ExtractTagsFromText(noteText) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range strings.Fields(userTagsStr) {
		if !strings.HasPrefix(t, "#") {
			t = "#" + t
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}
