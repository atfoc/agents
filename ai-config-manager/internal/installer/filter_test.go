package installer

import "testing"

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name   string
		item   Item
		filter string
		want   bool
	}{
		{
			name:   "agent bare name matches filter without extension",
			item:   Item{Kind: KindAgent, Name: "scout.md"},
			filter: "scout",
			want:   true,
		},
		{
			// Deliberate product decision, not an oversight: users type the
			// bare name, so a filter that still carries ".md" is a literal
			// mismatch against the trimmed name. Do not "fix" this to also
			// accept "scout.md" — that would make the filter accept two
			// spellings for the same item instead of one.
			name:   "agent name with extension in filter does not match",
			item:   Item{Kind: KindAgent, Name: "scout.md"},
			filter: "scout.md",
			want:   false,
		},
		{
			name:   "skill name matches filter exactly",
			item:   Item{Kind: KindSkill, Name: "research"},
			filter: "research",
			want:   true,
		},
		{
			name:   "skill name does not gain an implicit extension",
			item:   Item{Kind: KindSkill, Name: "research"},
			filter: "research.md",
			want:   false,
		},
		{
			name:   "comparison is case sensitive",
			item:   Item{Kind: KindAgent, Name: "scout.md"},
			filter: "Scout",
			want:   false,
		},
		{
			name:   "agent name without .md suffix trims as a no-op",
			item:   Item{Kind: KindAgent, Name: "plain"},
			filter: "plain",
			want:   true,
		},
		{
			name:   "empty filter does not match a normally named item",
			item:   Item{Kind: KindAgent, Name: "scout.md"},
			filter: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesFilter(tt.item, tt.filter); got != tt.want {
				t.Errorf("matchesFilter(%+v, %q) = %v, want %v", tt.item, tt.filter, got, tt.want)
			}
		})
	}
}

func TestFilterItems(t *testing.T) {
	t.Run("narrows a mixed slice to the one matching item", func(t *testing.T) {
		items := []Item{
			{Kind: KindAgent, Name: "scout.md"},
			{Kind: KindAgent, Name: "writer.md"},
			{Kind: KindSkill, Name: "research"},
		}

		got := filterItems(items, "writer")

		if len(got) != 1 {
			t.Fatalf("filterItems() returned %d items, want 1: %+v", len(got), got)
		}
		if got[0].Name != "writer.md" {
			t.Errorf("filterItems()[0].Name = %q, want %q", got[0].Name, "writer.md")
		}
	})

	t.Run("returns nothing when no item matches", func(t *testing.T) {
		items := []Item{
			{Kind: KindAgent, Name: "scout.md"},
			{Kind: KindSkill, Name: "research"},
		}

		got := filterItems(items, "nonexistent")

		if len(got) != 0 {
			t.Errorf("filterItems() = %+v, want empty", got)
		}
	})

	t.Run("preserves original relative order across kinds", func(t *testing.T) {
		agent := Item{Kind: KindAgent, Name: "x.md"}
		skill := Item{Kind: KindSkill, Name: "x"}
		other := Item{Kind: KindAgent, Name: "y.md"}
		items := []Item{agent, other, skill}

		got := filterItems(items, "x")

		if len(got) != 2 {
			t.Fatalf("filterItems() returned %d items, want 2: %+v", len(got), got)
		}
		if got[0].Kind != KindAgent || got[1].Kind != KindSkill {
			t.Errorf("filterItems() order = %+v, want agent x.md before skill x", got)
		}
	})

	t.Run("does not mutate or reorder the input slice", func(t *testing.T) {
		items := []Item{
			{Kind: KindAgent, Name: "x.md"},
			{Kind: KindAgent, Name: "y.md"},
			{Kind: KindSkill, Name: "x"},
		}
		original := make([]Item, len(items))
		copy(original, items)

		_ = filterItems(items, "x")

		for i := range items {
			if items[i] != original[i] {
				t.Errorf("input slice mutated at index %d: got %+v, want %+v", i, items[i], original[i])
			}
		}
	})
}
