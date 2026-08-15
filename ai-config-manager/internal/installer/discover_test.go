package installer

import (
	"os"
	"path/filepath"
	"testing"
)

// mustWriteFile creates path (and any missing parent directories) holding
// content.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mustMkdir creates path (and any missing parents) as a directory.
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// mustSymlink creates newname as a symlink pointing at oldname.
func mustSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(newname), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(newname), err)
	}
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatalf("symlink %s -> %s: %v", newname, oldname, err)
	}
}

func itemNames(items []Item) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names
}

func TestDiscoverAgents_FiltersByExtensionAndKind(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(srcRoot, "agents", "scout.md"), "scout")
	mustWriteFile(t, filepath.Join(srcRoot, "agents", "notes.txt"), "not an agent")
	mustWriteFile(t, filepath.Join(srcRoot, "agents", "README"), "not an agent either")
	// The trap: a directory whose name ends in .md must not be mistaken for
	// an agent file.
	mustMkdir(t, filepath.Join(srcRoot, "agents", "something.md"))

	items, err := discoverAgents(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverAgents: %v", err)
	}
	if got := itemNames(items); len(got) != 1 || got[0] != "scout.md" {
		t.Fatalf("discoverAgents = %v, want [scout.md]", got)
	}
}

func TestDiscoverAgents_SortedByName(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	for _, name := range []string{"zeta.md", "alpha.md", "mid.md"} {
		mustWriteFile(t, filepath.Join(srcRoot, "agents", name), name)
	}

	items, err := discoverAgents(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverAgents: %v", err)
	}
	want := []string{"alpha.md", "mid.md", "zeta.md"}
	if got := itemNames(items); !slicesEqual(got, want) {
		t.Fatalf("discoverAgents order = %v, want %v", got, want)
	}
}

func TestDiscoverAgents_SymlinkToFileIncluded(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	real := filepath.Join(t.TempDir(), "actual.md")
	mustWriteFile(t, real, "real content")
	mustSymlink(t, real, filepath.Join(srcRoot, "agents", "linked.md"))

	items, err := discoverAgents(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverAgents: %v", err)
	}
	if got := itemNames(items); len(got) != 1 || got[0] != "linked.md" {
		t.Fatalf("discoverAgents = %v, want [linked.md]", got)
	}
}

func TestDiscoverAgents_MissingDirIsNotError(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	items, err := discoverAgents(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverAgents: %v", err)
	}
	if items != nil {
		t.Fatalf("discoverAgents = %v, want nil", items)
	}
}

func TestDiscoverAgents_SrcDstPathsAndZeroValues(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(srcRoot, "agents", "scout.md"), "scout")

	items, err := discoverAgents(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverAgents: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("discoverAgents returned %d items, want 1", len(items))
	}
	item := items[0]

	wantSrc := filepath.Join(srcRoot, "agents", "scout.md")
	wantDst := filepath.Join(dstRoot, "agents", "scout.md")
	if item.Src != wantSrc {
		t.Errorf("Src = %q, want %q", item.Src, wantSrc)
	}
	if item.Dst != wantDst {
		t.Errorf("Dst = %q, want %q", item.Dst, wantDst)
	}
	if item.Kind != KindAgent {
		t.Errorf("Kind = %v, want KindAgent", item.Kind)
	}
	if item.Status != StatusUpdated {
		t.Errorf("Status = %v, want StatusUpdated (zero value)", item.Status)
	}
}

func TestDiscoverAgents_AgentsPathIsRegularFile(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// agents/ exists but as a plain file, not a directory: this must be a
	// real error, not a silent empty result indistinguishable from "no
	// agents/ directory at all".
	mustWriteFile(t, filepath.Join(srcRoot, "agents"), "not a directory")

	items, err := discoverAgents(srcRoot, dstRoot)
	if err == nil {
		t.Fatalf("discoverAgents = %v, %v, want an error", items, err)
	}
}

func TestDiscoverSkills_FiltersByKind(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(srcRoot, "skills", "research", "SKILL.md"), "research skill")
	mustWriteFile(t, filepath.Join(srcRoot, "skills", "loose.txt"), "not a skill")

	items, err := discoverSkills(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverSkills: %v", err)
	}
	if got := itemNames(items); len(got) != 1 || got[0] != "research" {
		t.Fatalf("discoverSkills = %v, want [research]", got)
	}
}

func TestDiscoverSkills_SortedByName(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	for _, name := range []string{"zeta", "alpha", "mid"} {
		mustWriteFile(t, filepath.Join(srcRoot, "skills", name, "SKILL.md"), name)
	}

	items, err := discoverSkills(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverSkills: %v", err)
	}
	want := []string{"alpha", "mid", "zeta"}
	if got := itemNames(items); !slicesEqual(got, want) {
		t.Fatalf("discoverSkills order = %v, want %v", got, want)
	}
}

func TestDiscoverSkills_SymlinkToDirIncluded(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	realDir := filepath.Join(t.TempDir(), "actual-skill")
	mustWriteFile(t, filepath.Join(realDir, "SKILL.md"), "real skill")
	mustSymlink(t, realDir, filepath.Join(srcRoot, "skills", "linked-skill"))

	items, err := discoverSkills(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverSkills: %v", err)
	}
	if got := itemNames(items); len(got) != 1 || got[0] != "linked-skill" {
		t.Fatalf("discoverSkills = %v, want [linked-skill]", got)
	}
}

func TestDiscoverSkills_MissingDirIsNotError(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	items, err := discoverSkills(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverSkills: %v", err)
	}
	if items != nil {
		t.Fatalf("discoverSkills = %v, want nil", items)
	}
}

func TestDiscoverSkills_SrcDstPathsAndKind(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(srcRoot, "skills", "research", "SKILL.md"), "research skill")

	items, err := discoverSkills(srcRoot, dstRoot)
	if err != nil {
		t.Fatalf("discoverSkills: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("discoverSkills returned %d items, want 1", len(items))
	}
	item := items[0]

	wantSrc := filepath.Join(srcRoot, "skills", "research")
	wantDst := filepath.Join(dstRoot, "skills", "research")
	if item.Src != wantSrc {
		t.Errorf("Src = %q, want %q", item.Src, wantSrc)
	}
	if item.Dst != wantDst {
		t.Errorf("Dst = %q, want %q", item.Dst, wantDst)
	}
	if item.Kind != KindSkill {
		t.Errorf("Kind = %v, want KindSkill", item.Kind)
	}
	if item.Status != StatusUpdated {
		t.Errorf("Status = %v, want StatusUpdated (zero value)", item.Status)
	}
}

func TestDiscoverSkills_SkillsPathIsRegularFile(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(srcRoot, "skills"), "not a directory")

	items, err := discoverSkills(srcRoot, dstRoot)
	if err == nil {
		t.Fatalf("discoverSkills = %v, %v, want an error", items, err)
	}
}

func TestKindString(t *testing.T) {
	cases := []struct {
		kind Kind
		want string
	}{
		{KindAgent, "agents"},
		{KindSkill, "skills"},
		{Kind(7), "Kind(7)"},
	}
	for _, c := range cases {
		if got := c.kind.String(); got != c.want {
			t.Errorf("Kind(%d).String() = %q, want %q", c.kind, got, c.want)
		}
	}
}

func TestStatusString(t *testing.T) {
	cases := []struct {
		status Status
		want   string
	}{
		{StatusUpdated, "updated"},
		{StatusUnchanged, "unchanged"},
		{Status(7), "Status(7)"},
	}
	for _, c := range cases {
		if got := c.status.String(); got != c.want {
			t.Errorf("Status(%d).String() = %q, want %q", c.status, got, c.want)
		}
	}
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
