package kit

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Init runs "git init" inside dir, creating it if needed.
func Init(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating dir %s: %w", dir, err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init: %w\n%s", err, out)
	}
	return nil
}

// HasChanges returns true if there are uncommitted changes in dir.
func HasChanges(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git status: %w\n%s", err, out)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// LastCommitTime returns the time of the most recent commit, or zero if none.
func LastCommitTime(dir string) (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return time.Time{}, nil
	}
	ts := strings.TrimSpace(string(out))
	if ts == "" {
		return time.Time{}, nil
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing commit time: %w", err)
	}
	return time.Unix(unix, 0), nil
}

// MaybeCommit commits all changes if conditions are met.
// If isFlush is true, always commits (message: "flush: <tool>").
// Otherwise commits only if the last commit is older than intervalMinutes.
func MaybeCommit(dir, message string, isFlush bool, intervalMinutes int) error {
	changed, err := HasChanges(dir)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if !isFlush {
		last, err := LastCommitTime(dir)
		if err != nil {
			return err
		}
		if !last.IsZero() && time.Since(last) < time.Duration(intervalMinutes)*time.Minute {
			return nil
		}
	}
	add := exec.Command("git", "add", "-A")
	add.Dir = dir
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}
	commit := exec.Command("git", "commit", "-m", message)
	commit.Dir = dir
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	return nil
}
