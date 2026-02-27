package commands

import (
	"testing"
)

func TestQueryCmd(t *testing.T) {
	t.Run("has correct use", func(t *testing.T) {
		cmd := queryCmd()
		if cmd.Use != "query [tags...]" {
			t.Errorf("expected Use %q, got %q", "query [tags...]", cmd.Use)
		}
	})
}
