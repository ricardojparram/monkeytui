// Package store persists monkeytui preferences and test history under the
// user's config directory. All operations are best-effort: failures fall back
// to in-memory defaults and never crash a running test.
package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
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

// Record is one finished test, appended as a line to history.jsonl.
type Record struct {
	Time        time.Time `json:"time"`
	Mode        string    `json:"mode"`
	TimeLimit   int       `json:"timeLimit"`
	WordCount   int       `json:"wordCount"`
	Punctuation bool      `json:"punctuation"`
	Numbers     bool      `json:"numbers"`
	WPM         float64   `json:"wpm"`
	Raw         float64   `json:"raw"`
	Accuracy    float64   `json:"acc"`
	Consistency float64   `json:"consistency"`
}

// Bucket groups comparable tests for personal-best lookups.
type Bucket struct {
	Mode        string
	TimeLimit   int
	WordCount   int
	Punctuation bool
	Numbers     bool
}

// normalize zeroes fields that don't distinguish a bucket for the given mode,
// so e.g. a 30s time test ignores WordCount.
func normalize(b Bucket) Bucket {
	switch b.Mode {
	case "time":
		b.WordCount = 0
	case "words":
		b.TimeLimit = 0
	case "quote":
		b.TimeLimit, b.WordCount = 0, 0
		b.Punctuation, b.Numbers = false, false
	}
	return b
}

// RecordBucket derives the normalized bucket a record belongs to.
func RecordBucket(r Record) Bucket {
	return normalize(Bucket{
		Mode: r.Mode, TimeLimit: r.TimeLimit, WordCount: r.WordCount,
		Punctuation: r.Punctuation, Numbers: r.Numbers,
	})
}

func historyPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "history.jsonl"), nil
}

// AppendRecord appends one JSON line to history.jsonl.
func AppendRecord(r Record) error {
	path, err := historyPath()
	if err != nil {
		return err
	}
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// LoadHistory parses history.jsonl, silently skipping malformed lines.
func LoadHistory() []Record {
	path, err := historyPath()
	if err != nil {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var r Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

// BestWPM returns the highest WPM recorded in the given bucket, if any.
func BestWPM(h []Record, b Bucket) (float64, bool) {
	target := normalize(b)
	best, found := 0.0, false
	for _, r := range h {
		if RecordBucket(r) != target {
			continue
		}
		if !found || r.WPM > best {
			best, found = r.WPM, true
		}
	}
	return best, found
}
