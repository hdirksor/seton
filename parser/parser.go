package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Parser is an interface implemented by NoteParser
type Parser interface {
	Parse(dir string, walker Walker) ([]ParsedFile, error)
}

// NoteParser implements Parser
type NoteParser struct {
	FileUtils FileUtils
}

type FileUtils interface {
	walkFilePath(filename string) error
}

// Parse is the primary parsing method for NoteParser
func (n NoteParser) Parse(dir string, walker Walker) ([]ParsedFile, error) {

	// Walk through the directory using the walkFn from the walker
	err := filepath.Walk(dir, walker.walkFn)
	if err != nil {
		return nil, err
	}

	if len(walker.getFiles()) == 0 {
		fmt.Printf("No notes found")
		return nil, nil
	}

	return walker.getFiles(), nil
}

type Walker interface {
	walkFn(path string, info os.FileInfo, err error) error
	getFiles() []ParsedFile
}

type WalkerImpl struct {
	files []ParsedFile
}

func (w *WalkerImpl) getFiles() []ParsedFile {
	return w.files
}

func NewWalker() Walker {
	return &WalkerImpl{}
}

func (w *WalkerImpl) walkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	// Only process files (ignore directories)
	if !info.IsDir() {
		// Read the file and find notes
		parsedFile, err := extractNotesReadOnce(path)
		if err != nil {
			return err
		}
		w.files = append(w.files, parsedFile)
	}

	return nil

}

func readFile(path string) (string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("error opening file")
		return "", err
	}

	fileText := string(file)
	return fileText, nil

}

func extractNotesReadOnce(path string) (ParsedFile, error) {
	// Regex to match properly closed tags and capture everything inside
	tagRegex := regexp.MustCompile(`(?s)~!(.*?)!~`)

	fileText, err := readFile(path)

	if err != nil {
		return ParsedFile{Path: path}, err

	}

	matches := tagRegex.FindAllStringSubmatch(fileText, -1)

	if matches == nil {
		return ParsedFile{Path: path}, nil
	}

	var notes []Note
	for n := 0; n < len(matches); n++ {
		tagContent := matches[n]

		notes = append(notes, *newNote(tagContent[0]))

	}
	return ParsedFile{Path: path, Notes: notes}, nil
}
