package commands

import (
	"strings"
	"testing"
)

func TestNoteSummary(t *testing.T) {
	t.Run("short text no tags", func(t *testing.T) {
		got := noteSummary("hello", nil)
		if got != "hello" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("text over 40 runes is truncated with ellipsis", func(t *testing.T) {
		text := strings.Repeat("a", 50)
		got := noteSummary(text, nil)
		if !strings.HasSuffix(got, "…") {
			t.Errorf("expected trailing ellipsis, got %q", got)
		}
		if len([]rune(strings.TrimSuffix(got, "…"))) != 40 {
			t.Errorf("expected 40 runes before ellipsis, got %q", got)
		}
	})

	t.Run("newlines are collapsed to spaces", func(t *testing.T) {
		got := noteSummary("line1\nline2", nil)
		if strings.Contains(got, "\n") {
			t.Errorf("expected newlines collapsed, got %q", got)
		}
	})

	t.Run("tags truncated to 3 chars after hash", func(t *testing.T) {
		got := noteSummary("note", []string{"#software", "#coding", "#algorithms"})
		if !strings.Contains(got, "#sof…") {
			t.Errorf("expected #sof…, got %q", got)
		}
		if !strings.Contains(got, "#cod…") {
			t.Errorf("expected #cod…, got %q", got)
		}
		if !strings.Contains(got, "#alg…") {
			t.Errorf("expected #alg…, got %q", got)
		}
	})

	t.Run("short tags are not truncated", func(t *testing.T) {
		got := noteSummary("note", []string{"#go", "#ai"})
		if !strings.Contains(got, "#go") || !strings.Contains(got, "#ai") {
			t.Errorf("expected short tags unchanged, got %q", got)
		}
	})

	t.Run("only first three tags shown", func(t *testing.T) {
		got := noteSummary("note", []string{"#one", "#two", "#three", "#four"})
		if strings.Contains(got, "#fou") {
			t.Errorf("expected at most 3 tags, got %q", got)
		}
	})
}
