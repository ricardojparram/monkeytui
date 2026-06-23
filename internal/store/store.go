// Package store persists monkeytui preferences and test history under the
// user's config directory. All operations are best-effort: failures fall back
// to in-memory defaults and never crash a running test.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Prefs is the set of settings remembered between launches.
type Prefs struct {
	Mode        string `json:"mode"` // "time" | "words" | "quote"
	TimeLimit   int    `json:"time"`
	WordCount   int    `json:"words"`
	Theme       string `json:"theme"`
	Punctuation bool   `json:"punctuation"`
	Numbers     bool   `json:"numbers"`
}

// DefaultPrefs mirrors the CLI defaults used before any file exists.
func DefaultPrefs() Prefs {
	return Prefs{Mode: "time", TimeLimit: 30, WordCount: 25, Theme: "yellow"}
}

const appDir = "monkeytui"

// Dir resolves (and creates) the monkeytui config directory.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, appDir)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

func prefsPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// LoadPrefs returns saved preferences, or DefaultPrefs on any failure.
func LoadPrefs() Prefs {
	p, err := prefsPath()
	if err != nil {
		return DefaultPrefs()
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return DefaultPrefs()
	}
	out := DefaultPrefs()
	if err := json.Unmarshal(data, &out); err != nil {
		return DefaultPrefs()
	}
	return out
}

// SavePrefs writes preferences, creating the config dir if needed.
func SavePrefs(p Prefs) error {
	path, err := prefsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
