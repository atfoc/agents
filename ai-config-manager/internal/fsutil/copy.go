package fsutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// tmpPattern is used for both the temp file staged by ReplaceFile and the
// temp directory staged by ReplaceTree, so that any crash-litter left
// behind is easy to recognize and clean up.
const tmpPattern = ".acm-*.tmp"

// ReplaceFile leaves dst holding a fresh copy of src, atomically replacing
// whatever was previously at dst (including a directory or a symlink).
func ReplaceFile(src, dst string) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", dst, err)
	}

	// Stat, not Lstat: a src that is itself a symlink is followed, matching
	// SameFile's convention, and its mode gives us the permission bits to
	// carry over to dst.
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	// Write into a temp file in dst's own directory first, so the final
	// rename is same-filesystem (and therefore atomic) and a crash mid-copy
	// never leaves a half-written file where a good one used to be.
	tmp, err := os.CreateTemp(dir, tmpPattern)
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpPath)
		}
	}()

	if err := copyFileContent(src, tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpPath, err)
	}

	if err := os.Chmod(tmpPath, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("chmod %s: %w", tmpPath, err)
	}

	// RemoveAll before Rename is required, not optional: if dst is currently
	// a directory a plain rename would fail, and if dst is a symlink a
	// rename can behave like writing through the link instead of replacing
	// it. RemoveAll handles both uniformly by clearing dst first.
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove %s: %w", dst, err)
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, dst, err)
	}

	ok = true
	return nil
}

// ReplaceTree leaves dst holding a fresh copy of the directory tree rooted
// at src, atomically replacing whatever was previously at dst. Anything
// that existed under dst but not under src is gone afterwards, since the
// tree is built fresh in an empty staging directory rather than merged in
// place.
func ReplaceTree(src, dst string) error {
	parent := filepath.Dir(dst)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", dst, err)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	// Stage the copy as a sibling of dst so the final rename stays on the
	// same filesystem. MkdirTemp creates it as 0700; that's corrected to
	// match src below, before it ever becomes visible at dst's path.
	stage, err := os.MkdirTemp(parent, tmpPattern)
	if err != nil {
		return fmt.Errorf("create staging dir in %s: %w", parent, err)
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(stage)
		}
	}()

	if err := copyTree(src, stage); err != nil {
		return err
	}

	if err := os.Chmod(stage, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("chmod %s: %w", stage, err)
	}

	// See ReplaceFile: RemoveAll first so a dst that is currently a plain
	// file, a symlink, or an existing directory is uniformly cleared before
	// the rename takes its place.
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove %s: %w", dst, err)
	}
	if err := os.Rename(stage, dst); err != nil {
		return fmt.Errorf("rename %s to %s: %w", stage, dst, err)
	}

	ok = true
	return nil
}

// copyTree recursively copies the contents of src into dst. dst must
// already exist. Symlinks in src are resolved rather than recreated: a
// symlink to a regular file is copied as that file's content, and a
// symlink to a directory is refused (following it risks a cycle, and
// copying it verbatim would leave a dangling link once the source tree
// moves or is removed).
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		target := filepath.Join(dst, rel)

		switch {
		case d.IsDir():
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("stat %s: %w", path, err)
			}
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			return nil

		case d.Type()&fs.ModeSymlink != 0:
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("stat %s: %w", path, err)
			}
			if info.IsDir() {
				return fmt.Errorf("symlinked directory in source is not supported: %s", path)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported file type at %s: %s", path, info.Mode())
			}
			return copyRegularFile(path, target, info.Mode().Perm())

		case d.Type().IsRegular():
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("stat %s: %w", path, err)
			}
			return copyRegularFile(path, target, info.Mode().Perm())

		default:
			mode := d.Type()
			if info, statErr := d.Info(); statErr == nil {
				mode = info.Mode()
			}
			return fmt.Errorf("unsupported file type at %s: %s", path, mode)
		}
	})
}

// copyRegularFile copies the content of the file at src to a new file at
// dst, creating or truncating it, and sets dst's permission bits to perm.
func copyRegularFile(src, dst string, perm fs.FileMode) error {
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}

	if err := copyFileContent(src, out); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dst, err)
	}

	// OpenFile's perm argument is filtered by umask, so set the permission
	// bits explicitly to make sure the copy actually matches the source.
	if err := os.Chmod(dst, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", dst, err)
	}
	return nil
}

// copyFileContent copies the content of the file at src into the already
// open file dstFile.
func copyFileContent(src string, dstFile *os.File) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	if _, err := io.Copy(dstFile, in); err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	return nil
}
