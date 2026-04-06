package kit

import (
	"fmt"
	"os"
	"time"
)

// FileOlderThan returns true if the file at path has an mtime older than d.
func FileOlderThan(path string, d time.Duration) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	return time.Since(info.ModTime()) > d, nil
}
