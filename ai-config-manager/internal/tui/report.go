package tui

import (
	"fmt"
	"io"
	"sort"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

// RenderReport writes the summary of an interactive install to w.
//
// It exists as its own function, called by the caller rather than by the
// Bubble Tea program itself, because the interactive program renders
// inline (no alternate screen buffer) and Bubble Tea v2 erases its last
// frame on exit. That means the program's own final frame can never be
// the record of what happened — by the time the user could read it, it
// would already be gone. So the model only collects an Outcome, and the
// caller prints this report to stdout after the program has quit.
//
// The format mirrors installer.Render (the non-interactive report): an
// AGENTS section, a SKILLS section, each a heading, a six-dash rule, one
// line per item or "(none)", a blank line, then a single summary line.
//
// One deliberate divergence: installer.Render only ever says "updated" or
// "unchanged" for an item, because Item.Status collapses "missing at the
// target" and "present but different" into the same StatusUpdated case.
// This report instead distinguishes "installed" (new) from "updated"
// (replaced), because the interactive UI already groups selected items
// that way before the user confirms — a summary that then said "updated"
// for something the UI had shown as new would contradict what the user
// just saw.
func RenderReport(w io.Writer, out Outcome) error {
	agentsInstalled, agentsUpdated, agentsFailed, err := renderSection(w, "AGENTS", out.Applied, installer.KindAgent)
	if err != nil {
		return err
	}
	skillsInstalled, skillsUpdated, skillsFailed, err := renderSection(w, "SKILLS", out.Applied, installer.KindSkill)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d installed, %d updated, %d failed\n",
		agentsInstalled+skillsInstalled, agentsUpdated+skillsUpdated, agentsFailed+skillsFailed); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// renderSection writes one heading, its rule, the outcomes for kind (or
// "(none)"), and a trailing blank line, and returns how many of its items
// were installed, updated, and failed.
func renderSection(w io.Writer, heading string, applied []ItemOutcome, kind installer.Kind) (installed, updated, failed int, err error) {
	if _, err := fmt.Fprintf(w, "%s\n------\n", heading); err != nil {
		return 0, 0, 0, fmt.Errorf("write report: %w", err)
	}

	sorted := sortedOutcomes(applied, kind)

	if len(sorted) == 0 {
		if _, err := fmt.Fprintf(w, "(none)\n\n"); err != nil {
			return 0, 0, 0, fmt.Errorf("write report: %w", err)
		}
		return 0, 0, 0, nil
	}

	for _, oc := range sorted {
		switch {
		case oc.Err != nil:
			if _, err := fmt.Fprintf(w, "%-10s %s: %v\n", "failed", oc.Item.Name, oc.Err); err != nil {
				return 0, 0, 0, fmt.Errorf("write report: %w", err)
			}
			failed++
		case oc.Item.Exists:
			if _, err := fmt.Fprintf(w, "%-10s %s\n", "updated", oc.Item.Name); err != nil {
				return 0, 0, 0, fmt.Errorf("write report: %w", err)
			}
			updated++
		default:
			if _, err := fmt.Fprintf(w, "%-10s %s\n", "installed", oc.Item.Name); err != nil {
				return 0, 0, 0, fmt.Errorf("write report: %w", err)
			}
			installed++
		}
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return 0, 0, 0, fmt.Errorf("write report: %w", err)
	}
	return installed, updated, failed, nil
}

// outcomeRank orders outcomes for display within a section: failures
// first, since they are the part of the report a user must act on, then
// freshly installed items, then updated ones.
func outcomeRank(oc ItemOutcome) int {
	switch {
	case oc.Err != nil:
		return 0
	case !oc.Item.Exists:
		return 1
	default:
		return 2
	}
}

// sortedOutcomes returns the outcomes for kind, reordered for display:
// all failed items first, then all installed items, then all updated
// items, each group sorted by Item.Name. It copies before sorting so the
// caller's slice is never reordered out from under it — the same
// precaution installer.Render takes, and for the same reason: callers may
// still hold onto that slice after RenderReport returns.
func sortedOutcomes(applied []ItemOutcome, kind installer.Kind) []ItemOutcome {
	var sorted []ItemOutcome
	for _, oc := range applied {
		if oc.Item.Kind == kind {
			sorted = append(sorted, oc)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		ri, rj := outcomeRank(sorted[i]), outcomeRank(sorted[j])
		if ri != rj {
			return ri < rj
		}
		return sorted[i].Item.Name < sorted[j].Item.Name
	})
	return sorted
}
