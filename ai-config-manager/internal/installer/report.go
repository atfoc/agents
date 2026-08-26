package installer

import (
	"fmt"
	"io"
	"sort"
)

// Render writes a human-readable report of res to w: an AGENTS section, a
// SKILLS section, and a summary line. It touches nothing on disk — it only
// formats the Result it is given.
//
// If dryRun is true, a banner is written first to make clear that the
// report describes what would happen, not what did happen.
func Render(w io.Writer, res Result, dryRun bool) error {
	if dryRun {
		if _, err := fmt.Fprintf(w, "DRY RUN — nothing written\n\n"); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	agentsUpdated, agentsUnchanged, err := renderSection(w, "AGENTS", res.Agents)
	if err != nil {
		return err
	}
	skillsUpdated, skillsUnchanged, err := renderSection(w, "SKILLS", res.Skills)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d updated, %d unchanged\n",
		agentsUpdated+skillsUpdated, agentsUnchanged+skillsUnchanged); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// renderSection writes one heading, its rule, its items (or "(none)"), and a
// trailing blank line, and returns how many of its items were updated and
// unchanged.
func renderSection(w io.Writer, heading string, items []Item) (updated, unchanged int, err error) {
	if _, err := fmt.Fprintf(w, "%s\n------\n", heading); err != nil {
		return 0, 0, fmt.Errorf("write report: %w", err)
	}

	sorted := sortedItems(items)

	if len(sorted) == 0 {
		if _, err := fmt.Fprintf(w, "(none)\n\n"); err != nil {
			return 0, 0, fmt.Errorf("write report: %w", err)
		}
		return 0, 0, nil
	}

	for _, item := range sorted {
		if _, err := fmt.Fprintf(w, "%-10s %s\n", item.Status, item.Name); err != nil {
			return 0, 0, fmt.Errorf("write report: %w", err)
		}
		if item.Status == StatusUpdated {
			updated++
		} else {
			unchanged++
		}
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return 0, 0, fmt.Errorf("write report: %w", err)
	}
	return updated, unchanged, nil
}

// sortedItems returns items reordered for display: all StatusUpdated items
// first, then all StatusUnchanged items, each group sorted by Name. It
// copies before sorting so the caller's slice — which Plan/Apply may still
// hold onto — is never reordered out from under it.
func sortedItems(items []Item) []Item {
	sorted := make([]Item, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Status != sorted[j].Status {
			return sorted[i].Status == StatusUpdated
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// RenderRemoved writes a human-readable report of a removal run to w: an
// AGENTS section, a SKILLS section, and a summary line. It mirrors Render's
// shape exactly — same headings, same six-dash rule, same "(none)", same
// %-10s status column — so the two reports read as one format. It touches
// nothing on disk.
//
// If dryRun is true, a banner is written first to make clear that the
// report describes what would happen, not what did happen.
func RenderRemoved(w io.Writer, res TargetResult, dryRun bool) error {
	if dryRun {
		if _, err := fmt.Fprintf(w, "DRY RUN — nothing removed\n\n"); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	agentsRemoved, err := renderRemovedSection(w, "AGENTS", res.Agents)
	if err != nil {
		return err
	}
	skillsRemoved, err := renderRemovedSection(w, "SKILLS", res.Skills)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d removed\n", agentsRemoved+skillsRemoved); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// renderRemovedSection writes one heading, its rule, its items (or
// "(none)"), and a trailing blank line, and returns how many it listed.
func renderRemovedSection(w io.Writer, heading string, items []TargetItem) (removed int, err error) {
	if _, err := fmt.Fprintf(w, "%s\n------\n", heading); err != nil {
		return 0, fmt.Errorf("write report: %w", err)
	}

	sorted := sortedTargetItems(items)

	if len(sorted) == 0 {
		if _, err := fmt.Fprintf(w, "(none)\n\n"); err != nil {
			return 0, fmt.Errorf("write report: %w", err)
		}
		return 0, nil
	}

	for _, item := range sorted {
		if _, err := fmt.Fprintf(w, "%-10s %s\n", "removed", item.Name); err != nil {
			return 0, fmt.Errorf("write report: %w", err)
		}
		removed++
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return 0, fmt.Errorf("write report: %w", err)
	}
	return removed, nil
}

// sortedTargetItems returns items sorted by Name, copying before sorting so
// the caller's slice is never reordered out from under it. There is no
// status to group by here: every item in a removal report has the same
// outcome.
func sortedTargetItems(items []TargetItem) []TargetItem {
	sorted := make([]TargetItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
