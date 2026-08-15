package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/atfoc/agents/ai-config-manager/internal/fsutil"
)

// Plan computes what a run would do — which agents and skills need writing
// and which already match the target — without touching the filesystem in
// any way that mutates it. This is what makes --dry-run trustworthy: dry-run
// is literally "call Plan and skip Apply", so the report it prints can never
// drift from what a real run would actually do.
func Plan(opts Options) (Result, error) {
	if opts.Source == "" {
		return Result{}, fmt.Errorf("installer: Options.Source is empty, no source directory given")
	}
	if opts.Target == "" {
		return Result{}, fmt.Errorf("installer: Options.Target is empty, no target directory given")
	}

	srcAbs, err := filepath.Abs(opts.Source)
	if err != nil {
		return Result{}, fmt.Errorf("resolve source %q: %w", opts.Source, err)
	}
	srcResolved, err := filepath.EvalSymlinks(srcAbs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Result{}, fmt.Errorf("source directory %q does not exist", opts.Source)
		}
		return Result{}, fmt.Errorf("resolve source %q: %w", opts.Source, err)
	}
	srcInfo, err := os.Stat(srcResolved)
	if err != nil {
		return Result{}, fmt.Errorf("stat source %q: %w", srcResolved, err)
	}
	if !srcInfo.IsDir() {
		return Result{}, fmt.Errorf("source %q is not a directory", srcResolved)
	}

	// Unlike the source, the target need not exist yet: a first-ever run
	// installs into a brand-new directory. resolveTarget copes with that by
	// resolving only the longest prefix that does exist.
	dstResolved, err := resolveTarget(opts.Target)
	if err != nil {
		return Result{}, fmt.Errorf("resolve target %q: %w", opts.Target, err)
	}

	// Both paths are now clean and absolute, with every symlink component
	// resolved, so a direct comparison and a separator-aware prefix check
	// are enough to catch the tool copying into its own input.
	if srcResolved == dstResolved {
		return Result{}, fmt.Errorf(
			"source %q and target %q both resolve to %q: they must be different directories",
			opts.Source, opts.Target, srcResolved)
	}
	if contains(srcResolved, dstResolved) {
		return Result{}, fmt.Errorf(
			"target %q is inside source %q: refusing to install into a subdirectory of the source",
			dstResolved, srcResolved)
	}
	if contains(dstResolved, srcResolved) {
		return Result{}, fmt.Errorf(
			"source %q is inside target %q: refusing to install from a subdirectory of the target",
			srcResolved, dstResolved)
	}

	// A source that ships neither agents/ nor skills/ has nothing at all to
	// install and is almost certainly the wrong directory. A single missing
	// subdirectory, though, is not an error: it just means that section is
	// empty (e.g. a source that only ships skills).
	_, agentsErr := os.Stat(filepath.Join(srcResolved, "agents"))
	_, skillsErr := os.Stat(filepath.Join(srcResolved, "skills"))
	if errors.Is(agentsErr, fs.ErrNotExist) && errors.Is(skillsErr, fs.ErrNotExist) {
		return Result{}, fmt.Errorf("source %q contains neither agents/ nor skills/", srcResolved)
	}

	agents, err := discoverAgents(srcResolved, dstResolved)
	if err != nil {
		return Result{}, fmt.Errorf("discover agents: %w", err)
	}
	skills, err := discoverSkills(srcResolved, dstResolved)
	if err != nil {
		return Result{}, fmt.Errorf("discover skills: %w", err)
	}

	for i := range agents {
		same, err := fsutil.SameFile(agents[i].Src, agents[i].Dst)
		if err != nil {
			return Result{}, fmt.Errorf("compare agent %q: %w", agents[i].Name, err)
		}
		if same {
			agents[i].Status = StatusUnchanged
		} else {
			agents[i].Status = StatusUpdated
		}
	}
	for i := range skills {
		same, err := fsutil.SameTree(skills[i].Src, skills[i].Dst)
		if err != nil {
			return Result{}, fmt.Errorf("compare skill %q: %w", skills[i].Name, err)
		}
		if same {
			skills[i].Status = StatusUnchanged
		} else {
			skills[i].Status = StatusUpdated
		}
	}

	return Result{Agents: agents, Skills: skills}, nil
}

// resolveTarget turns path into a clean, absolute, symlink-resolved path,
// even when path does not exist yet. It walks up to the longest prefix of
// path that does exist, resolves symlinks in that (existing) prefix only,
// and re-joins the non-existent tail onto it as plain text — there is
// nothing to resolve in path components that are not there yet.
func resolveTarget(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", path, err)
	}

	cur := abs
	for {
		// Lstat, not Stat: we only care whether something exists at cur, not
		// whether a symlink there points at something that does.
		if _, err := os.Lstat(cur); err == nil {
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("stat %s: %w", cur, err)
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			// Reached the root without finding anything that exists. This
			// should never happen in practice, but stops the loop from
			// spinning forever instead of relying on that assumption.
			break
		}
		cur = parent
	}

	resolved, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", cur, err)
	}

	tail, err := filepath.Rel(cur, abs)
	if err != nil {
		return "", fmt.Errorf("compute relative path from %s to %s: %w", cur, abs, err)
	}
	// filepath.Join cleans "." away on its own, so this also covers the case
	// where abs itself already existed (tail == ".") and resolved is the
	// whole answer.
	return filepath.Join(resolved, tail), nil
}

// contains reports whether child names a path strictly inside the directory
// tree rooted at parent. Both arguments must already be clean, absolute
// paths. A raw strings.HasPrefix(child, parent) would be wrong here: it
// would treat a mere sibling like "/tmp/foo-backup" as being inside
// "/tmp/foo" just because the strings share a prefix. Requiring the parent's
// separator right after the prefix rules that out.
func contains(parent, child string) bool {
	if parent == child {
		return false
	}
	return strings.HasPrefix(child, parent+string(filepath.Separator))
}

// Apply performs the writes that a Result describes: every item still
// StatusUpdated is written to its Dst; every StatusUnchanged item is left
// untouched. Skipping unchanged items is deliberate, not just an
// optimization — rewriting identical content would churn mtimes for no
// benefit, and it is what makes repeat runs of the tool cheap.
//
// Apply aborts on the first error it hits rather than pressing on to the
// remaining items: a write failure is almost always something structural
// like a permissions problem or a full disk, which the next item is not
// going to fix either. Partial application is an accepted consequence of
// that choice.
func Apply(res Result) error {
	for _, item := range res.Agents {
		if item.Status == StatusUnchanged {
			continue
		}
		if err := fsutil.ReplaceFile(item.Src, item.Dst); err != nil {
			return fmt.Errorf("install agent %q: %w", item.Name, err)
		}
	}
	for _, item := range res.Skills {
		if item.Status == StatusUnchanged {
			continue
		}
		if err := fsutil.ReplaceTree(item.Src, item.Dst); err != nil {
			return fmt.Errorf("install skill %q: %w", item.Name, err)
		}
	}
	return nil
}

// Run plans and, unless opts.DryRun is set, applies an installation. It
// never prints anything itself: the caller (main) is responsible for
// rendering the returned Result, so that Run stays testable as a pure
// function of Options in, (Result, error) out.
func Run(opts Options) (Result, error) {
	res, err := Plan(opts)
	if err != nil {
		return Result{}, err
	}

	if !opts.DryRun {
		if err := Apply(res); err != nil {
			// res is still returned here (not Result{}): even on a partial
			// failure, the caller can render what was planned so the user
			// can see how far the run got.
			return res, err
		}
	}

	return res, nil
}
