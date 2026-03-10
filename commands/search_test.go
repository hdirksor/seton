package commands

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdicksonjr/seton/store"
)

var noopQuery = func(_ []string) ([]store.Note, error) { return nil, nil }

func TestFilterTags(t *testing.T) {
	all := []string{"#auth", "#authentication", "#bug", "#todo"}

	t.Run("empty query returns all", func(t *testing.T) {
		got := filterTags(all, "")
		if len(got) != len(all) {
			t.Errorf("expected %d, got %d", len(all), len(got))
		}
	})

	t.Run("filters by substring", func(t *testing.T) {
		got := filterTags(all, "auth")
		if len(got) != 2 {
			t.Errorf("expected 2, got %d: %v", len(got), got)
		}
	})

	t.Run("strips hash prefix before matching", func(t *testing.T) {
		got := filterTags(all, "#bug")
		if len(got) != 1 || got[0] != "#bug" {
			t.Errorf("expected [#bug], got %v", got)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		got := filterTags(all, "AUTH")
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		got := filterTags(all, "xyz")
		if len(got) != 0 {
			t.Errorf("expected 0, got %d", len(got))
		}
	})
}

func TestSearchModelInitialState(t *testing.T) {
	tags := []string{"#auth", "#bug", "#todo"}
	m := initialSearchModel(tags, noopQuery)

	if m.phase != searchPhaseSelect {
		t.Errorf("expected initial phase to be select")
	}
	if m.focus != searchFocusInput {
		t.Errorf("expected initial focus on input")
	}
	if len(m.filtered) != len(tags) {
		t.Errorf("expected all tags in filtered list, got %d", len(m.filtered))
	}
	if len(m.selected) != 0 {
		t.Errorf("expected no selections initially")
	}
}

func TestSearchModelKeyHandling(t *testing.T) {
	tags := []string{"#auth", "#bug", "#todo"}

	t.Run("down moves focus to list", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		updated := result.(searchModel)
		if updated.focus != searchFocusList {
			t.Errorf("expected focus on list after down, got %v", updated.focus)
		}
		if updated.cursor != 0 {
			t.Errorf("expected cursor at 0, got %d", updated.cursor)
		}
	})

	t.Run("down navigates list", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
		m3 := result2.(searchModel)
		if m3.cursor != 1 {
			t.Errorf("expected cursor at 1, got %d", m3.cursor)
		}
	})

	t.Run("down does not go past end of list", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		// move to list and to last item
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyDown})
		m4 := result3.(searchModel)
		result4, _ := m4.Update(tea.KeyMsg{Type: tea.KeyDown})
		m5 := result4.(searchModel)
		if m5.cursor != len(tags)-1 {
			t.Errorf("expected cursor capped at %d, got %d", len(tags)-1, m5.cursor)
		}
	})

	t.Run("up from top of list returns focus to input", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyUp})
		m3 := result2.(searchModel)
		if m3.focus != searchFocusInput {
			t.Errorf("expected focus back on input, got %v", m3.focus)
		}
	})

	t.Run("up navigates list upward", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
		m3 := result2.(searchModel) // cursor = 1
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyUp})
		m4 := result3.(searchModel)
		if m4.cursor != 0 {
			t.Errorf("expected cursor at 0, got %d", m4.cursor)
		}
		if m4.focus != searchFocusList {
			t.Errorf("expected focus to remain on list")
		}
	})

	t.Run("space toggles selection", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)

		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		if !m3.selected["#auth"] {
			t.Errorf("expected #auth to be selected after space")
		}

		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeySpace})
		m4 := result3.(searchModel)
		if m4.selected["#auth"] {
			t.Errorf("expected #auth to be deselected after second space")
		}
	})

	t.Run("selection persists when query changes", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		// select #auth
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)

		// go back to input and change query — #auth may leave filtered list
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyUp})
		m4 := result3.(searchModel)
		// type "bug" into input via rune messages
		for _, ch := range "bug" {
			result4, _ := m4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
			m4 = result4.(searchModel)
		}
		// #auth should still be selected even though it's no longer in filtered
		if !m4.selected["#auth"] {
			t.Errorf("expected #auth to remain selected after query change")
		}
	})

	t.Run("enter with no selection stays in select phase", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		updated := result.(searchModel)
		if updated.phase != searchPhaseSelect {
			t.Errorf("expected phase to remain select when nothing selected")
		}
	})

	t.Run("enter with selection transitions to results phase", func(t *testing.T) {
		notes := []store.Note{{ID: 1, Text: "note", Tags: []string{"#auth"}}}
		queryFn := func(_ []string) ([]store.Note, error) { return notes, nil }
		m := initialSearchModel(tags, queryFn)
		// select #auth via list
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
		updated := result3.(searchModel)

		if updated.phase != searchPhaseResults {
			t.Errorf("expected phase=results after enter with selection")
		}
		if len(updated.notes) != 1 {
			t.Errorf("expected 1 note, got %d", len(updated.notes))
		}
	})
}

func TestSearchModelResultsPhase(t *testing.T) {
	notes := []store.Note{{ID: 1, Text: "a note", Tags: []string{"#auth"}}}
	queryFn := func(_ []string) ([]store.Note, error) { return notes, nil }

	// helper: put model into results phase with #auth selected
	inResults := func() searchModel {
		tags := []string{"#auth"}
		m := initialSearchModel(tags, queryFn)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return result3.(searchModel)
	}

	t.Run("ctrl+e sets export=true", func(t *testing.T) {
		m := inResults()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
		updated := result.(searchModel)
		if !updated.export {
			t.Errorf("expected export=true after ctrl+e in results phase")
		}
	})

	t.Run("q quits without export", func(t *testing.T) {
		m := inResults()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		updated := result.(searchModel)
		if updated.export {
			t.Errorf("expected export=false after q in results phase")
		}
	})

	t.Run("enter quits without export", func(t *testing.T) {
		m := inResults()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		updated := result.(searchModel)
		if updated.export {
			t.Errorf("expected export=false after enter in results phase")
		}
	})
}
