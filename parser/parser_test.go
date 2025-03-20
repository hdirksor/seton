package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type MockWalker struct {
	shouldError bool
	files       []ParsedFile
}

func (m *MockWalker) walkFn(path string, info os.FileInfo, err error) error {
	if m.shouldError {
		return fmt.Errorf("mock error during file walk")
	}
	return nil
}

func (m *MockWalker) getFiles() []ParsedFile {
	return m.files
}

func TestParse(t *testing.T) {
	t.Run("valid notes in multiple files", func(t *testing.T) {
		noteParser := NoteParser{}
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "testdir")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir) // Clean up the directory after the test

		// Create temporary files with valid notes
		fileContents := []string{
			"~!Note 1!~\n~!Note 2!~",
			"~!Note 3!~\n~!Note 4!~",
		}

		for i, content := range fileContents {
			tempFile := filepath.Join(tempDir, "file"+string(rune('A'+i))+".txt")
			if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
		}

		// Call the Parse function
		parsedFiles, err := noteParser.Parse(tempDir, &WalkerImpl{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Validate the results
		expectedNotes := 4
		actualNotes := 0
		for _, parsedFile := range parsedFiles {
			actualNotes += len(parsedFile.Notes)
		}

		if actualNotes != expectedNotes {
			t.Fatalf("expected %d notes, got %d", expectedNotes, actualNotes)
		}
	})
}

func TestParse_ErrorDuringWalk(t *testing.T) {
	// Create a NoteParser
	noteParser := NoteParser{}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // Clean up the directory after the test

	// Create a custom walker that simulates an error
	mockWalker := &MockWalker{shouldError: true}

	// Call the Parse function with the custom walker
	_, err = noteParser.Parse(tempDir, mockWalker)
	if err == nil || err.Error() != "mock error during file walk" {
		t.Fatalf("expected error 'mock error during file walk', got %v", err)
	}
}
