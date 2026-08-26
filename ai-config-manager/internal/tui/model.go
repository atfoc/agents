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
	stateConflict
	stateApplying
	stateDone
)

// tabCount is the number of tabs: Agents, Skills, Uninstall Agents,
// Uninstall Skills.
const tabCount = 4

// action says what marking a row in a tab means.
type action int

const (
	actionInstall action = iota
	actionRemove
)

// tabDef is the static description of one tab. Keeping the four in a table,
// rather than in branches on the tab index, is what lets tab-agnostic code
// ask "what does this tab list?" instead of enumerating cases.
type tabDef struct {
	label  string
	kind   installer.Kind
	action action
}

var tabs = [tabCount]tabDef{
	{label: "Agents", kind: installer.KindAgent, action: actionInstall},
	{label: "Skills", kind: installer.KindSkill, action: actionInstall},
	{label: "Uninstall Agents", kind: installer.KindAgent, action: actionRemove},
	{label: "Uninstall Skills", kind: installer.KindSkill, action: actionRemove},
}

// entry is one row's worth of list data, flattened from either an
// installer.Item or an installer.TargetItem. The two are different types
// with different groupings, but every mechanic below this point — the
// cursor, scrolling, the search filter, the checkbox — needs only a name, a
// heading to sit under, and whether it can be chosen. Flattening here is
// what keeps that machinery written once instead of twice.
type entry struct {
	kind installer.Kind
	name string
	// group is the heading text this entry sits under ("to update",
	// "only in target", …) and groupRank is that heading's display order
	// within the tab.
	group     string
	groupRank int
	// selectable is false only for an install tab's up-to-date items:
	// installing a byte-identical item is a no-op, so offering a checkbox
	// would imply an effect that does not exist. Every item on an uninstall
	// tab is selectable — it is there, so it can go.
	selectable bool
}

// selKey identifies a marked item across tabs. Keying a mark by kind and
// name — rather than by position — is what makes it survive switching tabs
// and searching: the row may move or vanish from view, the key does not.
type selKey struct {
	Kind installer.Kind
	Name string
}

// row is one line of the scrolled list. A heading row carries only
// heading (index is -1); an item row carries only entry and its position in
// that tab's visibleEntries. Keeping headings as rows, rather than drawing
// them separately from the scrolled content, is what lets the same offset
// arithmetic scroll headings and items together.
type row struct {
	heading string
	entry   entry
	index   int
}

// Model is the interactive picker's state.
type Model struct {
	res     installer.Result       // source side
	tgt     installer.TargetResult // target side
	install InstallFunc
	remove  RemoveFunc

	state state
	tab   int // index into tabs

	cursor [tabCount]int    // per-tab cursor, remembered across tab switches
	offset [tabCount]int    // per-tab scroll offset, in rows
	query  [tabCount]string // per-tab committed search text

	search     textinput.Model
	searchOpen bool // the input is focused and taking keystrokes

	// Two maps, not one keyed by {action, kind, name}. A conflict is
	// precisely "the same selKey is in both", so detection is a set
	// intersection; one map would make it a scan, and would turn each tab's
	// count into a filtered walk.
	selected map[selKey]bool // marked for install, on tabs 0 and 1
	marked   map[selKey]bool // marked for removal, on tabs 2 and 3

	outcomes  []ItemOutcome
	confirmed bool

	width  int
	height int
}

// New builds the interactive model. install and remove are called once per
// marked item when the user confirms; production callers pass
// installer.ApplyItem and installer.RemoveItem, tests pass stand-ins that
// never touch disk.
func New(res installer.Result, tgt installer.TargetResult, install InstallFunc, remove RemoveFunc) Model {
	search := textinput.New()
	search.Placeholder = "search"

	return Model{
		res:      res,
		tgt:      tgt,
		install:  install,
		remove:   remove,
		state:    stateBrowse,
		search:   search,
		selected: make(map[selKey]bool),
		marked:   make(map[selKey]bool),
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

// tabEntries returns the tab's full underlying list as entries, unfiltered
// and unsorted. It is the only place in the package that branches on item
// type.
func (m Model) tabEntries(tab int) []entry {
	d := tabs[tab]
	var out []entry
	switch d.action {
	case actionInstall:
		src := m.res.Agents
		if d.kind == installer.KindSkill {
			src = m.res.Skills
		}
		for _, it := range src {
			g := it.Group()
			out = append(out, entry{
				kind:       it.Kind,
				name:       it.Name,
				group:      g.String(),
				groupRank:  int(g),
				selectable: g != installer.GroupUpToDate,
			})
		}
	case actionRemove:
		src := m.tgt.Agents
		if d.kind == installer.KindSkill {
			src = m.tgt.Skills
		}
		for _, it := range src {
			g := it.Group()
			out = append(out, entry{
				kind:       it.Kind,
				name:       it.Name,
				group:      g.String(),
				groupRank:  int(g),
				selectable: true,
			})
		}
	}
	return out
}

// visibleEntries returns the entries of the given tab, narrowed by that
// tab's query, ordered by display group.
func (m Model) visibleEntries(tab int) []entry {
	src := m.tabEntries(tab)

	// This is a browse aid, not the command line's --install: --install is
	// an exact name match, whereas this is a loose, case-insensitive
	// substring match so the user can narrow the list while typing without
	// knowing an item's exact name up front. The two are deliberately not
	// the same kind of thing, so don't confuse them.
	q := strings.ToLower(m.query[tab])
	entries := make([]entry, 0, len(src))
	for _, e := range src {
		if q == "" || strings.Contains(strings.ToLower(e.name), q) {
			entries = append(entries, e)
		}
	}

	// Sorting on this copy — never on the underlying slices — by groupRank
	// and then name is what makes selectableCount a plain prefix count:
	// GroupToUpdate (0) and GroupToInstall (1) always sort ahead of
	// GroupUpToDate (2), so every selectable item ends up in a contiguous
	// prefix and every up-to-date item ends up at the tail. On an uninstall
	// tab nothing is unselectable, so that prefix is the whole list.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].groupRank != entries[j].groupRank {
			return entries[i].groupRank < entries[j].groupRank
		}
		return entries[i].name < entries[j].name
	})
	return entries
}

// selectableCount returns how many of the tab's visible entries can be
// chosen. Up-to-date items are shown but never selectable: installing a
// byte-identical item is a no-op, so offering a checkbox would imply an
// effect that does not exist.
func (m Model) selectableCount(tab int) int {
	n := 0
	for _, e := range m.visibleEntries(tab) {
		if e.selectable {
			n++
		}
	}
	return n
}

// rows walks visibleEntries(tab) in order, emitting a heading row each time
// the group changes, then a row per entry.
func (m Model) rows(tab int) []row {
	entries := m.visibleEntries(tab)
	rows := make([]row, 0, len(entries)+3)

	cur := ""
	seenGroup := false
	for i, e := range entries {
		if !seenGroup || e.group != cur {
			rows = append(rows, row{heading: e.group, index: -1})
			cur = e.group
			seenGroup = true
		}
		rows = append(rows, row{entry: e, index: i})
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
// not a marked name: the box's two border lines, the blank line under the
// names, the count line, the blank line under that, and the key hints.
const confirmChrome = 6

// conflictChrome is how many lines viewConflict spends on anything that is
// not a conflicting name: the box's two border lines, the header line, the
// blank line under the names, and the key hint.
const conflictChrome = 5

// tabRowCount is how many rows a tab's list has with nothing filtered out:
// one per entry, plus one heading per group present, or the single
// "no agents found" line for an empty tab. It deliberately counts the tab's
// whole underlying list rather than visibleEntries, because it feeds
// frameHeight, which must not move when the user types a search.
func (m Model) tabRowCount(tab int) int {
	entries := m.tabEntries(tab)
	if len(entries) == 0 {
		return 1
	}

	// A map rather than a fixed-size array: install tabs have three groups
	// and uninstall tabs two.
	seen := make(map[int]bool, 3)
	groups := 0
	for _, e := range entries {
		if !seen[e.groupRank] {
			seen[e.groupRank] = true
			groups++
		}
	}
	return len(entries) + groups
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
// The height is the tallest frame any state could want — the tallest tab's
// list, or the confirm box with everything marked — so that padding is
// the exception rather than the rule, and it never depends on what the user
// has marked or searched for, which would make it move again. It is
// capped one line short of the terminal: a frame that fills the terminal
// exactly leaves no room for whatever the shell printed above it, so every
// repaint scrolls the screen by a line.
func (m Model) frameHeight() int {
	// Four tabs now, not two.
	rows := 0
	for tab := 0; tab < tabCount; tab++ {
		rows = max(rows, m.tabRowCount(tab))
	}
	natural := browseChrome + rows

	// Worst case for the confirm box: everything marked on both sides, so
	// neither section is omitted. The +2 is the two section headers.
	confirm := confirmChrome + 2 +
		len(m.res.Agents) + len(m.res.Skills) +
		len(m.tgt.Agents) + len(m.tgt.Skills)
	natural = max(natural, confirm)

	// Worst case for the conflict box: every item appearing on both sides is
	// conflicted, which cannot exceed the smaller side.
	conflict := conflictChrome + min(
		len(m.res.Agents)+len(m.res.Skills),
		len(m.tgt.Agents)+len(m.tgt.Skills))
	natural = max(natural, conflict)

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
// tab's selectable entries out from under the cursor, namely typing a
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

// cursorDown moves the cursor one selectable entry down, without wrapping.
func (m *Model) cursorDown() {
	if n := m.selectableCount(m.tab); n > 0 && m.cursor[m.tab] < n-1 {
		m.cursor[m.tab]++
	}
	m.syncScroll()
}

// cursorUp moves the cursor one selectable entry up, without wrapping.
func (m *Model) cursorUp() {
	if m.cursor[m.tab] > 0 {
		m.cursor[m.tab]--
	}
	m.syncScroll()
}

// marks returns the map the given action writes into. Maps are reference
// types, so the caller mutates the model's own map through it.
func (m Model) marks(a action) map[selKey]bool {
	if a == actionRemove {
		return m.marked
	}
	return m.selected
}

// markCount returns how many of tab's kind are marked in tab's own map,
// used by the tab bar so a mark made on a tab the user isn't looking at
// stays visible as a count.
func (m Model) markCount(tab int) int {
	d := tabs[tab]
	n := 0
	for k := range m.marks(d.action) {
		if k.Kind == d.kind {
			n++
		}
	}
	return n
}

// inConflict reports whether key is marked for install and for removal at
// the same time. A conflict is same kind + same name: an agent named
// "research" and a skill named "research" are unrelated items that merely
// share a string, and never conflict. Both maps key on selKey and both
// sides use the same naming convention ("scout.md", "research"), which is
// why installer.TargetItem.Name keeps the ".md" suffix.
func (m Model) inConflict(key selKey) bool {
	return m.selected[key] && m.marked[key]
}

// conflicts returns every conflicted key, sorted by Kind then Name so the
// list reads the same way twice.
func (m Model) conflicts() []selKey {
	var out []selKey
	for k := range m.selected {
		if m.marked[k] {
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// toggleMark flips the mark on the entry under the cursor, in whichever map
// the current tab writes to. It is a no-op when the tab has no selectable
// entries — clampCursor keeps the cursor at 0 in that case, but that 0 may
// still point at an up-to-date item, which must never become selectable
// just because the cursor sits on it.
//
// It deliberately does not refuse to create a conflict: refusing here would
// mean a keypress silently doing nothing. The conflict instead becomes
// visible the instant it is made — see renderRow — and blocks only at
// enter.
func (m *Model) toggleMark() {
	if m.selectableCount(m.tab) == 0 {
		return
	}
	entries := m.visibleEntries(m.tab)
	idx := m.cursor[m.tab]
	if idx < 0 || idx >= len(entries) {
		return
	}
	e := entries[idx]
	key := selKey{Kind: e.kind, Name: e.name}
	set := m.marks(tabs[m.tab].action)
	if set[key] {
		delete(set, key)
	} else {
		set[key] = true
	}
}

// appliedMsg carries the result of applying every marked change. It is what
// applyCmd's command sends back once all of them are done.
type appliedMsg struct {
	outcomes []ItemOutcome
}

// applyCmd returns a command that performs every marked change: removals
// first, then installs. Conflicts are impossible by this point, so the order
// cannot change the outcome — but "clean out, then put in" is the order the
// confirmation box reads in, and matching it is what keeps the report
// legible against what the user just approved.
//
// It runs as a tea.Cmd — off the Update call stack, on Bubble Tea's own
// goroutine — so a slow InstallFunc or RemoveFunc cannot freeze the UI. And
// unlike installer.Apply/Remove, a failure here does not stop the loop: the
// interactive report has to account for every item the user marked, not
// just the ones before the first failure.
func (m Model) applyCmd() tea.Cmd {
	selected := make(map[selKey]bool, len(m.selected))
	for k := range m.selected {
		selected[k] = true
	}
	marked := make(map[selKey]bool, len(m.marked))
	for k := range m.marked {
		marked[k] = true
	}
	res, tgt := m.res, m.tgt
	install, remove := m.install, m.remove

	return func() tea.Msg {
		var outcomes []ItemOutcome

		// Removals: target agents then target skills, each in
		// TargetResult's own order — not visibleEntries' order, which a
		// search query or the display grouping can reshuffle.
		for _, group := range [][]installer.TargetItem{tgt.Agents, tgt.Skills} {
			for _, it := range group {
				if !marked[selKey{Kind: it.Kind, Name: it.Name}] {
					continue
				}
				outcomes = append(outcomes, ItemOutcome{
					Action: actionRemove, Kind: it.Kind, Name: it.Name, Err: remove(it),
				})
			}
		}

		// Installs: agents then skills, each in Result's own order.
		for _, group := range [][]installer.Item{res.Agents, res.Skills} {
			for _, it := range group {
				if !selected[selKey{Kind: it.Kind, Name: it.Name}] {
					continue
				}
				outcomes = append(outcomes, ItemOutcome{
					Action: actionInstall, Kind: it.Kind, Name: it.Name,
					Exists: it.Exists, Err: install(it),
				})
			}
		}

		return appliedMsg{outcomes: outcomes}
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

	case appliedMsg:
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
// including mid-search and mid-apply — without disturbing confirmed:
// confirmed is already true if an apply has started, and stays false
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
	case stateConflict:
		return m.handleConflictKey(key)
	default:
		// stateApplying and stateDone: every key but ctrl+c (handled
		// above) is ignored — there is nothing left for the user to do
		// but wait for the apply to finish.
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
		// Deliberately does not start an apply: a stray enter while
		// typing a search must never install or remove anything.
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

	case "h", "left":
		// Four tabs, so h and l are genuinely different directions now,
		// rather than the same toggle. Both still wrap.
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.clampCursor()
		m.syncScroll()

	case "l", "right":
		m.tab = (m.tab + 1) % tabCount
		m.clampCursor()
		m.syncScroll()

	case " ", "space":
		m.toggleMark()

	case "enter":
		if len(m.conflicts()) > 0 {
			m.state = stateConflict
			return m, nil
		}
		if len(m.selected)+len(m.marked) > 0 {
			m.state = stateConfirm
		}
		// Nothing marked: stay in stateBrowse. The help line's own text
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

// handleConflictKey is reached in stateConflict. It accepts only esc.
// There is no key that proceeds: the rule is hard-blocking with no
// override, so the screen offers no override to press. There is no
// defensible ordering — installing then deleting wastes the write, deleting
// then installing makes the delete meaningless — so the only correct
// response is to unmark one of the two.
func (m Model) handleConflictKey(key string) (tea.Model, tea.Cmd) {
	if key == "esc" {
		m.state = stateBrowse
	}
	return m, nil
}

// handleConfirmKey is reached in stateConfirm.
func (m Model) handleConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		m.state = stateApplying
		m.confirmed = true
		return m, m.applyCmd()

	case "esc":
		m.state = stateBrowse
	}
	return m, nil
}
