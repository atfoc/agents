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
// entry which is (or resolves to, through a symlink) a directory. A loose
// regular file directly under skills/ is skipped, whatever it is named.
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
		items = append(items, Item{
			Kind: KindSkill,
			Name: name,
			Src:  path,
			Dst:  filepath.Join(dstRoot, "skills", name),
		})
	}
	return items, nil
}
