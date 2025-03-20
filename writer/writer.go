package writer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hdicksonjr/seton/parser"
	"gopkg.in/yaml.v2"
)

// Writer is interface for the Writer package
type Writer interface {
	writeNotesToFile([]parser.Note)
}

// NoteWriter implements Writer
type NoteWriter struct{}

// WriteNotesToFile is primary method for writing notes data set to a file.
func (n NoteWriter) WriteNotesToFile(outputFile string, notes []parser.Note) error {
	var noteContents []string
	for _, note := range notes {
		noteContents = append(noteContents, note.Text)
	}

	// Create the .archive directory inside the original directory
	originalDir := filepath.Dir(outputFile)
	archiveDir := filepath.Join(originalDir, ".archive")
	err := os.MkdirAll(archiveDir, 0755) // Ensure the .archive directory exists
	if err != nil {
		return err
	}

	// Construct the destination file path inside the .archive directory
	destinationFile := filepath.Join(archiveDir, strings.TrimSuffix(filepath.Base(outputFile), filepath.Ext(outputFile))+".yaml")

	// Marshal the notes into YAML format
	contentBytes, err := yaml.Marshal(notes)
	if err != nil {
		return err
	}

	// Write the YAML content to the destination file
	return ioutil.WriteFile(destinationFile, contentBytes, 0644)
}
