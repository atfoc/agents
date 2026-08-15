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
