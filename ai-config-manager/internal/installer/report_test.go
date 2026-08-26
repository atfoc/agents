package installer

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// errWriter is an io.Writer whose Write always fails, used to verify that
// Render surfaces the underlying writer's error rather than swallowing it.
type errWriter struct{}

var errWriterFailed = errors.New("boom")

func (errWriter) Write(p []byte) (int, error) {
	return 0, errWriterFailed
}

func sampleResult() Result {
	return Result{
		Agents: []Item{
			{Kind: KindAgent, Name: "scout.md", Status: StatusUpdated},
			{Kind: KindAgent, Name: "thinker.md", Status: StatusUpdated},
			{Kind: KindAgent, Name: "worker.md", Status: StatusUnchanged},
		},
		Skills: []Item{
			{Kind: KindSkill, Name: "make-plan", Status: StatusUpdated},
			{Kind: KindSkill, Name: "implement-plan", Status: StatusUnchanged},
			{Kind: KindSkill, Name: "research", Status: StatusUnchanged},
		},
	}
}

const wantDryRunOutput = `DRY RUN — nothing written

AGENTS
------
updated    scout.md
updated    thinker.md
unchanged  worker.md

SKILLS
------
updated    make-plan
unchanged  implement-plan
unchanged  research

3 updated, 3 unchanged
`

const wantNoDryRunOutput = `AGENTS
------
updated    scout.md
updated    thinker.md
unchanged  worker.md

SKILLS
------
updated    make-plan
unchanged  implement-plan
unchanged  research

3 updated, 3 unchanged
`

func TestRender_DryRunExactOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleResult(), true); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := buf.String(); got != wantDryRunOutput {
		t.Fatalf("Render output mismatch.\ngot:\n%s\nwant:\n%s", got, wantDryRunOutput)
	}
}

func TestRender_NoDryRunExactOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleResult(), false); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := buf.String(); got != wantNoDryRunOutput {
		t.Fatalf("Render output mismatch.\ngot:\n%s\nwant:\n%s", got, wantNoDryRunOutput)
	}
}

// TestRender_SortsRegardlessOfInputOrder proves that Render does its own
// grouping and sorting: fed items in the opposite of the required order
// (unchanged before updated, names reverse-alphabetical), the output must
// still come out grouped-then-sorted.
func TestRender_SortsRegardlessOfInputOrder(t *testing.T) {
	res := Result{
		Agents: []Item{
			{Kind: KindAgent, Name: "worker.md", Status: StatusUnchanged},
			{Kind: KindAgent, Name: "thinker.md", Status: StatusUpdated},
			{Kind: KindAgent, Name: "scout.md", Status: StatusUpdated},
		},
		Skills: []Item{
			{Kind: KindSkill, Name: "research", Status: StatusUnchanged},
			{Kind: KindSkill, Name: "implement-plan", Status: StatusUnchanged},
			{Kind: KindSkill, Name: "make-plan", Status: StatusUpdated},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, res, true); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := buf.String(); got != wantDryRunOutput {
		t.Fatalf("Render output mismatch with scrambled input.\ngot:\n%s\nwant:\n%s", got, wantDryRunOutput)
	}
}

// TestRender_DoesNotMutateInput ensures Render copies before sorting rather
// than reordering the caller's slices in place.
func TestRender_DoesNotMutateInput(t *testing.T) {
	res := Result{
		Agents: []Item{
			{Kind: KindAgent, Name: "worker.md", Status: StatusUnchanged},
			{Kind: KindAgent, Name: "thinker.md", Status: StatusUpdated},
			{Kind: KindAgent, Name: "scout.md", Status: StatusUpdated},
		},
		Skills: []Item{
			{Kind: KindSkill, Name: "research", Status: StatusUnchanged},
			{Kind: KindSkill, Name: "implement-plan", Status: StatusUnchanged},
			{Kind: KindSkill, Name: "make-plan", Status: StatusUpdated},
		},
	}
	wantAgents := itemNames(res.Agents)
	wantSkills := itemNames(res.Skills)

	var buf bytes.Buffer
	if err := Render(&buf, res, true); err != nil {
		t.Fatalf("Render: %v", err)
	}

	if got := itemNames(res.Agents); !slicesEqual(got, wantAgents) {
		t.Errorf("Agents order mutated: got %v, want %v", got, wantAgents)
	}
	if got := itemNames(res.Skills); !slicesEqual(got, wantSkills) {
		t.Errorf("Skills order mutated: got %v, want %v", got, wantSkills)
	}
}

func TestRender_BothSectionsEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Result{}, false); err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := "AGENTS\n------\n(none)\n\nSKILLS\n------\n(none)\n\n0 updated, 0 unchanged\n"
	if got := buf.String(); got != want {
		t.Fatalf("Render output mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRender_OnlyAgentsNoSkills(t *testing.T) {
	res := Result{
		Agents: []Item{
			{Kind: KindAgent, Name: "scout.md", Status: StatusUpdated},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, false); err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := "AGENTS\n------\nupdated    scout.md\n\nSKILLS\n------\n(none)\n\n1 updated, 0 unchanged\n"
	if got := buf.String(); got != want {
		t.Fatalf("Render output mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRender_AllUnchangedSummary(t *testing.T) {
	res := Result{
		Agents: []Item{
			{Kind: KindAgent, Name: "scout.md", Status: StatusUnchanged},
		},
		Skills: []Item{
			{Kind: KindSkill, Name: "research", Status: StatusUnchanged},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, false); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := buf.String(); !strings.HasSuffix(got, "0 updated, 2 unchanged\n") {
		t.Fatalf("Render summary = %q, want suffix %q", got, "0 updated, 2 unchanged\n")
	}
}

func TestRender_WriterErrorIsReturned(t *testing.T) {
	err := Render(errWriter{}, sampleResult(), true)
	if err == nil {
		t.Fatal("Render with a failing writer returned nil error, want non-nil")
	}
}

// TestRender_BannerUsesEmDash guards against a careless future edit
// downgrading the required em dash (U+2014) to a plain hyphen.
func TestRender_BannerUsesEmDash(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Result{}, true); err != nil {
		t.Fatalf("Render: %v", err)
	}
	line := strings.SplitN(buf.String(), "\n", 2)[0]
	if !strings.Contains(line, "—") {
		t.Fatalf("banner line %q does not contain an em dash (U+2014)", line)
	}
	if strings.Contains(line, " - ") {
		t.Fatalf("banner line %q contains a plain hyphen instead of an em dash", line)
	}
}

func sampleTargetResult() TargetResult {
	return TargetResult{
		Agents: []TargetItem{
			{Kind: KindAgent, Name: "scout.md"},
			{Kind: KindAgent, Name: "worker.md"},
		},
		Skills: []TargetItem{
			{Kind: KindSkill, Name: "make-plan"},
			{Kind: KindSkill, Name: "research"},
		},
	}
}

const wantRemovedOutput = `AGENTS
------
removed    scout.md
removed    worker.md

SKILLS
------
removed    make-plan
removed    research

4 removed
`

func TestRenderRemoved_ExactOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderRemoved(&buf, sampleTargetResult(), false); err != nil {
		t.Fatalf("RenderRemoved: %v", err)
	}
	if got := buf.String(); got != wantRemovedOutput {
		t.Fatalf("RenderRemoved output mismatch.\ngot:\n%s\nwant:\n%s", got, wantRemovedOutput)
	}
}

// TestRenderRemoved_DryRunBanner pins both the wording and the em dash
// (U+2014), matching Render's own banner.
func TestRenderRemoved_DryRunBanner(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderRemoved(&buf, sampleTargetResult(), true); err != nil {
		t.Fatalf("RenderRemoved: %v", err)
	}
	want := "DRY RUN — nothing removed\n\n" + wantRemovedOutput
	if got := buf.String(); got != want {
		t.Fatalf("RenderRemoved output mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
	line := strings.SplitN(buf.String(), "\n", 2)[0]
	if !strings.Contains(line, "—") {
		t.Fatalf("banner line %q does not contain an em dash (U+2014)", line)
	}
	if strings.Contains(line, " - ") {
		t.Fatalf("banner line %q contains a plain hyphen instead of an em dash", line)
	}
}

func TestRenderRemoved_BothSectionsEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderRemoved(&buf, TargetResult{}, false); err != nil {
		t.Fatalf("RenderRemoved: %v", err)
	}
	want := "AGENTS\n------\n(none)\n\nSKILLS\n------\n(none)\n\n0 removed\n"
	if got := buf.String(); got != want {
		t.Fatalf("RenderRemoved output mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderRemoved_SortsByNameRegardlessOfInputOrder(t *testing.T) {
	res := TargetResult{
		Agents: []TargetItem{
			{Kind: KindAgent, Name: "worker.md"},
			{Kind: KindAgent, Name: "scout.md"},
		},
		Skills: []TargetItem{
			{Kind: KindSkill, Name: "research"},
			{Kind: KindSkill, Name: "make-plan"},
		},
	}

	var buf bytes.Buffer
	if err := RenderRemoved(&buf, res, false); err != nil {
		t.Fatalf("RenderRemoved: %v", err)
	}
	if got := buf.String(); got != wantRemovedOutput {
		t.Fatalf("RenderRemoved output mismatch with scrambled input.\ngot:\n%s\nwant:\n%s", got, wantRemovedOutput)
	}
}

func TestRenderRemoved_DoesNotMutateInput(t *testing.T) {
	res := TargetResult{
		Agents: []TargetItem{
			{Kind: KindAgent, Name: "worker.md"},
			{Kind: KindAgent, Name: "scout.md"},
		},
		Skills: []TargetItem{
			{Kind: KindSkill, Name: "research"},
			{Kind: KindSkill, Name: "make-plan"},
		},
	}
	wantAgents := targetItemNames(res.Agents)
	wantSkills := targetItemNames(res.Skills)

	var buf bytes.Buffer
	if err := RenderRemoved(&buf, res, false); err != nil {
		t.Fatalf("RenderRemoved: %v", err)
	}

	if got := targetItemNames(res.Agents); !slicesEqual(got, wantAgents) {
		t.Errorf("Agents order mutated: got %v, want %v", got, wantAgents)
	}
	if got := targetItemNames(res.Skills); !slicesEqual(got, wantSkills) {
		t.Errorf("Skills order mutated: got %v, want %v", got, wantSkills)
	}
}

func TestRenderRemoved_WriterErrorIsReturned(t *testing.T) {
	err := RenderRemoved(errWriter{}, sampleTargetResult(), true)
	if err == nil {
		t.Fatal("RenderRemoved with a failing writer returned nil error, want non-nil")
	}
}
