package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdicksonjr/seton/store"
)

var noopQuery = func(_ []string) ([]store.Note, error) { return nil, nil }

func fixedSearchProg(m searchModel) func(tea.Model) (tea.Model, error) {
	return func(_ tea.Model) (tea.Model, error) { return m, nil }
}

func seedNote(t *testing.T) {
	t.Helper()
	db, err := store.Open()
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer db.Close()
	if err := store.SaveNote(db, "a note #todo", ""); err != nil {
		t.Fatalf("saving note: %v", err)
	}
}

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

func TestSearchModelInit(t *testing.T) {
	m := initialSearchModel([]string{"#auth"}, noopQuery)
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a non-nil cmd")
	}
}

func TestSearchModelUpdateMissingBranches(t *testing.T) {
	tags := []string{"#auth", "#bug", "#todo"}

	t.Run("ctrl+c in select phase quits", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Error("expected quit cmd from ctrl+c")
		}
	})

	t.Run("esc in input focus quits", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected quit cmd from esc in input focus")
		}
	})

	t.Run("esc in list focus quits", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected quit cmd from esc in list focus")
		}
	})

	t.Run("q rune in list focus quits", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Error("expected quit cmd from q in list focus")
		}
	})

	t.Run("down with empty filtered list stays in input focus", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		// type something that matches nothing to empty the filtered list
		for _, ch := range "zzz" {
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
			m = result.(searchModel)
		}
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		updated := result.(searchModel)
		if updated.focus != searchFocusInput {
			t.Errorf("expected focus to remain on input when filtered list is empty")
		}
	})

	t.Run("non-key message is ignored", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, cmd := m.Update("not a key message")
		if result.(searchModel).phase != searchPhaseSelect {
			t.Error("expected phase unchanged after non-key message")
		}
		if cmd != nil {
			t.Error("expected nil cmd from non-key message")
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

	t.Run("esc quits without export", func(t *testing.T) {
		m := inResults()
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected quit cmd from esc in results phase")
		}
	})

	t.Run("ctrl+c quits without export", func(t *testing.T) {
		m := inResults()
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Error("expected quit cmd from ctrl+c in results phase")
		}
		if result.(searchModel).export {
			t.Error("expected export=false after ctrl+c")
		}
	})

	t.Run("non-q rune in results phase is ignored", func(t *testing.T) {
		m := inResults()
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		if result.(searchModel).export {
			t.Error("expected export=false after non-q rune")
		}
		if cmd != nil {
			t.Error("expected nil cmd from non-q rune in results phase")
		}
	})
}

func TestSearchModelView(t *testing.T) {
	tags := []string{"#auth", "#bug"}

	t.Run("select phase view contains header and input", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		view := m.View()
		if !strings.Contains(view, "Search tags") {
			t.Errorf("expected view to contain 'Search tags', got: %s", view)
		}
		if !strings.Contains(view, "enter search") {
			t.Errorf("expected view to contain hint, got: %s", view)
		}
	})

	t.Run("select phase view lists tags", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		view := m.View()
		if !strings.Contains(view, "#auth") || !strings.Contains(view, "#bug") {
			t.Errorf("expected view to contain tags, got: %s", view)
		}
	})

	t.Run("select phase view with no matching tags", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		for _, ch := range "zzz" {
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
			m = result.(searchModel)
		}
		view := m.View()
		if !strings.Contains(view, "no matching tags") {
			t.Errorf("expected 'no matching tags' in view, got: %s", view)
		}
	})

	t.Run("results phase view contains notes", func(t *testing.T) {
		notes := []store.Note{{ID: 1, Text: "my note", Tags: []string{"#auth"}}}
		queryFn := func(_ []string) ([]store.Note, error) { return notes, nil }
		m := initialSearchModel([]string{"#auth"}, queryFn)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m4 := result3.(searchModel)

		view := m4.View()
		if !strings.Contains(view, "my note") {
			t.Errorf("expected view to contain note text, got: %s", view)
		}
		if !strings.Contains(view, "ctrl+e export") {
			t.Errorf("expected view to contain export hint, got: %s", view)
		}
	})

	t.Run("results phase view with no notes", func(t *testing.T) {
		m := initialSearchModel([]string{"#auth"}, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m4 := result3.(searchModel)

		view := m4.View()
		if !strings.Contains(view, "No notes found") {
			t.Errorf("expected 'No notes found' in view, got: %s", view)
		}
	})

	t.Run("select phase view shows selected checkbox", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		// select #auth
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		view := m3.View()
		if !strings.Contains(view, "[x]") {
			t.Errorf("expected selected checkbox [x] in view, got: %s", view)
		}
	})

	t.Run("select phase view shows cursor in list focus", func(t *testing.T) {
		m := initialSearchModel(tags, noopQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		view := m2.View()
		if !strings.Contains(view, ">") {
			t.Errorf("expected cursor > in list focus view, got: %s", view)
		}
	})

	t.Run("results phase view with multiple notes shows both notes in cards", func(t *testing.T) {
		notes := []store.Note{
			{ID: 1, Text: "first note", Tags: []string{"#auth"}},
			{ID: 2, Text: "second note", Tags: []string{"#auth"}},
		}
		queryFn := func(_ []string) ([]store.Note, error) { return notes, nil }
		m := initialSearchModel([]string{"#auth"}, queryFn)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m4 := result3.(searchModel)

		view := m4.View()
		if !strings.Contains(view, "first note") || !strings.Contains(view, "second note") {
			t.Errorf("expected both notes in view, got: %s", view)
		}
	})

	t.Run("results phase view with query error", func(t *testing.T) {
		errQuery := func(_ []string) ([]store.Note, error) {
			return nil, fmt.Errorf("db failed")
		}
		m := initialSearchModel([]string{"#auth"}, errQuery)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(searchModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
		m3 := result2.(searchModel)
		result3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m4 := result3.(searchModel)

		view := m4.View()
		if !strings.Contains(view, "Error:") {
			t.Errorf("expected 'Error:' in view, got: %s", view)
		}
	})
}

func TestRunSearch(t *testing.T) {
	t.Run("no tags in db prints message and returns nil", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := runSearch(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("runProg error is returned", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		seedNote(t)
		stub := func(_ tea.Model) (tea.Model, error) { return nil, errors.New("terminal error") }
		if err := runSearch(stub); err == nil {
			t.Error("expected error from runProg")
		}
	})

	t.Run("tui exits without results returns nil", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		seedNote(t)
		stub := fixedSearchProg(searchModel{phase: searchPhaseSelect})
		if err := runSearch(stub); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("results with export=false returns nil", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		seedNote(t)
		result := searchModel{
			phase:  searchPhaseResults,
			export: false,
			notes:  []store.Note{{ID: 1, Text: "a note", Tags: []string{"#todo"}}},
		}
		if err := runSearch(fixedSearchProg(result)); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("results with no notes and export=true returns nil", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		seedNote(t)
		result := searchModel{phase: searchPhaseResults, export: true, notes: nil}
		if err := runSearch(fixedSearchProg(result)); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("results with export=true writes export file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		seedNote(t)
		result := searchModel{
			phase:    searchPhaseResults,
			export:   true,
			notes:    []store.Note{{ID: 1, Text: "a note", Tags: []string{"#todo"}}},
			selected: map[string]bool{"#todo": true},
		}
		if err := runSearch(fixedSearchProg(result)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entries, err := os.ReadDir(filepath.Join(home, "seton", "exports"))
		if err != nil || len(entries) == 0 {
			t.Errorf("expected export file to be created")
		}
	})
}

func TestRunSearchErrorPaths(t *testing.T) {
	t.Run("store open error is returned", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		os.WriteFile(filepath.Join(home, ".seton"), []byte("x"), 0644)
		if err := runSearch(nil); err == nil {
			t.Error("expected error from store.Open")
		}
	})

	t.Run("exportNotesFile error is returned", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		seedNote(t)
		// Place a regular file at the exports path so os.MkdirAll inside exportNotesFile fails.
		os.MkdirAll(filepath.Join(home, "seton"), 0755)
		os.WriteFile(filepath.Join(home, "seton", "exports"), []byte("x"), 0644)
		result := searchModel{
			phase:    searchPhaseResults,
			export:   true,
			notes:    []store.Note{{ID: 1, Text: "a note", Tags: []string{"#todo"}}},
			selected: map[string]bool{"#todo": true},
		}
		if err := runSearch(fixedSearchProg(result)); err == nil {
			t.Error("expected error from exportNotesFile")
		}
	})
}
