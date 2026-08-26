// Package tui implements the interactive (-i) mode: an inline Bubble Tea
// program for choosing which agents and skills to install, and which ones
// already at the target to remove.
package tui

import "github.com/atfoc/agents/ai-config-manager/internal/installer"

// InstallFunc writes one item to its target. The model takes it as a
// field rather than calling installer.ApplyItem directly so tests can
// drive the whole flow, failures included, without touching the disk.
type InstallFunc func(installer.Item) error

// RemoveFunc deletes one target item. The model takes it as a field, the
// same way it takes InstallFunc, so tests can drive the whole flow —
// failures included — without touching the disk.
type RemoveFunc func(installer.TargetItem) error

// ItemOutcome is what happened to one marked item. Like entry, it is
// flattened from either an installer.Item or an installer.TargetItem: by
// the time a report is written, all that matters is what was done, to what,
// and whether it worked. Keeping two typed slices instead would still force
// a merge into a common shape inside RenderReport, in order to sort
// failures first across both — so the flattening only moves earlier.
type ItemOutcome struct {
	Action action
	Kind   installer.Kind
	Name   string
	// Exists records whether an installed item replaced something already
	// at the target, which is what separates "updated" from "installed" in
	// the report. It is meaningless for a removal and left false there.
	Exists bool
	Err    error
}

// Outcome is what the interactive session did. Confirmed is false when
// the user quit without applying anything, in which case Applied is
// empty and the caller prints nothing.
type Outcome struct {
	Confirmed bool
	Applied   []ItemOutcome
}
