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

// TargetItem is a single agent or skill found in the target directory,
// considered for removal. It has no Src: removal is target-driven, and what
// the source ships has no bearing on what can be removed.
type TargetItem struct {
	Kind Kind
	// Name is the item's name as it appears under its Kind's directory:
	// "scout.md" for an agent, "research" for a skill. This is the same
	// convention as Item.Name, deliberately: it is what lets the bare-name
	// rules for --uninstall be the same code as the ones for --install, and
	// what lets the interactive mode detect an install/remove conflict as a
	// plain equality on {Kind, Name}.
	Name string
	// Path is the absolute path to the target file (agent) or directory
	// (skill) to be removed. It is called Path rather than Dst because Dst
	// only means something opposite a Src, and there is none here.
	Path string
	// InSource records whether the source also ships an item of this Kind
	// and Name. It drives display grouping only; it never affects whether
	// the item can be removed. It is stored rather than derived because it
	// is computed once, during planning, where both lists are in hand;
	// deriving it later would mean handing the source list to display code
	// that otherwise has no business with it.
	InSource bool
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

// UninstallGroup is how the interactive mode buckets a target item for
// display. Like Group, it is derived on demand and never stored.
type UninstallGroup int

const (
	GroupAlsoInSource UninstallGroup = iota
	GroupOnlyInTarget
)

// String returns the heading an UninstallGroup is shown under.
func (g UninstallGroup) String() string {
	switch g {
	case GroupAlsoInSource:
		return "also in source"
	case GroupOnlyInTarget:
		return "only in target"
	default:
		return fmt.Sprintf("UninstallGroup(%d)", g)
	}
}

// Group buckets the target item for display. GroupAlsoInSource is the zero
// value so that it sorts first, which is the order the feature calls for:
// the group where a conflict with an install selection is possible comes
// before the group that is the point of the feature.
func (t TargetItem) Group() UninstallGroup {
	if t.InSource {
		return GroupAlsoInSource
	}
	return GroupOnlyInTarget
}

// Result is the outcome of a planning pass: every agent and every skill
// considered, each with its computed Status.
type Result struct {
	Agents []Item
	Skills []Item
}

// TargetResult is the outcome of a target-scanning pass: every agent and
// every skill currently present in the target.
type TargetResult struct {
	Agents []TargetItem
	Skills []TargetItem
}

// nameKey identifies an item across the source and target lists.
type nameKey struct {
	Kind Kind
	Name string
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
	// Install, when non-empty, narrows the run to exactly these item names.
	// Empty means "everything the source ships". It is only meaningful
	// together with OnlyAgents or OnlySkills.
	Install []string
	// Uninstall names the items to remove from the target. A non-empty
	// Uninstall makes this an uninstall run: it is mutually exclusive with
	// Install, and with Source being set at all. Like Install, it is only
	// meaningful together with OnlyAgents or OnlySkills.
	Uninstall []string
	// Interactive selects the terminal UI instead of a straight-through run.
	Interactive bool
}
