package linter

import (
	"fmt"
	"testing"
)

type MockFileReader struct {
	shouldError       bool
	shouldReturnLints bool
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.shouldError {
		return nil, fmt.Errorf("FileRead Error")
	}
	if m.shouldReturnLints {
		return []byte("~! hur ~! dur #tag !~"), nil
	}
	return []byte("~! this is a fine note #tag !~"), nil
}

func TestLint(t *testing.T) {

	t.Run("errors if file can't be opened", func(t *testing.T) {
		mockReader := &MockFileReader{shouldError: true}
		_, err := mockReader.ReadFile("somefile.txt")
		if err == nil {
			t.Errorf("expected an error but got none")
		}
		expectedErr := "FileRead Error"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q but got %q", expectedErr, err.Error())
		}
	})

	t.Run("throws warning if there is a violation", func(t *testing.T) {
		mockReader := &MockFileReader{shouldError: false, shouldReturnLints: true}
		noteLinter := NoteLinter{FileReader: mockReader}
		warnings, err := noteLinter.Lint("somefile.txt")
		if err != nil {
			t.Fatalf("expected no error but got %v", err)
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning but got %d", len(warnings))
		}
		expectedMessage := "hurdur"
		if warnings[0].Message != expectedMessage {
			t.Errorf("expected warning message %q but got %q", expectedMessage, warnings[0].Message)
		}
		expectedLine := 0
		if warnings[0].Line != expectedLine {
			t.Errorf("expected warning line %d but got %d", expectedLine, warnings[0].Line)
		}
	})

	t.Run("finishes successfully if no violations", func(t *testing.T) {
		mockReader := &MockFileReader{shouldError: false, shouldReturnLints: false}
		noteLinter := NoteLinter{FileReader: mockReader}
		warnings, err := noteLinter.Lint("somefile.txt")
		if err != nil {
			t.Fatalf("expected no error but got %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("expected no warnings but got %d", len(warnings))
		}
	})

}
