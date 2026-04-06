package kit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	junkSession = "junk drawer"

	// Commands for session navigation. To be made configurable.
	junkRecordVisitCmd = "tmux-hometown record-visit"
	junkPrevSessionCmd = "tmux-hometown previous-session"
	junkKillSessionCmd = "tmux-hometown kill-session -y"
)

// JunkNew kills the junk drawer session if it exists and creates a fresh one.
// If currently in the junk drawer, uses tmux-hometown to switch away before
// killing. Otherwise kills it directly with tmux.
func JunkNew(cfg Config) error {
	if CurrentSession() == junkSession {
		if err := runCmd(junkKillSessionCmd); err != nil {
			return err
		}
	} else if SessionExists(junkSession) {
		exec.Command("tmux", "kill-session", "-t", junkSession).Run()
	}
	return JunkToggle(cfg)
}

// JunkToggle switches to the junk drawer session, creating it if necessary,
// or returns to the previous session if the junk drawer is already focused.
// The session opens the editor on the junk directory.
func JunkToggle(cfg Config) error {
	if CurrentSession() == junkSession {
		if err := runCmd(junkPrevSessionCmd); err != nil {
			exec.Command("tmux", "switch-client", "-l").Run()
		}
		return nil
	}

	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}
	junkDir := filepath.Join(dataDir, "junk")

	if !SessionExists(junkSession) {
		if err := os.MkdirAll(junkDir, 0755); err != nil {
			return fmt.Errorf("creating junk dir: %w", err)
		}
		editorCmd := fmt.Sprintf("%s .; %s", cfg.Editor, junkKillSessionCmd)
		if err := NewSessionWithWindow(junkSession, "junk", junkDir, editorCmd); err != nil {
			return err
		}
	}

	if err := SwitchClient(junkSession); err != nil {
		return err
	}
	runCmd(junkRecordVisitCmd) // best-effort
	return nil
}

// runCmd runs a command string via sh -c, discarding output.
func runCmd(cmd string) error {
	return exec.Command("sh", "-c", cmd).Run()
}
