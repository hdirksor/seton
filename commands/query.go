package commands

import (
	"fmt"
	"strings"

	"github.com/hdicksonjr/seton/store"
	"github.com/spf13/cobra"
)

func queryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query [tags...]",
		Short: "Query notes by tags (AND logic)",
		RunE:  runQuery,
	}
}

func runQuery(_ *cobra.Command, args []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	notes, err := store.QueryNotes(db, args)
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return nil
	}

	for i, n := range notes {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 40))
		}
		fmt.Printf("[%d] %s  tags: %s\n\n%s\n", n.ID, n.CreatedAt, n.Tags, n.Text)
	}
	return nil
}
