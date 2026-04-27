package commands

import (
	"fmt"
	"strings"

	"github.com/hdirksor/seton/store"
	"github.com/spf13/cobra"
)

// normalizeTags ensures every tag has a leading '#', allowing users to omit it
// on the command line (which avoids shell comment interpretation of bare '#').
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, len(tags))
	for i, t := range tags {
		if !strings.HasPrefix(t, "#") {
			t = "#" + t
		}
		out[i] = t
	}
	return out
}

func queryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query [tags...]",
		Short: "Query notes by tags (AND logic)",
		RunE:  runQuery,
	}
}

func runQuery(cmd *cobra.Command, args []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	notes, err := store.QueryNotes(db, normalizeTags(args))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if len(notes) == 0 {
		fmt.Fprintln(out, "No notes found.")
		return nil
	}

	for i, n := range notes {
		if i > 0 {
			fmt.Fprintln(out, strings.Repeat("-", 40))
		}
		fmt.Fprintf(out, "[%d] %s  tags: %s\n\n%s\n", n.ID, n.CreatedAt, strings.Join(n.Tags, " "), n.Text)
	}
	return nil
}
