package kit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const junkSession = "junk drawer"

// JunkNew kills the junk drawer session if it exists and creates a fresh one.
// If currently in the junk drawer, uses tmux-hometown to switch away before
// killing. Otherwise kills it directly with tmux.
func JunkNew(cfg Config) error {
	if CurrentSession() == junkSession {
		if err := runCmd(fullscreenKillSession); err != nil {
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
		if err := runCmd(fullscreenPrevSession); err != nil {
			exec.Command("tmux", "switch-client", "-l").Run()
		}
		return nil
	}

	// If a popup is currently showing, close it before switching to a
	// fullscreen session.
	if CurrentSession() == cfg.PopupSession {
		DetachClient()
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
		editorCmd := fmt.Sprintf("%s .; %s", cfg.Editor, fullscreenKillSession)
		if err := NewSessionWithWindow(junkSession, "junk", junkDir, editorCmd); err != nil {
			return err
		}
	}

	if err := SwitchClient(junkSession); err != nil {
		return err
	}
	runCmd(fullscreenRecordVisit) // best-effort
	return nil
}

// runCmd runs a command string via sh -c, discarding output.
func runCmd(cmd string) error {
	return exec.Command("sh", "-c", cmd).Run()
}
