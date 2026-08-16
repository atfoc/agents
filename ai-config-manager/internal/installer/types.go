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
// yet" and "there but different" are both cases that require a write. Where
// that distinction does matter for display, it is carried separately by
// Item.Exists, not folded back into Status.
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
	// Exists records whether anything was already present at Dst when the
	// item was planned. Status alone cannot answer that: an item that is
	// missing from the target and an item that is present but different are
	// both StatusUpdated. Only the interactive mode's grouping needs the
	// difference, which is why it is kept here rather than folded into
	// Status.
	Exists bool
	// Status is left at its zero value (StatusUpdated) by discovery; it is
	// filled in later once the item has actually been compared against the
	// target.
	Status Status
}

// Group is how the interactive mode buckets an item for display. It is
// derived from Status and Exists on demand and never stored: it is a
// presentation concern, not part of the plan.
type Group int

const (
	GroupToUpdate Group = iota
	GroupToInstall
	GroupUpToDate
)

// String returns the heading a Group is shown under.
func (g Group) String() string {
	switch g {
	case GroupToUpdate:
		return "to update"
	case GroupToInstall:
		return "to install"
	case GroupUpToDate:
		return "up to date"
	default:
		return fmt.Sprintf("Group(%d)", g)
	}
}

// Group buckets the item for display: an unchanged item is up to date
// regardless of Exists; among the rest, one that already existed at Dst is
// being updated, and one that did not is being installed for the first time.
func (i Item) Group() Group {
	switch {
	case i.Status == StatusUnchanged:
		return GroupUpToDate
	case i.Exists:
		return GroupToUpdate
	default:
		return GroupToInstall
	}
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
	// OnlyAgents and OnlySkills restrict a run to a single kind. They are
	// mutually exclusive; the command line rejects both at once.
	OnlyAgents bool
	OnlySkills bool
	// Filter, when set, narrows the run to the single item with this exact
	// name. It is only meaningful together with OnlyAgents or OnlySkills.
	Filter string
	// Interactive selects the terminal UI instead of a straight-through run.
	Interactive bool
}
