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

func TestHistoryAppendAndLoadSkipsMalformed(t *testing.T) {
	dir := withTempConfig(t)
	r1 := Record{Mode: "time", TimeLimit: 30, WPM: 80, Accuracy: 96}
	r2 := Record{Mode: "words", WordCount: 25, WPM: 90, Accuracy: 98}
	if err := AppendRecord(r1); err != nil {
		t.Fatal(err)
	}
	if err := AppendRecord(r2); err != nil {
		t.Fatal(err)
	}
	// Corrupt one line in the middle.
	hp := filepath.Join(dir, "monkeytui", "history.jsonl")
	data, _ := os.ReadFile(hp)
	if err := os.WriteFile(hp, append([]byte("{garbage\n"), data...), 0o644); err != nil {
		t.Fatal(err)
	}
	h := LoadHistory()
	if len(h) != 2 {
		t.Fatalf("want 2 valid records, got %d", len(h))
	}
}

func TestBestWPMBucketing(t *testing.T) {
	h := []Record{
		{Mode: "time", TimeLimit: 30, WPM: 70},
		{Mode: "time", TimeLimit: 30, WPM: 85},
		{Mode: "time", TimeLimit: 60, WPM: 99},          // different bucket
		{Mode: "words", WordCount: 25, WPM: 88},
		{Mode: "time", TimeLimit: 30, Punctuation: true, WPM: 200}, // flags differ
	}
	if best, ok := BestWPM(h, Bucket{Mode: "time", TimeLimit: 30}); !ok || best != 85 {
		t.Fatalf("time30 best: got %v ok=%v want 85", best, ok)
	}
	if _, ok := BestWPM(h, Bucket{Mode: "words", WordCount: 50}); ok {
		t.Fatalf("words50 should have no record")
	}
	// WordCount on a time bucket is ignored.
	if best, _ := BestWPM(h, Bucket{Mode: "time", TimeLimit: 30, WordCount: 999}); best != 85 {
		t.Fatalf("time bucket must ignore WordCount, got %v", best)
	}
}
