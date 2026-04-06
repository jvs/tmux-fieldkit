package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	notesSession = "field notes"
	notesWindow  = "recorder"
)

// NotesToggle shows the field notes session, or hides it if already visible.
// Auto-flushes files from previous days before opening.
func NotesToggle(cfg Config) error {
	if CurrentSession() == notesSession {
		if err := runCmd(fullscreenPrevSession); err != nil {
			runCmd("tmux switch-client -l")
		}
		return nil
	}

	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}

	// If a popup is currently showing, close it before switching to a
	// fullscreen session.
	if CurrentSession() == cfg.PopupSession {
		DetachClient()
	}

	if err := notesMaybeAutoFlush(dataDir, cfg.CommitInterval); err != nil {
		return err
	}

	if SessionExists(notesSession) {
		if err := SwitchClient(notesSession); err != nil {
			return err
		}
		runCmd(fullscreenRecordVisit) // best-effort
		return nil
	}

	target, err := notesResolveTarget(dataDir)
	if err != nil {
		return err
	}
	return notesOpenWindow(cfg, dataDir, target)
}

// NotesNew creates a new timestamped notes file and opens it, replacing the
// current notes window if one exists.
func NotesNew(cfg Config) error {
	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return err
	}
	target, err := notesNewFile(filepath.Join(dataDir, "notes", "today"))
	if err != nil {
		return err
	}
	return notesOpenWindow(cfg, dataDir, target)
}

// notesOpenWindow opens target in the field notes session. Replaces the notes
// window if the session already exists, creates the session if it doesn't.
func notesOpenWindow(cfg Config, dataDir, target string) error {
	notesDir := filepath.Join(dataDir, "notes")
	cmd := fmt.Sprintf("cd '%s' && %s '%s'; %s", notesDir, cfg.Editor, target, fullscreenKillSession)

	if !SessionExists(notesSession) {
		if err := NewSessionWithWindow(notesSession, notesWindow, notesDir, cmd); err != nil {
			return err
		}
		if err := SwitchClient(notesSession); err != nil {
			return err
		}
		runCmd(fullscreenRecordVisit) // best-effort
		return nil
	}

	// Kill the existing window without detaching clients (safe to call even if
	// the window doesn't exist).
	KillWindow(notesSession, notesWindow)

	// Killing the last window may destroy the session — recreate if needed.
	if !SessionExists(notesSession) {
		if err := NewSessionWithWindow(notesSession, notesWindow, notesDir, cmd); err != nil {
			return err
		}
		if err := SwitchClient(notesSession); err != nil {
			return err
		}
		runCmd(fullscreenRecordVisit) // best-effort
		return nil
	}

	if err := NewDetachedWindow(notesSession, notesWindow, notesDir, cmd); err != nil {
		return err
	}
	SelectWindow(notesSession, notesWindow)
	if err := SwitchClient(notesSession); err != nil {
		return err
	}
	runCmd(fullscreenRecordVisit) // best-effort
	return nil
}

// notesResolveTarget returns the most recent today file, creating one if none exist.
func notesResolveTarget(dataDir string) (string, error) {
	todayDir := filepath.Join(dataDir, "notes", "today")
	today := time.Now().Format("2006-01-02")

	entries, err := os.ReadDir(todayDir)
	if os.IsNotExist(err) {
		return notesNewFile(todayDir)
	}
	if err != nil {
		return "", fmt.Errorf("reading today dir: %w", err)
	}

	var matches []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), today) && strings.HasSuffix(e.Name(), ".md") {
			matches = append(matches, e.Name())
		}
	}
	if len(matches) == 0 {
		return notesNewFile(todayDir)
	}
	sort.Strings(matches)
	return filepath.Join(todayDir, matches[len(matches)-1]), nil
}

// notesNewFile creates a new empty timestamped file in todayDir.
func notesNewFile(todayDir string) (string, error) {
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		return "", fmt.Errorf("creating today dir: %w", err)
	}
	path := filepath.Join(todayDir, time.Now().Format("2006-01-02-150405.md"))
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		return "", fmt.Errorf("creating notes file: %w", err)
	}
	return path, nil
}

// notesMaybeAutoFlush flushes any files in today/ from a previous day.
func notesMaybeAutoFlush(dataDir string, commitInterval int) error {
	todayDir := filepath.Join(dataDir, "notes", "today")
	today := time.Now().Format("2006-01-02")

	entries, err := os.ReadDir(todayDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading today dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && !strings.HasPrefix(e.Name(), today) {
			return notesDoFlush(dataDir, commitInterval)
		}
	}
	return nil
}

// notesDoFlush processes all files in notes/today/ and routes their content
// to topic files.
func notesDoFlush(dataDir string, commitInterval int) error {
	todayDir := filepath.Join(dataDir, "notes", "today")
	topicsDir := filepath.Join(dataDir, "notes", "topics")

	entries, err := os.ReadDir(todayDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading today dir: %w", err)
	}

	var flushed bool
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(todayDir, e.Name())
		date := e.Name()[:10] // YYYY-MM-DD
		if err := notesProcessFile(path, date, topicsDir); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}
		flushed = true
	}
	if flushed {
		return MaybeCommit(dataDir, "flush: notes", true, commitInterval)
	}
	return nil
}

func notesProcessFile(path, date, topicsDir string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	for _, sec := range notesParseSections(string(data)) {
		if err := notesAppendToTopic(topicsDir, sec.slug, sec.heading, date, sec.content); err != nil {
			return err
		}
	}
	return nil
}

type notesSection struct {
	slug    string
	heading string
	content string
}

// notesParseSections splits content by # headings. Content before the first
// heading goes to the implicit "notes" topic. Duplicate slugs are merged.
func notesParseSections(content string) []notesSection {
	type entry struct {
		heading string
		buf     strings.Builder
	}
	var order []string
	entries := map[string]*entry{}

	get := func(slug, heading string) *entry {
		if e, ok := entries[slug]; ok {
			return e
		}
		e := &entry{heading: heading}
		entries[slug] = e
		order = append(order, slug)
		return e
	}

	cur := get("notes", "Notes")
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			heading := strings.TrimPrefix(line, "# ")
			cur = get(notesSlugify(heading), heading)
		} else {
			cur.buf.WriteString(line)
			cur.buf.WriteByte('\n')
		}
	}

	var out []notesSection
	for _, slug := range order {
		e := entries[slug]
		if body := strings.TrimSpace(e.buf.String()); body != "" {
			out = append(out, notesSection{slug: slug, heading: e.heading, content: body})
		}
	}
	return out
}

var notesNonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

func notesSlugify(s string) string {
	s = strings.ToLower(strings.ReplaceAll(s, " ", "-"))
	s = strings.Trim(notesNonAlphaNum.ReplaceAllString(s, ""), "-")
	if s == "" {
		return "notes"
	}
	return s
}

// notesAppendToTopic appends content to a topic file under a ## date subheader,
// creating the file with an h1 heading if it's new.
func notesAppendToTopic(topicsDir, slug, heading, date, content string) error {
	if err := os.MkdirAll(topicsDir, 0755); err != nil {
		return fmt.Errorf("creating topics dir: %w", err)
	}
	path := filepath.Join(topicsDir, slug+".md")

	existing, err := os.ReadFile(path)
	isNew := os.IsNotExist(err)
	if err != nil && !isNew {
		return fmt.Errorf("reading topic file: %w", err)
	}

	var buf strings.Builder
	if isNew {
		fmt.Fprintf(&buf, "# %s\n", heading)
	} else {
		buf.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			buf.WriteByte('\n')
		}
	}
	fmt.Fprintf(&buf, "\n## %s\n\n%s\n", date, content)

	return os.WriteFile(path, []byte(buf.String()), 0644)
}
