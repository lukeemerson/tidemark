// Package config persists tidemark's settings in ~/.config/tidemark/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Layout string `json:"layout"`
}

// Path honours XDG_CONFIG_HOME and otherwise uses ~/.config, like the rest of the dotfiles.
func Path() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "tidemark", "config.json")
}

// Load returns the saved config; a missing or unreadable file gives the zero Config.
func Load() Config {
	var c Config
	if b, err := os.ReadFile(Path()); err == nil {
		json.Unmarshal(b, &c)
	}
	return c
}

func Save(c Config) error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, append(b, '\n'), 0o644)
}
