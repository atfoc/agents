package installer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// sourceAgents and sourceSkillFiles are the fixed contents makeSourceTree
// writes for each agent/skill, keyed by name. Tests reuse these maps rather
// than hardcoding content strings so that "what the source ships" and "what
// a test expects to find" can never silently drift apart.
var sourceAgents = map[string]string{
	"scout.md":   "scout content",
	"thinker.md": "thinker content",
	"worker.md":  "worker content",
}

var sourceSkillFiles = map[string]string{
	"make-plan":      "make-plan skill",
	"implement-plan": "implement-plan skill",
	"research":       "research skill",
}

// makeSourceTree builds a realistic source directory: three agents and
// three skills, each skill a directory containing a single SKILL.md.
func makeSourceTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range sourceAgents {
		mustWriteFile(t, filepath.Join(root, "agents", name), content)
	}
	for name, content := range sourceSkillFiles {
		mustWriteFile(t, filepath.Join(root, "skills", name, "SKILL.md"), content)
	}
	return root
}

// findItem returns the item named name from items, if present.
func findItem(items []Item, name string) (Item, bool) {
	for _, it := range items {
		if it.Name == name {
			return it, true
		}
	}
	return Item{}, false
}

// allItems flattens a Result's two sections into one slice for tests that
// want to assert something about every item regardless of Kind.
func allItems(res Result) []Item {
	items := make([]Item, 0, len(res.Agents)+len(res.Skills))
	items = append(items, res.Agents...)
	items = append(items, res.Skills...)
	return items
}

// fileSnapshot captures everything about one path that a dry run, or a
// no-op rerun, must leave untouched: its kind, its content, its permission
// bits, and its modification time. modTime is what actually distinguishes
// "left alone" from "rewritten with identical bytes" — a rewrite creates a
// fresh file even when its content ends up the same.
type fileSnapshot struct {
	isDir   bool
	content string
	mode    fs.FileMode
	modTime time.Time
}

// snapshotTree walks root and records every entry under it. A root that
// does not exist at all yields an empty, valid snapshot rather than an
// error, since "the target does not exist yet" is itself a state a snapshot
// must be able to represent and compare.
func snapshotTree(t *testing.T, root string) map[string]fileSnapshot {
	t.Helper()
	snap := make(map[string]fileSnapshot)

	if _, err := os.Lstat(root); errors.Is(err, fs.ErrNotExist) {
		return snap
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			snap[rel] = fileSnapshot{isDir: true, mode: info.Mode()}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snap[rel] = fileSnapshot{content: string(content), mode: info.Mode(), modTime: info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return snap
}

func TestRun_FreshTarget_AllUpdatedAndFilesLand(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Agents) != len(sourceAgents) || len(res.Skills) != len(sourceSkillFiles) {
		t.Fatalf("Result = %+v, want %d agents and %d skills", res, len(sourceAgents), len(sourceSkillFiles))
	}
	for _, it := range allItems(res) {
		if it.Status != StatusUpdated {
			t.Errorf("%s %s Status = %v, want StatusUpdated", it.Kind, it.Name, it.Status)
		}
	}

	for name, content := range sourceAgents {
		got, err := os.ReadFile(filepath.Join(dst, "agents", name))
		if err != nil {
			t.Fatalf("read installed agent %s: %v", name, err)
		}
		if string(got) != content {
			t.Errorf("agent %s content = %q, want %q", name, got, content)
		}
	}
	for name, content := range sourceSkillFiles {
		got, err := os.ReadFile(filepath.Join(dst, "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatalf("read installed skill %s: %v", name, err)
		}
		if string(got) != content {
			t.Errorf("skill %s content = %q, want %q", name, got, content)
		}
	}
}

func TestRun_RerunOnPopulatedTarget_AllUnchangedAndNothingRewritten(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	if _, err := Run(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	before := snapshotTree(t, dst)

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	for _, it := range allItems(res) {
		if it.Status != StatusUnchanged {
			t.Errorf("%s %s Status = %v, want StatusUnchanged", it.Kind, it.Name, it.Status)
		}
	}

	after := snapshotTree(t, dst)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("target tree changed on a no-op rerun (mtimes or content differ).\nbefore: %+v\nafter:  %+v", before, after)
	}
}

func TestRun_ChangedAgentBytes_OnlyThatAgentUpdated(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	if _, err := Run(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout content v2")

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	for _, it := range res.Agents {
		want := StatusUnchanged
		if it.Name == "scout.md" {
			want = StatusUpdated
		}
		if it.Status != want {
			t.Errorf("agent %s Status = %v, want %v", it.Name, it.Status, want)
		}
	}
	for _, it := range res.Skills {
		if it.Status != StatusUnchanged {
			t.Errorf("skill %s Status = %v, want StatusUnchanged", it.Name, it.Status)
		}
	}

	got, err := os.ReadFile(filepath.Join(dst, "agents", "scout.md"))
	if err != nil {
		t.Fatalf("read updated agent: %v", err)
	}
	if string(got) != "scout content v2" {
		t.Errorf("content = %q, want %q", got, "scout content v2")
	}
}

func TestRun_ChangedFileInsideSkill_OnlyThatSkillUpdated(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	if _, err := Run(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	mustWriteFile(t, filepath.Join(src, "skills", "research", "SKILL.md"), "research skill v2")

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	for _, it := range res.Skills {
		want := StatusUnchanged
		if it.Name == "research" {
			want = StatusUpdated
		}
		if it.Status != want {
			t.Errorf("skill %s Status = %v, want %v", it.Name, it.Status, want)
		}
	}
	for _, it := range res.Agents {
		if it.Status != StatusUnchanged {
			t.Errorf("agent %s Status = %v, want StatusUnchanged", it.Name, it.Status)
		}
	}
}

func TestRun_StaleFileInsideTargetSkill_MarksUpdatedAndRemovesStaleFile(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	if _, err := Run(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	stale := filepath.Join(dst, "skills", "research", "stale.txt")
	mustWriteFile(t, stale, "leftover from a previous version")

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	item, ok := findItem(res.Skills, "research")
	if !ok {
		t.Fatalf("research skill missing from Result: %+v", res.Skills)
	}
	if item.Status != StatusUpdated {
		t.Errorf("research skill Status = %v, want StatusUpdated", item.Status)
	}

	if _, err := os.Stat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stale.txt still exists after Run (err = %v)", err)
	}
}

// TestRun_PreservesUntrackedTargetItems is the single most important
// behavioral guarantee of the installer: items already in the target that
// the source does not ship must never be read, written, deleted, or
// reported.
func TestRun_PreservesUntrackedTargetItems(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	mineAgent := filepath.Join(dst, "agents", "mine.md")
	mustWriteFile(t, mineAgent, "not from source")
	mineSkillFile := filepath.Join(dst, "skills", "mine", "SKILL.md")
	mustWriteFile(t, mineSkillFile, "not from source either")

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, found := findItem(res.Agents, "mine.md"); found {
		t.Errorf("mine.md appears in Result.Agents, want it absent")
	}
	if _, found := findItem(res.Skills, "mine"); found {
		t.Errorf("mine appears in Result.Skills, want it absent")
	}

	got, err := os.ReadFile(mineAgent)
	if err != nil {
		t.Fatalf("read %s: %v", mineAgent, err)
	}
	if string(got) != "not from source" {
		t.Errorf("mine.md content = %q, want unchanged", got)
	}
	got, err = os.ReadFile(mineSkillFile)
	if err != nil {
		t.Fatalf("read %s: %v", mineSkillFile, err)
	}
	if string(got) != "not from source either" {
		t.Errorf("mine/SKILL.md content = %q, want unchanged", got)
	}
}

func TestRun_DryRun_FreshTargetTouchesNothing(t *testing.T) {
	src := makeSourceTree(t)
	dst := filepath.Join(t.TempDir(), "does-not-exist-yet")

	before := snapshotTree(t, dst)
	if len(before) != 0 {
		t.Fatalf("snapshot of a nonexistent target is not empty: %+v", before)
	}

	res, err := Run(Options{Source: src, Target: dst, DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	after := snapshotTree(t, dst)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("dry run touched the target.\nbefore: %+v\nafter:  %+v", before, after)
	}

	for _, it := range allItems(res) {
		if it.Status != StatusUpdated {
			t.Errorf("%s %s Status = %v, want StatusUpdated", it.Kind, it.Name, it.Status)
		}
	}
}

func TestRun_DryRun_PartiallyPopulatedTargetTouchesNothing(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	if _, err := Run(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("seed Run: %v", err)
	}
	// Change one source agent so a real run against this target would report
	// a mix of updated and unchanged items, not a uniform one.
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout content v2")

	before := snapshotTree(t, dst)
	res, err := Run(Options{Source: src, Target: dst, DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	after := snapshotTree(t, dst)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("dry run touched the target.\nbefore: %+v\nafter:  %+v", before, after)
	}

	scout, ok := findItem(res.Agents, "scout.md")
	if !ok || scout.Status != StatusUpdated {
		t.Errorf("scout.md = %+v, ok=%v, want StatusUpdated", scout, ok)
	}
	thinker, ok := findItem(res.Agents, "thinker.md")
	if !ok || thinker.Status != StatusUnchanged {
		t.Errorf("thinker.md = %+v, ok=%v, want StatusUnchanged", thinker, ok)
	}
	for _, it := range res.Skills {
		if it.Status != StatusUnchanged {
			t.Errorf("skill %s Status = %v, want StatusUnchanged", it.Name, it.Status)
		}
	}
}

func TestRun_TargetDirectoryCreatedWhenMissing(t *testing.T) {
	src := makeSourceTree(t)
	dst := filepath.Join(t.TempDir(), "brand-new-target")

	if _, err := os.Stat(dst); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("target %s already exists before Run", dst)
	}

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Agents) != len(sourceAgents) || len(res.Skills) != len(sourceSkillFiles) {
		t.Fatalf("Result = %+v, want everything discovered and installed", res)
	}

	got, err := os.ReadFile(filepath.Join(dst, "agents", "scout.md"))
	if err != nil {
		t.Fatalf("read installed agent: %v", err)
	}
	if string(got) != sourceAgents["scout.md"] {
		t.Errorf("content = %q, want %q", got, sourceAgents["scout.md"])
	}
}

func TestRun_SourceHasOnlyAgents(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout content")
	dst := t.TempDir()

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Agents) != 1 {
		t.Fatalf("Agents = %v, want 1 item", res.Agents)
	}
	if len(res.Skills) != 0 {
		t.Fatalf("Skills = %v, want empty", res.Skills)
	}
	// No skills shipped means the skills/ section has nothing to write, so
	// the target's skills/ directory must never come into existence.
	if _, err := os.Stat(filepath.Join(dst, "skills")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("target skills/ was created despite source shipping no skills")
	}
}

func TestRun_SourceHasOnlySkills(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "skills", "research", "SKILL.md"), "research skill")
	dst := t.TempDir()

	res, err := Run(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Skills) != 1 {
		t.Fatalf("Skills = %v, want 1 item", res.Skills)
	}
	if len(res.Agents) != 0 {
		t.Fatalf("Agents = %v, want empty", res.Agents)
	}
	if _, err := os.Stat(filepath.Join(dst, "agents")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("target agents/ was created despite source shipping no agents")
	}
}

func TestApply_SkipsUnchangedItems(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	res, err := Plan(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// Force one agent and one skill to StatusUnchanged without having
	// actually written them to dst, to prove Apply trusts Status rather than
	// checking the filesystem itself.
	for i := range res.Agents {
		if res.Agents[i].Name == "scout.md" {
			res.Agents[i].Status = StatusUnchanged
		}
	}
	for i := range res.Skills {
		if res.Skills[i].Name == "research" {
			res.Skills[i].Status = StatusUnchanged
		}
	}

	if err := Apply(res); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "agents", "scout.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("scout.md was written despite StatusUnchanged")
	}
	if _, err := os.Stat(filepath.Join(dst, "skills", "research")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("research skill was written despite StatusUnchanged")
	}
	if _, err := os.Stat(filepath.Join(dst, "agents", "thinker.md")); err != nil {
		t.Errorf("thinker.md (StatusUpdated) was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "skills", "make-plan")); err != nil {
		t.Errorf("make-plan skill (StatusUpdated) was not written: %v", err)
	}
}

func TestPlan_ErrorSourceDoesNotExist(t *testing.T) {
	dst := t.TempDir()
	missing := filepath.Join(t.TempDir(), "nope")

	_, err := Plan(Options{Source: missing, Target: dst})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q does not mention %q", err, missing)
	}
}

func TestPlan_ErrorSourceIsRegularFile(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "src.txt")
	mustWriteFile(t, srcFile, "not a directory")
	dst := t.TempDir()

	_, err := Plan(Options{Source: srcFile, Target: dst})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "src.txt") {
		t.Errorf("error %q does not mention %q", err, "src.txt")
	}
}

func TestPlan_ErrorSourceHasNeitherAgentsNorSkills(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "README.md"), "nothing installable here")
	dst := t.TempDir()

	_, err := Plan(Options{Source: src, Target: dst})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), filepath.Base(src)) {
		t.Errorf("error %q does not mention source path %q", err, src)
	}
	if !strings.Contains(err.Error(), "agents/") || !strings.Contains(err.Error(), "skills/") {
		t.Errorf("error %q does not explain that neither agents/ nor skills/ was found", err)
	}
}

func TestPlan_ErrorSourceEqualsTarget(t *testing.T) {
	dir := t.TempDir()

	_, err := Plan(Options{Source: dir, Target: dir})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), filepath.Base(dir)) {
		t.Errorf("error %q does not mention path %q", err, dir)
	}
}

func TestPlan_ErrorTargetNestedInsideSource(t *testing.T) {
	src := makeSourceTree(t)
	dst := filepath.Join(src, "nested-target")

	_, err := Plan(Options{Source: src, Target: dst})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "nested-target") {
		t.Errorf("error %q does not mention the nested target %q", err, dst)
	}
}

func TestPlan_ErrorSourceNestedInsideTarget(t *testing.T) {
	dst := t.TempDir()
	src := filepath.Join(dst, "nested-source")
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout")

	_, err := Plan(Options{Source: src, Target: dst})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "nested-source") {
		t.Errorf("error %q does not mention the nested source %q", err, src)
	}
}

func TestPlan_ErrorEmptySource(t *testing.T) {
	dst := t.TempDir()

	_, err := Plan(Options{Source: "", Target: dst})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "Source") {
		t.Errorf("error %q does not mention Source", err)
	}
}

func TestPlan_ErrorEmptyTarget(t *testing.T) {
	src := makeSourceTree(t)

	_, err := Plan(Options{Source: src, Target: ""})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "Target") {
		t.Errorf("error %q does not mention Target", err)
	}
}

// TestPlan_SiblingPrefixIsNotTreatedAsContained guards the exact trap the
// spec calls out: a raw strings.HasPrefix check would wrongly treat
// ".../foo-backup" as being inside ".../foo" merely because one string is a
// textual prefix of the other. Two real sibling directories, run through
// the real Plan entry point, survive a refactor of contains() in a way a
// unit test of contains() alone would not.
func TestPlan_SiblingPrefixIsNotTreatedAsContained(t *testing.T) {
	parent := t.TempDir()
	src := filepath.Join(parent, "foo")
	dst := filepath.Join(parent, "foo-backup")
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout content")
	mustMkdir(t, dst)

	if _, err := Plan(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("Plan returned an error for sibling directories %q and %q: %v", src, dst, err)
	}
}

// TestPlan_TargetSymlinkToSourceTriggersIdenticalPathGuard proves the
// identical-path guard compares fully symlink-resolved paths, not raw
// strings: a target that is merely a symlink pointing at the source must
// still be caught.
func TestPlan_TargetSymlinkToSourceTriggersIdenticalPathGuard(t *testing.T) {
	parent := t.TempDir()
	src := filepath.Join(parent, "src")
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout content")
	link := filepath.Join(parent, "link-to-src")
	mustSymlink(t, src, link)

	_, err := Plan(Options{Source: src, Target: link})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil (target symlink resolves to source)")
	}
	if !strings.Contains(err.Error(), "resolve to") {
		t.Errorf("error %q does not describe the identical-path guard", err)
	}
}

func TestContains(t *testing.T) {
	cases := []struct {
		parent, child string
		want          bool
	}{
		{"/tmp/foo", "/tmp/foo/bar", true},
		{"/tmp/foo", "/tmp/foo-backup", false},
		{"/tmp/foo-backup", "/tmp/foo", false},
		{"/tmp/foo", "/tmp/foo", false},
		{"/tmp/foo", "/tmp/bar", false},
	}
	for _, c := range cases {
		if got := contains(c.parent, c.child); got != c.want {
			t.Errorf("contains(%q, %q) = %v, want %v", c.parent, c.child, got, c.want)
		}
	}
}

func TestResolveTarget_ExistingPathIsFullyResolved(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(filepath.Dir(real), "link-"+filepath.Base(real))
	mustSymlink(t, real, link)

	got, err := resolveTarget(link)
	if err != nil {
		t.Fatalf("resolveTarget: %v", err)
	}
	want, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if got != want {
		t.Errorf("resolveTarget(%q) = %q, want %q", link, got, want)
	}
}

func TestPlan_EmptyTarget_AllItemsExistsFalse(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	res, err := Plan(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, it := range allItems(res) {
		if it.Exists {
			t.Errorf("%s %s Exists = true, want false", it.Kind, it.Name)
		}
	}
}

func TestPlan_PopulatedTarget_ExistsTrueEvenWhenContentDiffers(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	if _, err := Run(Options{Source: src, Target: dst}); err != nil {
		t.Fatalf("seed Run: %v", err)
	}
	// Change the source after seeding so a second Plan sees a target item
	// that both already exists AND differs from the source — the exact
	// combination (StatusUpdated, Exists true) the interactive "to update"
	// group depends on.
	mustWriteFile(t, filepath.Join(src, "agents", "scout.md"), "scout content v2")

	res, err := Plan(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	scout, ok := findItem(res.Agents, "scout.md")
	if !ok {
		t.Fatalf("scout.md missing from Result: %+v", res.Agents)
	}
	if scout.Status != StatusUpdated {
		t.Errorf("scout.md Status = %v, want StatusUpdated", scout.Status)
	}
	if !scout.Exists {
		t.Errorf("scout.md Exists = false, want true")
	}

	// An unchanged item that was already installed must also report Exists
	// true, since Exists tracks presence at Dst, not Status.
	thinker, ok := findItem(res.Agents, "thinker.md")
	if !ok {
		t.Fatalf("thinker.md missing from Result: %+v", res.Agents)
	}
	if thinker.Status != StatusUnchanged {
		t.Errorf("thinker.md Status = %v, want StatusUnchanged", thinker.Status)
	}
	if !thinker.Exists {
		t.Errorf("thinker.md Exists = false, want true")
	}

	for _, it := range allItems(res) {
		if !it.Exists {
			t.Errorf("%s %s Exists = false, want true (target was fully seeded)", it.Kind, it.Name)
		}
	}
}

func TestPlan_OnlyAgents_SkillsEmptyAgentsPopulated(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	res, err := Plan(Options{Source: src, Target: dst, OnlyAgents: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(res.Skills) != 0 {
		t.Errorf("Skills = %v, want empty", res.Skills)
	}
	if len(res.Agents) != len(sourceAgents) {
		t.Errorf("Agents = %v, want %d items", res.Agents, len(sourceAgents))
	}
}

func TestPlan_OnlySkills_AgentsEmptySkillsPopulated(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	res, err := Plan(Options{Source: src, Target: dst, OnlySkills: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(res.Agents) != 0 {
		t.Errorf("Agents = %v, want empty", res.Agents)
	}
	if len(res.Skills) != len(sourceSkillFiles) {
		t.Errorf("Skills = %v, want %d items", res.Skills, len(sourceSkillFiles))
	}
}

func TestPlan_Install_NarrowsToOneItem(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	t.Run("agent", func(t *testing.T) {
		// Agents match on the bare name: the file is "scout.md" but the
		// name is typed without the extension.
		res, err := Plan(Options{Source: src, Target: dst, OnlyAgents: true, Install: []string{"scout"}})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		if len(res.Skills) != 0 {
			t.Errorf("Skills = %v, want empty", res.Skills)
		}
		if len(res.Agents) != 1 || res.Agents[0].Name != "scout.md" {
			t.Errorf("Agents = %v, want exactly [scout.md]", res.Agents)
		}
	})

	t.Run("skill", func(t *testing.T) {
		res, err := Plan(Options{Source: src, Target: dst, OnlySkills: true, Install: []string{"research"}})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		if len(res.Agents) != 0 {
			t.Errorf("Agents = %v, want empty", res.Agents)
		}
		if len(res.Skills) != 1 || res.Skills[0].Name != "research" {
			t.Errorf("Skills = %v, want exactly [research]", res.Skills)
		}
	})
}

func TestPlan_ErrorInstallMatchesNothing(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	_, err := Plan(Options{Source: src, Target: dst, Install: []string{"does-not-exist"}})
	if err == nil {
		t.Fatal("Plan = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("error %q does not mention the missing name %q", err, "does-not-exist")
	}

	if _, err := os.Stat(filepath.Join(dst, "agents")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("target agents/ was created despite a no-match --install")
	}
	if _, err := os.Stat(filepath.Join(dst, "skills")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("target skills/ was created despite a no-match --install")
	}
}

func TestApplyItem_WritesRegardlessOfStatus(t *testing.T) {
	src := makeSourceTree(t)
	dst := t.TempDir()

	res, err := Plan(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	agent, ok := findItem(res.Agents, "scout.md")
	if !ok {
		t.Fatalf("scout.md missing from Result: %+v", res.Agents)
	}
	// Force StatusUnchanged even though nothing has been written to dst yet,
	// to prove ApplyItem writes regardless of Status — unlike Apply, which
	// would skip this item.
	agent.Status = StatusUnchanged
	if err := ApplyItem(agent); err != nil {
		t.Fatalf("ApplyItem(agent): %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "agents", "scout.md"))
	if err != nil {
		t.Fatalf("read installed agent: %v", err)
	}
	if string(got) != sourceAgents["scout.md"] {
		t.Errorf("agent content = %q, want %q", got, sourceAgents["scout.md"])
	}

	skill, ok := findItem(res.Skills, "research")
	if !ok {
		t.Fatalf("research skill missing from Result: %+v", res.Skills)
	}
	skill.Status = StatusUnchanged
	if err := ApplyItem(skill); err != nil {
		t.Fatalf("ApplyItem(skill): %v", err)
	}
	got, err = os.ReadFile(filepath.Join(dst, "skills", "research", "SKILL.md"))
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if string(got) != sourceSkillFiles["research"] {
		t.Errorf("skill content = %q, want %q", got, sourceSkillFiles["research"])
	}
}

func TestResolveTarget_NonExistentTailAppendedAsPlainText(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "does", "not", "exist", "yet")

	got, err := resolveTarget(target)
	if err != nil {
		t.Fatalf("resolveTarget: %v", err)
	}
	wantBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	want := filepath.Join(wantBase, "does", "not", "exist", "yet")
	if got != want {
		t.Errorf("resolveTarget(%q) = %q, want %q", target, got, want)
	}
}

// targetAgents and targetSkillFiles are the fixed contents makeTargetTree
// writes, keyed by name. As with sourceAgents/sourceSkillFiles, tests read
// them rather than hardcoding strings so "what the target holds" and "what a
// test expects to find" cannot drift apart.
var targetAgents = map[string]string{
	"scout.md":  "installed scout",
	"stale.md":  "installed stale",
	"worker.md": "installed worker",
}

var targetSkillFiles = map[string]string{
	"research":  "installed research skill",
	"stale":     "installed stale skill",
	"make-plan": "installed make-plan skill",
}

// makeTargetTree builds a realistic target directory: three agents and three
// skills, each skill a directory containing a single SKILL.md.
func makeTargetTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range targetAgents {
		mustWriteFile(t, filepath.Join(root, "agents", name), content)
	}
	for name, content := range targetSkillFiles {
		mustWriteFile(t, filepath.Join(root, "skills", name, "SKILL.md"), content)
	}
	return root
}

// findTargetItem returns the target item named name from items, if present.
func findTargetItem(items []TargetItem, name string) (TargetItem, bool) {
	for _, it := range items {
		if it.Name == name {
			return it, true
		}
	}
	return TargetItem{}, false
}

// TestPlanTarget_NoSourceRequired is load-bearing: it proves the sourceless
// path exists at all. Every other PlanTarget test would still pass if
// PlanTarget quietly demanded a Source the way Plan does.
func TestPlanTarget_NoSourceRequired(t *testing.T) {
	dst := makeTargetTree(t)

	res, err := PlanTarget(Options{Target: dst})
	if err != nil {
		t.Fatalf("PlanTarget with an empty Source: %v", err)
	}
	if len(res.Agents) != len(targetAgents) || len(res.Skills) != len(targetSkillFiles) {
		t.Fatalf("PlanTarget = %d agents, %d skills; want %d and %d",
			len(res.Agents), len(res.Skills), len(targetAgents), len(targetSkillFiles))
	}
}

func TestPlanTarget_ListsTargetItems(t *testing.T) {
	dst := makeTargetTree(t)

	res, err := PlanTarget(Options{Target: dst})
	if err != nil {
		t.Fatalf("PlanTarget: %v", err)
	}

	wantAgents := []string{"scout.md", "stale.md", "worker.md"}
	if got := targetItemNames(res.Agents); !slicesEqual(got, wantAgents) {
		t.Errorf("agents = %v, want %v", got, wantAgents)
	}
	wantSkills := []string{"make-plan", "research", "stale"}
	if got := targetItemNames(res.Skills); !slicesEqual(got, wantSkills) {
		t.Errorf("skills = %v, want %v", got, wantSkills)
	}

	for _, it := range res.Agents {
		if it.Kind != KindAgent {
			t.Errorf("agent %s Kind = %v, want KindAgent", it.Name, it.Kind)
		}
		if info, err := os.Stat(it.Path); err != nil || !info.Mode().IsRegular() {
			t.Errorf("agent %s Path = %q, which is not a regular file (err %v)", it.Name, it.Path, err)
		}
	}
	for _, it := range res.Skills {
		if it.Kind != KindSkill {
			t.Errorf("skill %s Kind = %v, want KindSkill", it.Name, it.Kind)
		}
		if info, err := os.Stat(it.Path); err != nil || !info.IsDir() {
			t.Errorf("skill %s Path = %q, which is not a directory (err %v)", it.Name, it.Path, err)
		}
	}
}

// TestPlanTarget_MissingTargetIsEmptyNotError pins the divergence from Plan:
// a target that does not exist yet has nothing to remove, which is not an
// error the way a missing source is.
func TestPlanTarget_MissingTargetIsEmptyNotError(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "not-created-yet")

	res, err := PlanTarget(Options{Target: dst})
	if err != nil {
		t.Fatalf("PlanTarget on a nonexistent target: %v", err)
	}
	if len(res.Agents) != 0 || len(res.Skills) != 0 {
		t.Errorf("PlanTarget = %+v, want empty lists", res)
	}
}

// TestPlanTarget_EmptyTargetDirsAreEmptyNotError covers the other half of the
// divergence: Plan rejects a source shipping neither agents/ nor skills/,
// but a target with neither is just a fresh target.
func TestPlanTarget_EmptyTargetDirsAreEmptyNotError(t *testing.T) {
	dst := t.TempDir()

	res, err := PlanTarget(Options{Target: dst})
	if err != nil {
		t.Fatalf("PlanTarget on a target with neither agents/ nor skills/: %v", err)
	}
	if len(res.Agents) != 0 || len(res.Skills) != 0 {
		t.Errorf("PlanTarget = %+v, want empty lists", res)
	}
}

func TestPlanTarget_ErrorEmptyTargetOption(t *testing.T) {
	_, err := PlanTarget(Options{})
	if err == nil {
		t.Fatal("PlanTarget with an empty Target returned nil error, want an error")
	}
	if !strings.Contains(err.Error(), "Target") {
		t.Errorf("error %q does not name Target as the problem", err.Error())
	}
}

func TestPlanTarget_OnlyAgents(t *testing.T) {
	dst := makeTargetTree(t)

	res, err := PlanTarget(Options{Target: dst, OnlyAgents: true})
	if err != nil {
		t.Fatalf("PlanTarget: %v", err)
	}
	if len(res.Agents) != len(targetAgents) {
		t.Errorf("agents = %v, want all %d", targetItemNames(res.Agents), len(targetAgents))
	}
	if len(res.Skills) != 0 {
		t.Errorf("skills = %v, want none with --agents", targetItemNames(res.Skills))
	}
}

func TestPlanTarget_OnlySkills(t *testing.T) {
	dst := makeTargetTree(t)

	res, err := PlanTarget(Options{Target: dst, OnlySkills: true})
	if err != nil {
		t.Fatalf("PlanTarget: %v", err)
	}
	if len(res.Skills) != len(targetSkillFiles) {
		t.Errorf("skills = %v, want all %d", targetItemNames(res.Skills), len(targetSkillFiles))
	}
	if len(res.Agents) != 0 {
		t.Errorf("agents = %v, want none with --skills", targetItemNames(res.Agents))
	}
}

func TestPlanTarget_UninstallNarrowsToNamedItems(t *testing.T) {
	dst := makeTargetTree(t)

	res, err := PlanTarget(Options{Target: dst, OnlyAgents: true, Uninstall: []string{"scout", "worker"}})
	if err != nil {
		t.Fatalf("PlanTarget: %v", err)
	}
	// The bare name is what the user types: "scout", never "scout.md".
	want := []string{"scout.md", "worker.md"}
	if got := targetItemNames(res.Agents); !slicesEqual(got, want) {
		t.Errorf("agents = %v, want %v", got, want)
	}
	if len(res.Skills) != 0 {
		t.Errorf("skills = %v, want none", targetItemNames(res.Skills))
	}
}

func TestPlanTarget_ErrorUninstallMatchesNothing(t *testing.T) {
	dst := makeTargetTree(t)

	_, err := PlanTarget(Options{Target: dst, OnlySkills: true, Uninstall: []string{"research", "nope"}})
	if err == nil {
		t.Fatal("PlanTarget with an unmatched --uninstall name returned nil error, want an error")
	}
	for _, want := range []string{"nope", "skills", "nothing was removed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err.Error(), want)
		}
	}
	// The name that did match must not be blamed.
	if strings.Contains(err.Error(), `"research"`) {
		t.Errorf("error %q blames %q, which does exist in the target", err.Error(), "research")
	}
}

func TestPlanTarget_TargetSymlinkResolved(t *testing.T) {
	real := makeTargetTree(t)
	link := filepath.Join(t.TempDir(), "link")
	mustSymlink(t, real, link)

	res, err := PlanTarget(Options{Target: link})
	if err != nil {
		t.Fatalf("PlanTarget: %v", err)
	}

	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", real, err)
	}
	scout, ok := findTargetItem(res.Agents, "scout.md")
	if !ok {
		t.Fatalf("scout.md missing from %v", targetItemNames(res.Agents))
	}
	want := filepath.Join(resolved, "agents", "scout.md")
	if scout.Path != want {
		t.Errorf("scout.md Path = %q, want the symlink-resolved %q", scout.Path, want)
	}
}

func TestRemoveItem_RemovesAgentFile(t *testing.T) {
	dst := makeTargetTree(t)
	path := filepath.Join(dst, "agents", "scout.md")

	if err := RemoveItem(TargetItem{Kind: KindAgent, Name: "scout.md", Path: path}); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("agent still present at %s (err %v)", path, err)
	}
	// Its neighbours are untouched.
	if _, err := os.Stat(filepath.Join(dst, "agents", "worker.md")); err != nil {
		t.Errorf("removing scout.md also disturbed worker.md: %v", err)
	}
}

// TestRemoveItem_RemovesWholeSkillDirIncludingUntrackedFiles pins the
// decision that a skill is one unit: a file this tool never installed goes
// with it rather than being preserved.
func TestRemoveItem_RemovesWholeSkillDirIncludingUntrackedFiles(t *testing.T) {
	dst := makeTargetTree(t)
	path := filepath.Join(dst, "skills", "research")
	mustWriteFile(t, filepath.Join(path, "notes.txt"), "hand-added, never installed by this tool")
	mustWriteFile(t, filepath.Join(path, "scratch", "deep.md"), "nested and hand-added too")

	if err := RemoveItem(TargetItem{Kind: KindSkill, Name: "research", Path: path}); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("skill directory still present at %s (err %v)", path, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "skills", "make-plan", "SKILL.md")); err != nil {
		t.Errorf("removing research also disturbed make-plan: %v", err)
	}
}

// TestRemoveItem_LeavesParentDirInPlace pins the "empty directory stays"
// rule: removing the last agent must not take <target>/agents with it.
func TestRemoveItem_LeavesParentDirInPlace(t *testing.T) {
	dst := t.TempDir()
	mustWriteFile(t, filepath.Join(dst, "agents", "only.md"), "the last one")

	if err := RemoveItem(TargetItem{
		Kind: KindAgent, Name: "only.md", Path: filepath.Join(dst, "agents", "only.md"),
	}); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}

	info, err := os.Stat(filepath.Join(dst, "agents"))
	if err != nil {
		t.Fatalf("%s/agents is gone after removing its last item: %v", dst, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s/agents is no longer a directory", dst)
	}
	entries, err := os.ReadDir(filepath.Join(dst, "agents"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("agents/ = %v, want empty", entries)
	}
}

// TestRemoveItem_MissingAgentErrors pins the os.Remove choice for agents:
// unlike RemoveAll, it reports a path that is not there.
func TestRemoveItem_MissingAgentErrors(t *testing.T) {
	dst := t.TempDir()
	path := filepath.Join(dst, "agents", "ghost.md")

	err := RemoveItem(TargetItem{Kind: KindAgent, Name: "ghost.md", Path: path})
	if err == nil {
		t.Fatal("RemoveItem on a nonexistent agent returned nil error, want an error")
	}
	if !strings.Contains(err.Error(), "ghost.md") {
		t.Errorf("error %q does not name the item", err.Error())
	}
}

func TestRemoveItem_UnknownKindErrors(t *testing.T) {
	err := RemoveItem(TargetItem{Kind: Kind(42), Name: "weird", Path: filepath.Join(t.TempDir(), "weird")})
	if err == nil {
		t.Fatal("RemoveItem with an unknown Kind returned nil error, want an error")
	}
	if !strings.Contains(err.Error(), "unknown item kind") {
		t.Errorf("error %q does not report an unknown kind", err.Error())
	}
}

// TestRemoveItem_SymlinkedSkillRemovesLinkNotTarget pins the documented
// consequence of RemoveAll not following symlinks: the link goes, whatever
// it pointed at stays.
func TestRemoveItem_SymlinkedSkillRemovesLinkNotTarget(t *testing.T) {
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	mustWriteFile(t, filepath.Join(elsewhere, "SKILL.md"), "lives outside the target")

	dst := t.TempDir()
	link := filepath.Join(dst, "skills", "linked")
	mustSymlink(t, elsewhere, link)

	if err := RemoveItem(TargetItem{Kind: KindSkill, Name: "linked", Path: link}); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if _, err := os.Lstat(link); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("symlink still present at %s (err %v)", link, err)
	}
	if _, err := os.Stat(filepath.Join(elsewhere, "SKILL.md")); err != nil {
		t.Errorf("the pointed-at directory was removed along with the link: %v", err)
	}
}

// TestRemove_StopsAtFirstError pins Remove's abort-on-first-error policy:
// agents are removed before skills, so a failing agent must leave the skills
// alone rather than the loop pressing on.
func TestRemove_StopsAtFirstError(t *testing.T) {
	dst := makeTargetTree(t)
	skillPath := filepath.Join(dst, "skills", "research")

	res := TargetResult{
		Agents: []TargetItem{
			{Kind: KindAgent, Name: "ghost.md", Path: filepath.Join(dst, "agents", "ghost.md")},
			{Kind: KindAgent, Name: "scout.md", Path: filepath.Join(dst, "agents", "scout.md")},
		},
		Skills: []TargetItem{
			{Kind: KindSkill, Name: "research", Path: skillPath},
		},
	}

	if err := Remove(res); err == nil {
		t.Fatal("Remove returned nil error, want the first item's failure")
	}
	if _, err := os.Stat(filepath.Join(dst, "agents", "scout.md")); err != nil {
		t.Errorf("Remove pressed on past the first error and removed scout.md: %v", err)
	}
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("Remove pressed on past the first error and removed the skill: %v", err)
	}
}

func TestRunUninstall_RemovesNamedItemsOnly(t *testing.T) {
	dst := makeTargetTree(t)

	res, err := RunUninstall(Options{Target: dst, OnlySkills: true, Uninstall: []string{"stale"}})
	if err != nil {
		t.Fatalf("RunUninstall: %v", err)
	}
	if got := targetItemNames(res.Skills); !slicesEqual(got, []string{"stale"}) {
		t.Errorf("removed skills = %v, want [stale]", got)
	}
	if len(res.Agents) != 0 {
		t.Errorf("removed agents = %v, want none", targetItemNames(res.Agents))
	}

	if _, err := os.Lstat(filepath.Join(dst, "skills", "stale")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("skill stale still present (err %v)", err)
	}
	// Every other item in the target survives.
	for name := range targetAgents {
		if _, err := os.Stat(filepath.Join(dst, "agents", name)); err != nil {
			t.Errorf("agent %s was removed but was never named: %v", name, err)
		}
	}
	for name := range targetSkillFiles {
		if name == "stale" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dst, "skills", name, "SKILL.md")); err != nil {
			t.Errorf("skill %s was removed but was never named: %v", name, err)
		}
	}
}

func TestRunUninstall_DryRunTouchesNothing(t *testing.T) {
	dst := makeTargetTree(t)
	before := snapshotTree(t, dst)

	res, err := RunUninstall(Options{
		Target: dst, OnlySkills: true, DryRun: true, Uninstall: []string{"research", "stale"},
	})
	if err != nil {
		t.Fatalf("RunUninstall: %v", err)
	}

	after := snapshotTree(t, dst)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("dry run touched the target.\nbefore: %+v\nafter:  %+v", before, after)
	}
	// The report still describes what would have gone.
	want := []string{"research", "stale"}
	if got := targetItemNames(res.Skills); !slicesEqual(got, want) {
		t.Errorf("dry-run result skills = %v, want %v", got, want)
	}
}

// TestRunUninstall_PartialNameFailureRemovesNothing is the design's central
// safety promise: a typo in the third name never leaves the first two
// already gone.
func TestRunUninstall_PartialNameFailureRemovesNothing(t *testing.T) {
	dst := makeTargetTree(t)
	before := snapshotTree(t, dst)

	_, err := RunUninstall(Options{
		Target: dst, OnlySkills: true, Uninstall: []string{"research", "stale", "typo"},
	})
	if err == nil {
		t.Fatal("RunUninstall with one bad name returned nil error, want an error")
	}
	if !strings.Contains(err.Error(), "typo") {
		t.Errorf("error %q does not name the mistyped name", err.Error())
	}

	after := snapshotTree(t, dst)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("a run that failed name verification still removed something.\nbefore: %+v\nafter:  %+v", before, after)
	}
}

// TestRunUninstall_NeverTouchesAnythingOutsideAgentsAndSkills guards the
// promise in the usage text: nothing outside the target's agents/ and
// skills/ is ever read, written, or removed.
func TestRunUninstall_NeverTouchesAnythingOutsideAgentsAndSkills(t *testing.T) {
	dst := makeTargetTree(t)
	mustWriteFile(t, filepath.Join(dst, "settings.json"), `{"kept": true}`)
	mustWriteFile(t, filepath.Join(dst, "projects", "x", "notes.md"), "kept too")

	if _, err := RunUninstall(Options{
		Target: dst, OnlyAgents: true, Uninstall: []string{"scout", "stale", "worker"},
	}); err != nil {
		t.Fatalf("RunUninstall: %v", err)
	}

	if got, err := os.ReadFile(filepath.Join(dst, "settings.json")); err != nil {
		t.Errorf("settings.json did not survive: %v", err)
	} else if string(got) != `{"kept": true}` {
		t.Errorf("settings.json content = %q, want it unchanged", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "projects", "x", "notes.md")); err != nil {
		t.Errorf("projects/x/notes.md did not survive: %v", err)
	}
	// And the skills the run was not scoped to are all still there.
	for name := range targetSkillFiles {
		if _, err := os.Stat(filepath.Join(dst, "skills", name, "SKILL.md")); err != nil {
			t.Errorf("skill %s was removed by an --agents-scoped run: %v", name, err)
		}
	}
}

// TestPlanInteractive_MarksInSource is the cross-reference the -i picker
// runs on: every target item the source also ships is flagged, and the ones
// only the target has are not.
func TestPlanInteractive_MarksInSource(t *testing.T) {
	src := makeSourceTree(t)
	dst := makeTargetTree(t)

	res, tgt, err := PlanInteractive(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("PlanInteractive: %v", err)
	}

	// The source side must be exactly what Plan alone produces.
	if len(res.Agents) != len(sourceAgents) || len(res.Skills) != len(sourceSkillFiles) {
		t.Fatalf("source side = %d agents, %d skills; want %d and %d",
			len(res.Agents), len(res.Skills), len(sourceAgents), len(sourceSkillFiles))
	}

	wantAgents := map[string]bool{"scout.md": true, "worker.md": true, "stale.md": false}
	for name, want := range wantAgents {
		it, ok := findTargetItem(tgt.Agents, name)
		if !ok {
			t.Fatalf("target agent %q missing from the plan", name)
		}
		if it.InSource != want {
			t.Errorf("target agent %q InSource = %v, want %v", name, it.InSource, want)
		}
		wantGroup := GroupOnlyInTarget
		if want {
			wantGroup = GroupAlsoInSource
		}
		if got := it.Group(); got != wantGroup {
			t.Errorf("target agent %q Group() = %v, want %v", name, got, wantGroup)
		}
	}

	wantSkills := map[string]bool{"research": true, "make-plan": true, "stale": false}
	for name, want := range wantSkills {
		it, ok := findTargetItem(tgt.Skills, name)
		if !ok {
			t.Fatalf("target skill %q missing from the plan", name)
		}
		if it.InSource != want {
			t.Errorf("target skill %q InSource = %v, want %v", name, it.InSource, want)
		}
	}
}

// TestPlanInteractive_InSourceIsPerKind pins the matching to the {Kind,
// Name} pair rather than the name alone: a skill directory named
// "alpha.md" must not be flagged just because the source ships an *agent*
// by that name, and vice versa.
func TestPlanInteractive_InSourceIsPerKind(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "agents", "alpha.md"), "alpha agent")
	mustWriteFile(t, filepath.Join(src, "skills", "beta", "SKILL.md"), "beta skill")

	dst := t.TempDir()
	// A skill directory whose name collides with the source's agent name,
	// and an agent file whose name collides with the source's skill name.
	mustWriteFile(t, filepath.Join(dst, "skills", "alpha.md", "SKILL.md"), "not the same thing")
	mustWriteFile(t, filepath.Join(dst, "agents", "beta.md"), "not the same thing either")

	_, tgt, err := PlanInteractive(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("PlanInteractive: %v", err)
	}

	skill, ok := findTargetItem(tgt.Skills, "alpha.md")
	if !ok {
		t.Fatalf("target skill %q missing from the plan", "alpha.md")
	}
	if skill.InSource {
		t.Errorf(`skill "alpha.md" InSource = true; the source ships an agent by that name, not a skill`)
	}

	agent, ok := findTargetItem(tgt.Agents, "beta.md")
	if !ok {
		t.Fatalf("target agent %q missing from the plan", "beta.md")
	}
	if agent.InSource {
		t.Errorf(`agent "beta.md" InSource = true; the source ships no agent by that name`)
	}
}

// TestPlanInteractive_PropagatesPlanError checks that -i still gets the
// full source validation: a bad source fails the whole call rather than
// quietly yielding an empty install side next to a usable uninstall side.
func TestPlanInteractive_PropagatesPlanError(t *testing.T) {
	dst := makeTargetTree(t)

	for _, tc := range []struct {
		name string
		opts Options
	}{
		{"empty source", Options{Target: dst}},
		{"source does not exist", Options{Source: filepath.Join(t.TempDir(), "nope"), Target: dst}},
		{"source ships neither agents/ nor skills/", Options{Source: t.TempDir(), Target: dst}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, tgt, err := PlanInteractive(tc.opts)
			if err == nil {
				t.Fatalf("PlanInteractive(%+v) = nil error, want an error", tc.opts)
			}
			if len(res.Agents) != 0 || len(res.Skills) != 0 {
				t.Errorf("Result on error = %+v, want zero value", res)
			}
			if len(tgt.Agents) != 0 || len(tgt.Skills) != 0 {
				t.Errorf("TargetResult on error = %+v, want zero value", tgt)
			}
		})
	}
}

// TestPlanInteractive_EmptyTargetGivesEmptyTargetResult covers the
// first-run case: a target that does not exist yet has nothing to remove,
// which is not an error, and the install side still plans normally.
func TestPlanInteractive_EmptyTargetGivesEmptyTargetResult(t *testing.T) {
	src := makeSourceTree(t)
	dst := filepath.Join(t.TempDir(), "not-created-yet")

	res, tgt, err := PlanInteractive(Options{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("PlanInteractive against a fresh target: %v", err)
	}
	if len(tgt.Agents) != 0 || len(tgt.Skills) != 0 {
		t.Errorf("TargetResult = %+v, want empty lists", tgt)
	}
	if len(res.Agents) != len(sourceAgents) || len(res.Skills) != len(sourceSkillFiles) {
		t.Fatalf("source side = %d agents, %d skills; want %d and %d",
			len(res.Agents), len(res.Skills), len(sourceAgents), len(sourceSkillFiles))
	}
	for _, it := range allItems(res) {
		if it.Exists {
			t.Errorf("item %q Exists = true against a fresh target", it.Name)
		}
	}
}
