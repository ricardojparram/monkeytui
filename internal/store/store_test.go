// internal/store/store_test.go
package store

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempConfig points os.UserConfigDir at a temp dir for the duration of t.
func withTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)   // Linux/UserConfigDir honors this
	t.Setenv("HOME", dir)              // macOS fallback path
	return dir
}

func TestPrefsRoundtrip(t *testing.T) {
	withTempConfig(t)
	in := Prefs{Mode: "words", TimeLimit: 60, WordCount: 50, Theme: "cyan", Punctuation: true, Numbers: true}
	if err := SavePrefs(in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := LoadPrefs()
	if got != in {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", got, in)
	}
}

func TestLoadPrefsMissingReturnsDefaults(t *testing.T) {
	withTempConfig(t)
	if got := LoadPrefs(); got != DefaultPrefs() {
		t.Fatalf("missing file: got %+v want defaults %+v", got, DefaultPrefs())
	}
}

func TestLoadPrefsCorruptReturnsDefaults(t *testing.T) {
	dir := withTempConfig(t)
	d := filepath.Join(dir, "monkeytui")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadPrefs(); got != DefaultPrefs() {
		t.Fatalf("corrupt file: got %+v want defaults", got)
	}
}
