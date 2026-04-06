package kit

import (
	"os"
)

const (
	calcWindow        = "kit-calc"
	calcKillWindowCmd = "tmux-hometown kill-window -y"
)

// CalcToggle shows or hides the calculator popup.
// Creates a new python3 window if one doesn't exist.
func CalcToggle(cfg Config) error {
	if CurrentSession() == cfg.PopupSession {
		if CurrentWindow() == calcWindow {
			return DetachClient()
		}
		if err := DetachClient(); err != nil {
			return err
		}
	}

	if err := EnsureSession(cfg.PopupSession); err != nil {
		return err
	}

	if !WindowExists(cfg.PopupSession, calcWindow) {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cmd := "python3; tmux detach-client"
		if err := NewDetachedWindow(cfg.PopupSession, calcWindow, home, cmd); err != nil {
			return err
		}
	}

	return ShowSessionPopup(cfg.PopupSession, calcWindow, "#[align=right] calculator ")
}

// CalcNew kills the calculator window if it exists and creates a fresh one.
func CalcNew(cfg Config) error {
	if CurrentWindow() == calcWindow {
		if err := runCmd(calcKillWindowCmd); err != nil {
			return err
		}
	} else if WindowExists(cfg.PopupSession, calcWindow) {
		KillWindow(cfg.PopupSession, calcWindow)
	}
	return CalcToggle(cfg)
}
