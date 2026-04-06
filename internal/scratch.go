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
// If the window doesn't exist, run cleanup, then create a new timestamped
// scratch file in scratch/stage/ and open it in a new popup window.
func ScratchToggle(cfg Config) error {
	if CurrentSession() == cfg.PopupSession {
		if CurrentWindow() == scratchWindow {
			return DetachClient()
		}
		if err := DetachClient(); err != nil {
			return err
		}
	}

	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}
	scratchDir := filepath.Join(dataDir, "scratch")
	stageDir := filepath.Join(scratchDir, "stage")
	trashDir := filepath.Join(scratchDir, "trash")

	if err := EnsureSession(cfg.PopupSession); err != nil {
		return err
	}

	if !WindowExists(cfg.PopupSession, scratchWindow) {
		if err := scratchCleanup(stageDir, trashDir); err != nil {
			return err
		}

		if err := os.MkdirAll(stageDir, 0755); err != nil {
			return fmt.Errorf("creating scratch stage dir: %w", err)
		}

		name := time.Now().Format("scratch-2006-01-02-150405.md")
		target := filepath.Join(stageDir, name)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.WriteFile(target, []byte{}, 0644); err != nil {
				return fmt.Errorf("creating scratch file: %w", err)
			}
		}

		cmd := fmt.Sprintf("cd '%s' && %s '%s'; tmux detach-client", stageDir, cfg.Editor, target)
		if err := NewDetachedWindow(cfg.PopupSession, scratchWindow, stageDir, cmd); err != nil {
			return err
		}
	}

	return ShowSessionPopup(cfg.PopupSession, scratchWindow, "#[align=right] scratch pad ")
}

// scratchCleanup moves stage files older than 30 days to trash, and deletes
// trash files older than 60 days.
func scratchCleanup(stageDir, trashDir string) error {
	const stageMaxAge = 30 * 24 * time.Hour
	const trashMaxAge = 60 * 24 * time.Hour

	entries, err := os.ReadDir(stageDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading scratch stage: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > stageMaxAge {
			if err := os.MkdirAll(trashDir, 0755); err != nil {
				return fmt.Errorf("creating scratch trash dir: %w", err)
			}
			src := filepath.Join(stageDir, entry.Name())
			dst := filepath.Join(trashDir, entry.Name())
			os.Rename(src, dst)
		}
	}

	entries, err = os.ReadDir(trashDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading scratch trash: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > trashMaxAge {
			os.Remove(filepath.Join(trashDir, entry.Name()))
		}
	}

	return nil
}
