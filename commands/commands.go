package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/hdicksonjr/seton/parser"
	"github.com/hdicksonjr/seton/writer"
	"github.com/spf13/cobra"
)

const banner = `░█▀█░█▀█░▀█▀░█▀▀░█▀▀
░█░█░█░█░░█░░█▀▀░▀▀█
░▀░▀░▀▀▀░░▀░░▀▀▀░▀▀▀

`

func extractCmd(parserImpl parser.Parser) *cobra.Command {
	return &cobra.Command{
		Use:   "extract [directory] [tag]",
		Short: "Extracts notes from files in a directory based on the tag",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {

			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("53")). // Red color
				Bold(true).                       // Bold to add emphasis
				Padding(1, 2).                    // Add padding for extra effect
				Blink(true)

			directory := args[0]
			ParsedFiles, err := parserImpl.Parse(directory, &parser.WalkerImpl{})

			if err != nil {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Error: %v", err)))
				return
			}

			noteWriter := writer.NoteWriter{}
			for _, file := range ParsedFiles {
				err := noteWriter.WriteNotesToFile(file.Path, file.Notes)
				if err != nil {
					fmt.Println(errorStyle.Render(fmt.Sprintf("Error writing file %s: %v", file.Path, err)))
				}
			}
		},
	}
}

// InitRootCmd Initializes entire CLI interface
func InitRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "note management",
		Short: "A CLI to search a directory for notes and extract them based on a tag.",
		Long:  `Searches a directory for code notes enclosed by '~~!' and '!~~' with a specific tag (e.g. #tag), and extracts them to a new file named after the tag.`,
	}

	rootCmd.AddCommand(extractCmd(parser.NoteParser{}))
	rootCmd.AddCommand(jotCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(importCmd())

	return rootCmd
}
