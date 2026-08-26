package installer

import (
	"reflect"
	"testing"
)

func TestBareName(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		in   string
		want string
	}{
		{
			name: "agent loses its .md suffix",
			kind: KindAgent,
			in:   "scout.md",
			want: "scout",
		},
		{
			name: "agent without a suffix trims as a no-op",
			kind: KindAgent,
			in:   "scout",
			want: "scout",
		},
		{
			name: "skill is returned unchanged",
			kind: KindSkill,
			in:   "research",
			want: "research",
		},
		{
			// Deliberate: only an agent is a markdown file, so a skill
			// directory that happens to be named "research.md" keeps that
			// name whole. Trimming here would make the two kinds disagree
			// about what the user typed.
			kind: KindSkill,
			name: "skill does not lose a .md suffix",
			in:   "research.md",
			want: "research.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bareName(tt.kind, tt.in); got != tt.want {
				t.Errorf("bareName(%v, %q) = %q, want %q", tt.kind, tt.in, got, tt.want)
			}
		})
	}
}

func TestSelectInstall_MatchesAcrossKinds(t *testing.T) {
	agents := []Item{
		{Kind: KindAgent, Name: "scout.md"},
		{Kind: KindAgent, Name: "writer.md"},
	}
	skills := []Item{
		{Kind: KindSkill, Name: "research"},
		{Kind: KindSkill, Name: "review"},
	}

	a, s, missing := selectInstall(agents, skills, []string{"scout", "review"})

	if len(a) != 1 || a[0].Name != "scout.md" {
		t.Errorf("agents = %+v, want exactly [scout.md]", a)
	}
	if len(s) != 1 || s[0].Name != "review" {
		t.Errorf("skills = %+v, want exactly [review]", s)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}

func TestSelectInstall_PreservesOriginalOrder(t *testing.T) {
	agents := []Item{
		{Kind: KindAgent, Name: "a.md"},
		{Kind: KindAgent, Name: "b.md"},
		{Kind: KindAgent, Name: "c.md"},
	}

	// The names are given in the reverse of the discovered order: the result
	// must follow the list's order, not the caller's.
	a, _, missing := selectInstall(agents, nil, []string{"c", "a"})

	if len(missing) != 0 {
		t.Fatalf("missing = %v, want empty", missing)
	}
	got := []string{}
	for _, it := range a {
		got = append(got, it.Name)
	}
	want := []string{"a.md", "c.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("agents = %v, want %v", got, want)
	}
}

func TestSelectInstall_MissingNamesInUserOrder(t *testing.T) {
	agents := []Item{{Kind: KindAgent, Name: "scout.md"}}

	a, s, missing := selectInstall(agents, nil, []string{"z", "a"})

	if len(a) != 0 || len(s) != 0 {
		t.Errorf("agents = %+v, skills = %+v, want both empty", a, s)
	}
	// Reported the way the command line was typed, not sorted: the error
	// message reads back the user's own order.
	want := []string{"z", "a"}
	if !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %v, want %v", missing, want)
	}
}

func TestSelectInstall_NameMatchedByEitherKindIsNotMissing(t *testing.T) {
	agents := []Item{{Kind: KindAgent, Name: "x.md"}}
	skills := []Item{{Kind: KindSkill, Name: "y"}}

	a, s, missing := selectInstall(agents, skills, []string{"x", "y"})

	if len(a) != 1 || a[0].Name != "x.md" {
		t.Errorf("agents = %+v, want exactly [x.md]", a)
	}
	if len(s) != 1 || s[0].Name != "y" {
		t.Errorf("skills = %+v, want exactly [y]", s)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty: a name matched by either list is not missing", missing)
	}
}

func TestSelectInstall_EmptyNamesMatchNothing(t *testing.T) {
	agents := []Item{{Kind: KindAgent, Name: "scout.md"}}
	skills := []Item{{Kind: KindSkill, Name: "research"}}

	// Documents the contract callers rely on: with no names, everything is
	// dropped rather than everything kept, which is why every caller guards
	// the call with len(names) > 0.
	a, s, missing := selectInstall(agents, skills, nil)

	if len(a) != 0 || len(s) != 0 {
		t.Errorf("agents = %+v, skills = %+v, want both empty", a, s)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}

func TestSelectUninstall_MatchesAcrossKinds(t *testing.T) {
	agents := []TargetItem{
		{Kind: KindAgent, Name: "scout.md"},
		{Kind: KindAgent, Name: "writer.md"},
	}
	skills := []TargetItem{
		{Kind: KindSkill, Name: "research"},
		{Kind: KindSkill, Name: "review"},
	}

	a, s, missing := selectUninstall(agents, skills, []string{"scout", "review"})

	if len(a) != 1 || a[0].Name != "scout.md" {
		t.Errorf("agents = %+v, want exactly [scout.md]", a)
	}
	if len(s) != 1 || s[0].Name != "review" {
		t.Errorf("skills = %+v, want exactly [review]", s)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}

func TestSelectUninstall_PreservesOriginalOrder(t *testing.T) {
	agents := []TargetItem{
		{Kind: KindAgent, Name: "a.md"},
		{Kind: KindAgent, Name: "b.md"},
		{Kind: KindAgent, Name: "c.md"},
	}

	a, _, missing := selectUninstall(agents, nil, []string{"c", "a"})

	if len(missing) != 0 {
		t.Fatalf("missing = %v, want empty", missing)
	}
	got := []string{}
	for _, it := range a {
		got = append(got, it.Name)
	}
	want := []string{"a.md", "c.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("agents = %v, want %v", got, want)
	}
}

func TestSelectUninstall_MissingNamesInUserOrder(t *testing.T) {
	agents := []TargetItem{{Kind: KindAgent, Name: "scout.md"}}

	a, s, missing := selectUninstall(agents, nil, []string{"z", "a"})

	if len(a) != 0 || len(s) != 0 {
		t.Errorf("agents = %+v, skills = %+v, want both empty", a, s)
	}
	want := []string{"z", "a"}
	if !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %v, want %v", missing, want)
	}
}

func TestSelectUninstall_NameMatchedByEitherKindIsNotMissing(t *testing.T) {
	agents := []TargetItem{{Kind: KindAgent, Name: "x.md"}}
	skills := []TargetItem{{Kind: KindSkill, Name: "y"}}

	a, s, missing := selectUninstall(agents, skills, []string{"x", "y"})

	if len(a) != 1 || a[0].Name != "x.md" {
		t.Errorf("agents = %+v, want exactly [x.md]", a)
	}
	if len(s) != 1 || s[0].Name != "y" {
		t.Errorf("skills = %+v, want exactly [y]", s)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty: a name matched by either list is not missing", missing)
	}
}

func TestSelectUninstall_EmptyNamesMatchNothing(t *testing.T) {
	agents := []TargetItem{{Kind: KindAgent, Name: "scout.md"}}
	skills := []TargetItem{{Kind: KindSkill, Name: "research"}}

	a, s, missing := selectUninstall(agents, skills, nil)

	if len(a) != 0 || len(s) != 0 {
		t.Errorf("agents = %+v, skills = %+v, want both empty", a, s)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}

func TestKindWord(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{"agents", Options{OnlyAgents: true}, "agents"},
		{"skills", Options{OnlySkills: true}, "skills"},
		{
			// Unreachable from the command line, which requires one of the
			// two scoping flags; only a caller building Options directly can
			// get here.
			name: "neither",
			opts: Options{},
			want: "agents or skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := kindWord(tt.opts); got != tt.want {
				t.Errorf("kindWord(%+v) = %q, want %q", tt.opts, got, tt.want)
			}
		})
	}
}

func TestQuoteList(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  string
	}{
		{"one name", []string{"a"}, `"a"`},
		{"several names", []string{"a", "b", "c"}, `"a", "b", "c"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteList(tt.names); got != tt.want {
				t.Errorf("quoteList(%v) = %s, want %s", tt.names, got, tt.want)
			}
		})
	}
}
