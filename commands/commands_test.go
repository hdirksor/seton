package commands

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/hdicksonjr/seton/parser"
)

// MockParser is a mock implementation of the parser.Parser interface
type MockParser struct {
	shouldError bool
}

func (m *MockParser) Parse(directory string, walker parser.Walker) ([]parser.ParsedFile, error) {
	if m.shouldError {
		fmt.Printf("should error did Error")
		return nil, fmt.Errorf("parse error")
	}
	return nil, nil
}

func TestExtractCmd(t *testing.T) {
	t.Run("successful extraction", func(t *testing.T) {
		mockParser := &MockParser{}
		cmd := extractCmd(mockParser)

		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{"testdir"})

		err := cmd.Execute()

		if err != nil {
			t.Errorf("unexpected Error: %q", err)
		}
	})

	t.Run("parser error", func(t *testing.T) {
		mockParser := &MockParser{shouldError: true}
		cmd := extractCmd(mockParser)

		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{"testdir"})

		stdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd.Execute()

		w.Close()
		os.Stdout = stdout

		var buf bytes.Buffer
		buf.ReadFrom(r)

		if !bytes.Contains(buf.Bytes(), []byte("parse error")) {
			t.Logf("output buffer: %s", buf.String())
			t.Errorf("expected error message not found in output: %q", buf.String())

		}
	})
}

