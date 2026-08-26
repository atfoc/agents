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

func TestDiscoverSkills_DirWithoutSkillMdExcluded(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(srcRoot, "skills", "research", "SKILL.md"), "research skill")
	// A directory under skills/ with no SKILL.md isn't a skill, no matter
	// what else it contains.
	mustWriteFile(t, filepath.Join(srcRoot, "skills", "not-a-skill", "notes.md"), "just notes")

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

func targetItemNames(items []TargetItem) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names
}

func TestDiscoverTargetAgents_FiltersByExtensionAndKind(t *testing.T) {
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(dstRoot, "agents", "scout.md"), "scout")
	mustWriteFile(t, filepath.Join(dstRoot, "agents", "notes.txt"), "not an agent")
	// The trap: a directory whose name ends in .md must not be mistaken for
	// an agent file.
	mustMkdir(t, filepath.Join(dstRoot, "agents", "c.md"))

	items, err := discoverTargetAgents(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetAgents: %v", err)
	}
	if got := targetItemNames(items); len(got) != 1 || got[0] != "scout.md" {
		t.Fatalf("discoverTargetAgents = %v, want [scout.md]", got)
	}
}

func TestDiscoverTargetAgents_SortedByName(t *testing.T) {
	dstRoot := t.TempDir()

	for _, name := range []string{"zeta.md", "alpha.md", "mid.md"} {
		mustWriteFile(t, filepath.Join(dstRoot, "agents", name), name)
	}

	items, err := discoverTargetAgents(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetAgents: %v", err)
	}
	want := []string{"alpha.md", "mid.md", "zeta.md"}
	if got := targetItemNames(items); !slicesEqual(got, want) {
		t.Fatalf("discoverTargetAgents order = %v, want %v", got, want)
	}
}

func TestDiscoverTargetAgents_SymlinkToFileIncluded(t *testing.T) {
	dstRoot := t.TempDir()

	real := filepath.Join(t.TempDir(), "actual.md")
	mustWriteFile(t, real, "real content")
	mustSymlink(t, real, filepath.Join(dstRoot, "agents", "linked.md"))

	items, err := discoverTargetAgents(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetAgents: %v", err)
	}
	if got := targetItemNames(items); len(got) != 1 || got[0] != "linked.md" {
		t.Fatalf("discoverTargetAgents = %v, want [linked.md]", got)
	}
}

func TestDiscoverTargetAgents_DanglingSymlinkSkipped(t *testing.T) {
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(dstRoot, "agents", "scout.md"), "scout")
	// Deliberate divergence from discoverAgents, which treats a failing Stat
	// as fatal: the target is not under this tool's control, so junk left
	// there must not stop the run.
	mustSymlink(t, filepath.Join(t.TempDir(), "gone.md"), filepath.Join(dstRoot, "agents", "dangling.md"))

	items, err := discoverTargetAgents(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetAgents: %v", err)
	}
	if got := targetItemNames(items); len(got) != 1 || got[0] != "scout.md" {
		t.Fatalf("discoverTargetAgents = %v, want [scout.md]", got)
	}
}

func TestDiscoverTargetAgents_MissingDirIsNotError(t *testing.T) {
	dstRoot := t.TempDir()

	items, err := discoverTargetAgents(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetAgents: %v", err)
	}
	if items != nil {
		t.Fatalf("discoverTargetAgents = %v, want nil", items)
	}
}

func TestDiscoverTargetAgents_PathAndZeroValues(t *testing.T) {
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(dstRoot, "agents", "scout.md"), "scout")

	items, err := discoverTargetAgents(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetAgents: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("discoverTargetAgents returned %d items, want 1", len(items))
	}
	item := items[0]

	wantPath := filepath.Join(dstRoot, "agents", "scout.md")
	if item.Path != wantPath {
		t.Errorf("Path = %q, want %q", item.Path, wantPath)
	}
	if item.Kind != KindAgent {
		t.Errorf("Kind = %v, want KindAgent", item.Kind)
	}
	if item.InSource {
		t.Errorf("InSource = true, want false (only the planner fills it in)")
	}
}

func TestDiscoverTargetAgents_AgentsPathIsRegularFile(t *testing.T) {
	dstRoot := t.TempDir()

	// agents/ exists but as a plain file, not a directory: a real error, not
	// a silent empty result indistinguishable from "no agents/ at all".
	mustWriteFile(t, filepath.Join(dstRoot, "agents"), "not a directory")

	items, err := discoverTargetAgents(dstRoot)
	if err == nil {
		t.Fatalf("discoverTargetAgents = %v, %v, want an error", items, err)
	}
}

func TestDiscoverTargetSkills_FiltersByKind(t *testing.T) {
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(dstRoot, "skills", "research", "SKILL.md"), "research skill")
	mustWriteFile(t, filepath.Join(dstRoot, "skills", "loose.txt"), "not a skill")

	items, err := discoverTargetSkills(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetSkills: %v", err)
	}
	if got := targetItemNames(items); len(got) != 1 || got[0] != "research" {
		t.Fatalf("discoverTargetSkills = %v, want [research]", got)
	}
}

func TestDiscoverTargetSkills_DirWithoutSkillMdExcluded(t *testing.T) {
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(dstRoot, "skills", "research", "SKILL.md"), "research skill")
	// A directory under skills/ with no SKILL.md isn't a skill, no matter
	// what else it contains, and this tool never offers it for removal.
	mustWriteFile(t, filepath.Join(dstRoot, "skills", "not-a-skill", "notes.md"), "just notes")

	items, err := discoverTargetSkills(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetSkills: %v", err)
	}
	if got := targetItemNames(items); len(got) != 1 || got[0] != "research" {
		t.Fatalf("discoverTargetSkills = %v, want [research]", got)
	}
}

func TestDiscoverTargetSkills_SymlinkToDirIncluded(t *testing.T) {
	dstRoot := t.TempDir()

	realDir := filepath.Join(t.TempDir(), "actual-skill")
	mustWriteFile(t, filepath.Join(realDir, "SKILL.md"), "real skill")
	mustSymlink(t, realDir, filepath.Join(dstRoot, "skills", "linked-skill"))

	items, err := discoverTargetSkills(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetSkills: %v", err)
	}
	if got := targetItemNames(items); len(got) != 1 || got[0] != "linked-skill" {
		t.Fatalf("discoverTargetSkills = %v, want [linked-skill]", got)
	}
}

func TestDiscoverTargetSkills_SortedByName(t *testing.T) {
	dstRoot := t.TempDir()

	for _, name := range []string{"zeta", "alpha", "mid"} {
		mustWriteFile(t, filepath.Join(dstRoot, "skills", name, "SKILL.md"), name)
	}

	items, err := discoverTargetSkills(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetSkills: %v", err)
	}
	want := []string{"alpha", "mid", "zeta"}
	if got := targetItemNames(items); !slicesEqual(got, want) {
		t.Fatalf("discoverTargetSkills order = %v, want %v", got, want)
	}
}

func TestDiscoverTargetSkills_MissingDirIsNotError(t *testing.T) {
	dstRoot := t.TempDir()

	items, err := discoverTargetSkills(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetSkills: %v", err)
	}
	if items != nil {
		t.Fatalf("discoverTargetSkills = %v, want nil", items)
	}
}

func TestDiscoverTargetSkills_PathAndKind(t *testing.T) {
	dstRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(dstRoot, "skills", "research", "SKILL.md"), "research skill")

	items, err := discoverTargetSkills(dstRoot)
	if err != nil {
		t.Fatalf("discoverTargetSkills: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("discoverTargetSkills returned %d items, want 1", len(items))
	}
	item := items[0]

	wantPath := filepath.Join(dstRoot, "skills", "research")
	if item.Path != wantPath {
		t.Errorf("Path = %q, want %q", item.Path, wantPath)
	}
	if item.Kind != KindSkill {
		t.Errorf("Kind = %v, want KindSkill", item.Kind)
	}
	if item.InSource {
		t.Errorf("InSource = true, want false (only the planner fills it in)")
	}
}
