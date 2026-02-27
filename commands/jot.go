package commands

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/hdicksonjr/seton/store"
	"github.com/spf13/cobra"
)

func jotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jot",
		Short: "Interactively write and save a note",
		Args:  cobra.NoArgs,
		RunE:  runJot,
	}
}

func runJot(_ *cobra.Command, _ []string) error {
	var noteText, tags string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Note").
				Placeholder("Write your note here...").
				Value(&noteText),
			huh.NewInput().
				Title("Tags").
				Description("Space-separated, e.g. #todo #refactor").
				Value(&tags),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := store.SaveNote(db, noteText, tags); err != nil {
		return err
	}

	fmt.Println("Note saved.")
	return nil
}
