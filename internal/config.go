package kit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// localConfig holds machine-specific settings, stored in ~/.kitrc.
// It is not synced to git.
type localConfig struct {
	DataDir string `toml:"data_dir"`
	Editor  string `toml:"editor"`
}

// Config is the full merged configuration. DataDir and Editor come from
// ~/.kitrc; all other fields come from <data_dir>/kit.toml (git-synced).
type Config struct {
	DataDir string
	Editor  string

	TodoFlushTimeout int    `toml:"todo_flush_timeout"`
	CommitInterval   int    `toml:"commit_interval"`
	PopupSession     string `toml:"popup_session"`
	CleanupKeys      string `toml:"cleanup_keys"`
}

// Defaults returns a Config with all default values populated.
func Defaults() Config {
	return Config{
		DataDir:          "~/kit",
		Editor:           "nvim",
		TodoFlushTimeout: 120,
		CommitInterval:   60,
		PopupSession:     "__kit__",
	}
}

// Load reads ~/.kitrc for data_dir and editor, then <data_dir>/kit.toml for
// the remaining settings. Missing files are not errors — defaults are used.
func Load() (Config, error) {
	cfg := Defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, fmt.Errorf("resolving home dir: %w", err)
	}

	local := localConfig{DataDir: cfg.DataDir, Editor: cfg.Editor}
	kitrcPath := filepath.Join(home, ".kitrc")
	if data, err := os.ReadFile(kitrcPath); err == nil {
		if _, err := toml.Decode(string(data), &local); err != nil {
			return cfg, fmt.Errorf("parsing ~/.kitrc: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("reading ~/.kitrc: %w", err)
	}
	cfg.DataDir = local.DataDir
	cfg.Editor = local.Editor

	dataDir, err := DataDirPath(cfg.DataDir)
	if err != nil {
		return cfg, err
	}
	kitTomlPath := filepath.Join(dataDir, "config", "kit.toml")
	if data, err := os.ReadFile(kitTomlPath); err == nil {
		if _, err := toml.Decode(string(data), &cfg); err != nil {
			return cfg, fmt.Errorf("parsing kit.toml: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("reading kit.toml: %w", err)
	}

	return cfg, nil
}

// SaveLocal writes data_dir and editor to ~/.kitrc.
func SaveLocal(dataDir, editor string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home dir: %w", err)
	}
	local := localConfig{DataDir: dataDir, Editor: editor}
	var buf bytes.Buffer
	buf.WriteString("# Field Kit local config — machine-specific, not synced\n\n")
	if err := toml.NewEncoder(&buf).Encode(local); err != nil {
		return fmt.Errorf("encoding local config: %w", err)
	}
	return os.WriteFile(filepath.Join(home, ".kitrc"), buf.Bytes(), 0644)
}

// DataDirPath expands ~ and returns an absolute path.
func DataDirPath(raw string) (string, error) {
	if raw == "" {
		raw = "~/kit"
	}
	if len(raw) >= 2 && raw[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home dir: %w", err)
		}
		return filepath.Join(home, raw[2:]), nil
	}
	return filepath.Abs(raw)
}
