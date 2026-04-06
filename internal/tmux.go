package kit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SessionExists returns true if a tmux session with the given name exists.
func SessionExists(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

// EnsureSessionAt creates a detached tmux session rooted at cwd if it doesn't
// already exist.
func EnsureSessionAt(name, cwd string) error {
	if SessionExists(name) {
		return nil
	}
	out, err := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", cwd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating session %q: %w\n%s", name, err, out)
	}
	return nil
}

// EnsureSession creates a detached tmux session if it doesn't already exist.
func EnsureSession(name string) error {
	if SessionExists(name) {
		return nil
	}
	out, err := exec.Command("tmux", "new-session", "-d", "-s", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating session %q: %w\n%s", name, err, out)
	}
	return nil
}

// NewSessionWithWindow creates a detached session with a single named window
// running shellCmd.
func NewSessionWithWindow(session, window, cwd, shellCmd string) error {
	out, err := exec.Command("tmux", "new-session", "-d", "-s", session, "-n", window, "-c", cwd, shellCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating session %q: %w\n%s", session, err, out)
	}
	return nil
}

// WindowExists returns true if the named window exists in the session.
func WindowExists(session, window string) bool {
	out, err := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_name}").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == window {
			return true
		}
	}
	return false
}

// NewDetachedWindow creates a new window in session running shellCmd with cwd.
func NewDetachedWindow(session, window, cwd, shellCmd string) error {
	out, err := exec.Command("tmux", "new-window", "-d", "-t", session, "-n", window, "-c", cwd, shellCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating window %q in %q: %w\n%s", window, session, err, out)
	}
	return nil
}

// ShowSessionPopup displays session:window as a popup overlay with an
// optional title. Pass an empty string for no title.
func ShowSessionPopup(session, window, title string) error {
	target := session + ":" + window
	args := []string{"display-popup", "-h", "80%", "-w", "80%", "-EE"}
	if title != "" {
		args = append(args, "-T", title)
	}
	args = append(args, "tmux attach-session -t '"+target+"'")
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("showing popup %q: %w\n%s", target, err, out)
	}
	return nil
}

// CurrentSession returns the name of the current tmux session.
func CurrentSession() string {
	out, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CurrentWindow returns the name of the current tmux window.
func CurrentWindow() string {
	out, err := exec.Command("tmux", "display-message", "-p", "#{window_name}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// DetachClient detaches the current tmux client.
func DetachClient() error {
	return exec.Command("tmux", "detach-client").Run()
}

// SwitchClient switches the current tmux client to the named session.
func SwitchClient(session string) error {
	out, err := exec.Command("tmux", "switch-client", "-t", session).CombinedOutput()
	if err != nil {
		return fmt.Errorf("switching to session %q: %w\n%s", session, err, out)
	}
	return nil
}

// SelectWindow switches to a named window within a session.
func SelectWindow(session, window string) error {
	out, err := exec.Command("tmux", "select-window", "-t", session+":"+window).CombinedOutput()
	if err != nil {
		return fmt.Errorf("selecting window %q in %q: %w\n%s", window, session, err, out)
	}
	return nil
}

// KillWindow kills the named window without detaching clients.
func KillWindow(session, window string) {
	exec.Command("tmux", "kill-window", "-t", session+":"+window).Run()
}

// ForceCloseWindow detaches all clients from session and kills the named window.
func ForceCloseWindow(session, window string) {
	exec.Command("tmux", "detach-client", "-s", session).Run()
	exec.Command("tmux", "kill-window", "-t", session+":"+window).Run()
}

// PaneCurrentCommand returns the foreground process name in the named window.
func PaneCurrentCommand(session, window string) string {
	out, err := exec.Command("tmux", "display-message", "-t", session+":"+window,
		"-p", "#{pane_current_command}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// SendKeys sends a key sequence to a tmux window.
func SendKeys(session, window, keys string) error {
	out, err := exec.Command("tmux", "send-keys", "-t", session+":"+window, keys, "").CombinedOutput()
	if err != nil {
		return fmt.Errorf("send-keys to %s:%s: %w\n%s", session, window, err, out)
	}
	return nil
}

// GracefulCloseWindow sends cleanupKeys to the editor if it is the foreground
// process, then force-closes the window.
func GracefulCloseWindow(session, window, editor, cleanupKeys string) {
	if cleanupKeys != "" && editor != "" {
		editorBin := filepath.Base(strings.Fields(editor)[0])
		if PaneCurrentCommand(session, window) == editorBin {
			SendKeys(session, window, cleanupKeys)
			time.Sleep(200 * time.Millisecond)
		}
	}
	ForceCloseWindow(session, window)
}

// RunDirect runs shellCmd via sh -c with stdio inherited from the parent.
func RunDirect(shellCmd string) error {
	c := exec.Command("sh", "-c", shellCmd)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
