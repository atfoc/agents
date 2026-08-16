package installer

import "strings"

// matchesFilter reports whether item is the item named by filter.
//
// The comparison is exact, against the item's bare name. Agents and skills
// disagree about what "bare" means: an agent is a single markdown file, so
// its Name carries the file's ".md" suffix ("scout.md"), while a skill is a
// directory and carries no such suffix ("research"). Asking the user to type
// the extension for one kind but not the other is the kind of inconsistency
// that trips people up every time, so the suffix is stripped before
// comparing an agent's Name — the name a user types is always "scout", never
// "scout.md".
func matchesFilter(item Item, filter string) bool {
	switch item.Kind {
	case KindAgent:
		return strings.TrimSuffix(item.Name, ".md") == filter
	case KindSkill:
		return item.Name == filter
	default:
		return false
	}
}

// filterItems returns the items matching filter, in their original order.
func filterItems(items []Item, filter string) []Item {
	var matched []Item
	for _, item := range items {
		if matchesFilter(item, filter) {
			matched = append(matched, item)
		}
	}
	return matched
}
