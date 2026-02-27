package commands

import (
	"testing"
)

func TestJotCmd(t *testing.T) {
	t.Run("has correct use", func(t *testing.T) {
		cmd := jotCmd()
		if cmd.Use != "jot" {
			t.Errorf("expected Use %q, got %q", "jot", cmd.Use)
		}
	})

	t.Run("rejects extra args", func(t *testing.T) {
		cmd := jotCmd()
		cmd.SetArgs([]string{"unexpected"})
		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when args provided, got nil")
		}
	})
}
