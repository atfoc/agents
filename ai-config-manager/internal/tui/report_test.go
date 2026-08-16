package tui

import (
	"bytes"
	"errors"
	"testing"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

// TestRenderReport checks the exact rendered output of RenderReport,
// byte-for-byte, across the shapes the interactive session can produce:
// a normal run with a mix of installed/updated/failed items, a run where
// one section has nothing in it, a run where nothing was applied at all,
// and a run whose failures need their error text visible. Each case
// compares the full output against a literal, not just a substring, so a
// change to spacing, the six-dash rule, or the blank lines would show up
// here immediately.
func TestRenderReport(t *testing.T) {
	tests := []struct {
		name string
		out  Outcome
		want string
	}{
		{
			name: "both sections populated with installed, updated, and failed items",
			out: Outcome{
				Confirmed: true,
				Applied: []ItemOutcome{
					{Item: installer.Item{Kind: installer.KindAgent, Name: "scout.md", Exists: false}},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "thinker.md", Exists: true}},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "worker.md"}, Err: errors.New("permission denied")},
					{Item: installer.Item{Kind: installer.KindSkill, Name: "research", Exists: false}},
					{Item: installer.Item{Kind: installer.KindSkill, Name: "make-plan", Exists: true}},
					{Item: installer.Item{Kind: installer.KindSkill, Name: "implement-plan"}, Err: errors.New("disk full")},
				},
			},
			want: `AGENTS
------
failed     worker.md: permission denied
installed  scout.md
updated    thinker.md

SKILLS
------
failed     implement-plan: disk full
installed  research
updated    make-plan

2 installed, 2 updated, 2 failed
`,
		},
		{
			name: "skills section empty prints (none)",
			out: Outcome{
				Confirmed: true,
				Applied: []ItemOutcome{
					{Item: installer.Item{Kind: installer.KindAgent, Name: "scout.md", Exists: false}},
				},
			},
			want: `AGENTS
------
installed  scout.md

SKILLS
------
(none)

1 installed, 0 updated, 0 failed
`,
		},
		{
			// A zero-value Outcome is what a quit-without-installing session
			// produces before the caller even decides whether to call
			// RenderReport at all; it must still render cleanly.
			name: "empty outcome prints (none) for both sections",
			out:  Outcome{},
			want: `AGENTS
------
(none)

SKILLS
------
(none)

0 installed, 0 updated, 0 failed
`,
		},
		{
			// Err takes priority over Exists: an item that already existed
			// but failed to write is reported as failed, never as updated.
			name: "failure line carries the error text and wins over Exists",
			out: Outcome{
				Confirmed: true,
				Applied: []ItemOutcome{
					{
						Item: installer.Item{Kind: installer.KindAgent, Name: "scout.md", Exists: true},
						Err:  errors.New("permission denied: /tmp/x"),
					},
				},
			},
			want: `AGENTS
------
failed     scout.md: permission denied: /tmp/x

SKILLS
------
(none)

0 installed, 0 updated, 1 failed
`,
		},
		{
			// Fed in scrambled order with two items in each outcome group,
			// the output must still come out failed-then-installed-then-
			// updated, alphabetical by name within each group.
			name: "sorts failed before installed before updated, alphabetically within each",
			out: Outcome{
				Confirmed: true,
				Applied: []ItemOutcome{
					{Item: installer.Item{Kind: installer.KindAgent, Name: "yankee", Exists: true}},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "zeta", Exists: false}},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "charlie"}, Err: errors.New("err-c")},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "bravo", Exists: true}},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "whiskey"}, Err: errors.New("err-w")},
					{Item: installer.Item{Kind: installer.KindAgent, Name: "alpha", Exists: false}},
				},
			},
			want: `AGENTS
------
failed     charlie: err-c
failed     whiskey: err-w
installed  alpha
installed  zeta
updated    bravo
updated    yankee

SKILLS
------
(none)

2 installed, 2 updated, 2 failed
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := RenderReport(&buf, tt.out); err != nil {
				t.Fatalf("RenderReport: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("RenderReport output mismatch.\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// TestRenderReport_DoesNotMutateInput ensures RenderReport copies before
// sorting rather than reordering out.Applied in place, the same
// precaution installer.Render takes with its own slices: a caller may
// still hold onto and reuse that slice after RenderReport returns.
func TestRenderReport_DoesNotMutateInput(t *testing.T) {
	out := Outcome{
		Confirmed: true,
		Applied: []ItemOutcome{
			{Item: installer.Item{Kind: installer.KindAgent, Name: "worker.md", Exists: true}},
			{Item: installer.Item{Kind: installer.KindAgent, Name: "thinker.md"}, Err: errors.New("boom")},
			{Item: installer.Item{Kind: installer.KindAgent, Name: "scout.md", Exists: false}},
		},
	}
	wantOrder := itemNames(out.Applied)

	var buf bytes.Buffer
	if err := RenderReport(&buf, out); err != nil {
		t.Fatalf("RenderReport: %v", err)
	}

	if got := itemNames(out.Applied); !slicesEqual(got, wantOrder) {
		t.Errorf("Applied order mutated: got %v, want %v", got, wantOrder)
	}
}

func itemNames(applied []ItemOutcome) []string {
	names := make([]string, len(applied))
	for i, oc := range applied {
		names[i] = oc.Item.Name
	}
	return names
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
