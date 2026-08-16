package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExists(t *testing.T) {
	t.Run("path does not exist", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "does-not-exist")

		got, err := Exists(path)
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if got != false {
			t.Errorf("Exists(%s) = %v, want false", path, got)
		}
	})

	t.Run("regular file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "file.txt")
		writeFile(t, path, "content", 0o644)

		got, err := Exists(path)
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if got != true {
			t.Errorf("Exists(%s) = %v, want true", path, got)
		}
	})

	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "subdir")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		got, err := Exists(path)
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if got != true {
			t.Errorf("Exists(%s) = %v, want true", path, got)
		}
	})

	t.Run("dangling symlink", func(t *testing.T) {
		// This is the whole reason Exists uses Lstat rather than Stat: a
		// symlink whose target does not exist is still something present
		// at path, and Stat would follow the link and misreport it as
		// missing.
		dir := t.TempDir()
		target := filepath.Join(dir, "does-not-exist")
		link := filepath.Join(dir, "dangling-link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		got, err := Exists(link)
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if got != true {
			t.Errorf("Exists(%s) = %v, want true", link, got)
		}
	})

	t.Run("symlink to an existing file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.txt")
		link := filepath.Join(dir, "link")
		writeFile(t, target, "content", 0o644)
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		got, err := Exists(link)
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if got != true {
			t.Errorf("Exists(%s) = %v, want true", link, got)
		}
	})
}
