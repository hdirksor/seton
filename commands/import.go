package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hdicksonjr/seton/config"
	"github.com/hdicksonjr/seton/store"
	"github.com/spf13/cobra"
)

// parseBlocks extracts the text between each open/close delimiter pair.
// The returned strings have leading/trailing whitespace trimmed.
// Empty blocks (whitespace only) are omitted.
func parseBlocks(content, open, close string) []string {
	var blocks []string
	for {
		start := strings.Index(content, open)
		if start == -1 {
			break
		}
		rest := content[start+len(open):]
		end := strings.Index(rest, close)
		if end == -1 {
			break
		}
		block := strings.TrimSpace(rest[:end])
		if block != "" {
			blocks = append(blocks, block)
		}
		content = rest[end+len(close):]
	}
	return blocks
}

// archiveFile moves src to archiveDir, prefixes the filename with date (YYYY-MM-DD_),
// and makes the destination read-only.
func archiveFile(src, archiveDir, date string) error {
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("creating archive dir: %w", err)
	}
	dst := filepath.Join(archiveDir, date+"_"+filepath.Base(src))
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("archiving file: %w", err)
	}
	return os.Chmod(dst, 0444)
}

// importModel is a bubbletea model for the step-by-step import wizard.
type importModel struct {
	blocks   []string
	current  int
	tagInput textinput.Model
	save     func(text, tags string) error
	saved    int
	skipped  int
	completed bool
	quitting  bool
	saveErr   error
}

func newImportModel(blocks []string, saveFn func(string, string) error) importModel {
	ti := textinput.New()
	ti.Placeholder = "#tag1 #tag2"
	ti.Focus()

	m := importModel{
		blocks:   blocks,
		save:     saveFn,
		tagInput: ti,
	}
	m.tagInput.SetValue(blockTags(blocks, 0))
	return m
}

// blockTags returns the space-joined extracted tags for blocks[idx], or "" if out of range.
func blockTags(blocks []string, idx int) string {
	if idx >= len(blocks) {
		return ""
	}
	tags := store.ExtractTagsFromText(blocks[idx])
	return strings.Join(tags, " ")
}

func (m importModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m importModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlS:
			if m.current < len(m.blocks) {
				if err := m.save(m.blocks[m.current], m.tagInput.Value()); err != nil {
					m.saveErr = err
					return m, nil
				}
				m.saveErr = nil
				m.saved++
				m.current++
				m.tagInput.SetValue(blockTags(m.blocks, m.current))
				if m.current >= len(m.blocks) {
					m.completed = true
					return m, tea.Quit
				}
			}
			return m, nil

		case tea.KeyCtrlK:
			if m.current < len(m.blocks) {
				m.skipped++
				m.current++
				m.tagInput.SetValue(blockTags(m.blocks, m.current))
				if m.current >= len(m.blocks) {
					m.completed = true
					return m, tea.Quit
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

var (
	importHeaderStyle = lipgloss.NewStyle().Bold(true)
	importBlockStyle  = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1).
				Foreground(lipgloss.AdaptiveColor{Dark: "#CED4DA", Light: "#343A40"})
	importHintStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Dark: "#9E9E9E", Light: "#616161"})
	importErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Dark: "#EF5350", Light: "#B71C1C"}).Bold(true)
)

func (m importModel) View() string {
	if m.quitting {
		return fmt.Sprintf("\nAborted. %d saved, %d skipped.\n", m.saved, m.skipped)
	}
	if m.completed {
		return fmt.Sprintf("\nDone. %d saved, %d skipped.\n", m.saved, m.skipped)
	}
	if m.current >= len(m.blocks) {
		return ""
	}

	var b strings.Builder
	b.WriteString(banner)
	b.WriteString(importHeaderStyle.Render(
		fmt.Sprintf("Block %d / %d", m.current+1, len(m.blocks))) + "\n\n")
	b.WriteString(importBlockStyle.Render(m.blocks[m.current]) + "\n\n")
	b.WriteString("Tags: " + m.tagInput.View() + "\n")
	if m.saveErr != nil {
		b.WriteString(importErrStyle.Render("Error: "+m.saveErr.Error()) + "\n")
	}
	b.WriteString(importHintStyle.Render("ctrl+s save · ctrl+k skip · ctrl+c quit"))
	return b.String()
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file>",
		Short: "Review and save notes from a file containing ~! !~ blocks",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runImport(func(m tea.Model) (tea.Model, error) {
				return tea.NewProgram(m).Run()
			}, args)
		},
	}
}

func runImport(runProg func(tea.Model) (tea.Model, error), args []string) error {
	path := args[0]
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	blocks := parseBlocks(string(content), cfg.Delimiters.Open, cfg.Delimiters.Close)
	if len(blocks) == 0 {
		return fmt.Errorf("no note blocks found in %s", path)
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	m := newImportModel(blocks, func(text, tags string) error {
		return store.SaveNote(db, text, tags)
	})

	result, err := runProg(m)
	if err != nil {
		return err
	}

	final := result.(importModel)
	if final.completed {
		date := time.Now().Format("2006-01-02")
		dst := filepath.Join(cfg.Paths.Archive(), date+"_"+filepath.Base(path))
		if err := archiveFile(path, cfg.Paths.Archive(), date); err != nil {
			return fmt.Errorf("archiving: %w", err)
		}
		fmt.Printf("Archived to %s\n", dst)
	}
	return nil
}
