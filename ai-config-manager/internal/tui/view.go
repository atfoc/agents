package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

var (
	activeTabStyle = lipgloss.NewStyle().Bold(true)
	headingStyle   = lipgloss.NewStyle().Bold(true)
	dimStyle       = lipgloss.NewStyle().Faint(true)
	helpStyle      = lipgloss.NewStyle().Faint(true)
	// No colour: the existing palette is Bold/Faint only, and "[!]" plus
	// the word "conflict" carries the meaning on a monochrome terminal
	// where a colour would not.
	conflictStyle = lipgloss.NewStyle().Bold(true)
	confirmBox    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
)

// View renders the current frame.
//
// AltScreen is deliberately left at its zero value (false) here, in every
// state. That is the whole mechanism that keeps this program inline,
// drawing directly below wherever the command was run, instead of taking
// over the terminal the way the alternate screen buffer would — there is
// no "AltScreen: false" line below because not setting it is the point.
func (m Model) View() tea.View {
	var s string
	switch m.state {
	case stateBrowse:
		s = m.viewBrowse()
	case stateConfirm:
		s = m.viewConfirm()
	case stateConflict:
		s = m.viewConflict()
	case stateApplying:
		s = m.viewApplying()
	case stateDone:
		s = m.viewDone()
	}
	return tea.View{Content: fitHeight(s, m.frameHeight())}
}

// fitHeight pads s with blank lines, or drops lines off its end, until it
// is exactly h lines tall. Every frame goes through it, which is what holds
// each state's frame to the one height frameHeight picked — see there for
// why a frame that shrinks breaks the inline renderer.
//
// The clipping half is only a backstop: visibleRows sizes the list to fit
// and viewConfirm bounds its own box, so nothing should reach here too
// tall. It matters anyway, because a frame taller than the terminal sends
// the renderer into a redraw loop that never settles.
func fitHeight(s string, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// viewBrowse renders the four-tab, grouped, scrollable, searchable list.
func (m Model) viewBrowse() string {
	lines := []string{m.tabBar(), ""}

	switch {
	case m.searchOpen:
		lines = append(lines, "search: "+m.search.View())
	case m.query[m.tab] != "":
		lines = append(lines, fmt.Sprintf("filter: %q (esc or / to clear)", m.query[m.tab]))
	}

	lines = append(lines, m.listLines()...)
	lines = append(lines, "", helpStyle.Render(m.helpLine()))

	return strings.Join(lines, "\n")
}

// tabBar renders all four tabs with the active one bold. An install tab's
// count is signed "+" and an uninstall tab's "-": one number means "will be
// written", the other "will be erased", and a sign carries that distinction
// without colour.
//
// The separator is two spaces rather than three, and the whole bar is
// capped at the terminal width. A tab bar that wraps onto a second line
// would add a line to the frame, which is precisely what frameHeight exists
// to prevent.
func (m Model) tabBar() string {
	var parts []string
	for i, d := range tabs {
		sign := "+"
		if d.action == actionRemove {
			sign = "-"
		}
		label := fmt.Sprintf("%s (%s%d)", d.label, sign, m.markCount(i))
		if i == m.tab {
			label = activeTabStyle.Render(label)
		}
		parts = append(parts, label)
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(strings.Join(parts, "  "))
}

// helpLine lists the real keys. Its wording changes when nothing is marked
// (enter has nothing to do) and when a conflict exists (enter will refuse).
func (m Model) helpLine() string {
	switch {
	case len(m.conflicts()) > 0:
		return "j/k move · h/l tabs · space mark · enter (conflicts) · / search · esc quit"
	case len(m.selected)+len(m.marked) == 0:
		return "j/k move · h/l tabs · space mark · enter (nothing marked yet) · / search · esc quit"
	default:
		return "j/k move · h/l tabs · space mark · enter apply · / search · esc quit"
	}
}

// listLines renders the tab's content below the tab bar and search/filter
// line: either an explanatory line (the tab has no items at all, or the
// current query matches none of them) or the scrolled window of rows plus
// its scroll indicators.
func (m Model) listLines() []string {
	// The kind word and the underlying list both come from the tab table,
	// so a missing or empty target directory simply shows an empty
	// uninstall tab rather than an error, and the install tabs keep
	// working normally.
	if len(m.tabEntries(m.tab)) == 0 {
		return []string{fmt.Sprintf("no %s found", tabs[m.tab].kind)}
	}

	allRows := m.rows(m.tab)
	if len(allRows) == 0 {
		return []string{fmt.Sprintf("no items match %q", m.query[m.tab])}
	}

	vr := m.visibleRows()
	off := m.offset[m.tab]
	end := off + vr
	if end > len(allRows) {
		end = len(allRows)
	}
	window := allRows[off:end]

	lines := make([]string, 0, len(window)+2)
	for _, r := range window {
		lines = append(lines, m.renderRow(r))
	}
	if off > 0 {
		lines = append(lines, fmt.Sprintf("↑ %d more", off))
	}
	if end < len(allRows) {
		lines = append(lines, fmt.Sprintf("↓ %d more", len(allRows)-end))
	}
	return lines
}

// renderRow draws one row: a styled heading, or a "cursor + checkbox +
// name" item row (e.g. "> [x] scout.md"). Up-to-date rows are dimmed and
// drawn without a checkbox, since they cannot be chosen. A conflicted row
// is marked in both tabs at once — inConflict does not care which tab is
// asking — so the user sees it while still picking rather than after
// marking twenty things.
func (m Model) renderRow(r row) string {
	if r.heading != "" {
		return headingStyle.Render(r.heading + ":")
	}

	mark := "  "
	if r.index == m.cursor[m.tab] {
		mark = "> "
	}

	if !r.entry.selectable {
		return dimStyle.Render(mark + r.entry.name)
	}

	key := selKey{Kind: r.entry.kind, Name: r.entry.name}
	box := "[ ]"
	switch {
	case m.inConflict(key):
		box = "[!]"
	case m.marks(tabs[m.tab].action)[key]:
		box = "[x]"
	}
	line := mark + box + " " + r.entry.name
	if m.inConflict(key) {
		line += conflictStyle.Render("  (conflict)")
	}
	return line
}

// viewConfirm renders the confirmation step: a bordered box listing the
// marked items in two sections, in place of the list — not composited or
// overlaid on top of it.
func (m Model) viewConfirm() string {
	lines := clampNames(m.markedLines(), m.frameHeight()-confirmChrome)

	var b strings.Builder
	for _, line := range lines {
		fmt.Fprintln(&b, line)
	}
	fmt.Fprintf(&b, "\n%d to remove, %d to install\n\nenter apply · esc cancel",
		len(m.marked), len(m.selected))

	return confirmBox.Render(b.String())
}

// viewConflict renders the hard block on marking the same item for install
// and for deletion. It is a state of its own, in the same bordered-box
// shape as viewConfirm, rather than a warning line under the list: the
// design requires that enter list every conflicting item by name, and an
// arbitrary number of names cannot go on one line without either wrapping —
// which breaks the fixed frame height — or truncating, which drops the very
// names it promised to list.
func (m Model) viewConflict() string {
	lines := []string{headingStyle.Render("marked for both install and removal — unmark one side of each:")}
	for _, k := range m.conflicts() {
		lines = append(lines, fmt.Sprintf("  %-6s %s", kindNoun(k.Kind), k.Name))
	}
	lines = clampNames(lines, m.frameHeight()-conflictChrome)

	var b strings.Builder
	for _, line := range lines {
		fmt.Fprintln(&b, line)
	}
	fmt.Fprint(&b, "\nesc back")

	return confirmBox.Render(b.String())
}

// markedLines returns the confirmation box's item lines: a "remove" section
// then an "install" section, each name prefixed with the same sign the tab
// bar uses and labelled with its kind. A section with nothing in it is
// omitted entirely rather than printed as "(none)" — an empty destructive
// section is noise on a screen whose whole job is to make the destructive
// part impossible to miss.
//
// The destructive marking is carried by three things at once — the section
// coming first, the "-" sign, and the words "deletes these from the target"
// — rather than by colour, matching the rest of the UI's Bold/Faint-only
// palette.
func (m Model) markedLines() []string {
	var lines []string

	if removals := sortedKeys(m.marked); len(removals) > 0 {
		lines = append(lines, headingStyle.Render("remove — deletes these from the target:"))
		for _, k := range removals {
			lines = append(lines, fmt.Sprintf("  - %-6s %s", kindNoun(k.Kind), k.Name))
		}
	}
	if installs := sortedKeys(m.selected); len(installs) > 0 {
		lines = append(lines, headingStyle.Render("install:"))
		for _, k := range installs {
			lines = append(lines, fmt.Sprintf("  + %-6s %s", kindNoun(k.Kind), k.Name))
		}
	}
	return lines
}

// sortedKeys returns set's keys ordered by Kind then Name, so the confirm
// box lists them deterministically.
func sortedKeys(set map[selKey]bool) []selKey {
	keys := make([]selKey, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Kind != keys[j].Kind {
			return keys[i].Kind < keys[j].Kind
		}
		return keys[i].Name < keys[j].Name
	})
	return keys
}

// kindNoun names one item of a Kind ("agent", "skill"), unlike
// Kind.String(), which names the directory the kind lives under.
func kindNoun(k installer.Kind) string {
	if k == installer.KindSkill {
		return "skill"
	}
	return "agent"
}

// clampNames drops names off the end until the block fits in budget lines,
// replacing what it dropped with a count of them. frameHeight is sized so
// that the full list fits whenever the terminal is tall enough to show it,
// so this only bites on a terminal too short for what is marked — where the
// alternative is a box whose bottom border and, worse, whose "esc cancel"
// hint are off the frame.
func clampNames(names []string, budget int) []string {
	if len(names) <= budget {
		return names
	}
	if budget < 1 {
		return nil
	}
	hidden := len(names) - (budget - 1)
	return append(names[:budget-1], fmt.Sprintf("  … %d more", hidden))
}

// viewApplying renders the brief in-progress state between confirming and
// the apply command finishing.
func (m Model) viewApplying() string {
	return fmt.Sprintf("applying %d changes…", len(m.selected)+len(m.marked))
}

// viewDone renders the single frame between appliedMsg arriving and the
// program actually exiting. It is necessarily brief: the real report is
// printed by the caller, via RenderReport, only after Run returns —
// because Bubble Tea v2's inline renderer erases this program's last frame
// on quit, whatever this function draws would be gone before anyone could
// read it.
func (m Model) viewDone() string {
	return "done."
}
