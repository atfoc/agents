// Package installer plans and applies the installation of agents and skills
// from a source directory into a target directory.
package installer

import "fmt"

// Kind distinguishes the two kinds of items the installer manages.
type Kind int

const (
	KindAgent Kind = iota
	KindSkill
)

// String returns the name of the directory a Kind lives under, both in the
// source tree and the target tree ("agents" or "skills").
func (k Kind) String() string {
	switch k {
	case KindAgent:
		return "agents"
	case KindSkill:
		return "skills"
	default:
		return fmt.Sprintf("Kind(%d)", k)
	}
}

// Status reports whether an item needed to be written to the target.
//
// There are deliberately only two states: an item is either updated or
// unchanged. An item that does not yet exist at the target is StatusUpdated,
// not some third "new" state — from the product's point of view, "not there
// yet" and "there but different" are both cases that require a write.
type Status int

const (
	// StatusUpdated is the zero value so that a freshly discovered Item,
	// before its Status has been computed, reads as "needs writing" rather
	// than silently claiming to already match the target.
	StatusUpdated Status = iota
	StatusUnchanged
)

// String returns the human-readable word used in reports ("updated" or
// "unchanged").
func (s Status) String() string {
	switch s {
	case StatusUpdated:
		return "updated"
	case StatusUnchanged:
		return "unchanged"
	default:
		return fmt.Sprintf("Status(%d)", s)
	}
}

// Item is a single agent or skill considered for installation.
type Item struct {
	Kind Kind
	// Name is the item's name as it appears under its Kind's directory:
	// "scout.md" for an agent, "research" for a skill.
	Name string
	// Src is the absolute path to the source file (agent) or directory
	// (skill).
	Src string
	// Dst is the absolute path to the corresponding destination file or
	// directory.
	Dst string
	// Status is left at its zero value (StatusUpdated) by discovery; it is
	// filled in later once the item has actually been compared against the
	// target.
	Status Status
}

// Result is the outcome of a planning pass: every agent and every skill
// considered, each with its computed Status.
type Result struct {
	Agents []Item
	Skills []Item
}

// Options configures a planning/apply run.
type Options struct {
	Source string
	Target string
	DryRun bool
}
