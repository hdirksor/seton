package commands

import (
	"strings"

	"github.com/spf13/cobra"
)

// noteSummary returns a one-line preview for exit messages: the first 40 runes
// of text (newlines collapsed, trailing ellipsis if truncated) followed by up
// to three tags each showing only their first three characters after '#'.
func noteSummary(text string, tags []string) string {
	flat := strings.ReplaceAll(strings.TrimSpace(text), "\n", " ")
	runes := []rune(flat)
	preview := string(runes)
	if len(runes) > 40 {
		preview = string(runes[:40]) + "…"
	}

	var tagParts []string
	for i, tag := range tags {
		if i >= 3 {
			break
		}
		name := []rune(strings.TrimPrefix(tag, "#"))
		if len(name) > 3 {
			tagParts = append(tagParts, "#"+string(name[:3])+"…")
		} else {
			tagParts = append(tagParts, tag)
		}
	}

	if len(tagParts) == 0 {
		return preview
	}
	return preview + "  " + strings.Join(tagParts, "  ")
}

// InitRootCmd Initializes entire CLI interface
func InitRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "Note Management",
		Short: "A CLI to manage notes",
		Long:  `A CLI to make taking and managing notes more simple`,
	}

	rootCmd.AddCommand(jotCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(importCmd())

	return rootCmd
}
