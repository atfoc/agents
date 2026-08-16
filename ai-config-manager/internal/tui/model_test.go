package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

// key builds a tea.KeyPressMsg the way Bubble Tea's own terminal input
// decoder builds one, so that feeding it to Update is equivalent to a real
// keypress. Verified by reading
// $(go env GOMODCACHE)/github.com/charmbracelet/ultraviolet@.../decoder.go
// and key.go against bubbletea v2.0.8 / ultraviolet
// v0.0.0-20260811164956-006e29f97886, and by TestKeyHelper below, which
// checks every case's .String() against what the caller asked for:
//
//   - A plain printable rune (e.g. "j", "/") carries both Text and Code —
//     Key.String() prefers Text when set.
//   - Named keys (enter, esc, up, down, left, right, backspace, delete,
//     home, end) carry only Code; String() falls back to Key.Keystroke(),
//     which maps the Code to its name.
//   - Space is Code: KeySpace, Text: " " — decoder.go's legacy-input path
//     sets Text to " " for a bare space, but Key.String() special-cases
//     Text == " " and falls back to Keystroke() anyway, which spells it
//     "space", not " ".
//   - ctrl+<letter> carries Code plus ModCtrl and, like the named keys, no
//     Text: decoder.go's legacy ctrl parsing never sets Text for these, and
//     if it did, String() would report the bare letter instead of
//     "ctrl+<letter>", since a non-empty Text short-circuits Keystroke().
func key(name string) tea.KeyPressMsg {
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+w":
		return tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl}
	default:
		// A single printable rune, e.g. "j", "k", "h", "l", "/", "a".
		r := []rune(name)[0]
		return tea.KeyPressMsg{Text: name, Code: r}
	}
}

// TestKeyHelper verifies that key(name).String() actually reports name, so
// every other test in this file can trust that feeding key("x") to Update
// is equivalent to the user pressing x.
func TestKeyHelper(t *testing.T) {
	names := []string{
		"enter", "esc", "up", "down", "left", "right", "backspace",
		"delete", "home", "end", "space", "ctrl+c", "ctrl+w",
		"j", "k", "h", "l", "/", "a",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if got := key(name).String(); got != name {
				t.Errorf("key(%q).String() = %q, want %q", name, got, name)
			}
		})
	}
}

// noopInstall is an InstallFunc that always succeeds and never touches
// disk, for tests that don't care about the install outcome itself.
func noopInstall(installer.Item) error { return nil }

// smallResult is the standard fixture: three agents and three skills, one
// item of each of the three groups per tab (to-update, to-install, up to
// date), sorted so that within a tab the selectable prefix is
// [to-update item, to-install item] and the up-to-date item is last.
func smallResult() installer.Result {
	return installer.Result{
		Agents: []installer.Item{
			{Kind: installer.KindAgent, Name: "alpha.md", Exists: true, Status: installer.StatusUpdated},
			{Kind: installer.KindAgent, Name: "bravo.md", Exists: false, Status: installer.StatusUpdated},
			{Kind: installer.KindAgent, Name: "charlie.md", Exists: true, Status: installer.StatusUnchanged},
		},
		Skills: []installer.Item{
			{Kind: installer.KindSkill, Name: "delta", Exists: true, Status: installer.StatusUpdated},
			{Kind: installer.KindSkill, Name: "echo", Exists: false, Status: installer.StatusUpdated},
			{Kind: installer.KindSkill, Name: "foxtrot", Exists: true, Status: installer.StatusUnchanged},
		},
	}
}

// manyAgentsResult builds n agents, all StatusUpdated/!Exists (so all
// selectable, all in one group), for exercising scrolling.
func manyAgentsResult(n int) installer.Result {
	items := make([]installer.Item, n)
	for i := 0; i < n; i++ {
		items[i] = installer.Item{
			Kind:   installer.KindAgent,
			Name:   fmt.Sprintf("item%02d.md", i),
			Exists: false,
			Status: installer.StatusUpdated,
		}
	}
	return installer.Result{Agents: items}
}

// update is a small test helper: it calls m.Update(msg), type-asserts the
// result back to Model (failing the test if that ever doesn't hold — it
// always should, since Model is the only tea.Model this package produces),
// and returns the new model and command.
func update(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	nm, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", next)
	}
	return nm, cmd
}

func sized(m Model, w, h int) Model {
	m.width, m.height = w, h
	return m
}

// 1. View().AltScreen must be false in every state — the hard requirement
// that keeps the program rendering inline instead of taking over the
// terminal.
func TestViewNeverSetsAltScreen(t *testing.T) {
	m := sized(New(smallResult(), noopInstall), 80, 24)

	checkNotAlt := func(t *testing.T, label string, m Model) {
		t.Helper()
		if v := m.View(); v.AltScreen {
			t.Errorf("%s: View().AltScreen = true, want false", label)
		}
	}

	checkNotAlt(t, "browse", m)

	m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] = true
	m.state = stateConfirm
	checkNotAlt(t, "confirm", m)

	m.state = stateInstalling
	checkNotAlt(t, "installing", m)

	m.state = stateDone
	checkNotAlt(t, "done", m)
}

// 2. space toggles selection; pressing it twice deselects.
func TestSpaceTogglesSelection(t *testing.T) {
	m := sized(New(smallResult(), noopInstall), 80, 24)
	k := selKey{Kind: installer.KindAgent, Name: "alpha.md"}

	m, _ = update(t, m, key("space"))
	if !m.selected[k] {
		t.Fatalf("after one space, alpha.md not selected")
	}

	m, _ = update(t, m, key("space"))
	if m.selected[k] {
		t.Fatalf("after two spaces, alpha.md still selected")
	}
}

// 3. a selection made on the Agents tab survives switching to Skills and
// back, and shows up in the confirm step alongside a Skills selection.
func TestSelectionSurvivesTabSwitch(t *testing.T) {
	m := sized(New(smallResult(), noopInstall), 80, 24)

	m, _ = update(t, m, key("space")) // select alpha.md on Agents
	m, _ = update(t, m, key("l"))     // -> Skills
	if m.tab != 1 {
		t.Fatalf("tab = %d, want 1 after l", m.tab)
	}
	m, _ = update(t, m, key("h")) // -> Agents
	if m.tab != 0 {
		t.Fatalf("tab = %d, want 0 after h", m.tab)
	}
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Fatalf("alpha.md selection lost after a tab round-trip")
	}

	m, _ = update(t, m, key("l"))     // -> Skills
	m, _ = update(t, m, key("space")) // select delta

	m, _ = update(t, m, key("enter"))
	if m.state != stateConfirm {
		t.Fatalf("state = %v, want stateConfirm", m.state)
	}
	if got := m.selectedNames(installer.KindAgent); len(got) != 1 || got[0] != "alpha.md" {
		t.Errorf("selectedNames(agent) = %v, want [alpha.md]", got)
	}
	if got := m.selectedNames(installer.KindSkill); len(got) != 1 || got[0] != "delta" {
		t.Errorf("selectedNames(skill) = %v, want [delta]", got)
	}
}

// 4. the cursor never lands on an up-to-date item, and space is a no-op on
// a tab with no selectable items.
func TestCursorSkipsUpToDateItems(t *testing.T) {
	m := sized(New(smallResult(), noopInstall), 80, 24)
	// selectableCount(0) == 2 (alpha.md, bravo.md); charlie.md (up to
	// date) must never be reachable by moving the cursor down.
	for i := 0; i < 5; i++ {
		m, _ = update(t, m, key("j"))
	}
	if m.cursor[0] != 1 {
		t.Fatalf("cursor[0] = %d, want 1 (clamped below charlie.md)", m.cursor[0])
	}

	upToDateOnly := installer.Result{
		Agents: []installer.Item{
			{Kind: installer.KindAgent, Name: "solo.md", Exists: true, Status: installer.StatusUnchanged},
		},
	}
	m2 := sized(New(upToDateOnly, noopInstall), 80, 24)
	if n := m2.selectableCount(0); n != 0 {
		t.Fatalf("selectableCount(0) = %d, want 0", n)
	}
	m2, _ = update(t, m2, key("space"))
	if len(m2.selected) != 0 {
		t.Errorf("space on an all-up-to-date tab selected something: %v", m2.selected)
	}
	if m2.cursor[0] != 0 {
		t.Errorf("cursor[0] = %d, want 0", m2.cursor[0])
	}
}

// 5. the cursor clamps at both ends without wrapping; tabs DO wrap.
func TestCursorClampsTabsWrap(t *testing.T) {
	m := sized(New(smallResult(), noopInstall), 80, 24)

	m, _ = update(t, m, key("k")) // already at top
	if m.cursor[0] != 0 {
		t.Fatalf("cursor[0] = %d after up at top, want 0 (no wrap)", m.cursor[0])
	}

	for i := 0; i < 5; i++ {
		m, _ = update(t, m, key("j"))
	}
	if m.cursor[0] != 1 {
		t.Fatalf("cursor[0] = %d after repeated down, want 1 (no wrap, selectableCount=2)", m.cursor[0])
	}

	m, _ = update(t, m, key("h"))
	if m.tab != 1 {
		t.Fatalf("tab = %d after h from tab 0, want 1 (wrap)", m.tab)
	}
	m, _ = update(t, m, key("h"))
	if m.tab != 0 {
		t.Fatalf("tab = %d after h from tab 1, want 0 (wrap)", m.tab)
	}
	m, _ = update(t, m, key("l"))
	if m.tab != 1 {
		t.Fatalf("tab = %d after l from tab 0, want 1 (wrap)", m.tab)
	}
}

// 6. scrolling: offset clamps at both ends, the cursor's row stays inside
// the window while moving down and back up, and visibleRows() floors at 3.
func TestScrolling(t *testing.T) {
	m := New(manyAgentsResult(20), noopInstall)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})

	checkWindow := func(t *testing.T, m Model) {
		t.Helper()
		rows := m.rows(m.tab)
		vr := m.visibleRows()
		off := m.offset[m.tab]
		if off < 0 {
			t.Fatalf("offset = %d, want >= 0", off)
		}
		maxOff := len(rows) - vr
		if maxOff < 0 {
			maxOff = 0
		}
		if off > maxOff {
			t.Fatalf("offset = %d, want <= %d", off, maxOff)
		}
		cursorRow := -1
		for i, r := range rows {
			if r.index == m.cursor[m.tab] {
				cursorRow = i
				break
			}
		}
		if cursorRow < off || cursorRow >= off+vr {
			t.Fatalf("cursor row %d outside window [%d, %d)", cursorRow, off, off+vr)
		}
	}

	for i := 0; i < 25; i++ {
		m, _ = update(t, m, key("j"))
		checkWindow(t, m)
	}
	for i := 0; i < 25; i++ {
		m, _ = update(t, m, key("k"))
		checkWindow(t, m)
	}
	// The cursor is back on the tab's first item, which sits at row 1 (row
	// 0 is the group heading). syncScroll only ever pulls the offset down
	// to the cursor's own row, never further just to reveal a heading, so
	// 1 - not 0 - is the correct rock-bottom offset here.
	if m.offset[m.tab] != 1 {
		t.Errorf("offset after returning to the top = %d, want 1", m.offset[m.tab])
	}

	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 1})
	if vr := m.visibleRows(); vr != 3 {
		t.Errorf("visibleRows() with height=1 = %d, want 3 (floor)", vr)
	}
}

// 7. search: opening, live narrowing, case-insensitive substring matching,
// commit-and-unfocus on enter, clear-and-close on esc, per-tab isolation,
// and survival across a tab round-trip.
func TestSearch(t *testing.T) {
	t.Run("slash opens search", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		if !m.searchOpen {
			t.Fatalf("searchOpen = false after /, want true")
		}
		if !m.search.Focused() {
			t.Errorf("search not focused after /")
		}
	})

	t.Run("typing narrows items, case-insensitively", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "ALPHA")

		items := m.visibleItems(0)
		if len(items) != 1 || items[0].Name != "alpha.md" {
			t.Fatalf("visibleItems(0) = %v, want just alpha.md", items)
		}
	})

	t.Run("enter commits and unfocuses, without installing", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "bravo")
		m, _ = update(t, m, key("enter"))

		if m.searchOpen {
			t.Errorf("searchOpen = true after enter, want false")
		}
		if m.search.Focused() {
			t.Errorf("search still focused after enter")
		}
		if m.query[0] != "bravo" {
			t.Errorf("query[0] = %q, want %q", m.query[0], "bravo")
		}
		if m.state != stateBrowse {
			t.Errorf("state = %v after enter in search, want stateBrowse", m.state)
		}
	})

	t.Run("esc closes and clears the query", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "alpha")
		m, _ = update(t, m, key("esc"))

		if m.searchOpen {
			t.Errorf("searchOpen = true after esc, want false")
		}
		if m.query[0] != "" {
			t.Errorf("query[0] = %q after esc, want empty", m.query[0])
		}
		if m.search.Value() != "" {
			t.Errorf("search.Value() = %q after esc, want empty", m.search.Value())
		}
	})

	t.Run("a query on one tab does not filter the other", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "alpha")
		m, _ = update(t, m, key("enter"))

		if items := m.visibleItems(1); len(items) != 3 {
			t.Errorf("visibleItems(1) with a tab-0 query set = %d items, want 3 (unfiltered)", len(items))
		}
	})

	t.Run("a committed query survives a tab round-trip", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "alpha")
		m, _ = update(t, m, key("enter"))

		m, _ = update(t, m, key("l"))
		m, _ = update(t, m, key("h"))

		if m.query[0] != "alpha" {
			t.Errorf("query[0] = %q after round-trip, want %q", m.query[0], "alpha")
		}
		if items := m.visibleItems(0); len(items) != 1 {
			t.Errorf("visibleItems(0) after round-trip = %d items, want 1", len(items))
		}
	})
}

// 8. key forwarding while the search input is open: j/k/h/l/space insert
// text and never move the cursor or change tabs; left/right/backspace/
// delete/home/end reach the input; esc/enter/up/down never reach it.
func TestSearchKeyForwarding(t *testing.T) {
	open := func(t *testing.T) Model {
		t.Helper()
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("/"))
		return m
	}

	t.Run("j k h l space insert text without moving the cursor or tab", func(t *testing.T) {
		m := open(t)
		startCursor, startTab := m.cursor[0], m.tab

		for _, k := range []string{"j", "k", "h", "l", "space"} {
			m, _ = update(t, m, key(k))
		}
		if got, want := m.search.Value(), "jkhl "; got != want {
			t.Fatalf("search.Value() = %q, want %q", got, want)
		}
		if m.cursor[0] != startCursor {
			t.Errorf("cursor[0] = %d while searching, want unchanged %d", m.cursor[0], startCursor)
		}
		if m.tab != startTab {
			t.Errorf("tab = %d while searching, want unchanged %d", m.tab, startTab)
		}
	})

	t.Run("left right backspace delete home end reach the input", func(t *testing.T) {
		m := open(t)
		for _, r := range "abc" {
			m, _ = update(t, m, key(string(r)))
		}
		if m.search.Value() != "abc" {
			t.Fatalf("setup: search.Value() = %q, want %q", m.search.Value(), "abc")
		}

		m, _ = update(t, m, key("backspace"))
		if m.search.Value() != "ab" {
			t.Fatalf("after backspace: search.Value() = %q, want %q", m.search.Value(), "ab")
		}

		m, _ = update(t, m, key("home"))
		if m.search.Position() != 0 {
			t.Fatalf("after home: cursor position = %d, want 0", m.search.Position())
		}

		m, _ = update(t, m, key("delete"))
		if m.search.Value() != "b" {
			t.Fatalf("after delete at start: search.Value() = %q, want %q", m.search.Value(), "b")
		}

		m, _ = update(t, m, key("end"))
		if want := len([]rune(m.search.Value())); m.search.Position() != want {
			t.Fatalf("after end: cursor position = %d, want %d", m.search.Position(), want)
		}

		m, _ = update(t, m, key("left"))
		if want := len([]rune(m.search.Value())) - 1; m.search.Position() != want {
			t.Fatalf("after left: cursor position = %d, want %d", m.search.Position(), want)
		}

		m, _ = update(t, m, key("right"))
		if want := len([]rune(m.search.Value())); m.search.Position() != want {
			t.Fatalf("after right: cursor position = %d, want %d", m.search.Position(), want)
		}
	})

	t.Run("esc enter up down never reach the input", func(t *testing.T) {
		m := open(t)

		// Deliberately typing nothing yet: smallResult's names contain
		// neither "x" nor "y", so typing "xy" here would filter the tab
		// down to zero selectable items and make the cursor correctly
		// refuse to move for an unrelated reason (see
		// TestCursorSkipsUpToDateItems), which would confuse this check.
		// down/up need at least one selectable item to prove they moved
		// the cursor at all.
		//
		// textinput's own default keymap binds "down"/"up" to cycling
		// suggestions, not text editing, so the only way to tell whether
		// they reached the model instead is that the model's cursor
		// actually moved.
		if m.cursor[0] != 0 {
			t.Fatalf("setup: cursor[0] = %d, want 0", m.cursor[0])
		}
		m, _ = update(t, m, key("down"))
		if m.cursor[0] != 1 {
			t.Fatalf("down while searching: cursor[0] = %d, want 1 (moved by the model)", m.cursor[0])
		}
		if m.search.Value() != "" {
			t.Fatalf("down mutated the input: %q, want unchanged empty", m.search.Value())
		}
		if !m.searchOpen {
			t.Fatalf("down closed the search, want still open")
		}

		m, _ = update(t, m, key("up"))
		if m.cursor[0] != 0 {
			t.Fatalf("up while searching: cursor[0] = %d, want 0", m.cursor[0])
		}
		if m.search.Value() != "" {
			t.Fatalf("up mutated the input: %q, want unchanged empty", m.search.Value())
		}

		m, _ = update(t, m, key("esc"))
		if m.searchOpen {
			t.Fatalf("esc did not close the search")
		}
		if m.query[0] != "" {
			t.Fatalf("esc did not clear the query: %q", m.query[0])
		}

		m, _ = update(t, m, key("/"))
		m = type_(t, m, "xy")
		m, _ = update(t, m, key("enter"))
		if m.searchOpen {
			t.Fatalf("enter did not close the search")
		}
		if m.query[0] != "xy" {
			t.Fatalf("query[0] = %q after enter, want %q", m.query[0], "xy")
		}
		if m.state != stateBrowse {
			t.Fatalf("state = %v after enter in search, want stateBrowse", m.state)
		}
	})
}

// type_ is a shared helper for TestSearchKeyForwarding, mirroring the one
// local to TestSearch.
func type_(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m, _ = update(t, m, key(string(r)))
	}
	return m
}

// 9. a selection made before searching survives being filtered out of
// view, and still installs.
func TestSelectionSurvivesFilterAndInstalls(t *testing.T) {
	var applied []string
	install := func(it installer.Item) error {
		applied = append(applied, it.Name)
		return nil
	}

	m := sized(New(smallResult(), install), 80, 24)

	m, _ = update(t, m, key("j"))     // cursor -> bravo.md
	m, _ = update(t, m, key("space")) // select bravo.md
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] {
		t.Fatalf("setup: bravo.md not selected")
	}

	m, _ = update(t, m, key("/"))
	m = type_(t, m, "alpha")
	m, _ = update(t, m, key("enter"))

	for _, it := range m.visibleItems(0) {
		if it.Name == "bravo.md" {
			t.Fatalf("bravo.md still visible under query %q", m.query[0])
		}
	}
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] {
		t.Fatalf("selection lost once bravo.md was filtered out of view")
	}

	m, _ = update(t, m, key("/"))
	m, _ = update(t, m, key("esc")) // clear the filter

	m, _ = update(t, m, key("enter")) // -> confirm
	if m.state != stateConfirm {
		t.Fatalf("state = %v, want stateConfirm", m.state)
	}
	var cmd tea.Cmd
	m, cmd = update(t, m, key("enter")) // -> installing
	if m.state != stateInstalling {
		t.Fatalf("state = %v, want stateInstalling", m.state)
	}
	if cmd == nil {
		t.Fatalf("no install command returned")
	}
	m, _ = update(t, m, cmd())

	if len(applied) != 1 || applied[0] != "bravo.md" {
		t.Fatalf("applied = %v, want [bravo.md]", applied)
	}
	if len(m.outcomes) != 1 || m.outcomes[0].Item.Name != "bravo.md" {
		t.Fatalf("outcomes = %v, want one outcome for bravo.md", m.outcomes)
	}
}

// 10. enter/esc transitions between stateBrowse and stateConfirm.
func TestBrowseConfirmTransitions(t *testing.T) {
	t.Run("enter with nothing selected stays in browse", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("enter"))
		if m.state != stateBrowse {
			t.Errorf("state = %v, want stateBrowse", m.state)
		}
	})

	t.Run("enter with a selection goes to confirm", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("space"))
		m, _ = update(t, m, key("enter"))
		if m.state != stateConfirm {
			t.Errorf("state = %v, want stateConfirm", m.state)
		}
	})

	t.Run("esc in confirm returns to browse without confirming", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("space"))
		m, _ = update(t, m, key("enter"))
		m, _ = update(t, m, key("esc"))
		if m.state != stateBrowse {
			t.Errorf("state = %v, want stateBrowse", m.state)
		}
		if m.confirmed {
			t.Errorf("confirmed = true after cancelling from confirm, want false")
		}
	})

	t.Run("enter in confirm starts the install", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("space"))
		m, _ = update(t, m, key("enter"))
		var cmd tea.Cmd
		m, cmd = update(t, m, key("enter"))
		if m.state != stateInstalling {
			t.Errorf("state = %v, want stateInstalling", m.state)
		}
		if !m.confirmed {
			t.Errorf("confirmed = false after starting the install, want true")
		}
		if cmd == nil {
			t.Errorf("no install command returned")
		}
	})
}

// 11. the full install flow: every selected item gets an outcome, one
// item's failure doesn't stop the rest, and Outcome().Confirmed reflects
// whether an install actually happened.
func TestInstallFlow(t *testing.T) {
	t.Run("outcomes recorded for every selected item; one failure doesn't stop the rest", func(t *testing.T) {
		failErr := errors.New("boom")
		install := func(it installer.Item) error {
			if it.Name == "bravo.md" {
				return failErr
			}
			return nil
		}

		m := sized(New(smallResult(), install), 80, 24)

		m, _ = update(t, m, key("space")) // select alpha.md
		m, _ = update(t, m, key("j"))
		m, _ = update(t, m, key("space")) // select bravo.md
		m, _ = update(t, m, key("l"))     // -> Skills
		m, _ = update(t, m, key("space")) // select delta

		m, _ = update(t, m, key("enter")) // -> confirm
		var cmd tea.Cmd
		m, cmd = update(t, m, key("enter")) // -> installing
		m, _ = update(t, m, cmd())

		if len(m.outcomes) != 3 {
			t.Fatalf("len(outcomes) = %d, want 3", len(m.outcomes))
		}
		errByName := map[string]error{}
		for _, oc := range m.outcomes {
			errByName[oc.Item.Name] = oc.Err
		}
		if errByName["alpha.md"] != nil {
			t.Errorf("alpha.md outcome err = %v, want nil", errByName["alpha.md"])
		}
		if !errors.Is(errByName["bravo.md"], failErr) {
			t.Errorf("bravo.md outcome err = %v, want %v", errByName["bravo.md"], failErr)
		}
		if errByName["delta"] != nil {
			t.Errorf("delta outcome err = %v, want nil", errByName["delta"])
		}
		if !m.Outcome().Confirmed {
			t.Errorf("Outcome().Confirmed = false after installing, want true")
		}
	})

	t.Run("Outcome().Confirmed is false when the user quits from browse", func(t *testing.T) {
		m := sized(New(smallResult(), noopInstall), 80, 24)
		m, _ = update(t, m, key("space"))
		m, _ = update(t, m, key("esc"))
		if m.Outcome().Confirmed {
			t.Errorf("Outcome().Confirmed = true after quitting from browse, want false")
		}
	})
}

// frameLines is how many lines the model's current frame actually occupies
// on screen.
func frameLines(m Model) int {
	return strings.Count(m.View().Content, "\n") + 1
}

// 12. every frame is exactly frameHeight lines tall, in every state, on
// either tab, with or without a search query, whatever is selected. This is
// the invariant that keeps the picker repainting in place.
//
// Bubble Tea v2.0.8's inline renderer moves the cursor back up by the *new*
// frame's height when a frame is shorter than the one before it, rather
// than by the old one's, so it starts redrawing part-way down the previous
// frame and leaves that frame's top lines behind. A frame that never
// shrinks never trips it. Before this was enforced, leaving the confirm box
// or switching from the taller tab to the shorter one walked the whole UI
// down the terminal a few lines at a time, one copy per keypress.
func TestFrameHeightIsConstant(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  installer.Result
		h    int
	}{
		{"small result, 24 rows", smallResult(), 24},
		{"small result, 10 rows", smallResult(), 10},
		{"list taller than the terminal", manyAgentsResult(60), 24},
		{"empty result", installer.Result{}, 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := sized(New(tc.res, noopInstall), 80, tc.h)
			want := m.frameHeight()
			if want > tc.h-1 && tc.h > 1 {
				t.Fatalf("frameHeight() = %d, want <= %d (one line short of the terminal)", want, tc.h-1)
			}

			check := func(label string, m Model) {
				t.Helper()
				if got := frameLines(m); got != want {
					t.Errorf("%s: frame is %d lines, want %d", label, got, want)
				}
			}

			check("browse, agents tab", m)

			m.tab = 1
			check("browse, skills tab", m)
			m.tab = 0

			m.query[0] = "zzz-matches-nothing"
			check("browse, query matching nothing", m)
			m.query[0] = ""

			m.searchOpen = true
			check("browse, search open", m)
			m.searchOpen = false

			// One selection, then everything selected: the confirm box grows
			// with the selection, and must still land on the same height.
			for _, kind := range []installer.Kind{installer.KindAgent, installer.KindSkill} {
				for _, it := range tc.res.Agents {
					if it.Kind == kind {
						m.selected[selKey{Kind: it.Kind, Name: it.Name}] = true
						break
					}
				}
			}
			m.state = stateConfirm
			check("confirm, one selection", m)

			for _, group := range [][]installer.Item{tc.res.Agents, tc.res.Skills} {
				for _, it := range group {
					m.selected[selKey{Kind: it.Kind, Name: it.Name}] = true
				}
			}
			check("confirm, everything selected", m)

			m.state = stateInstalling
			check("installing", m)

			m.state = stateDone
			check("done", m)
		})
	}
}

// 13. frameHeight depends only on the terminal size and the result — never
// on what the user has done. If browsing, searching or selecting could move
// it, "the frame never shrinks" would stop being true the moment the user
// backed out of any of them.
func TestFrameHeightIgnoresUserState(t *testing.T) {
	m := sized(New(smallResult(), noopInstall), 80, 24)
	want := m.frameHeight()

	steps := []string{"l", "j", "space", "h", "space", "/", "d", "e", "enter", "esc", "enter"}
	for _, k := range steps {
		m, _ = update(t, m, key(k))
		if got := m.frameHeight(); got != want {
			t.Fatalf("frameHeight() = %d after %q, want %d", got, k, want)
		}
	}
}

// 14. the confirm box stays inside the frame even when the terminal is too
// short to list every selected name, keeping its bottom border and — what
// actually matters — the "esc cancel" hint that gets the user back out. A
// box taller than the terminal sends the inline renderer into a redraw loop
// that never settles.
func TestConfirmBoxFitsShortTerminal(t *testing.T) {
	m := sized(New(manyAgentsResult(30), noopInstall), 80, 12)
	for _, it := range m.res.Agents {
		m.selected[selKey{Kind: it.Kind, Name: it.Name}] = true
	}
	m.state = stateConfirm

	content := m.View().Content
	if got, want := frameLines(m), m.frameHeight(); got != want {
		t.Errorf("confirm frame is %d lines, want %d", got, want)
	}
	for _, want := range []string{"… ", "esc cancel", "└"} {
		if !strings.Contains(content, want) {
			t.Errorf("confirm frame does not contain %q:\n%s", want, content)
		}
	}
}
