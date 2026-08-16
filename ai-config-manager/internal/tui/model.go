package tui

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

// state is which screen of the flow the model is currently showing.
type state int

const (
	stateBrowse state = iota
	stateConfirm
	stateInstalling
	stateDone
)

// selKey identifies a selected item across tabs. Keying selection by kind
// and name — rather than by position — is what makes a selection survive
// switching tabs and searching: the row may move or vanish from view, the
// key does not.
type selKey struct {
	Kind installer.Kind
	Name string
}

// row is one line of the scrolled list. A heading row carries only
// Heading (Index is -1); an item row carries only Item and its position in
// that tab's visibleItems. Keeping headings as rows, rather than drawing
// them separately from the scrolled content, is what lets the same offset
// arithmetic scroll headings and items together.
type row struct {
	heading string
	item    installer.Item
	index   int
}

// Model is the interactive picker's state.
type Model struct {
	res     installer.Result
	install InstallFunc

	state state
	tab   int // 0 = Agents, 1 = Skills

	cursor [2]int    // per-tab cursor, remembered across tab switches
	offset [2]int    // per-tab scroll offset, in rows
	query  [2]string // per-tab committed search text

	search     textinput.Model
	searchOpen bool // the input is focused and taking keystrokes

	selected  map[selKey]bool
	outcomes  []ItemOutcome
	confirmed bool

	width  int
	height int
}

// New builds the interactive model for res. install is called once per
// selected item when the user confirms; production callers pass
// installer.ApplyItem, tests pass a stand-in that never touches disk.
func New(res installer.Result, install InstallFunc) Model {
	search := textinput.New()
	search.Placeholder = "search"

	return Model{
		res:      res,
		install:  install,
		state:    stateBrowse,
		search:   search,
		selected: make(map[selKey]bool),
		// 80x24 is the traditional terminal default. Setting it here, not
		// leaving it zero, is what makes the model render something sane
		// on the one frame that can be drawn before tea.WindowSizeMsg
		// first arrives.
		width:  80,
		height: 24,
	}
}

// Outcome reports what the interactive session did, for the caller to
// render after the program has quit.
func (m Model) Outcome() Outcome {
	return Outcome{Confirmed: m.confirmed, Applied: m.outcomes}
}

// Init requests no initial command: New already sets defaults good enough
// to render before the first tea.WindowSizeMsg shows up.
func (m Model) Init() tea.Cmd {
	return nil
}

// visibleItems returns the items of the given tab, narrowed by that tab's
// query, ordered by display group.
func (m Model) visibleItems(tab int) []installer.Item {
	src := m.res.Agents
	if tab == 1 {
		src = m.res.Skills
	}

	// This is a browse aid, not the command line's --filter: --filter is
	// an exact name match, whereas this is a loose, case-insensitive
	// substring match so the user can narrow the list while typing without
	// knowing an item's exact name up front. The two are deliberately not
	// the same kind of thing, so don't confuse them.
	q := strings.ToLower(m.query[tab])
	items := make([]installer.Item, 0, len(src))
	for _, it := range src {
		if q == "" || strings.Contains(strings.ToLower(it.Name), q) {
			items = append(items, it)
		}
	}

	// Sorting on this copy — never on m.res's own slices — by Group() and
	// then Name is what makes selectableCount a plain prefix count:
	// GroupToUpdate (0) and GroupToInstall (1) always sort ahead of
	// GroupUpToDate (2), so every selectable item ends up in a contiguous
	// prefix and every up-to-date item ends up at the tail.
	sort.Slice(items, func(i, j int) bool {
		gi, gj := items[i].Group(), items[j].Group()
		if gi != gj {
			return gi < gj
		}
		return items[i].Name < items[j].Name
	})
	return items
}

// selectableCount returns how many of the tab's visible items can be
// chosen. Up-to-date items are shown but never selectable: installing a
// byte-identical item is a no-op, so offering a checkbox would imply an
// effect that does not exist.
func (m Model) selectableCount(tab int) int {
	n := 0
	for _, it := range m.visibleItems(tab) {
		if it.Group() != installer.GroupUpToDate {
			n++
		}
	}
	return n
}

// rows walks visibleItems(tab) in order, emitting a heading row each time
// the group changes, then a row per item.
func (m Model) rows(tab int) []row {
	items := m.visibleItems(tab)
	rows := make([]row, 0, len(items)+3)

	var cur installer.Group
	seenGroup := false
	for i, it := range items {
		g := it.Group()
		if !seenGroup || g != cur {
			rows = append(rows, row{heading: g.String(), index: -1})
			cur = g
			seenGroup = true
		}
		rows = append(rows, row{item: it, index: i})
	}
	return rows
}

// browseChrome is how many lines viewBrowse spends on anything that is not
// a list row: the tab bar, the blank line under it, the search/filter line,
// the blank line above the help line, and the help line. The search/filter
// line is counted even for the frames that don't draw one, so that opening
// the search doesn't change how tall the frame wants to be.
const browseChrome = 5

// confirmChrome is how many lines viewConfirm spends on anything that is
// not a selected name: the box's two border lines, the blank line under the
// names, the count line, the blank line under that, and the key hints.
const confirmChrome = 6

// tabRowCount is how many rows a tab's list has with nothing filtered out:
// one per item, plus one heading per group present, or the single
// "no agents found" line for an empty tab. It deliberately counts the tab's
// whole underlying list rather than visibleItems, because it feeds
// frameHeight, which must not move when the user types a search.
func (m Model) tabRowCount(tab int) int {
	src := m.res.Agents
	if tab == 1 {
		src = m.res.Skills
	}
	if len(src) == 0 {
		return 1
	}

	var seen [3]bool
	groups := 0
	for _, it := range src {
		if g := it.Group(); int(g) < len(seen) && !seen[g] {
			seen[g] = true
			groups++
		}
	}
	return len(src) + groups
}

// frameHeight is how many lines every frame occupies — the same number in
// every state, which is the whole point of it.
//
// Bubble Tea v2.0.8's inline renderer gets a *shrinking* frame wrong: when
// the new frame is shorter than the last one it clamps its remembered
// cursor row to the new, smaller height before working out how far back up
// to move (ultraviolet's TerminalRenderer.move), so it moves up too few
// lines, redraws part-way down the previous frame, and leaves that frame's
// top lines on screen. That is what makes the display march down the
// terminal a few lines at a time instead of repainting in place — switching
// from the taller tab to the shorter one, or leaving the confirm box, is
// enough to trigger it. Growing a frame is handled correctly; only
// shrinking is not. Pinning every frame to one height means no frame ever
// shrinks.
//
// The height is the tallest frame any state could want — the taller tab's
// list, or the confirm box with everything selected — so that padding is
// the exception rather than the rule, and it never depends on what the user
// has selected or searched for, which would make it move again. It is
// capped one line short of the terminal: a frame that fills the terminal
// exactly leaves no room for whatever the shell printed above it, so every
// repaint scrolls the screen by a line.
func (m Model) frameHeight() int {
	natural := browseChrome + max(m.tabRowCount(0), m.tabRowCount(1))

	// Worst case for the confirm box: every item selected, so neither kind
	// falls back to its "(none)" line. The max(…, 1) covers a kind with no
	// items at all, which does draw one.
	confirm := confirmChrome + 2 + max(len(m.res.Agents), 1) + max(len(m.res.Skills), 1)
	natural = max(natural, confirm)

	if maxH := m.height - 1; natural > maxH {
		natural = maxH
	}
	if natural < 1 {
		natural = 1
	}
	return natural
}

// visibleRows is how many rows of the list fit inside frameHeight below the
// chrome: the tab bar line, the blank line under it, the search/query line
// (only when shown), the blank line before the help line, the help line
// itself, and a flat reservation of two lines for the scroll indicators.
//
// The indicators are reserved unconditionally, even on frames where the
// list isn't clipped and neither one is actually drawn, because whether
// they're needed depends on the row count, which depends on this very
// number — reserving their worst case sidesteps that circularity.
func (m Model) visibleRows() int {
	chrome := 2 // tab bar line + blank line
	if m.searchOpen || m.query[m.tab] != "" {
		chrome++ // search/query line
	}
	chrome += 2 // blank line before help + help line
	chrome += 2 // up to two scroll-indicator lines, reserved flat

	vr := m.frameHeight() - chrome
	if vr < 3 {
		vr = 3
	}
	return vr
}

// syncScroll keeps the cursor's row inside the visible window
// [offset, offset+visibleRows), adjusting the offset in whichever
// direction the cursor moved past it, then clamps the offset itself to
// [0, max(0, len(rows)-visibleRows)]. It must be called after any change
// to the cursor, the active tab, or a query, since all three can move
// which row the cursor is on or how many rows there are to scroll through.
func (m *Model) syncScroll() {
	vr := m.visibleRows()
	rows := m.rows(m.tab)

	cursorRow := 0
	for i, r := range rows {
		if r.index == m.cursor[m.tab] {
			cursorRow = i
			break
		}
	}

	off := m.offset[m.tab]
	if cursorRow < off {
		off = cursorRow
	}
	if cursorRow >= off+vr {
		off = cursorRow - vr + 1
	}

	maxOff := len(rows) - vr
	if maxOff < 0 {
		maxOff = 0
	}
	if off > maxOff {
		off = maxOff
	}
	if off < 0 {
		off = 0
	}
	m.offset[m.tab] = off
}

// clampCursor keeps the cursor inside [0, selectableCount(tab)), or at 0
// when that count is 0. Needed after anything that can shrink or empty the
// tab's selectable items out from under the cursor, namely typing a
// narrower search query.
func (m *Model) clampCursor() {
	n := m.selectableCount(m.tab)
	switch {
	case n == 0:
		m.cursor[m.tab] = 0
	case m.cursor[m.tab] >= n:
		m.cursor[m.tab] = n - 1
	case m.cursor[m.tab] < 0:
		m.cursor[m.tab] = 0
	}
}

// cursorDown moves the cursor one selectable item down, without wrapping.
func (m *Model) cursorDown() {
	if n := m.selectableCount(m.tab); n > 0 && m.cursor[m.tab] < n-1 {
		m.cursor[m.tab]++
	}
	m.syncScroll()
}

// cursorUp moves the cursor one selectable item up, without wrapping.
func (m *Model) cursorUp() {
	if m.cursor[m.tab] > 0 {
		m.cursor[m.tab]--
	}
	m.syncScroll()
}

// toggleSelected flips the selection state of the item under the cursor.
// It is a no-op when the tab has no selectable items — clampCursor keeps
// the cursor at 0 in that case, but that 0 may still point at an
// up-to-date item (e.g. a tab whose only item is up to date), which must
// never become selectable just because the cursor sits on it.
func (m *Model) toggleSelected() {
	if m.selectableCount(m.tab) == 0 {
		return
	}
	items := m.visibleItems(m.tab)
	idx := m.cursor[m.tab]
	if idx < 0 || idx >= len(items) {
		return
	}
	it := items[idx]
	key := selKey{Kind: it.Kind, Name: it.Name}
	if m.selected[key] {
		delete(m.selected, key)
	} else {
		m.selected[key] = true
	}
}

// selectedNames returns the sorted names of selected items of kind, for
// deterministic display in the confirm step.
func (m Model) selectedNames(kind installer.Kind) []string {
	var names []string
	for k := range m.selected {
		if k.Kind == kind {
			names = append(names, k.Name)
		}
	}
	sort.Strings(names)
	return names
}

// selectedCount returns how many items of kind are currently selected,
// across both tabs — used by the tab bar so a selection made on the other
// tab stays visible as a count.
func (m Model) selectedCount(kind installer.Kind) int {
	n := 0
	for k := range m.selected {
		if k.Kind == kind {
			n++
		}
	}
	return n
}

// installedMsg carries the result of installing every selected item. It is
// what installCmd's command sends back once all of them are done.
type installedMsg struct {
	outcomes []ItemOutcome
}

// installCmd returns a command that installs every selected item and
// records an outcome for each of them. It runs as a tea.Cmd — off the
// Update call stack, on Bubble Tea's own goroutine — so a slow or blocking
// InstallFunc cannot freeze the UI.
func (m Model) installCmd() tea.Cmd {
	selected := make(map[selKey]bool, len(m.selected))
	for k := range m.selected {
		selected[k] = true
	}
	agents := m.res.Agents
	skills := m.res.Skills
	install := m.install

	return func() tea.Msg {
		var outcomes []ItemOutcome
		// Agents then skills, each in the Result's own order — not
		// visibleItems' order, which a search query or the display
		// grouping can reshuffle. And unlike installer.Apply, a failure
		// here does not stop the loop: the interactive report has to
		// account for every item the user chose, not just the ones before
		// the first failure.
		for _, it := range agents {
			if !selected[selKey{Kind: it.Kind, Name: it.Name}] {
				continue
			}
			outcomes = append(outcomes, ItemOutcome{Item: it, Err: install(it)})
		}
		for _, it := range skills {
			if !selected[selKey{Kind: it.Kind, Name: it.Name}] {
				continue
			}
			outcomes = append(outcomes, ItemOutcome{Item: it, Err: install(it)})
		}
		return installedMsg{outcomes: outcomes}
	}
}

// Update handles one message. It is the only place model state changes.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncScroll()
		return m, nil

	case installedMsg:
		m.outcomes = msg.outcomes
		m.state = stateDone
		return m, tea.Quit

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey dispatches a key press. ctrl+c is handled here, ahead of
// everything else, because it must quit from any state unconditionally —
// including mid-search and mid-install — without disturbing confirmed:
// confirmed is already true if an install has started, and stays false
// otherwise.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	if m.searchOpen {
		return m.handleSearchKey(msg, key)
	}

	switch m.state {
	case stateBrowse:
		return m.handleBrowseKey(key)
	case stateConfirm:
		return m.handleConfirmKey(key)
	default:
		// stateInstalling and stateDone: every key but ctrl+c (handled
		// above) is ignored — there is nothing left for the user to do
		// but wait for the install to finish.
		return m, nil
	}
}

// handleSearchKey is reached only while the search input is focused. It
// intercepts exactly esc, enter, up, and down (ctrl+c is intercepted
// earlier, in handleKey, ahead of this) and forwards everything else — j,
// k, h, l, space, left, right, backspace, delete, home, end, ctrl+w, and so
// on — to the textinput untouched. Those are its own default keymap (see
// textinput.DefaultKeyMap); reimplementing them here would mean getting
// word-motions and rune handling exactly right a second time just to match
// what the widget already does for free.
func (m Model) handleSearchKey(msg tea.KeyPressMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.query[m.tab] = ""
		m.search.SetValue("")
		m.search.Blur()
		m.searchOpen = false
		m.clampCursor()
		m.syncScroll()
		return m, nil

	case "enter":
		// Deliberately does not start an install: a stray enter while
		// typing a search must never install anything.
		m.query[m.tab] = m.search.Value()
		m.search.Blur()
		m.searchOpen = false
		m.clampCursor()
		m.syncScroll()
		return m, nil

	case "up":
		m.cursorUp()
		return m, nil

	case "down":
		m.cursorDown()
		return m, nil

	default:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.query[m.tab] = m.search.Value()
		m.clampCursor()
		m.syncScroll()
		return m, cmd
	}
}

// handleBrowseKey is reached in stateBrowse when the search input is not
// focused.
func (m Model) handleBrowseKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.cursorDown()

	case "k", "up":
		m.cursorUp()

	case "h", "left", "l", "right":
		// With exactly two tabs, moving to the "previous" one and the
		// "next" one are the same operation: toggle. Both directions wrap
		// for the same reason — there's nowhere else to go.
		m.tab = 1 - m.tab
		m.clampCursor()
		m.syncScroll()

	case " ", "space":
		m.toggleSelected()

	case "enter":
		if len(m.selected) > 0 {
			m.state = stateConfirm
		}
		// Nothing selected: stay in stateBrowse. The help line's own text
		// changes to say so; see viewBrowse.

	case "/":
		m.searchOpen = true
		m.search.SetValue(m.query[m.tab])
		m.syncScroll()
		return m, m.search.Focus()

	case "esc":
		m.confirmed = false
		return m, tea.Quit
	}
	return m, nil
}

// handleConfirmKey is reached in stateConfirm.
func (m Model) handleConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		m.state = stateInstalling
		m.confirmed = true
		return m, m.installCmd()

	case "esc":
		m.state = stateBrowse
	}
	return m, nil
}
