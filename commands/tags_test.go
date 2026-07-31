package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdirksor/seton/store"
)

func fixedTagsProg(m tagsModel) func(tea.Model) (tea.Model, error) {
	return func(_ tea.Model) (tea.Model, error) { return m, nil }
}

func noopSaveDescription(_, _ string) error { return nil }

func seedTaggedNotes(t *testing.T) {
	t.Helper()
	db, err := store.Open()
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer db.Close()
	if err := store.SaveNote(db, "a note", "#todo #auth"); err != nil {
		t.Fatalf("saving note: %v", err)
	}
}

// isQuit reports whether cmd is a tea.Quit command, distinguishing it from
// other non-nil commands (e.g. textinput's cursor-blink command).
func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// typeString sends each rune of s as a KeyRunes message and returns the
// resulting model.
func typeString(m tagsModel, s string) tagsModel {
	for _, ch := range s {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = result.(tagsModel)
	}
	return m
}

func TestInitialTagsModel(t *testing.T) {
	tags := []store.Tag{{Name: "#auth", NoteCount: 2}, {Name: "#todo", NoteCount: 1}}
	m := initialTagsModel(tags, noopSaveDescription)

	if m.phase != tagsPhaseList {
		t.Errorf("expected initial phase to be list")
	}
	if m.focus != tagsFocusInput {
		t.Errorf("expected initial focus to be the filter input")
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if len(m.filtered) != len(tags) {
		t.Errorf("expected filtered list to start with all tags, got %d", len(m.filtered))
	}
	if !m.filterInput.Focused() {
		t.Error("expected filter input to be focused by default")
	}
	if m.filterInput.Placeholder != "type to filter tags..." {
		t.Errorf("expected filter placeholder, got %q", m.filterInput.Placeholder)
	}
}

func TestFilterStoreTags(t *testing.T) {
	tags := []store.Tag{{Name: "#auth"}, {Name: "#bug"}, {Name: "#todo"}}

	t.Run("empty query returns all tags", func(t *testing.T) {
		out := filterStoreTags(tags, "")
		if len(out) != 3 {
			t.Errorf("expected all 3 tags, got %d", len(out))
		}
	})

	t.Run("matches substring case-insensitively", func(t *testing.T) {
		out := filterStoreTags(tags, "TO")
		if len(out) != 1 || out[0].Name != "#todo" {
			t.Errorf("expected only #todo, got %v", out)
		}
	})

	t.Run("strips # prefix from query before comparing", func(t *testing.T) {
		out := filterStoreTags(tags, "#bug")
		if len(out) != 1 || out[0].Name != "#bug" {
			t.Errorf("expected only #bug, got %v", out)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		out := filterStoreTags(tags, "zzz")
		if len(out) != 0 {
			t.Errorf("expected no matches, got %v", out)
		}
	})
}

func TestTagsModelFilterInput(t *testing.T) {
	tags := []store.Tag{{Name: "#auth"}, {Name: "#bug"}, {Name: "#todo"}}

	t.Run("typing narrows the filtered list", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		m = typeString(m, "to")
		if len(m.filtered) != 1 || m.filtered[0].Name != "#todo" {
			t.Errorf("expected only #todo, got %v", m.filtered)
		}
	})

	t.Run("down-arrow moves focus to the list when a tag is visible", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = result.(tagsModel)
		if m.focus != tagsFocusList {
			t.Errorf("expected focus to move to list, got %v", m.focus)
		}
		if m.cursor != 0 {
			t.Errorf("expected cursor to reset to 0, got %d", m.cursor)
		}
	})

	t.Run("down-arrow does not move focus when no tags are visible", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		m = typeString(m, "zzz")
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = result.(tagsModel)
		if m.focus != tagsFocusInput {
			t.Errorf("expected focus to remain on input, got %v", m.focus)
		}
	})

	t.Run("q is typed into the input, not treated as quit", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if isQuit(cmd) {
			t.Error("expected q to be forwarded to the filter input, not quit")
		}
		if result.(tagsModel).filterInput.Value() != "q" {
			t.Errorf("expected q typed into filter input, got %q", result.(tagsModel).filterInput.Value())
		}
	})

	t.Run("esc quits", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected quit cmd from esc")
		}
	})

	t.Run("ctrl+c quits", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Error("expected quit cmd from ctrl+c")
		}
	})
}

// focusList moves focus from the filter input to the tag list via down-arrow.
func focusList(m tagsModel) tagsModel {
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	return result.(tagsModel)
}

func TestTagsModelListNavigation(t *testing.T) {
	tags := []store.Tag{{Name: "#auth"}, {Name: "#bug"}, {Name: "#todo"}}

	t.Run("down moves cursor forward once list is focused", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		if result.(tagsModel).cursor != 1 {
			t.Errorf("expected cursor at 1, got %d", result.(tagsModel).cursor)
		}
	})

	t.Run("down does not go past end of list", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		for i := 0; i < 5; i++ {
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m = result.(tagsModel)
		}
		if m.cursor != len(tags)-1 {
			t.Errorf("expected cursor capped at %d, got %d", len(tags)-1, m.cursor)
		}
	})

	t.Run("up on first row moves focus back to filter input", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		updated := result.(tagsModel)
		if updated.focus != tagsFocusInput {
			t.Errorf("expected focus back on input, got %v", updated.focus)
		}
		if !updated.filterInput.Focused() {
			t.Error("expected filter input to be re-focused")
		}
	})

	t.Run("up moves cursor back when not on first row", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(tagsModel)
		result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyUp})
		updated := result2.(tagsModel)
		if updated.cursor != 0 {
			t.Errorf("expected cursor back at 0, got %d", updated.cursor)
		}
		if updated.focus != tagsFocusList {
			t.Errorf("expected focus to remain on list, got %v", updated.focus)
		}
	})
}

func TestTagsModelWindowSizeMsg(t *testing.T) {
	m := initialTagsModel([]store.Tag{{Name: "#auth"}}, noopSaveDescription)
	result, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if result.(tagsModel).height != 20 {
		t.Errorf("expected height to be set to 20, got %d", result.(tagsModel).height)
	}
	if cmd != nil {
		t.Error("expected nil cmd from window size message")
	}
}

func manyTags(n int) []store.Tag {
	tags := make([]store.Tag, n)
	for i := range tags {
		tags[i] = store.Tag{Name: fmt.Sprintf("#tag%02d", i)}
	}
	return tags
}

func TestTagsModelListView_ScrollsToKeepCursorVisible(t *testing.T) {
	m := initialTagsModel(manyTags(30), noopSaveDescription)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	m = result.(tagsModel)
	m = focusList(m)

	t.Run("short terminal does not render every tag", func(t *testing.T) {
		view := m.View()
		if strings.Contains(view, "#tag00") && strings.Contains(view, "#tag29") {
			t.Errorf("expected list to be windowed on a short terminal, but both ends were visible: %s", view)
		}
	})

	t.Run("moving cursor near the bottom scrolls the top out of view", func(t *testing.T) {
		cur := m
		for i := 0; i < 25; i++ {
			result, _ := cur.Update(tea.KeyMsg{Type: tea.KeyDown})
			cur = result.(tagsModel)
		}
		view := cur.View()
		if !strings.Contains(view, "#tag25") {
			t.Errorf("expected selected tag #tag25 to be visible, got: %s", view)
		}
		if strings.Contains(view, "#tag00") {
			t.Errorf("expected #tag00 to have scrolled out of view, got: %s", view)
		}
	})
}

func TestTagsModelQuitFromList(t *testing.T) {
	tags := []store.Tag{{Name: "#auth"}}

	t.Run("ctrl+c quits", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Error("expected quit cmd from ctrl+c")
		}
	})

	t.Run("esc quits", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected quit cmd from esc")
		}
	})

	t.Run("q quits", func(t *testing.T) {
		m := focusList(initialTagsModel(tags, noopSaveDescription))
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Error("expected quit cmd from q")
		}
	})
}

func TestTagsModelEnterOpensEntity(t *testing.T) {
	tags := []store.Tag{
		{Name: "#auth", Description: "auth stuff", NoteCount: 2},
		{Name: "#todo", NoteCount: 1},
	}
	m := focusList(initialTagsModel(tags, noopSaveDescription))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := result.(tagsModel)
	result2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	entity := result2.(tagsModel)

	if entity.phase != tagsPhaseEntity {
		t.Fatalf("expected phase to be entity, got %v", entity.phase)
	}
	if entity.cursor != 1 {
		t.Errorf("expected cursor to remain at selected tag, got %d", entity.cursor)
	}
	if entity.descInput.Value() != "" {
		t.Errorf("expected empty description prefilled, got %q", entity.descInput.Value())
	}
}

func tagsModelInEntity(tags []store.Tag, saveFn func(string, string) error) tagsModel {
	m := focusList(initialTagsModel(tags, saveFn))
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return result.(tagsModel)
}

func TestTagsModelEntityPhase(t *testing.T) {
	tags := []store.Tag{{Name: "#auth", Description: "existing", NoteCount: 3}}

	t.Run("descInput prefilled with existing description", func(t *testing.T) {
		m := tagsModelInEntity(tags, noopSaveDescription)
		if m.descInput.Value() != "existing" {
			t.Errorf("expected prefilled description, got %q", m.descInput.Value())
		}
	})

	t.Run("esc returns to list without saving", func(t *testing.T) {
		called := false
		saveFn := func(_, _ string) error { called = true; return nil }
		m := tagsModelInEntity(tags, saveFn)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		updated := result.(tagsModel)
		if updated.phase != tagsPhaseList {
			t.Errorf("expected phase back to list, got %v", updated.phase)
		}
		if called {
			t.Error("expected save not to be called on esc")
		}
	})

	t.Run("ctrl+c quits entirely, not just back to list", func(t *testing.T) {
		m := tagsModelInEntity(tags, noopSaveDescription)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Error("expected quit cmd from ctrl+c in entity phase")
		}
	})

	t.Run("ctrl+s saves description and returns to list", func(t *testing.T) {
		var savedName, savedDesc string
		saveFn := func(name, desc string) error {
			savedName, savedDesc = name, desc
			return nil
		}
		m := tagsModelInEntity(tags, saveFn)
		m = typeString(m, " updated")
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		updated := result.(tagsModel)

		if updated.phase != tagsPhaseList {
			t.Errorf("expected phase back to list after save, got %v", updated.phase)
		}
		if savedName != "#auth" {
			t.Errorf("expected save called with #auth, got %q", savedName)
		}
		if savedDesc != "existing updated" {
			t.Errorf("expected save called with updated description, got %q", savedDesc)
		}
		if updated.tags[0].Description != "existing updated" {
			t.Errorf("expected local tag description updated, got %q", updated.tags[0].Description)
		}
		if updated.filtered[0].Description != "existing updated" {
			t.Errorf("expected filtered tag description updated, got %q", updated.filtered[0].Description)
		}
	})

	t.Run("ctrl+s save error keeps entity phase and records error", func(t *testing.T) {
		saveFn := func(_, _ string) error { return errors.New("db error") }
		m := tagsModelInEntity(tags, saveFn)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		updated := result.(tagsModel)
		if updated.phase != tagsPhaseEntity {
			t.Errorf("expected phase to remain entity on save error")
		}
		if updated.saveErr == nil {
			t.Error("expected saveErr to be set")
		}
	})
}

func TestTagsModelInit(t *testing.T) {
	m := initialTagsModel([]store.Tag{{Name: "#auth"}}, noopSaveDescription)
	if m.Init() == nil {
		t.Error("expected Init to return a non-nil cmd")
	}
}

func TestTagsModelNonKeyMessageIgnored(t *testing.T) {
	m := initialTagsModel([]store.Tag{{Name: "#auth"}}, noopSaveDescription)
	result, cmd := m.Update("not a key message")
	if result.(tagsModel).phase != tagsPhaseList {
		t.Error("expected phase unchanged after non-key message")
	}
	if cmd != nil {
		t.Error("expected nil cmd from non-key message")
	}
}

func TestTagsModelView(t *testing.T) {
	tags := []store.Tag{
		{Name: "#auth", Description: "auth stuff", NoteCount: 2},
		{Name: "#todo", NoteCount: 1},
	}

	t.Run("list view shows tag names and note counts", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		view := m.View()
		if !strings.Contains(view, "#auth") || !strings.Contains(view, "#todo") {
			t.Errorf("expected view to contain tag names, got: %s", view)
		}
		if !strings.Contains(view, "2") || !strings.Contains(view, "1") {
			t.Errorf("expected view to contain note counts, got: %s", view)
		}
	})

	t.Run("list view shows the filter placeholder", func(t *testing.T) {
		m := initialTagsModel(tags, noopSaveDescription)
		view := m.View()
		if !strings.Contains(view, "type to filter tags...") {
			t.Errorf("expected view to contain filter placeholder, got: %s", view)
		}
	})

	t.Run("list view shows empty-list text when filter matches nothing", func(t *testing.T) {
		m := typeString(initialTagsModel(tags, noopSaveDescription), "zzz")
		view := m.View()
		if !strings.Contains(view, "no matching tags") {
			t.Errorf("expected view to contain empty-list text, got: %s", view)
		}
	})

	t.Run("entity view shows tag name, note count, and description input", func(t *testing.T) {
		m := tagsModelInEntity(tags, noopSaveDescription)
		view := m.View()
		if !strings.Contains(view, "#auth") {
			t.Errorf("expected view to contain tag name, got: %s", view)
		}
		if !strings.Contains(view, "2") {
			t.Errorf("expected view to contain note count, got: %s", view)
		}
		if !strings.Contains(view, "auth stuff") {
			t.Errorf("expected view to contain description, got: %s", view)
		}
	})

	t.Run("entity view shows save error", func(t *testing.T) {
		saveFn := func(_, _ string) error { return errors.New("boom") }
		m := tagsModelInEntity(tags, saveFn)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		view := result.(tagsModel).View()
		if !strings.Contains(view, "boom") {
			t.Errorf("expected view to contain save error, got: %s", view)
		}
	})
}

func TestRunTags(t *testing.T) {
	t.Run("no tags in db prints message and returns nil", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := runTags(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("store open error is returned", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		os.WriteFile(filepath.Join(home, ".seton"), []byte("x"), 0644)
		if err := runTags(nil); err == nil {
			t.Error("expected error from store.Open")
		}
	})

	t.Run("runProg error is returned", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		seedTaggedNotes(t)
		stub := func(_ tea.Model) (tea.Model, error) { return nil, errors.New("terminal error") }
		if err := runTags(stub); err == nil {
			t.Error("expected error from runProg")
		}
	})

	t.Run("successful run persists saved description", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		seedTaggedNotes(t)

		db, err := store.Open()
		if err != nil {
			t.Fatalf("opening db: %v", err)
		}
		tags, err := store.ListTagsWithCounts(db)
		if err != nil {
			t.Fatalf("listing tags: %v", err)
		}
		db.Close()

		entity := tagsModelInEntity(tags, func(name, desc string) error { return nil })
		entity = typeString(entity, "my notes")
		final, _ := entity.Update(tea.KeyMsg{Type: tea.KeyCtrlS})

		if err := runTags(fixedTagsProg(final.(tagsModel))); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
