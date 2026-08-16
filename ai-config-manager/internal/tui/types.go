// Package tui implements the interactive (-i) mode: an inline Bubble Tea
// program for choosing which agents and skills to install.
package tui

import "github.com/atfoc/agents/ai-config-manager/internal/installer"

// InstallFunc writes one item to its target. The model takes it as a
// field rather than calling installer.ApplyItem directly so tests can
// drive the whole flow, failures included, without touching the disk.
type InstallFunc func(installer.Item) error

// ItemOutcome is what happened to one selected item.
type ItemOutcome struct {
	Item installer.Item
	Err  error
}

// Outcome is what the interactive session did. Confirmed is false when
// the user quit without installing anything, in which case Applied is
// empty and the caller prints nothing.
type Outcome struct {
	Confirmed bool
	Applied   []ItemOutcome
}
