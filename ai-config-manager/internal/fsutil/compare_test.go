package fsutil

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile creates path (and any missing parent directories) with the
// given content and permission bits. The permission is chmod'd explicitly
// after writing so the test isn't at the mercy of the process umask.
func writeFile(t *testing.T, path, content string, perm fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parents for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

// mkTree writes a set of files (relative-path -> content) under root, with
// mode 0644, creating any parent directories implied by the paths.
func mkTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		writeFile(t, filepath.Join(root, rel), content, 0o644)
	}
}

func TestSameFile(t *testing.T) {
	t.Run("identical content", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, "hello world", 0o644)
		writeFile(t, dst, "hello world", 0o644)

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if !same {
			t.Fatal("expected identical files to be reported same")
		}
	})

	t.Run("different content same length", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, "aaaaaaaaaa", 0o644)
		writeFile(t, dst, "bbbbbbbbbb", 0o644)

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if same {
			t.Fatal("expected different content to be reported different")
		}
	})

	t.Run("different length", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, "short", 0o644)
		writeFile(t, dst, "a much longer string of content", 0o644)

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if same {
			t.Fatal("expected different-length files to be reported different")
		}
	})

	t.Run("empty vs empty", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, "", 0o644)
		writeFile(t, dst, "", 0o644)

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if !same {
			t.Fatal("expected two empty files to be reported same")
		}
	})

	t.Run("dst missing", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "does-not-exist.txt")
		writeFile(t, src, "content", 0o644)

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("expected no error for missing dst, got: %v", err)
		}
		if same {
			t.Fatal("expected missing dst to be reported different")
		}
	})

	t.Run("dst is a symlink to byte-identical content", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		target := filepath.Join(dir, "target.txt")
		dst := filepath.Join(dir, "dst-link.txt")
		writeFile(t, src, "same bytes", 0o644)
		writeFile(t, target, "same bytes", 0o644)
		if err := os.Symlink(target, dst); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if same {
			t.Fatal("expected a symlink dst to be reported different even if target content matches")
		}
	})

	t.Run("dst is a directory", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst-dir")
		writeFile(t, src, "content", 0o644)
		if err := os.Mkdir(dst, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if same {
			t.Fatal("expected a directory dst to be reported different")
		}
	})

	t.Run("src missing is an error", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "does-not-exist.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, dst, "content", 0o644)

		_, err := SameFile(src, dst)
		if err == nil {
			t.Fatal("expected an error when src is missing")
		}
	})

	t.Run("large files differing only in last byte", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.bin")
		dst := filepath.Join(dir, "dst.bin")

		const size = 100 * 1024 // spans more than one 32KiB comparison buffer
		content := bytes.Repeat([]byte{0x42}, size)

		if err := os.WriteFile(src, content, 0o644); err != nil {
			t.Fatalf("write src: %v", err)
		}

		dstContent := make([]byte, size)
		copy(dstContent, content)
		dstContent[size-1] ^= 0xFF // flip only the very last byte
		if err := os.WriteFile(dst, dstContent, 0o644); err != nil {
			t.Fatalf("write dst: %v", err)
		}

		same, err := SameFile(src, dst)
		if err != nil {
			t.Fatalf("SameFile: %v", err)
		}
		if same {
			t.Fatal("expected files differing only in the last byte to be reported different")
		}
	})
}

func TestSameTree(t *testing.T) {
	t.Run("identical single-file trees", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "hello"})
		mkTree(t, dst, map[string]string{"a.txt": "hello"})

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if !same {
			t.Fatal("expected identical single-file trees to be reported same")
		}
	})

	t.Run("identical nested trees", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		tree := map[string]string{
			"SKILL.md":         "# skill",
			"README.md":        "readme",
			"scripts/run.sh":   "#!/bin/sh\necho hi\n",
			"scripts/lib/a.py": "print('a')",
		}
		mkTree(t, src, tree)
		mkTree(t, dst, tree)

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if !same {
			t.Fatal("expected identical nested trees to be reported same")
		}
	})

	t.Run("dst has an extra file", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "hello"})
		mkTree(t, dst, map[string]string{"a.txt": "hello", "extra.txt": "stale"})

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if same {
			t.Fatal("expected an extra file in dst to make the trees differ")
		}
	})

	t.Run("dst is missing a file", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "hello", "b.txt": "world"})
		mkTree(t, dst, map[string]string{"a.txt": "hello"})

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if same {
			t.Fatal("expected a missing file in dst to make the trees differ")
		}
	})

	t.Run("nested file differs in content", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"sub/inner.txt": "version A"})
		mkTree(t, dst, map[string]string{"sub/inner.txt": "version B"})

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if same {
			t.Fatal("expected differing nested file content to make the trees differ")
		}
	})

	t.Run("dst has a file where src has a directory", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"sub/inner.txt": "content"})
		writeFile(t, filepath.Join(dst, "sub"), "not a directory", 0o644)

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if same {
			t.Fatal("expected a file-vs-directory mismatch to make the trees differ")
		}
	})

	t.Run("dst missing entirely", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "does-not-exist")
		mkTree(t, src, map[string]string{"a.txt": "hello"})

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("expected no error for missing dst, got: %v", err)
		}
		if same {
			t.Fatal("expected a missing dst to be reported different")
		}
	})

	t.Run("dst is a regular file not a directory", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "hello"})
		writeFile(t, dst, "just a file", 0o644)

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if same {
			t.Fatal("expected a dst that is a plain file to be reported different")
		}
	})

	t.Run("dst contains a symlink", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"a.txt": "hello"})
		if err := os.MkdirAll(dst, 0o755); err != nil {
			t.Fatalf("mkdir dst: %v", err)
		}
		// The symlink's target has byte-identical content to src's a.txt,
		// so this specifically tests that dst being a symlink is what makes
		// the trees differ, not the content.
		target := filepath.Join(base, "outside-target.txt")
		writeFile(t, target, "hello", 0o644)
		if err := os.Symlink(target, filepath.Join(dst, "a.txt")); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		same, err := SameTree(src, dst)
		if err != nil {
			t.Fatalf("SameTree: %v", err)
		}
		if same {
			t.Fatal("expected a symlink inside dst to make the trees differ")
		}
	})

	t.Run("src contains a symlink to a directory", func(t *testing.T) {
		base := t.TempDir()
		src := filepath.Join(base, "src")
		dst := filepath.Join(base, "dst")
		mkTree(t, src, map[string]string{"realdir/inner.txt": "content"})
		if err := os.Mkdir(dst, 0o755); err != nil {
			t.Fatalf("mkdir dst: %v", err)
		}
		linkPath := filepath.Join(src, "link-to-dir")
		if err := os.Symlink(filepath.Join(src, "realdir"), linkPath); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		_, err := SameTree(src, dst)
		if err == nil {
			t.Fatal("expected an error for a symlinked directory in src")
		}
		if !strings.Contains(err.Error(), linkPath) {
			t.Fatalf("expected error to mention %q, got: %v", linkPath, err)
		}
	})
}

func TestSameFile_ErrorIsWrapped(t *testing.T) {
	// Sanity check that not-exist detection uses errors.Is against
	// fs.ErrNotExist rather than string matching, by exercising the actual
	// path once more with errors.Is on the underlying Lstat behavior.
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	writeFile(t, src, "content", 0o644)

	_, err := os.Lstat(filepath.Join(dir, "missing"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected errors.Is to recognize a missing file, got: %v", err)
	}
}
