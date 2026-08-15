// Package fsutil provides the filesystem primitives used to install agents
// and skills: comparing files and directory trees for equality, and
// atomically replacing them.
package fsutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// bufSize is the size of the two buffers used for byte-by-byte comparison.
// Reading in fixed-size chunks keeps memory use flat regardless of file size.
const bufSize = 32 * 1024

// SameFile reports whether dst exists, is a regular file, and is
// byte-for-byte identical to src.
func SameFile(src, dst string) (bool, error) {
	// Lstat, not Stat: a symlink (or any other non-regular file) at dst must
	// NOT be followed here. The installer replaces whatever is at dst with a
	// real file, so a symlink at dst always counts as "different" even if it
	// happens to point at content identical to src.
	dstInfo, err := os.Lstat(dst)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", dst, err)
	}
	if !dstInfo.Mode().IsRegular() {
		return false, nil
	}

	// src, in contrast, is allowed to be a symlink: Stat follows it.
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", src, err)
	}

	// Cheap short-circuit before touching file contents.
	if srcInfo.Size() != dstInfo.Size() {
		return false, nil
	}

	return sameContent(src, dst)
}

// SameTree reports whether the directory tree rooted at dst exactly matches
// the one rooted at src: same set of relative paths, same file-vs-dir kind
// at each path, and byte-identical content for every regular file.
func SameTree(src, dst string) (bool, error) {
	dstInfo, err := os.Lstat(dst)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", dst, err)
	}
	// A symlink at the dst root, or a dst root that isn't a directory at
	// all, is never "the same tree" — it will be wiped and replaced.
	if dstInfo.Mode()&fs.ModeSymlink != 0 || !dstInfo.IsDir() {
		return false, nil
	}

	srcKinds, err := srcTreeKinds(src)
	if err != nil {
		return false, err
	}
	dstKinds, dstInvalid, err := dstTreeKinds(dst)
	if err != nil {
		return false, err
	}
	if dstInvalid {
		return false, nil
	}

	// Comparing lengths plus a one-directional walk is enough to detect
	// extra entries in dst too: if every src path is present in dst with a
	// matching kind, and the maps have the same size, dst cannot contain
	// anything src doesn't have.
	if len(srcKinds) != len(dstKinds) {
		return false, nil
	}
	for rel, isDir := range srcKinds {
		dstIsDir, ok := dstKinds[rel]
		if !ok || dstIsDir != isDir {
			return false, nil
		}
	}

	for rel, isDir := range srcKinds {
		if isDir {
			continue
		}
		same, err := sameContent(filepath.Join(src, rel), filepath.Join(dst, rel))
		if err != nil {
			return false, err
		}
		if !same {
			return false, nil
		}
	}

	return true, nil
}

// srcTreeKinds walks src and returns, for every entry other than the root
// itself, its path relative to src mapped to whether it is a directory.
//
// A symlink in src is resolved: if it points at a regular file it is
// recorded as a file (its content is compared by following the link), and
// if it points at a directory that is refused with an error rather than
// followed, because following it risks an infinite cycle.
func srcTreeKinds(root string) (map[string]bool, error) {
	kinds := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		if rel == "." {
			return nil
		}

		switch {
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
			kinds[rel] = false
		case d.IsDir():
			kinds[rel] = true
		case d.Type().IsRegular():
			kinds[rel] = false
		default:
			return fmt.Errorf("unsupported file type at %s: %s", path, d.Type())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return kinds, nil
}

// dstTreeKinds walks dst and returns, for every entry other than the root
// itself, its path relative to dst mapped to whether it is a directory.
//
// Unlike srcTreeKinds, a symlink or any other non-regular entry found inside
// dst is not an error: it simply makes the tree "not the same", since a
// stray symlink there will be wiped and replaced along with everything
// else. invalid reports that this happened.
func dstTreeKinds(root string) (kinds map[string]bool, invalid bool, err error) {
	kinds = make(map[string]bool)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			kinds[rel] = true
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			invalid = true
			return filepath.SkipAll
		}
		kinds[rel] = false
		return nil
	})
	if walkErr != nil {
		return nil, false, walkErr
	}
	return kinds, invalid, nil
}

// sameContent reports whether the files at a and b are byte-for-byte
// identical. It reads both in fixed-size chunks rather than loading either
// file into memory, so it scales to arbitrarily large files.
func sameContent(a, b string) (bool, error) {
	fa, err := os.Open(a)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", a, err)
	}
	defer fa.Close()

	fb, err := os.Open(b)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", b, err)
	}
	defer fb.Close()

	bufA := make([]byte, bufSize)
	bufB := make([]byte, bufSize)

	for {
		na, erra := io.ReadFull(fa, bufA)
		if erra != nil && !errors.Is(erra, io.EOF) && !errors.Is(erra, io.ErrUnexpectedEOF) {
			return false, fmt.Errorf("read %s: %w", a, erra)
		}
		nb, errb := io.ReadFull(fb, bufB)
		if errb != nil && !errors.Is(errb, io.EOF) && !errors.Is(errb, io.ErrUnexpectedEOF) {
			return false, fmt.Errorf("read %s: %w", b, errb)
		}

		if na != nb || !bytes.Equal(bufA[:na], bufB[:nb]) {
			return false, nil
		}

		aEOF := erra != nil
		bEOF := errb != nil
		if aEOF != bEOF {
			return false, nil
		}
		if aEOF && bEOF {
			return true, nil
		}
	}
}
