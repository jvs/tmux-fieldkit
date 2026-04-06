package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const scratchWindow = "kit-scratch"

const scratchKillWindowCmd = "tmux-hometown kill-window -y"

// ScratchNew kills the scratch window if it exists and creates a fresh one.
// If the scratch window is currently focused, uses tmux-hometown to navigate
// away before killing. Otherwise kills it directly with tmux.
func ScratchNew(cfg Config) error {
	if CurrentWindow() == scratchWindow {
		if err := runCmd(scratchKillWindowCmd); err != nil {
			return err
		}
	} else if WindowExists(cfg.PopupSession, scratchWindow) {
		KillWindow(cfg.PopupSession, scratchWindow)
	}
	return ScratchToggle(cfg)
}

// ScratchToggle shows or hides the scratch pad popup.
//
// If the scratch popup is currently showing, hide it. The editor keeps
// running in the background.
//
// If the window already exists (editor running in background), show it.
//
// If the window doesn't exist, create a new timestamped scratch file and
// open it in a new popup window.
func ScratchToggle(cfg Config) error {
	if CurrentSession() == cfg.PopupSession && CurrentWindow() == scratchWindow {
		return DetachClient()
	}

	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}
	scratchDir := filepath.Join(dataDir, "scratch")

	if err := EnsureSession(cfg.PopupSession); err != nil {
		return err
	}

	if !WindowExists(cfg.PopupSession, scratchWindow) {
		if err := os.MkdirAll(scratchDir, 0755); err != nil {
			return fmt.Errorf("creating scratch dir: %w", err)
		}

		name := time.Now().Format("scratch-2006-01-02-150405.md")
		target := filepath.Join(scratchDir, name)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.WriteFile(target, []byte{}, 0644); err != nil {
				return fmt.Errorf("creating scratch file: %w", err)
			}
		}

		cmd := fmt.Sprintf("cd '%s' && %s '%s'; tmux detach-client", scratchDir, cfg.Editor, target)
		if err := NewDetachedWindow(cfg.PopupSession, scratchWindow, scratchDir, cmd); err != nil {
			return err
		}
	}

	return ShowSessionPopup(cfg.PopupSession, scratchWindow, "#[align=right] scratch pad ")
}
