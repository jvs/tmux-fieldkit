package kit

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initGit(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git setup: %v\n%s", err, out)
		}
	}
}

func TestTodoFlush(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	todoDir := filepath.Join(dir, "todo")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatal(err)
	}

	stage := filepath.Join(todoDir, "stage.md")
	list := filepath.Join(todoDir, "list.md")

	os.WriteFile(stage, []byte("- buy milk\n- call dentist\n"), 0644)
	os.WriteFile(list, []byte("- old task\n"), 0644)

	cfg := Defaults()
	cfg.DataDir = dir

	if err := TodoFlush(cfg); err != nil {
		t.Fatalf("TodoFlush error: %v", err)
	}

	stageData, _ := os.ReadFile(stage)
	if len(stageData) != 0 {
		t.Errorf("stage.md should be empty after flush, got: %q", stageData)
	}

	listData, _ := os.ReadFile(list)
	got := string(listData)
	if got != "- buy milk\n- call dentist\n\n- old task\n" {
		t.Errorf("list.md wrong after flush:\n%q", got)
	}
}

func TestTodoFlushEmptyStage(t *testing.T) {
	dir := t.TempDir()
	todoDir := filepath.Join(dir, "todo")
	os.MkdirAll(todoDir, 0755)

	stage := filepath.Join(todoDir, "stage.md")
	list := filepath.Join(todoDir, "list.md")

	os.WriteFile(stage, []byte("   \n"), 0644) // whitespace only
	os.WriteFile(list, []byte("- keep me\n"), 0644)

	cfg := Defaults()
	cfg.DataDir = dir

	if err := TodoFlush(cfg); err != nil {
		t.Fatalf("TodoFlush error: %v", err)
	}

	listData, _ := os.ReadFile(list)
	if string(listData) != "- keep me\n" {
		t.Errorf("list.md should be unchanged for empty stage flush, got: %q", listData)
	}
}
