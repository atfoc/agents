package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Exists reports whether anything at all is present at path — a regular
// file, a directory, or even a dangling symlink.
func Exists(path string) (bool, error) {
	// Lstat, not Stat: only the presence of something at path is in
	// question here, not whether a symlink there resolves to a live
	// target. Stat would follow the link and report a dangling symlink as
	// not existing, which is wrong for a function whose whole job is to
	// answer "is there something at this path at all".
	_, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	return true, nil
}
