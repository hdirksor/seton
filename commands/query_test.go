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

func TestNormalizeTags(t *testing.T) {
	cases := []struct {
		input    []string
		expected []string
	}{
		{[]string{"todo", "auth"}, []string{"#todo", "#auth"}},
		{[]string{"#todo", "#auth"}, []string{"#todo", "#auth"}},
		{[]string{"todo", "#auth"}, []string{"#todo", "#auth"}},
		{nil, nil},
	}

	for _, c := range cases {
		got := normalizeTags(c.input)
		if len(got) != len(c.expected) {
			t.Errorf("input %v: expected len %d, got %d", c.input, len(c.expected), len(got))
			continue
		}
		for i := range got {
			if got[i] != c.expected[i] {
				t.Errorf("input %v: expected %q at index %d, got %q", c.input, c.expected[i], i, got[i])
			}
		}
	}
}
