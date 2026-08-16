package tui

import (
	"fmt"
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
	confirmBox     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
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
	case stateInstalling:
		s = m.viewInstalling()
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

// viewBrowse renders the two-tab, grouped, scrollable, searchable list.
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

// tabBar renders "Agents (n)" / "Skills (n)" with the active tab bold, so
// a selection made on the tab the user isn't looking at stays visible as a
// count.
func (m Model) tabBar() string {
	agents := fmt.Sprintf("Agents (%d)", m.selectedCount(installer.KindAgent))
	skills := fmt.Sprintf("Skills (%d)", m.selectedCount(installer.KindSkill))
	if m.tab == 0 {
		agents = activeTabStyle.Render(agents)
	} else {
		skills = activeTabStyle.Render(skills)
	}
	return agents + "   " + skills
}

// helpLine lists the real keys. Its wording changes when nothing is
// selected, since enter has nothing to do in that case — that changed
// wording is the hint the key-handling table promises.
func (m Model) helpLine() string {
	if len(m.selected) == 0 {
		return "j/k move · h/l tabs · space select · enter (nothing selected yet) · / search · esc quit"
	}
	return "j/k move · h/l tabs · space select · enter install · / search · esc quit"
}

// listLines renders the tab's content below the tab bar and search/filter
// line: either an explanatory line (the tab has no items at all, or the
// current query matches none of them) or the scrolled window of rows plus
// its scroll indicators.
func (m Model) listLines() []string {
	underlying := m.res.Agents
	kindWord := "agents"
	if m.tab == 1 {
		underlying = m.res.Skills
		kindWord = "skills"
	}
	if len(underlying) == 0 {
		return []string{fmt.Sprintf("no %s found", kindWord)}
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
// drawn without a checkbox, since they cannot be chosen.
func (m Model) renderRow(r row) string {
	if r.heading != "" {
		return headingStyle.Render(r.heading + ":")
	}

	mark := "  "
	if r.index == m.cursor[m.tab] {
		mark = "> "
	}

	if r.item.Group() == installer.GroupUpToDate {
		return dimStyle.Render(mark + r.item.Name)
	}

	box := "[ ]"
	if m.selected[selKey{Kind: r.item.Kind, Name: r.item.Name}] {
		box = "[x]"
	}
	return mark + box + " " + r.item.Name
}

// viewConfirm renders the confirmation step: a bordered box listing the
// selected items grouped by kind, in place of the list — not composited
// or overlaid on top of it.
func (m Model) viewConfirm() string {
	names := appendNames([]string{"agents:"}, m.selectedNames(installer.KindAgent))
	names = appendNames(append(names, "skills:"), m.selectedNames(installer.KindSkill))
	names = clampNames(names, m.frameHeight()-confirmChrome)

	var b strings.Builder
	for _, line := range names {
		fmt.Fprintln(&b, line)
	}
	fmt.Fprintf(&b, "\n%d selected\n\nenter install · esc cancel", len(m.selected))

	return confirmBox.Render(b.String())
}

// appendNames appends one indented line per name, or "(none)" when names is
// empty.
func appendNames(lines, names []string) []string {
	if len(names) == 0 {
		return append(lines, "  (none)")
	}
	for _, n := range names {
		lines = append(lines, "  "+n)
	}
	return lines
}

// clampNames drops names off the end until the block fits in budget lines,
// replacing what it dropped with a count of them. frameHeight is sized so
// that the full list fits whenever the terminal is tall enough to show it,
// so this only bites on a terminal too short for the selection — where the
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

// viewInstalling renders the brief in-progress state between confirming
// and the install command finishing.
func (m Model) viewInstalling() string {
	return fmt.Sprintf("installing %d items…", len(m.selected))
}

// viewDone renders the single frame between installedMsg arriving and the
// program actually exiting. It is necessarily brief: the real report is
// printed by the caller, via RenderReport, only after Run returns —
// because Bubble Tea v2's inline renderer erases this program's last frame
// on quit, whatever this function draws would be gone before anyone could
// read it.
func (m Model) viewDone() string {
	return "done."
}
