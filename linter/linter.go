package linter

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Linter is an interface implemented by NoteLinter
type Linter interface {
	Lint(filename string) ([]Warning, error)
}

// NoteLinter implements Linter
type NoteLinter struct {
	FileReader FileReader
}

// Warning represents a warning message
type Warning struct {
	Message string
	Line    int
}

// FileReader reads files
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// OSFileReader implements FileReader using the ReadFile from the os package.
type OSFileReader struct{}

// ReadFile implements ReadFile for OSFileReader using ReadFile from the os package
func (OSFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (w Warning) String() string {
	return fmt.Sprintf("Warning at line %d: %s", w.Line, w.Message)
}

// Lint is the primary linting method for NoteLinter
func (n NoteLinter) Lint(filename string) ([]Warning, error) {

	unclosedTagRegex := regexp.MustCompile(`(?s)~!([^!]*)(!([^~][^!]*))*~!`)

	file, err := n.FileReader.ReadFile(filename)

	if err != nil {
		fmt.Println("error opening file")
		return nil, err
	}

	fileText := string(file)

	matches := unclosedTagRegex.FindStringIndex(fileText)

	if matches != nil {
		lineCount := strings.Count(fileText[:matches[1]], "\n")
		fmt.Println("Something was linted")
		return []Warning{{Message: "hurdur", Line: lineCount}}, nil
	}

	fmt.Println("no matches")
	return nil, nil
}
