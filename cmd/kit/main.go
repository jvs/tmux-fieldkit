package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	kit "github.com/jvs/tmux-fieldkit/internal"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kit",
	Short: "Field Kit — everyday-carry terminal tools",
}

func init() {
	rootCmd.AddCommand(initCmd, toggleCmd, cycleCmd, newCmd, flushCmd)
}

// prompt prints a label with a default and reads a line from stdin.
// Returns the default if the user enters nothing.
func prompt(label, defaultVal string) (string, error) {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// --- kit init ---

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create data directory, scaffold files, configure local machine",
	Long: `Idempotent setup — safe to re-run on an existing install or on a new
machine pointing at an already-cloned data directory.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Load whatever exists already so prompts can default to current values.
	cfg, _ := kit.Load()

	dataDir, err := prompt("Data directory", cfg.DataDir)
	if err != nil {
		return err
	}
	editor, err := prompt("Editor", cfg.Editor)
	if err != nil {
		return err
	}

	expanded, err := kit.DataDirPath(dataDir)
	if err != nil {
		return err
	}

	// Scaffold directories.
	dirs := []string{
		filepath.Join(expanded, "config"),
		filepath.Join(expanded, "junk"),
		filepath.Join(expanded, "notes", "today"),
		filepath.Join(expanded, "notes", "topics"),
		filepath.Join(expanded, "scratch", "stage"),
		filepath.Join(expanded, "scratch", "trash"),
		filepath.Join(expanded, "todo"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}

	// Create empty files only if they don't exist yet.
	seedFiles := []string{
		filepath.Join(expanded, "todo", "stage.md"),
		filepath.Join(expanded, "todo", "list.md"),
	}
	for _, f := range seedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			if err := os.WriteFile(f, []byte{}, 0644); err != nil {
				return fmt.Errorf("creating %s: %w", f, err)
			}
		}
	}

	// Git init (no-op if already a repo).
	if err := kit.Init(expanded); err != nil {
		return err
	}
	if err := kit.EnsureGitignore(expanded, "scratch/"); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}
	fmt.Printf("Data directory: %s\n", expanded)

	// Write ~/.kitrc.
	if err := kit.SaveLocal(dataDir, editor); err != nil {
		return fmt.Errorf("writing ~/.kitrc: %w", err)
	}
	fmt.Println("Wrote ~/.kitrc")

	fmt.Println("\nDone. Add keybindings to ~/.tmux.conf — see README.")
	return nil
}

// --- kit cycle ---

var cycleCmd = &cobra.Command{
	Use:   "cycle <tool>",
	Short: "Flush staging area and open a fresh one",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := kit.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "todo":
			return kit.TodoCycle(cfg)
		default:
			return fmt.Errorf("unknown tool %q", args[0])
		}
	},
}

// --- kit new ---

var newCmd = &cobra.Command{
	Use:   "new <tool>",
	Short: "Kill a tool's session and create a fresh one",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := kit.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "junk":
			return kit.JunkNew(cfg)
		case "note":
			return kit.NotesNew(cfg)
		case "scratch":
			return kit.ScratchNew(cfg)
		default:
			return fmt.Errorf("unknown tool %q", args[0])
		}
	},
}

// --- kit toggle ---

var toggleCmd = &cobra.Command{
	Use:   "toggle <tool>",
	Short: "Toggle a tool's popup open or closed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := kit.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "todo":
			return kit.TodoToggle(cfg)
		case "notes":
			return kit.NotesToggle(cfg)
		case "junk":
			return kit.JunkToggle(cfg)
		case "scratch":
			return kit.ScratchToggle(cfg)
		default:
			return fmt.Errorf("unknown tool %q", args[0])
		}
	},
}

// --- kit flush ---

var flushCmd = &cobra.Command{
	Use:   "flush <tool>",
	Short: "Flush a tool's staging area",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := kit.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "todo":
			return kit.TodoFlush(cfg)
		default:
			return fmt.Errorf("unknown tool %q", args[0])
		}
	},
}
