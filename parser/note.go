package parser

import (
	"regexp"
	"strings"
)

type Note struct {
	RawText string
	Text    string
	Tags    []string
	File    string
}

func newNote(rawText string) *Note {
	hashTagRe := regexp.MustCompile(`#(\S+)`)

	re := regexp.MustCompile(`\s*#\S+`)
	n := Note{RawText: rawText,
		Text: strings.TrimSpace(re.ReplaceAllString(rawText, "")),
		File: "temp",
		Tags: hashTagRe.FindAllString(rawText, -1),
	}

	return &n
}

// ParsedFile represents the data derived from a file after it is parsed
type ParsedFile struct {
	Path  string
	Notes []Note
}
