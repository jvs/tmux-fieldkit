package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const todoWindow = "kit-todo"

// TodoToggle shows or hides the todo popup.
//
// If the todo popup is currently showing (we're in the popup session and the
// todo window is focused), hide it. The editor keeps running in the background.
//
// If a different kit popup is showing, hide it and then show the todo popup.
//
// If no kit popup is showing, show the todo popup, creating the window if
// necessary.
func TodoToggle(cfg Config) error {
	if CurrentSession() == cfg.PopupSession {
		if CurrentWindow() == todoWindow {
			// Todo is showing — hide it, leave the editor running.
			return DetachClient()
		}
		// A different kit popup is showing — hide it, then show todo.
		if err := DetachClient(); err != nil {
			return err
		}
	}
	return openTodo(cfg)
}

// TodoCycle flushes stage.md into list.md and switches the view:
//   - stage.md has content (user is looking at stage) -> flush -> show list.md
//   - stage.md is empty   (user is looking at list)   -> show fresh stage.md
func TodoCycle(cfg Config) error {
	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}

	stagePath := filepath.Join(dataDir, "todo", "stage.md")
	listPath := filepath.Join(dataDir, "todo", "list.md")

	// Read before flushing — this tells us which file the window is showing.
	stageData, _ := os.ReadFile(stagePath)
	stageHasContent := len(strings.TrimSpace(string(stageData))) > 0

	if os.Getenv("TMUX") != "" {
		GracefulCloseWindow(cfg.PopupSession, todoWindow, cfg.Editor, cfg.CleanupKeys)
	}

	if err := todoDoFlush(dataDir, stagePath, listPath, cfg.CommitInterval); err != nil {
		return err
	}

	var target string
	if stageHasContent {
		target = listPath
	} else {
		target = stagePath
	}

	if os.Getenv("TMUX") == "" {
		todoDir := filepath.Join(dataDir, "todo")
		return RunDirect(fmt.Sprintf("cd '%s' && %s '%s'", todoDir, cfg.Editor, target))
	}
	return todoShowWindow(cfg, dataDir, target)
}

// TodoFlush appends stage.md to the top of list.md and clears stage.md.
// Closes the todo window first so the editor isn't holding the file.
func TodoFlush(cfg Config) error {
	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}
	if os.Getenv("TMUX") != "" {
		GracefulCloseWindow(cfg.PopupSession, todoWindow, cfg.Editor, cfg.CleanupKeys)
	}
	stagePath := filepath.Join(dataDir, "todo", "stage.md")
	listPath := filepath.Join(dataDir, "todo", "list.md")
	return todoDoFlush(dataDir, stagePath, listPath, cfg.CommitInterval)
}

// openTodo shows the todo popup, creating the window if it doesn't exist.
// Falls back to opening the editor directly if not inside tmux.
func openTodo(cfg Config) error {
	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}
	if os.Getenv("TMUX") == "" {
		target, err := todoResolveTarget(cfg, dataDir)
		if err != nil {
			return err
		}
		todoDir := filepath.Join(dataDir, "todo")
		return RunDirect(fmt.Sprintf("cd '%s' && %s '%s'", todoDir, cfg.Editor, target))
	}
	return todoShowWindow(cfg, dataDir, "")
}

// todoShowWindow ensures the todo window exists in the popup session, then
// shows it as a popup overlay. If the window already exists (editor is
// running in the background), it is shown as-is. If it doesn't exist, the
// target file is resolved and a new editor window is created.
func todoShowWindow(cfg Config, dataDir, target string) error {
	todoDir := filepath.Join(dataDir, "todo")

	if err := EnsureSession(cfg.PopupSession); err != nil {
		return err
	}

	if !WindowExists(cfg.PopupSession, todoWindow) {
		if target == "" {
			var err error
			target, err = todoResolveTarget(cfg, dataDir)
			if err != nil {
				return err
			}
		}
		cmd := fmt.Sprintf("cd '%s' && %s '%s'; tmux detach-client", todoDir, cfg.Editor, target)
		if err := NewDetachedWindow(cfg.PopupSession, todoWindow, todoDir, cmd); err != nil {
			return err
		}
	}

	if err := ShowSessionPopup(cfg.PopupSession, todoWindow, "#[align=right] todo list "); err != nil {
		return err
	}

	return MaybeCommit(dataDir, "auto: todo", false, cfg.CommitInterval)
}

// todoResolveTarget decides which file to open when creating a new window.
// If stage.md is stale (older than TodoFlushTimeout) and non-empty, it is
// flushed automatically and list.md is returned. Otherwise stage.md is
// returned if it has content, else list.md.
func todoResolveTarget(cfg Config, dataDir string) (string, error) {
	stagePath := filepath.Join(dataDir, "todo", "stage.md")
	listPath := filepath.Join(dataDir, "todo", "list.md")

	timeout := time.Duration(cfg.TodoFlushTimeout) * time.Minute
	old, err := FileOlderThan(stagePath, timeout)
	if err == nil && old {
		stageData, err := os.ReadFile(stagePath)
		if err == nil && len(strings.TrimSpace(string(stageData))) > 0 {
			if err := todoDoFlush(dataDir, stagePath, listPath, cfg.CommitInterval); err != nil {
				return "", fmt.Errorf("auto-flush: %w", err)
			}
			return listPath, nil
		}
	}

	info, err := os.Stat(stagePath)
	if err == nil && info.Size() > 0 {
		return stagePath, nil
	}
	return listPath, nil
}

// todoDoFlush prepends the contents of stage.md to list.md, then clears
// stage.md. Does nothing if stage.md is empty or whitespace-only.
func todoDoFlush(dataDir, stagePath, listPath string, commitInterval int) error {
	stageData, err := os.ReadFile(stagePath)
	if err != nil {
		return fmt.Errorf("reading stage.md: %w", err)
	}
	if len(strings.TrimSpace(string(stageData))) == 0 {
		return nil
	}

	listData, err := os.ReadFile(listPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading list.md: %w", err)
	}

	combined := stageData
	if len(listData) > 0 {
		if !strings.HasSuffix(string(stageData), "\n") {
			combined = append(combined, '\n')
		}
		combined = append(combined, '\n')
		combined = append(combined, listData...)
	}

	if err := os.WriteFile(listPath, combined, 0644); err != nil {
		return fmt.Errorf("writing list.md: %w", err)
	}
	if err := os.WriteFile(stagePath, []byte{}, 0644); err != nil {
		return fmt.Errorf("clearing stage.md: %w", err)
	}

	return MaybeCommit(dataDir, "flush: todo", true, commitInterval)
}
