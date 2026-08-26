package installer

import (
	"fmt"
	"strings"
)

// bareName returns the name a user types on the command line for an item of
// this kind. Agents and skills disagree about what "bare" means: an agent is
// a single markdown file, so its Name carries the ".md" suffix
// ("scout.md"), while a skill is a directory and carries no such suffix
// ("research"). Asking the user to type the extension for one kind but not
// the other is the kind of inconsistency that trips people up every time,
// so the suffix is stripped for an agent — the name a user types is always
// "scout", never "scout.md".
func bareName(kind Kind, name string) string {
	if kind == KindAgent {
		return strings.TrimSuffix(name, ".md")
	}
	return name
}

// selectInstall narrows agents and skills to the items whose bare names
// appear in names, preserving each list's original order, and reports which
// names matched nothing at all.
//
// Both kinds are searched and missing is computed across both rather than
// per-kind: a name matched by either list is not missing. In practice
// --install always comes with --agents or --skills, so one of the two lists
// is already nil by the time this is called — but computing it this way
// means the function is correct without depending on that.
func selectInstall(agents, skills []Item, names []string) (a, s []Item, missing []string) {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	seen := map[string]bool{}

	for _, it := range agents {
		b := bareName(it.Kind, it.Name)
		if want[b] {
			a = append(a, it)
			seen[b] = true
		}
	}
	for _, it := range skills {
		b := bareName(it.Kind, it.Name)
		if want[b] {
			s = append(s, it)
			seen[b] = true
		}
	}
	// The user's own order, not the map's, so the error reads back the way
	// the command line was typed.
	for _, n := range names {
		if !seen[n] {
			missing = append(missing, n)
		}
	}
	return a, s, missing
}

// selectUninstall is selectInstall's twin for TargetItem.
func selectUninstall(agents, skills []TargetItem, names []string) (a, s []TargetItem, missing []string) {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	seen := map[string]bool{}

	for _, it := range agents {
		b := bareName(it.Kind, it.Name)
		if want[b] {
			a = append(a, it)
			seen[b] = true
		}
	}
	for _, it := range skills {
		b := bareName(it.Kind, it.Name)
		if want[b] {
			s = append(s, it)
			seen[b] = true
		}
	}
	for _, n := range names {
		if !seen[n] {
			missing = append(missing, n)
		}
	}
	return a, s, missing
}

// kindWord names the kind a run is scoped to, for error messages.
// --install and --uninstall both require --agents or --skills, so the
// "agents or skills" fallback is only reachable by a caller that builds
// Options directly.
func kindWord(opts Options) string {
	switch {
	case opts.OnlyAgents:
		return "agents"
	case opts.OnlySkills:
		return "skills"
	default:
		return "agents or skills"
	}
}

// quoteList renders names as a comma-separated, quoted list: `"a", "b"`.
func quoteList(names []string) string {
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%q", n))
	}
	return strings.Join(parts, ", ")
}
