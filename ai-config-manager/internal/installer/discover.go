package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// discoverAgents lists the agents available under srcRoot/agents: every
// entry whose name ends in ".md" and which is (or resolves to, through a
// symlink) a regular file. Every other entry — directories, non-".md" files,
// non-regular files — is skipped rather than reported.
func discoverAgents(srcRoot, dstRoot string) ([]Item, error) {
	dir := filepath.Join(srcRoot, "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// A source containing only skills (no agents/ directory at all) is a
		// legal, ordinary case, not an error.
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	// os.ReadDir already returns entries sorted by filename, which is what
	// gives us the required "sorted by Name" result below. We don't rely on
	// that silently, though: nothing later assumes it, so if ReadDir's
	// contract ever changed this loop would still need an explicit sort.
	var items []Item
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		// Stat, not the DirEntry's own type, so that a symlink pointing at a
		// regular .md file is included rather than skipped.
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		items = append(items, Item{
			Kind: KindAgent,
			Name: name,
			Src:  path,
			Dst:  filepath.Join(dstRoot, "agents", name),
		})
	}
	return items, nil
}

// discoverSkills lists the skills available under srcRoot/skills: every
// entry which is (or resolves to, through a symlink) a directory that itself
// contains a SKILL.md (or resolves to, through a symlink, a regular) file.
// A loose regular file directly under skills/, or a directory without a
// SKILL.md, is skipped, whatever it is named.
func discoverSkills(srcRoot, dstRoot string) ([]Item, error) {
	dir := filepath.Join(srcRoot, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// A source containing only agents (no skills/ directory at all) is a
		// legal, ordinary case, not an error.
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	// See discoverAgents: ReadDir's sorted order satisfies the "sorted by
	// Name" requirement, but nothing here depends on that silently.
	var items []Item
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)
		// Stat, not the DirEntry's own type, so that a symlink pointing at a
		// directory is included rather than skipped.
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if !info.IsDir() {
			continue
		}
		skillMD, err := os.Stat(filepath.Join(path, "SKILL.md"))
		if err != nil {
			// No SKILL.md (or it's unreadable): this directory isn't a skill.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", filepath.Join(path, "SKILL.md"), err)
		}
		if !skillMD.Mode().IsRegular() {
			continue
		}
		items = append(items, Item{
			Kind: KindSkill,
			Name: name,
			Src:  path,
			Dst:  filepath.Join(dstRoot, "skills", name),
		})
	}
	return items, nil
}

// discoverTargetAgents lists the agents present in dstRoot/agents: every
// entry whose name ends in ".md" and which is (or resolves to, through a
// symlink) a regular file. Every other entry is skipped rather than
// reported.
func discoverTargetAgents(dstRoot string) ([]TargetItem, error) {
	dir := filepath.Join(dstRoot, "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// A target with no agents/ at all is an ordinary case — a fresh
		// target, or one that only ever held skills. Not an error.
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	// As in discoverAgents, ReadDir's sorted-by-filename order is what gives
	// the name-sorted result; nothing later depends on it silently.
	var items []TargetItem
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		// Stat, not the DirEntry's own type, so that a symlink pointing at a
		// regular .md file is included rather than skipped.
		info, err := os.Stat(path)
		if err != nil {
			// Divergence from discoverAgents, which makes this fatal. The
			// source is this tool's own curated input; the target is not
			// under our control, and a dangling symlink someone left in
			// ~/.claude/agents must not stop the whole run.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		items = append(items, TargetItem{
			Kind: KindAgent,
			Name: name,
			Path: path,
			// InSource is left false; only the planner can fill it in.
		})
	}
	return items, nil
}

// discoverTargetSkills lists the skills present in dstRoot/skills: every
// entry which is (or resolves to, through a symlink) a directory that itself
// contains a SKILL.md (or a symlink resolving to a regular one).
func discoverTargetSkills(dstRoot string) ([]TargetItem, error) {
	dir := filepath.Join(dstRoot, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// A target with no skills/ at all is an ordinary case, not an error.
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	// See discoverTargetAgents on the ordering.
	var items []TargetItem
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)
		// Stat, not the DirEntry's own type, so that a symlink pointing at a
		// directory is included rather than skipped.
		info, err := os.Stat(path)
		if err != nil {
			// A dangling symlink in the target is skipped, not fatal: see
			// discoverTargetAgents.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if !info.IsDir() {
			continue
		}
		skillMD, err := os.Stat(filepath.Join(path, "SKILL.md"))
		if err != nil {
			// A directory without a SKILL.md is not a skill and this tool
			// does not touch it. There are no "invalid" skill folders to
			// clean up; fixing them is not this tool's job.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", filepath.Join(path, "SKILL.md"), err)
		}
		if !skillMD.Mode().IsRegular() {
			continue
		}
		items = append(items, TargetItem{
			Kind: KindSkill,
			Name: name,
			Path: path,
		})
	}
	return items, nil
}
