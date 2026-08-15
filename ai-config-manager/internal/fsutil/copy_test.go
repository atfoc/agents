package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assertNoTmpLitter checks that no ".acm-*.tmp" entries were left behind in
// dir, which would indicate a failure path forgot to clean up after itself.
func assertNoTmpLitter(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".acm-*.tmp"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no .acm-*.tmp litter in %s, found: %v", dir, matches)
	}
}

func TestReplaceFile(t *testing.T) {
	t.Run("creates dst when absent", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, "fresh content", 0o644)

		if err := ReplaceFile(src, dst); err != nil {
			t.Fatalf("ReplaceFile: %v", err)
		}

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if !same {
			t.Fatal("expected dst to match src after ReplaceFile")
		}
		assertNoTmpLitter(t, dir)
	})

	t.Run("overwrites existing dst with different content", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, "new content", 0o644)
		writeFile(t, dst, "stale content that is a different length", 0o644)

		if err := ReplaceFile(src, dst); err != nil {
			t.Fatalf("ReplaceFile: %v", err)
		}

		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("read dst: %v", err)
		}
		if string(got) != "new content" {
			t.Fatalf("expected dst content %q, got %q", "new content", got)
		}
		assertNoTmpLitter(t, dir)
	})

	t.Run("creates missing parent directories", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "a", "b", "c", "dst.txt")
		writeFile(t, src, "nested", 0o644)

		if err := ReplaceFile(src, dst); err != nil {
			t.Fatalf("ReplaceFile: %v", err)
		}

		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("read dst: %v", err)
		}
		if string(got) != "nested" {
			t.Fatalf("expected dst content %q, got %q", "nested", got)
		}
	})

	t.Run("preserves source permission bits", func(t *testing.T) {
		for _, perm := range []os.FileMode{0o600, 0o644} {
			perm := perm
			t.Run(perm.String(), func(t *testing.T) {
				dir := t.TempDir()
				src := filepath.Join(dir, "src.txt")
				dst := filepath.Join(dir, "dst.txt")
				writeFile(t, src, "content", perm)

				if err := ReplaceFile(src, dst); err != nil {
					t.Fatalf("ReplaceFile: %v", err)
				}

				info, err := os.Stat(dst)
				if err != nil {
					t.Fatalf("stat dst: %v", err)
				}
				if info.Mode().Perm() != perm {
					t.Fatalf("expected dst perm %v, got %v", perm, info.Mode().Perm())
				}
			})
		}
	})

	t.Run("replaces a dst that is currently a directory", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst-was-dir")
		writeFile(t, src, "now a file", 0o644)
		writeFile(t, filepath.Join(dst, "inner.txt"), "stale", 0o644)

		if err := ReplaceFile(src, dst); err != nil {
			t.Fatalf("ReplaceFile: %v", err)
		}

		info, err := os.Lstat(dst)
		if err != nil {
			t.Fatalf("lstat dst: %v", err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("expected dst to be a regular file, got mode %v", info.Mode())
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("read dst: %v", err)
		}
		if string(got) != "now a file" {
			t.Fatalf("expected dst content %q, got %q", "now a file", got)
		}
	})

	t.Run("replaces a dst that is currently a symlink", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		target := filepath.Join(dir, "link-target.txt")
		dst := filepath.Join(dir, "dst-was-link")
		writeFile(t, src, "replacement content", 0o644)
		writeFile(t, target, "original target content", 0o644)
		if err := os.Symlink(target, dst); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		if err := ReplaceFile(src, dst); err != nil {
			t.Fatalf("ReplaceFile: %v", err)
		}

		info, err := os.Lstat(dst)
		if err != nil {
			t.Fatalf("lstat dst: %v", err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			t.Fatal("expected dst to no longer be a symlink")
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("expected dst to be a regular file, got mode %v", info.Mode())
		}

		targetContent, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read original link target: %v", err)
		}
		if string(targetContent) != "original target content" {
			t.Fatalf("expected the old symlink's target to be left untouched, got %q", targetContent)
		}
		assertNoTmpLitter(t, dir)
	})
}

func TestReplaceTree(t *testing.T) {
	t.Run("creates dst when absent", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"SKILL.md": "# a skill"})

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if !same {
			t.Fatal("expected dst tree to match src after ReplaceTree")
		}
	})

	t.Run("replaces existing dst and removes stale files", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"keep.txt": "kept content"})
		mkTree(t, dst, map[string]string{"keep.txt": "old content", "stale.txt": "should be removed"})

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dst, "stale.txt")); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("expected stale.txt to be gone, stat returned: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dst, "keep.txt"))
		if err != nil {
			t.Fatalf("read keep.txt: %v", err)
		}
		if string(got) != "kept content" {
			t.Fatalf("expected keep.txt content %q, got %q", "kept content", got)
		}
	})

	t.Run("copies nested subdirectories", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{
			"SKILL.md":            "# skill",
			"scripts/run.sh":      "#!/bin/sh\n",
			"scripts/lib/util.py": "print('util')",
		})

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if !same {
			t.Fatal("expected nested subdirectories to be copied faithfully")
		}
	})

	t.Run("preserves file permission bits", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		writeFile(t, filepath.Join(src, "exec.sh"), "#!/bin/sh\n", 0o755)
		writeFile(t, filepath.Join(src, "data.txt"), "data", 0o640)

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		execInfo, err := os.Stat(filepath.Join(dst, "exec.sh"))
		if err != nil {
			t.Fatalf("stat exec.sh: %v", err)
		}
		if execInfo.Mode().Perm() != 0o755 {
			t.Fatalf("expected exec.sh perm 0755, got %v", execInfo.Mode().Perm())
		}

		dataInfo, err := os.Stat(filepath.Join(dst, "data.txt"))
		if err != nil {
			t.Fatalf("stat data.txt: %v", err)
		}
		if dataInfo.Mode().Perm() != 0o640 {
			t.Fatalf("expected data.txt perm 0640, got %v", dataInfo.Mode().Perm())
		}
	})

	t.Run("sets destination root perm bits from source", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		if err := os.Mkdir(src, 0o750); err != nil {
			t.Fatalf("mkdir src: %v", err)
		}
		if err := os.Chmod(src, 0o750); err != nil {
			t.Fatalf("chmod src: %v", err)
		}
		writeFile(t, filepath.Join(src, "a.txt"), "content", 0o644)

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat dst: %v", err)
		}
		if info.Mode().Perm() != 0o750 {
			t.Fatalf("expected dst root perm 0750 (not MkdirTemp's default 0700), got %v", info.Mode().Perm())
		}
	})

	t.Run("leaves no staging directory behind", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "content"})

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}
		assertNoTmpLitter(t, base)
	})

	t.Run("replaces a dst that is currently a regular file", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "content"})
		writeFile(t, dst, "dst used to be a plain file", 0o644)

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat dst: %v", err)
		}
		if !info.IsDir() {
			t.Fatalf("expected dst to become a directory, got mode %v", info.Mode())
		}
		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if !same {
			t.Fatal("expected dst tree to match src after replacing a regular file")
		}
	})
}

func TestCopyTree_Symlinks(t *testing.T) {
	t.Run("symlink to a file becomes a real regular file", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		target := filepath.Join(base, "outside-target.txt")
		writeFile(t, target, "resolved content", 0o644)
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatalf("mkdir src: %v", err)
		}
		if err := os.Symlink(target, filepath.Join(src, "link.txt")); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		if err := ReplaceTree(src, dst); err != nil {
			t.Fatalf("ReplaceTree: %v", err)
		}

		info, err := os.Lstat(filepath.Join(dst, "link.txt"))
		if err != nil {
			t.Fatalf("lstat dst/link.txt: %v", err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			t.Fatal("expected dst/link.txt to be a real file, not a symlink")
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("expected dst/link.txt to be a regular file, got mode %v", info.Mode())
		}
		got, err := os.ReadFile(filepath.Join(dst, "link.txt"))
		if err != nil {
			t.Fatalf("read dst/link.txt: %v", err)
		}
		if string(got) != "resolved content" {
			t.Fatalf("expected content %q, got %q", "resolved content", got)
		}
	})

	t.Run("symlink to a directory is an error", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"realdir/inner.txt": "content"})
		linkPath := filepath.Join(src, "link-to-dir")
		if err := os.Symlink(filepath.Join(src, "realdir"), linkPath); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		err := ReplaceTree(src, dst)
		if err == nil {
			t.Fatal("expected an error for a symlinked directory in src")
		}
		if !strings.Contains(err.Error(), linkPath) {
			t.Fatalf("expected error to mention %q, got: %v", linkPath, err)
		}
		assertNoTmpLitter(t, base)
	})
}
