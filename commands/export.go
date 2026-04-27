package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdirksor/seton/config"
	"github.com/hdirksor/seton/store"
	"github.com/hdirksor/seton/styles"
	"github.com/spf13/cobra"
)

func exportCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "export [tags...]",
		Short: "Query notes by tags and export them to a markdown file",
		RunE: func(_ *cobra.Command, args []string) error {
			return runExport(args, dir)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "directory to write the output file (default: ~/seton/exports)")
	return cmd
}

func runExport(tags []string, dir string) error {
	if dir == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		dir = cfg.Paths.Exports()
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	notes, err := store.QueryNotes(db, normalizeTags(tags))
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Println(styles.Err().Render("✗  No notes found for tags: " + strings.Join(tags, " ")))
		return fmt.Errorf("no notes found for tags: %s", strings.Join(tags, " "))
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	path, err := exportNotesFile(notes, tags, dir, timestamp)
	if err != nil {
		fmt.Println(styles.Err().Render("✗  Export failed: " + err.Error()))
		return err
	}

	fmt.Println(styles.Success().Render(fmt.Sprintf("✓  %d note(s) exported · %s", len(notes), path)))
	return nil
}

func exportNotesFile(notes []store.Note, tags []string, dir string, timestamp string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, buildFilename(tags, timestamp))

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	for i, n := range notes {
		if i > 0 {
			fmt.Fprint(f, "\n---\n\n")
		}
		fmt.Fprintf(f, "_%s · %s_\n\n%s\n", n.CreatedAt, strings.Join(n.Tags, " "), n.Text)
	}

	return path, nil
}

func buildFilename(tags []string, timestamp string) string {
	parts := make([]string, 0, len(tags)+1)
	for _, tag := range tags {
		parts = append(parts, strings.TrimPrefix(tag, "#"))
	}
	parts = append(parts, timestamp)
	return strings.Join(parts, "_") + ".md"
}
