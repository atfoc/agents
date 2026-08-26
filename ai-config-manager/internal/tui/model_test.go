package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

// noopRemove is the RemoveFunc counterpart to noopInstall.
func noopRemove(installer.TargetItem) error { return nil }

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

// sampleTarget is the standard target-side fixture, joining smallResult:
// two agents and two skills, one of each kind also shipped by the source
// (so it lands in "also in source") and one only present at the target (so
// it lands in "only in target").
func sampleTarget() installer.TargetResult {
	return installer.TargetResult{
		Agents: []installer.TargetItem{
			{Kind: installer.KindAgent, Name: "alpha.md", InSource: true},
			{Kind: installer.KindAgent, Name: "stale.md", InSource: false},
		},
		Skills: []installer.TargetItem{
			{Kind: installer.KindSkill, Name: "delta", InSource: true},
			{Kind: installer.KindSkill, Name: "old-thing", InSource: false},
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
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

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

	m.marked[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] = true
	m.state = stateConflict
	checkNotAlt(t, "conflict", m)

	m.state = stateApplying
	checkNotAlt(t, "applying", m)

	m.state = stateDone
	checkNotAlt(t, "done", m)
}

// 2. space toggles selection; pressing it twice deselects.
func TestSpaceTogglesSelection(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

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
	if len(m.selected) != 2 {
		t.Fatalf("selected = %v, want exactly alpha.md and delta", m.selected)
	}
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Errorf("alpha.md not selected")
	}
	if !m.selected[selKey{Kind: installer.KindSkill, Name: "delta"}] {
		t.Errorf("delta not selected")
	}
}

// 4. the cursor never lands on an up-to-date item, and space is a no-op on
// a tab with no selectable items.
func TestCursorSkipsUpToDateItems(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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
	m2 := sized(New(upToDateOnly, installer.TargetResult{}, noopInstall, noopRemove), 80, 24)
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
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

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
	if m.tab != tabCount-1 {
		t.Fatalf("tab = %d after h from tab 0, want %d (wrap)", m.tab, tabCount-1)
	}
	m, _ = update(t, m, key("l"))
	if m.tab != 0 {
		t.Fatalf("tab = %d after l from the last tab, want 0 (wrap)", m.tab)
	}
	m, _ = update(t, m, key("l"))
	if m.tab != 1 {
		t.Fatalf("tab = %d after l from tab 0, want 1", m.tab)
	}
}

// 6. scrolling: offset clamps at both ends, the cursor's row stays inside
// the window while moving down and back up, and visibleRows() floors at 3.
func TestScrolling(t *testing.T) {
	m := New(manyAgentsResult(20), installer.TargetResult{}, noopInstall, noopRemove)
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
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("/"))
		if !m.searchOpen {
			t.Fatalf("searchOpen = false after /, want true")
		}
		if !m.search.Focused() {
			t.Errorf("search not focused after /")
		}
	})

	t.Run("typing narrows items, case-insensitively", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "ALPHA")

		entries := m.visibleEntries(0)
		if len(entries) != 1 || entries[0].name != "alpha.md" {
			t.Fatalf("visibleEntries(0) = %v, want just alpha.md", entries)
		}
	})

	t.Run("enter commits and unfocuses, without installing", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "alpha")
		m, _ = update(t, m, key("enter"))

		if entries := m.visibleEntries(1); len(entries) != 3 {
			t.Errorf("visibleEntries(1) with a tab-0 query set = %d entries, want 3 (unfiltered)", len(entries))
		}
	})

	t.Run("a committed query survives a tab round-trip", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("/"))
		m = type_(t, m, "alpha")
		m, _ = update(t, m, key("enter"))

		m, _ = update(t, m, key("l"))
		m, _ = update(t, m, key("h"))

		if m.query[0] != "alpha" {
			t.Errorf("query[0] = %q after round-trip, want %q", m.query[0], "alpha")
		}
		if entries := m.visibleEntries(0); len(entries) != 1 {
			t.Errorf("visibleEntries(0) after round-trip = %d entries, want 1", len(entries))
		}
	})
}

// 8. key forwarding while the search input is open: j/k/h/l/space insert
// text and never move the cursor or change tabs; left/right/backspace/
// delete/home/end reach the input; esc/enter/up/down never reach it.
func TestSearchKeyForwarding(t *testing.T) {
	open := func(t *testing.T) Model {
		t.Helper()
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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

	m := sized(New(smallResult(), sampleTarget(), install, noopRemove), 80, 24)

	m, _ = update(t, m, key("j"))     // cursor -> bravo.md
	m, _ = update(t, m, key("space")) // select bravo.md
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] {
		t.Fatalf("setup: bravo.md not selected")
	}

	m, _ = update(t, m, key("/"))
	m = type_(t, m, "alpha")
	m, _ = update(t, m, key("enter"))

	for _, e := range m.visibleEntries(0) {
		if e.name == "bravo.md" {
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
	if m.state != stateApplying {
		t.Fatalf("state = %v, want stateApplying", m.state)
	}
	if cmd == nil {
		t.Fatalf("no install command returned")
	}
	m, _ = update(t, m, cmd())

	if len(applied) != 1 || applied[0] != "bravo.md" {
		t.Fatalf("applied = %v, want [bravo.md]", applied)
	}
	if len(m.outcomes) != 1 || m.outcomes[0].Name != "bravo.md" {
		t.Fatalf("outcomes = %v, want one outcome for bravo.md", m.outcomes)
	}
}

// 10. enter/esc transitions between stateBrowse and stateConfirm.
func TestBrowseConfirmTransitions(t *testing.T) {
	t.Run("enter with nothing selected stays in browse", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("enter"))
		if m.state != stateBrowse {
			t.Errorf("state = %v, want stateBrowse", m.state)
		}
	})

	t.Run("enter with a selection goes to confirm", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("space"))
		m, _ = update(t, m, key("enter"))
		if m.state != stateConfirm {
			t.Errorf("state = %v, want stateConfirm", m.state)
		}
	})

	t.Run("esc in confirm returns to browse without confirming", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m, _ = update(t, m, key("space"))
		m, _ = update(t, m, key("enter"))
		var cmd tea.Cmd
		m, cmd = update(t, m, key("enter"))
		if m.state != stateApplying {
			t.Errorf("state = %v, want stateApplying", m.state)
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

		m := sized(New(smallResult(), sampleTarget(), install, noopRemove), 80, 24)

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
			errByName[oc.Name] = oc.Err
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
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
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
		tgt  installer.TargetResult
		h    int
	}{
		{"small result, 24 rows", smallResult(), sampleTarget(), 24},
		{"small result, 10 rows", smallResult(), sampleTarget(), 10},
		{"list taller than the terminal", manyAgentsResult(60), sampleTarget(), 24},
		{"empty result", installer.Result{}, installer.TargetResult{}, 24},
		{"empty source, populated target", installer.Result{}, sampleTarget(), 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := sized(New(tc.res, tc.tgt, noopInstall, noopRemove), 80, tc.h)
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

			for tab := 0; tab < tabCount; tab++ {
				m.tab = tab
				check(fmt.Sprintf("browse, tab %d (%s)", tab, tabs[tab].label), m)
			}
			m.tab = 0

			m.query[0] = "zzz-matches-nothing"
			check("browse, query matching nothing", m)
			m.query[0] = ""

			m.searchOpen = true
			check("browse, search open", m)
			m.searchOpen = false

			// One mark on each side, then everything marked: the confirm box
			// grows with what is marked, and must still land on the same
			// height.
			if len(tc.res.Agents) > 0 {
				it := tc.res.Agents[0]
				m.selected[selKey{Kind: it.Kind, Name: it.Name}] = true
			}
			if len(tc.tgt.Agents) > 0 {
				it := tc.tgt.Agents[0]
				m.marked[selKey{Kind: it.Kind, Name: it.Name}] = true
			}
			m.state = stateConfirm
			check("confirm, one mark on each side", m)

			for _, group := range [][]installer.Item{tc.res.Agents, tc.res.Skills} {
				for _, it := range group {
					m.selected[selKey{Kind: it.Kind, Name: it.Name}] = true
				}
			}
			for _, group := range [][]installer.TargetItem{tc.tgt.Agents, tc.tgt.Skills} {
				for _, it := range group {
					m.marked[selKey{Kind: it.Kind, Name: it.Name}] = true
				}
			}
			check("confirm, everything marked", m)

			// Everything marked on both sides is also the worst case for
			// the conflict box: every item present on both sides is now
			// conflicted.
			m.state = stateConflict
			check("conflict, everything marked", m)

			m.state = stateApplying
			check("applying", m)

			m.state = stateDone
			check("done", m)
		})
	}
}

// 13. frameHeight depends only on the terminal size and the result — never
// on what the user has done. If browsing, marking, conflicting or searching
// could move it, "the frame never shrinks" would stop being true the moment
// the user backed out of any of them.
func TestFrameHeightIgnoresUserState(t *testing.T) {
	want := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24).frameHeight()

	for _, tc := range []struct {
		name  string
		steps []string
	}{
		{
			"browsing, marking, searching and confirming",
			[]string{"l", "j", "space", "h", "space", "/", "d", "e", "enter", "esc", "enter"},
		},
		{
			// space on tab 0 marks alpha.md for install, space on tab 2
			// marks the same alpha.md for removal, and the enter after
			// that lands in the conflict box.
			"marking both sides of one item, then the conflict box",
			[]string{"space", "l", "l", "space", "enter", "esc", "/", "a", "enter"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
			for _, k := range tc.steps {
				m, _ = update(t, m, key(k))
				if got := m.frameHeight(); got != want {
					t.Fatalf("frameHeight() = %d after %q, want %d", got, k, want)
				}
			}
		})
	}
}

// 14. the confirm box stays inside the frame even when the terminal is too
// short to list every selected name, keeping its bottom border and — what
// actually matters — the "esc cancel" hint that gets the user back out. A
// box taller than the terminal sends the inline renderer into a redraw loop
// that never settles.
func TestConfirmBoxFitsShortTerminal(t *testing.T) {
	m := sized(New(manyAgentsResult(30), sampleTarget(), noopInstall, noopRemove), 80, 12)
	for _, it := range m.res.Agents {
		m.selected[selKey{Kind: it.Kind, Name: it.Name}] = true
	}
	for _, it := range m.tgt.Agents {
		m.marked[selKey{Kind: it.Kind, Name: it.Name}] = true
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

// 15. h and l walk all four tabs in opposite directions, and both wrap.
// With two tabs they were the same operation; with four they are not, so
// this is the test that would catch one of them being left as a toggle.
func TestTabsCycleInBothDirections(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	for want := 1; want < tabCount; want++ {
		m, _ = update(t, m, key("l"))
		if m.tab != want {
			t.Fatalf("tab = %d after %d l presses, want %d", m.tab, want, want)
		}
	}
	m, _ = update(t, m, key("l"))
	if m.tab != 0 {
		t.Fatalf("tab = %d after l on the last tab, want 0 (wrap)", m.tab)
	}

	m, _ = update(t, m, key("h"))
	if m.tab != tabCount-1 {
		t.Fatalf("tab = %d after h on tab 0, want %d (wrap)", m.tab, tabCount-1)
	}
	for want := tabCount - 2; want >= 0; want-- {
		m, _ = update(t, m, key("h"))
		if m.tab != want {
			t.Fatalf("tab = %d walking back with h, want %d", m.tab, want)
		}
	}
}

// entryByName finds an entry by name in a tab's list.
func entryByName(entries []entry, name string) (entry, bool) {
	for _, e := range entries {
		if e.name == name {
			return e, true
		}
	}
	return entry{}, false
}

// 16. an install tab flattens the source's own three groups, and only its
// up-to-date items are unselectable.
func TestTabEntries_InstallTabUsesSourceGroups(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	for tab, want := range map[int]map[string]struct {
		group      string
		rank       int
		selectable bool
	}{
		0: {
			"alpha.md":   {"to update", int(installer.GroupToUpdate), true},
			"bravo.md":   {"to install", int(installer.GroupToInstall), true},
			"charlie.md": {"up to date", int(installer.GroupUpToDate), false},
		},
		1: {
			"delta":   {"to update", int(installer.GroupToUpdate), true},
			"echo":    {"to install", int(installer.GroupToInstall), true},
			"foxtrot": {"up to date", int(installer.GroupUpToDate), false},
		},
	} {
		entries := m.tabEntries(tab)
		if len(entries) != len(want) {
			t.Fatalf("tabEntries(%d) = %d entries, want %d", tab, len(entries), len(want))
		}
		for name, w := range want {
			e, ok := entryByName(entries, name)
			if !ok {
				t.Fatalf("tabEntries(%d) missing %q", tab, name)
			}
			if e.group != w.group || e.groupRank != w.rank {
				t.Errorf("tabEntries(%d)[%q] group = %q/%d, want %q/%d",
					tab, name, e.group, e.groupRank, w.group, w.rank)
			}
			if e.selectable != w.selectable {
				t.Errorf("tabEntries(%d)[%q] selectable = %v, want %v",
					tab, name, e.selectable, w.selectable)
			}
			if e.kind != tabs[tab].kind {
				t.Errorf("tabEntries(%d)[%q] kind = %v, want %v", tab, name, e.kind, tabs[tab].kind)
			}
		}
	}
}

// 17. every entry on an uninstall tab is selectable, including one the
// source also ships byte-identically: it is there, so it can go.
func TestTabEntries_UninstallTabAllSelectable(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	for _, tab := range []int{2, 3} {
		entries := m.tabEntries(tab)
		if len(entries) == 0 {
			t.Fatalf("tabEntries(%d) is empty", tab)
		}
		for _, e := range entries {
			if !e.selectable {
				t.Errorf("tabEntries(%d)[%q] selectable = false, want true", tab, e.name)
			}
			if e.kind != tabs[tab].kind {
				t.Errorf("tabEntries(%d)[%q] kind = %v, want %v", tab, e.name, e.kind, tabs[tab].kind)
			}
		}
		if n := m.selectableCount(tab); n != len(entries) {
			t.Errorf("selectableCount(%d) = %d, want %d", tab, n, len(entries))
		}
	}

	// "charlie.md" is up to date on the install tab, i.e. byte-identical to
	// the source, and must still be removable when it is in the target.
	tgt := installer.TargetResult{
		Agents: []installer.TargetItem{{Kind: installer.KindAgent, Name: "charlie.md", InSource: true}},
	}
	m2 := sized(New(smallResult(), tgt, noopInstall, noopRemove), 80, 24)
	e, ok := entryByName(m2.tabEntries(2), "charlie.md")
	if !ok {
		t.Fatalf("tabEntries(2) missing charlie.md")
	}
	if !e.selectable {
		t.Errorf("an up-to-date item is not selectable on the uninstall tab, want selectable")
	}
}

// 18. an uninstall tab groups "also in source" ahead of "only in target",
// and visibleEntries' sort puts the whole first group ahead of the second.
func TestTabEntries_UninstallGroupsAlsoInSourceFirst(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	entries := m.visibleEntries(2)
	var order []string
	for _, e := range entries {
		order = append(order, e.name+"/"+e.group)
	}
	want := []string{"alpha.md/also in source", "stale.md/only in target"}
	if len(order) != len(want) {
		t.Fatalf("visibleEntries(2) = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("visibleEntries(2) = %v, want %v", order, want)
		}
	}

	// rows() must emit one heading per group, in that order.
	var headings []string
	for _, r := range m.rows(2) {
		if r.index == -1 {
			headings = append(headings, r.heading)
		}
	}
	if len(headings) != 2 || headings[0] != "also in source" || headings[1] != "only in target" {
		t.Errorf("rows(2) headings = %v, want [also in source, only in target]", headings)
	}
}

// 19. space writes into the map its tab owns: the install tabs into
// selected, the uninstall tabs into marked. Two maps rather than one is
// what makes a conflict a set intersection, so which map a mark lands in
// has to be right.
func TestMarkGoesToTheRightMap(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	m, _ = update(t, m, key("space")) // tab 0, alpha.md -> selected
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Fatalf("space on tab 0 did not select alpha.md")
	}
	if len(m.marked) != 0 {
		t.Fatalf("space on tab 0 wrote into marked: %v", m.marked)
	}

	m, _ = update(t, m, key("l"))
	m, _ = update(t, m, key("l"))     // -> tab 2, Uninstall Agents
	m, _ = update(t, m, key("space")) // alpha.md -> marked
	if !m.marked[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Fatalf("space on tab 2 did not mark alpha.md for removal")
	}
	if len(m.selected) != 1 {
		t.Fatalf("space on tab 2 changed selected: %v", m.selected)
	}

	// Pressing it again on the uninstall tab clears only the removal mark.
	m, _ = update(t, m, key("space"))
	if len(m.marked) != 0 {
		t.Fatalf("second space on tab 2 did not unmark: %v", m.marked)
	}
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Fatalf("second space on tab 2 cleared the install selection")
	}
}

// 20. marks made on any of the four tabs survive switching tabs and being
// filtered out of view by a search.
func TestMarksSurviveTabSwitchAndSearch(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	// Mark one item on every tab.
	for tab := 0; tab < tabCount; tab++ {
		m.tab = tab
		m.cursor[tab] = 0
		m, _ = update(t, m, key("space"))
	}
	if len(m.selected) != 2 || len(m.marked) != 2 {
		t.Fatalf("after one mark per tab: selected = %v, marked = %v", m.selected, m.marked)
	}

	// A full lap of the tabs must not disturb them.
	for i := 0; i < tabCount; i++ {
		m, _ = update(t, m, key("l"))
	}
	if len(m.selected) != 2 || len(m.marked) != 2 {
		t.Fatalf("after a full tab lap: selected = %v, marked = %v", m.selected, m.marked)
	}

	// A query on the uninstall-agents tab that matches nothing must not
	// clear the mark it hides.
	m.tab = 2
	m, _ = update(t, m, key("/"))
	m = type_(t, m, "zzz")
	m, _ = update(t, m, key("enter"))
	if len(m.visibleEntries(2)) != 0 {
		t.Fatalf("query %q still matches entries on tab 2", m.query[2])
	}
	if !m.marked[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Errorf("removal mark lost once the row was filtered out of view")
	}
}

// 21. each of the four tabs remembers its own query, and one tab's query
// never filters another's list.
func TestQueryIsPerTabAcrossFourTabs(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)

	queries := [tabCount]string{"alpha", "delta", "stale", "old"}
	for tab := 0; tab < tabCount; tab++ {
		m.tab = tab
		m, _ = update(t, m, key("/"))
		m = type_(t, m, queries[tab])
		m, _ = update(t, m, key("enter"))
	}

	for tab := 0; tab < tabCount; tab++ {
		if m.query[tab] != queries[tab] {
			t.Errorf("query[%d] = %q, want %q", tab, m.query[tab], queries[tab])
		}
		if n := len(m.visibleEntries(tab)); n != 1 {
			t.Errorf("visibleEntries(%d) under query %q = %d entries, want 1",
				tab, m.query[tab], n)
		}
	}
}

// 22. the tab bar signs an install count "+" and a removal count "-", so
// the two can never be read as the same number.
func TestTabBarSignsCounts(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 200, 24)

	m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] = true
	m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] = true
	m.marked[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] = true
	m.marked[selKey{Kind: installer.KindAgent, Name: "stale.md"}] = true
	m.marked[selKey{Kind: installer.KindSkill, Name: "delta"}] = true

	bar := m.tabBar()
	for _, want := range []string{
		"Agents (+2)", "Skills (+0)", "Uninstall Agents (-2)", "Uninstall Skills (-1)",
	} {
		if !strings.Contains(bar, want) {
			t.Errorf("tabBar() = %q, want it to contain %q", bar, want)
		}
	}
}

// 23. the tab bar never wraps: a second line would add a line to the
// frame, which is exactly the invariant frameHeight exists to hold.
func TestTabBarNeverExceedsWidth(t *testing.T) {
	for _, w := range []int{40, 60, 80, 200} {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), w, 24)
		bar := m.tabBar()
		if strings.Contains(bar, "\n") {
			t.Errorf("width %d: tabBar() wrapped onto a second line: %q", w, bar)
		}
		if got := lipgloss.Width(bar); got > w {
			t.Errorf("width %d: tabBar() is %d columns wide, want <= %d", w, got, w)
		}
	}

	// The frame itself must still be exactly frameHeight lines at a width
	// that forces the bar to be truncated.
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 40, 24)
	if got, want := frameLines(m), m.frameHeight(); got != want {
		t.Errorf("frame at width 40 is %d lines, want %d", got, want)
	}
}

// 24. the confirm box puts the destructive section first, marks it as
// destructive, and signs its lines "-" against the installs' "+".
func TestConfirmBoxListsRemovalsFirst(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
	m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] = true
	m.marked[selKey{Kind: installer.KindAgent, Name: "stale.md"}] = true
	m.marked[selKey{Kind: installer.KindSkill, Name: "old-thing"}] = true
	m.state = stateConfirm

	content := m.View().Content

	removeAt := strings.Index(content, "remove — deletes these from the target:")
	installAt := strings.Index(content, "install:")
	if removeAt < 0 {
		t.Fatalf("confirm box has no remove section:\n%s", content)
	}
	if installAt < 0 {
		t.Fatalf("confirm box has no install section:\n%s", content)
	}
	if removeAt > installAt {
		t.Errorf("install section comes before the remove section:\n%s", content)
	}

	for _, want := range []string{
		"- agent  stale.md",
		"- skill  old-thing",
		"+ agent  bravo.md",
		"2 to remove, 1 to install",
		"enter apply · esc cancel",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("confirm box does not contain %q:\n%s", want, content)
		}
	}
}

// 25. a section with nothing in it is omitted entirely rather than printed
// as "(none)": an empty destructive section is noise on a screen whose
// whole job is to make the destructive part impossible to miss.
func TestConfirmBoxOmitsEmptySection(t *testing.T) {
	t.Run("installs only", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] = true
		m.state = stateConfirm

		content := m.View().Content
		if strings.Contains(content, "remove —") {
			t.Errorf("confirm box shows an empty remove section:\n%s", content)
		}
		if strings.Contains(content, "(none)") {
			t.Errorf("confirm box still prints (none):\n%s", content)
		}
		if !strings.Contains(content, "0 to remove, 1 to install") {
			t.Errorf("confirm box count line wrong:\n%s", content)
		}
	})

	t.Run("removals only", func(t *testing.T) {
		m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
		m.marked[selKey{Kind: installer.KindSkill, Name: "old-thing"}] = true
		m.state = stateConfirm

		content := m.View().Content
		if strings.Contains(content, "install:") {
			t.Errorf("confirm box shows an empty install section:\n%s", content)
		}
		if !strings.Contains(content, "1 to remove, 0 to install") {
			t.Errorf("confirm box count line wrong:\n%s", content)
		}
	})
}

// markEverything marks one agent and one skill on each side, so an apply
// exercises both halves and both kinds.
func markEverything(m Model) Model {
	m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] = true
	m.selected[selKey{Kind: installer.KindSkill, Name: "delta"}] = true
	m.marked[selKey{Kind: installer.KindAgent, Name: "stale.md"}] = true
	m.marked[selKey{Kind: installer.KindSkill, Name: "old-thing"}] = true
	return m
}

// applyNow drives the model from stateConfirm through the apply command
// and returns the resulting model.
func applyNow(t *testing.T, m Model) Model {
	t.Helper()
	m.state = stateConfirm
	var cmd tea.Cmd
	m, cmd = update(t, m, key("enter"))
	if m.state != stateApplying {
		t.Fatalf("state = %v after confirming, want stateApplying", m.state)
	}
	if cmd == nil {
		t.Fatal("no apply command returned")
	}
	m, _ = update(t, m, cmd())
	if m.state != stateDone {
		t.Fatalf("state = %v after applying, want stateDone", m.state)
	}
	return m
}

// 26. every removal runs before every install. Conflicts are impossible by
// then, so the order cannot change the outcome — but it is the order the
// confirmation box reads in, and the report reads back in.
func TestApplyOrder_RemovalsBeforeInstalls(t *testing.T) {
	var log []string
	install := func(it installer.Item) error {
		log = append(log, "install "+it.Name)
		return nil
	}
	remove := func(it installer.TargetItem) error {
		log = append(log, "remove "+it.Name)
		return nil
	}

	m := markEverything(sized(New(smallResult(), sampleTarget(), install, remove), 80, 24))
	m = applyNow(t, m)

	if len(log) != 4 {
		t.Fatalf("log = %v, want 4 entries", log)
	}
	lastRemove, firstInstall := -1, -1
	for i, l := range log {
		if strings.HasPrefix(l, "remove ") {
			lastRemove = i
		}
		if strings.HasPrefix(l, "install ") && firstInstall == -1 {
			firstInstall = i
		}
	}
	if lastRemove > firstInstall {
		t.Errorf("log = %v, want every remove before every install", log)
	}

	// The outcomes are recorded in the same order they were performed.
	for i, oc := range m.outcomes {
		wantAction := actionRemove
		if i >= 2 {
			wantAction = actionInstall
		}
		if oc.Action != wantAction {
			t.Errorf("outcomes[%d] (%s) Action = %v, want %v", i, oc.Name, oc.Action, wantAction)
		}
	}
}

// 27. an outcome is recorded for every marked item, and a failing removal
// does not stop the ones behind it — the report has to account for
// everything the user approved, not just what happened before the first
// failure.
func TestApplyRecordsOutcomeForEveryMarkedItem(t *testing.T) {
	failErr := errors.New("boom")
	remove := func(it installer.TargetItem) error {
		if it.Name == "stale.md" {
			return failErr
		}
		return nil
	}

	var installed []string
	install := func(it installer.Item) error {
		installed = append(installed, it.Name)
		return nil
	}

	m := markEverything(sized(New(smallResult(), sampleTarget(), install, remove), 80, 24))
	m = applyNow(t, m)

	if len(m.outcomes) != 4 {
		t.Fatalf("len(outcomes) = %d, want 4", len(m.outcomes))
	}
	errByName := map[string]error{}
	for _, oc := range m.outcomes {
		errByName[oc.Name] = oc.Err
	}
	if !errors.Is(errByName["stale.md"], failErr) {
		t.Errorf("stale.md outcome err = %v, want %v", errByName["stale.md"], failErr)
	}
	for _, name := range []string{"old-thing", "alpha.md", "delta"} {
		if _, ok := errByName[name]; !ok {
			t.Errorf("no outcome recorded for %q", name)
		}
		if errByName[name] != nil {
			t.Errorf("%s outcome err = %v, want nil", name, errByName[name])
		}
	}
	if len(installed) != 2 {
		t.Errorf("installs = %v, want both to have run despite the failed removal", installed)
	}
}

// 28. each outcome carries the action that produced it and, for installs
// only, whether it replaced something — the two fields the report needs to
// tell removed from installed from updated.
func TestApplyOutcomeCarriesActionAndExists(t *testing.T) {
	m := markEverything(sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24))
	m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] = true
	m = applyNow(t, m)

	byName := map[string]ItemOutcome{}
	for _, oc := range m.outcomes {
		byName[oc.Name] = oc
	}

	// alpha.md is Exists:true in smallResult (an update); bravo.md is
	// Exists:false (a fresh install).
	for name, want := range map[string]struct {
		action action
		kind   installer.Kind
		exists bool
	}{
		"stale.md":  {actionRemove, installer.KindAgent, false},
		"old-thing": {actionRemove, installer.KindSkill, false},
		"alpha.md":  {actionInstall, installer.KindAgent, true},
		"bravo.md":  {actionInstall, installer.KindAgent, false},
		"delta":     {actionInstall, installer.KindSkill, true},
	} {
		oc, ok := byName[name]
		if !ok {
			t.Fatalf("no outcome for %q", name)
		}
		if oc.Action != want.action {
			t.Errorf("%s Action = %v, want %v", name, oc.Action, want.action)
		}
		if oc.Kind != want.kind {
			t.Errorf("%s Kind = %v, want %v", name, oc.Kind, want.kind)
		}
		if oc.Exists != want.exists {
			t.Errorf("%s Exists = %v, want %v", name, oc.Exists, want.exists)
		}
	}
}

// 29. a missing or empty target is not an error: the uninstall tabs are
// simply empty, and the install tabs keep working normally.
func TestEmptyTargetShowsEmptyUninstallTabs(t *testing.T) {
	m := sized(New(smallResult(), installer.TargetResult{}, noopInstall, noopRemove), 80, 24)

	for tab, want := range map[int]string{2: "no agents found", 3: "no skills found"} {
		m.tab = tab
		lines := m.listLines()
		if len(lines) != 1 || lines[0] != want {
			t.Errorf("tab %d listLines() = %v, want [%q]", tab, lines, want)
		}
		if n := m.selectableCount(tab); n != 0 {
			t.Errorf("selectableCount(%d) = %d, want 0", tab, n)
		}
	}

	// space on an empty uninstall tab must do nothing at all.
	m.tab = 2
	m, _ = update(t, m, key("space"))
	if len(m.marked) != 0 {
		t.Errorf("space on an empty uninstall tab marked something: %v", m.marked)
	}

	// The install tabs are unaffected.
	m.tab = 0
	m, _ = update(t, m, key("space"))
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Errorf("the install tab stopped working with an empty target")
	}
}

// conflictedModel returns a model with alpha.md marked for install on tab 0
// and the same alpha.md marked for removal on tab 2 — the smallest real
// conflict the standard fixtures can make, since sampleTarget's alpha.md is
// the agent smallResult also ships.
func conflictedModel(t *testing.T) Model {
	t.Helper()
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
	m, _ = update(t, m, key("space")) // tab 0: install alpha.md
	m.tab = 2
	m, _ = update(t, m, key("space")) // tab 2: remove alpha.md
	return m
}

// 30. a conflict is one selKey sitting in both maps at once, and conflicts()
// lists them in a stable order. A mark on one side alone is not a conflict:
// that is the ordinary case the picker exists for.
func TestConflictDetected(t *testing.T) {
	agent := selKey{Kind: installer.KindAgent, Name: "alpha.md"}
	skill := selKey{Kind: installer.KindSkill, Name: "delta"}

	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
	if m.inConflict(agent) {
		t.Errorf("inConflict on an unmarked item = true, want false")
	}

	m.selected[agent] = true
	if m.inConflict(agent) {
		t.Errorf("inConflict on an install-only mark = true, want false")
	}
	if got := m.conflicts(); len(got) != 0 {
		t.Errorf("conflicts() = %v with nothing marked twice, want none", got)
	}

	m.marked[agent] = true
	if !m.inConflict(agent) {
		t.Errorf("inConflict on an item marked for install and removal = false, want true")
	}

	m.marked[skill] = true
	if m.inConflict(skill) {
		t.Errorf("inConflict on a removal-only mark = true, want false")
	}
	m.selected[skill] = true

	got := m.conflicts()
	want := []selKey{agent, skill} // KindAgent sorts before KindSkill
	if len(got) != len(want) {
		t.Fatalf("conflicts() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("conflicts()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// 31. a conflict is same kind + same name. An agent and a skill that merely
// share a name string are unrelated items and never conflict, so marking
// one for install and the other for removal must sail straight through to
// the confirm box.
func TestConflictIsKindScoped(t *testing.T) {
	res := installer.Result{
		Agents: []installer.Item{
			{Kind: installer.KindAgent, Name: "research", Exists: false, Status: installer.StatusUpdated},
		},
	}
	tgt := installer.TargetResult{
		Skills: []installer.TargetItem{
			{Kind: installer.KindSkill, Name: "research", InSource: false},
		},
	}

	m := sized(New(res, tgt, noopInstall, noopRemove), 80, 24)
	m, _ = update(t, m, key("space")) // tab 0: install the agent "research"
	m.tab = 3
	m, _ = update(t, m, key("space")) // tab 3: remove the skill "research"

	if !m.selected[selKey{Kind: installer.KindAgent, Name: "research"}] {
		t.Fatalf("the agent was not marked for install: %v", m.selected)
	}
	if !m.marked[selKey{Kind: installer.KindSkill, Name: "research"}] {
		t.Fatalf("the skill was not marked for removal: %v", m.marked)
	}
	if got := m.conflicts(); len(got) != 0 {
		t.Errorf("conflicts() = %v for an agent and a skill sharing a name, want none", got)
	}

	m, _ = update(t, m, key("enter"))
	if m.state != stateConfirm {
		t.Errorf("state = %v, want stateConfirm — a shared name across kinds must not block", m.state)
	}
}

// 32. the conflicted row is marked in both tabs the instant it happens, so
// the user sees it while still picking rather than after marking twenty
// things. "[!]" replaces the "[x]" it would otherwise carry.
func TestConflictRowRendersInBothTabs(t *testing.T) {
	m := conflictedModel(t)

	for _, tab := range []int{0, 2} {
		m.tab = tab
		content := m.View().Content
		for _, want := range []string{"[!] alpha.md", "(conflict)"} {
			if !strings.Contains(content, want) {
				t.Errorf("tab %d (%s) does not contain %q:\n%s", tab, tabs[tab].label, want, content)
			}
		}
		if strings.Contains(content, "[x] alpha.md") {
			t.Errorf("tab %d (%s) still draws the conflicted row as [x]:\n%s", tab, tabs[tab].label, content)
		}
	}

	// The help line says enter will refuse, rather than promising an apply.
	if got := m.helpLine(); !strings.Contains(got, "enter (conflicts)") {
		t.Errorf("helpLine() = %q, want it to mention the conflicts", got)
	}
}

// 33. enter refuses: it reaches the conflict box and never the confirm box,
// starts no apply command, and leaves confirmed false.
func TestConflictBlocksEnter(t *testing.T) {
	m := conflictedModel(t)

	m, cmd := update(t, m, key("enter"))
	if m.state != stateConflict {
		t.Fatalf("state = %v, want stateConflict", m.state)
	}
	if cmd != nil {
		t.Errorf("enter with a conflict returned a command, want none")
	}
	if m.confirmed {
		t.Errorf("confirmed = true after a refused enter, want false")
	}
}

// 34. the conflict box names every conflicting item, and only those: a mark
// that is not part of a conflict has nothing to unmark.
func TestConflictBoxListsEveryConflict(t *testing.T) {
	m := sized(New(smallResult(), sampleTarget(), noopInstall, noopRemove), 80, 24)
	for _, k := range []selKey{
		{Kind: installer.KindAgent, Name: "alpha.md"},
		{Kind: installer.KindSkill, Name: "delta"},
	} {
		m.selected[k] = true
		m.marked[k] = true
	}
	// Two marks that are not conflicts, one on each side.
	m.selected[selKey{Kind: installer.KindAgent, Name: "bravo.md"}] = true
	m.marked[selKey{Kind: installer.KindAgent, Name: "stale.md"}] = true

	m, _ = update(t, m, key("enter"))
	if m.state != stateConflict {
		t.Fatalf("state = %v, want stateConflict", m.state)
	}

	content := m.View().Content
	for _, want := range []string{
		"marked for both install and removal — unmark one side of each:",
		"agent  alpha.md",
		"skill  delta",
		"esc back",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("conflict box does not contain %q:\n%s", want, content)
		}
	}
	for _, unwanted := range []string{"bravo.md", "stale.md"} {
		if strings.Contains(content, unwanted) {
			t.Errorf("conflict box lists the unconflicted %q:\n%s", unwanted, content)
		}
	}

	// Sorted by kind then name, so the list reads the same way twice.
	if strings.Index(content, "alpha.md") > strings.Index(content, "delta") {
		t.Errorf("conflict box is not sorted by kind then name:\n%s", content)
	}
}

// 35. the block is hard, with no override: the box offers no key that
// proceeds, because there is no defensible ordering to proceed with. Only
// esc, which puts the user back where they can unmark one of the two.
func TestConflictBoxHasNoOverrideKey(t *testing.T) {
	base := conflictedModel(t)
	base, _ = update(t, base, key("enter"))
	if base.state != stateConflict {
		t.Fatalf("setup: state = %v, want stateConflict", base.state)
	}

	for _, k := range []string{"enter", "space", "y", "j", "l"} {
		t.Run(k, func(t *testing.T) {
			m, cmd := update(t, base, key(k))
			if m.state != stateConflict {
				t.Errorf("%q left stateConflict for %v, want to stay", k, m.state)
			}
			if cmd != nil {
				t.Errorf("%q returned a command, want none", k)
			}
			if m.confirmed {
				t.Errorf("%q set confirmed, want false", k)
			}
		})
	}

	t.Run("esc", func(t *testing.T) {
		m, _ := update(t, base, key("esc"))
		if m.state != stateBrowse {
			t.Errorf("esc left state = %v, want stateBrowse", m.state)
		}
		if m.confirmed {
			t.Errorf("esc set confirmed, want false")
		}
	})
}

// 36. unmarking one side is the correct response, and it works: the row
// stops being marked, and the very next enter reaches the confirm box.
func TestUnmarkingResolvesConflict(t *testing.T) {
	m := conflictedModel(t)
	m, _ = update(t, m, key("enter"))
	m, _ = update(t, m, key("esc")) // back to browsing, on tab 2

	m, _ = update(t, m, key("space")) // unmark the removal side
	if got := m.conflicts(); len(got) != 0 {
		t.Fatalf("conflicts() = %v after unmarking one side, want none", got)
	}
	if strings.Contains(m.View().Content, "(conflict)") {
		t.Errorf("tab 2 still marks the row as conflicted:\n%s", m.View().Content)
	}

	m, _ = update(t, m, key("enter"))
	if m.state != stateConfirm {
		t.Errorf("state = %v, want stateConfirm once the conflict is gone", m.state)
	}
	if !m.selected[selKey{Kind: installer.KindAgent, Name: "alpha.md"}] {
		t.Errorf("unmarking the removal side also dropped the install mark")
	}
}

// 37. the conflict box stays inside the frame even when the terminal is too
// short to name every conflict, keeping its bottom border and the "esc
// back" hint that is the only way out of the state. Mirror of
// TestConfirmBoxFitsShortTerminal.
func TestConflictBoxFitsShortTerminal(t *testing.T) {
	res := manyAgentsResult(30)
	var tgt installer.TargetResult
	for _, it := range res.Agents {
		tgt.Agents = append(tgt.Agents, installer.TargetItem{Kind: it.Kind, Name: it.Name, InSource: true})
	}

	m := sized(New(res, tgt, noopInstall, noopRemove), 80, 12)
	for _, it := range res.Agents {
		k := selKey{Kind: it.Kind, Name: it.Name}
		m.selected[k] = true
		m.marked[k] = true
	}
	if got := len(m.conflicts()); got != len(res.Agents) {
		t.Fatalf("conflicts() has %d entries, want %d", got, len(res.Agents))
	}
	m.state = stateConflict

	content := m.View().Content
	if got, want := frameLines(m), m.frameHeight(); got != want {
		t.Errorf("conflict frame is %d lines, want %d", got, want)
	}
	for _, want := range []string{"… ", "esc back", "└"} {
		if !strings.Contains(content, want) {
			t.Errorf("conflict frame does not contain %q:\n%s", want, content)
		}
	}
}
