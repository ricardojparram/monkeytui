# Config Persistence, History/Records, Punctuation & Numbers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist last-used settings, record every finished test with personal-best surfacing, and add optional punctuation/numbers decoration to generated words.

**Architecture:** A new isolated `internal/store` package owns the on-disk config dir (`os.UserConfigDir()/monkeytui/`) with `config.json` (prefs) and `history.jsonl` (append-only records). `internal/words` gains a pure `Decorate` function. `typing.Config` carries two new bools. The UI imports `store` for types + best-WPM lookup and persists via direct store calls guarded by a `persist` flag (off in tests). `main.go` merges saved prefs with explicitly-set flags and adds a `stats` subcommand.

**Tech Stack:** Go 1.21+, Bubble Tea, Lip Gloss, standard `encoding/json`.

## Global Constraints

- Go 1.21+ (uses `min`/`max` builtins already in the codebase).
- Store I/O is best-effort: never panic, never crash a test. Missing/corrupt files → defaults; malformed history lines skipped.
- Quote mode is never decorated and is excluded from punctuation/numbers buckets.
- `-seed` is never persisted.
- No new third-party dependencies.
- Match existing comment density and naming (lowercase unexported, doc comments on exported symbols).

---

## File Structure

- **Create** `internal/store/store.go` — `Prefs`, `Record`, `Bucket`, dir resolution, load/save/append, `BestWPM`.
- **Create** `internal/store/store_test.go` — roundtrip, corruption, bucketing tests.
- **Create** `internal/words/decorate.go` — `Decorate` + helpers.
- **Create** `internal/words/decorate_test.go`.
- **Modify** `internal/typing/typing.go` — `Config` fields + `New` decoration wiring.
- **Modify** `internal/ui/palette.go` — toggle command kinds + entries.
- **Modify** `internal/ui/model.go` — preserve flags in `apply`, toggles, persistence, PB on results, `WithStore`.
- **Modify** `internal/ui/view.go` — PB badge + type-summary line.
- **Modify** `main.go` — flag/pref merge, `-punctuation`/`-numbers`, wire store, persist on finish.
- **Modify** `selfupdate.go` — `stats` subcommand in `dispatchCommand`.
- **Create** `internal/store/stats_render.go` — `FormatStats(history []Record) string` (pure, testable; used by the subcommand).

---

## Task 1: `internal/store` types + prefs persistence

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Produces:
  - `type Prefs struct { Mode string; TimeLimit int; WordCount int; Theme string; Punctuation bool; Numbers bool }` (json tags: `mode,time,words,theme,punctuation,numbers`)
  - `func DefaultPrefs() Prefs`
  - `func Dir() (string, error)`
  - `func LoadPrefs() Prefs`
  - `func SavePrefs(p Prefs) error`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestPrefs -v`
Expected: FAIL — `undefined: SavePrefs` etc.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/store/store.go

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store/ -run TestPrefs -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): persist preferences to config.json"
```

---

## Task 2: history records + `BestWPM` bucketing

**Files:**
- Modify: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Consumes: `Dir()` from Task 1.
- Produces:
  - `type Record struct { Time time.Time; Mode string; TimeLimit int; WordCount int; Punctuation bool; Numbers bool; WPM float64; Raw float64; Accuracy float64; Consistency float64 }` (json tags `time,mode,timeLimit,wordCount,punctuation,numbers,wpm,raw,acc,consistency`)
  - `type Bucket struct { Mode string; TimeLimit int; WordCount int; Punctuation bool; Numbers bool }`
  - `func RecordBucket(r Record) Bucket`
  - `func AppendRecord(r Record) error`
  - `func LoadHistory() []Record`
  - `func BestWPM(h []Record, b Bucket) (float64, bool)`

- [ ] **Step 1: Write the failing test**

```go
// append to internal/store/store_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run 'TestHistory|TestBestWPM' -v`
Expected: FAIL — `undefined: AppendRecord`.

- [ ] **Step 3: Write minimal implementation**

```go
// append to internal/store/store.go imports: add "bufio", "time"
// (final import block: bufio, encoding/json, os, path/filepath, time)

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store/ -v`
Expected: PASS (all tests).

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): append test history and compute per-bucket best WPM"
```

---

## Task 3: `words.Decorate`

**Files:**
- Create: `internal/words/decorate.go`
- Test: `internal/words/decorate_test.go`

**Interfaces:**
- Produces: `func Decorate(in []string, punctuation, numbers bool, rng *rand.Rand) []string`

- [ ] **Step 1: Write the failing test**

```go
// internal/words/decorate_test.go
package words

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

func TestDecorateNoFlagsIsIdentity(t *testing.T) {
	in := []string{"the", "quick", "brown"}
	out := Decorate(in, false, false, rand.New(rand.NewSource(1)))
	if strings.Join(out, " ") != strings.Join(in, " ") {
		t.Fatalf("no-flags should be identity, got %v", out)
	}
}

func TestDecorateEmptySafe(t *testing.T) {
	if out := Decorate(nil, true, true, rand.New(rand.NewSource(1))); len(out) != 0 {
		t.Fatalf("nil input should stay empty, got %v", out)
	}
}

func TestDecorateDeterministicForSeed(t *testing.T) {
	in := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}
	a := Decorate(in, true, true, rand.New(rand.NewSource(42)))
	b := Decorate(in, true, true, rand.New(rand.NewSource(42)))
	if strings.Join(a, " ") != strings.Join(b, " ") {
		t.Fatalf("same seed must produce same output:\n%v\n%v", a, b)
	}
}

func TestDecoratePunctuationCapitalizesFirst(t *testing.T) {
	in := []string{"alpha", "beta", "gamma"}
	out := Decorate(in, true, false, rand.New(rand.NewSource(7)))
	if out[0][0] < 'A' || out[0][0] > 'Z' {
		t.Fatalf("first word must be capitalized, got %q", out[0])
	}
}

func TestDecorateNumbersAreDigitsAndNotAdjacent(t *testing.T) {
	in := make([]string, 40)
	for i := range in {
		in[i] = "word"
	}
	out := Decorate(in, false, true, rand.New(rand.NewSource(3)))
	numRe := regexp.MustCompile(`^[0-9]{1,4}$`)
	prevNum := false
	sawNum := false
	for _, w := range out {
		isNum := numRe.MatchString(w)
		if isNum {
			sawNum = true
			if prevNum {
				t.Fatalf("two number tokens adjacent in %v", out)
			}
		}
		prevNum = isNum
	}
	if !sawNum {
		t.Fatalf("expected at least one number token in %v", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/words/ -run TestDecorate -v`
Expected: FAIL — `undefined: Decorate`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/words/decorate.go

package words

import (
	"math/rand"
	"strings"
)

// sentenceEnders are trailing marks that start a new sentence (next word capitalized).
const sentenceEnders = ".?!"

// trailingMarks is the weighted pool of punctuation appended to a word: commas
// and periods are most common, then sentence enders and clause marks.
var trailingMarks = []string{".", ".", ".", ",", ",", ",", "?", "!", ";", ":"}

// wraps pairs an opening and closing decoration applied around a bare word.
var wraps = [][2]string{{`"`, `"`}, {"(", ")"}, {"[", "]"}, {"-", "-"}}

// Decorate optionally rewrites generated words with monkeytype-style
// punctuation and/or numbers. With both flags false it returns in unchanged.
// rng drives all randomness so callers can make output reproducible.
func Decorate(in []string, punctuation, numbers bool, rng *rand.Rand) []string {
	if len(in) == 0 || (!punctuation && !numbers) {
		return in
	}
	out := make([]string, len(in))
	prevNum := false
	capNext := true // first word starts a "sentence"
	for i, w := range in {
		token := w
		isNum := false

		// Numbers: replace ~15% of words, never two in a row.
		if numbers && !prevNum && rng.Intn(100) < 15 {
			n := 1 + rng.Intn(4) // 1..4 digits
			b := make([]byte, n)
			for j := range b {
				b[j] = byte('0' + rng.Intn(10))
			}
			token = string(b)
			isNum = true
		}
		prevNum = isNum

		if punctuation && !isNum {
			// Capitalize sentence starts plus ~10% of other words.
			if capNext || rng.Intn(100) < 10 {
				token = capitalize(token)
			}
			capNext = false
			// ~3% wrapped.
			if rng.Intn(100) < 3 {
				wr := wraps[rng.Intn(len(wraps))]
				token = wr[0] + token + wr[1]
			}
			// ~15% trailing mark.
			if rng.Intn(100) < 15 {
				mark := trailingMarks[rng.Intn(len(trailingMarks))]
				token += mark
				if strings.ContainsAny(mark, sentenceEnders) {
					capNext = true
				}
			}
		}
		out[i] = token
	}
	return out
}

// capitalize upper-cases the first rune of s.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/words/ -v`
Expected: PASS (all decorate tests + existing words tests).

- [ ] **Step 5: Commit**

```bash
git add internal/words/decorate.go internal/words/decorate_test.go
git commit -m "feat(words): add punctuation/numbers decoration generator"
```

---

## Task 4: wire decoration into `typing.Config` + `New`

**Files:**
- Modify: `internal/typing/typing.go`
- Test: `internal/typing/typing_test.go`

**Interfaces:**
- Consumes: `words.Decorate` (Task 3), `words` package rng.
- Produces: `Config` with `Punctuation bool` and `Numbers bool` fields; `typing.New` applies decoration for time/words modes.

- [ ] **Step 1: Write the failing test**

```go
// append to internal/typing/typing_test.go
func TestNewWordsModeDecorationDoesNotChangeCount(t *testing.T) {
	e := New(Config{Mode: ModeWords, WordCount: 30, Punctuation: true, Numbers: true})
	if got := len(e.TargetWords()); got != 30 {
		t.Fatalf("decoration must preserve word count: got %d want 30", got)
	}
}

func TestNewQuoteModeIgnoresDecoration(t *testing.T) {
	// Quote text must be untouched by the flags (no crash, words present).
	e := New(Config{Mode: ModeQuote, Punctuation: true, Numbers: true})
	if len(e.TargetWords()) == 0 {
		t.Fatal("quote mode should still produce target words")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/typing/ -run 'TestNewWordsModeDecoration|TestNewQuoteModeIgnores' -v`
Expected: FAIL — `unknown field 'Punctuation' in struct literal`.

- [ ] **Step 3: Write minimal implementation**

Add the two fields to `Config` (in `internal/typing/typing.go`):

```go
// Config describes the parameters of a single test.
type Config struct {
	Mode        Mode
	TimeLimit   int // seconds, for ModeTime
	WordCount   int // words, for ModeWords
	Punctuation bool
	Numbers     bool
}
```

Apply decoration in `New` for the word-generating modes (leave `ModeQuote` alone):

```go
func New(cfg Config) *Engine {
	e := &Engine{cfg: cfg, typedWords: []string{}}
	switch cfg.Mode {
	case ModeWords:
		e.targetWords = words.Decorate(words.Random(max(cfg.WordCount, 1)), cfg.Punctuation, cfg.Numbers, words.RNG())
	case ModeQuote:
		q := quotes.Random()
		e.targetWords = strings.Fields(q.Text)
		e.quoteSource = q.Source
	default: // ModeTime
		e.targetWords = words.Decorate(words.Random(60), cfg.Punctuation, cfg.Numbers, words.RNG())
	}
	return e
}
```

Expose the package rng from `internal/words/words.go` so decoration shares the seedable source (add below `Seed`):

```go
// RNG returns the package's random source so callers (e.g. decoration) share
// the same seedable stream as word generation.
func RNG() *rand.Rand { return rng }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/typing/ ./internal/words/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/typing/typing.go internal/words/words.go internal/typing/typing_test.go
git commit -m "feat(typing): apply punctuation/numbers decoration in time/words modes"
```

---

## Task 5: palette toggle commands

**Files:**
- Modify: `internal/ui/palette.go`

**Interfaces:**
- Produces: new `cmdKind` values `cmdTogglePunct`, `cmdToggleNumbers`; palette entries `punctuation` and `numbers`.

- [ ] **Step 1: Add the command kinds**

In `internal/ui/palette.go`, extend the `cmdKind` const block:

```go
const (
	cmdModeTime cmdKind = iota
	cmdModeWords
	cmdModeQuote
	cmdTheme
	cmdRestart
	cmdTogglePunct
	cmdToggleNumbers
	cmdQuit
)
```

- [ ] **Step 2: Add palette entries**

In `newPalette`, add two entries to the `items` slice (after the `quote` mode entry, before the theme loop):

```go
		{title: "quote", group: "mode", kind: cmdModeQuote},
		{title: "punctuation", group: "toggle", kind: cmdTogglePunct},
		{title: "numbers", group: "toggle", kind: cmdToggleNumbers},
	}
```

- [ ] **Step 3: Build to verify it compiles**

Run: `go build ./...`
Expected: PASS (no usage yet of the new kinds; unused const values are legal in Go).

- [ ] **Step 4: Commit**

```bash
git add internal/ui/palette.go
git commit -m "feat(ui): add punctuation/numbers toggle commands to palette"
```

---

## Task 6: model persistence + flag-preserving apply + PB tracking

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go` (create)

**Interfaces:**
- Consumes: `store.Prefs`, `store.Record`, `store.Bucket`, `store.BestWPM`, `store.SavePrefs`, `store.AppendRecord` (Tasks 1–2); palette `cmdTogglePunct`/`cmdToggleNumbers` (Task 5); `typing.Config` flags (Task 4).
- Produces:
  - `func (m Model) WithStore(history []store.Record) Model` — enables persistence and seeds history.
  - `Model` fields `themeName string`, `history []store.Record`, `persist bool`, `priorBest float64`, `isPB bool`.
  - `func (m Model) prefs() store.Prefs`

- [ ] **Step 1: Write the failing test**

```go
// internal/ui/model_test.go
package ui

import (
	"testing"

	"github.com/ricardojparram/monkeytui/internal/store"
	"github.com/ricardojparram/monkeytui/internal/typing"
)

func TestApplyTogglePunctuationPreservesMode(t *testing.T) {
	m := New(typing.Config{Mode: typing.ModeWords, WordCount: 25}, "yellow")
	out, _ := m.apply(command{kind: cmdTogglePunct})
	got := out.(Model)
	if !got.cfg.Punctuation {
		t.Fatal("toggle should enable punctuation")
	}
	if got.cfg.Mode != typing.ModeWords || got.cfg.WordCount != 25 {
		t.Fatalf("toggle must preserve mode/count, got %+v", got.cfg)
	}
}

func TestApplyModeSwitchPreservesFlags(t *testing.T) {
	m := New(typing.Config{Mode: typing.ModeWords, WordCount: 25, Punctuation: true, Numbers: true}, "yellow")
	out, _ := m.apply(command{kind: cmdModeTime, arg: 60})
	got := out.(Model)
	if got.cfg.Mode != typing.ModeTime || got.cfg.TimeLimit != 60 {
		t.Fatalf("mode switch failed: %+v", got.cfg)
	}
	if !got.cfg.Punctuation || !got.cfg.Numbers {
		t.Fatalf("mode switch must preserve flags, got %+v", got.cfg)
	}
}

func TestPrefsSnapshot(t *testing.T) {
	m := New(typing.Config{Mode: typing.ModeTime, TimeLimit: 15, WordCount: 25, Numbers: true}, "cyan")
	p := m.prefs()
	want := store.Prefs{Mode: "time", TimeLimit: 15, WordCount: 25, Theme: "cyan", Numbers: true}
	if p != want {
		t.Fatalf("prefs snapshot: got %+v want %+v", p, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run 'TestApply|TestPrefs' -v`
Expected: FAIL — `m.prefs undefined` / toggle cases not handled.

- [ ] **Step 3: Implement**

Add imports `"github.com/ricardojparram/monkeytui/internal/store"` to `internal/ui/model.go` (alongside existing imports).

Extend the `Model` struct with new fields:

```go
type Model struct {
	eng        *typing.Engine
	cfg        typing.Config
	th         theme.Theme
	themeName  string
	state      state
	result     stats.Result
	pal        palette
	now        time.Time
	width      int
	height     int
	caretShown bool

	history   []store.Record
	persist   bool
	priorBest float64
	isPB      bool
}
```

Store the theme name in `New`:

```go
func New(cfg typing.Config, themeName string) Model {
	return Model{
		eng:        typing.New(cfg),
		cfg:        cfg,
		th:         theme.ByName(themeName),
		themeName:  themeName,
		state:      stateTyping,
		pal:        newPalette(),
		now:        time.Now(),
		caretShown: true,
	}
}
```

Add the store wiring + snapshot helpers (place after `New`):

```go
// WithStore enables disk persistence and seeds the in-memory history used for
// personal-best comparisons. main.go calls this; tests leave persistence off.
func (m Model) WithStore(history []store.Record) Model {
	m.history = history
	m.persist = true
	return m
}

// prefs snapshots the current settings for persistence.
func (m Model) prefs() store.Prefs {
	return store.Prefs{
		Mode:        m.cfg.Mode.String(),
		TimeLimit:   m.cfg.TimeLimit,
		WordCount:   m.cfg.WordCount,
		Theme:       m.themeName,
		Punctuation: m.cfg.Punctuation,
		Numbers:     m.cfg.Numbers,
	}
}

// savePrefs persists the current settings when persistence is enabled.
func (m *Model) savePrefs() {
	if m.persist {
		_ = store.SavePrefs(m.prefs())
	}
}

// bucket identifies the current test's personal-best group.
func (m Model) bucket() store.Bucket {
	return store.Bucket{
		Mode: m.cfg.Mode.String(), TimeLimit: m.cfg.TimeLimit, WordCount: m.cfg.WordCount,
		Punctuation: m.cfg.Punctuation, Numbers: m.cfg.Numbers,
	}
}
```

Update `finishToResults` to compute PB and persist the record:

```go
func (m *Model) finishToResults() {
	m.result = m.eng.Result(m.now)
	m.state = stateResults

	prev, ok := store.BestWPM(m.history, m.bucket())
	m.priorBest = prev
	m.isPB = !ok || m.result.WPM > prev

	rec := store.Record{
		Time:        m.now,
		Mode:        m.cfg.Mode.String(),
		TimeLimit:   m.cfg.TimeLimit,
		WordCount:   m.cfg.WordCount,
		Punctuation: m.cfg.Punctuation,
		Numbers:     m.cfg.Numbers,
		WPM:         m.result.WPM,
		Raw:         m.result.Raw,
		Accuracy:    m.result.Accuracy,
		Consistency: m.result.Consistency,
	}
	m.history = append(m.history, rec)
	if m.persist {
		_ = store.AppendRecord(rec)
		m.savePrefs()
	}
}
```

Rewrite `apply` to mutate config fields (preserving flags) and handle toggles + persistence:

```go
// apply runs a chosen command.
func (m Model) apply(cmd command) (tea.Model, tea.Cmd) {
	switch cmd.kind {
	case cmdModeTime:
		m.cfg.Mode = typing.ModeTime
		m.cfg.TimeLimit = cmd.arg
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdModeWords:
		m.cfg.Mode = typing.ModeWords
		m.cfg.WordCount = cmd.arg
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdModeQuote:
		m.cfg.Mode = typing.ModeQuote
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdTogglePunct:
		m.cfg.Punctuation = !m.cfg.Punctuation
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdToggleNumbers:
		m.cfg.Numbers = !m.cfg.Numbers
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdTheme:
		m.th = theme.ByName(cmd.sarg)
		m.themeName = cmd.sarg
		m.savePrefs()
		return m, nil
	case cmdRestart:
		m.restart()
		return m, tick()
	case cmdQuit:
		return m, tea.Quit
	}
	return m, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/ -run 'TestApply|TestPrefs' -v`
Expected: PASS.

- [ ] **Step 5: Run the full UI suite (no regressions)**

Run: `go test ./internal/ui/`
Expected: PASS (existing spacing/render tests unaffected; persistence is off by default so no disk writes in tests).

- [ ] **Step 6: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): persist prefs/history, preserve flags on mode switch, track personal best"
```

---

## Task 7: results screen — PB badge + type summary

**Files:**
- Modify: `internal/ui/view.go`

**Interfaces:**
- Consumes: `Model.isPB`, `Model.priorBest`, `Model.cfg` flags (Task 6).
- Produces: `func typeSummary(cfg typing.Config) string` (mode label + active toggles).

- [ ] **Step 1: Add the type-summary helper**

Add near `modeLabel` in `internal/ui/view.go`:

```go
// typeSummary describes the test on the results screen: mode plus any active
// decoration toggles, e.g. "time 30 punctuation numbers".
func typeSummary(cfg typing.Config) string {
	s := modeLabel(cfg)
	if cfg.Mode != typing.ModeQuote {
		if cfg.Punctuation {
			s += " punctuation"
		}
		if cfg.Numbers {
			s += " numbers"
		}
	}
	return s
}
```

- [ ] **Step 2: Show PB + summary in the headline column**

In `renderResults`, replace the `left := lipgloss.JoinVertical(...)` block with one that appends a best/PB line under the wpm stat and uses `typeSummary`:

```go
	// Personal-best line under the wpm headline.
	var pbLine string
	if m.isPB {
		pbLine = th.Accent.Render("new pb")
	} else if m.priorBest > 0 {
		pbLine = th.Faint.Render(fmt.Sprintf("best %.0f", m.priorBest))
	}

	left := lipgloss.JoinVertical(lipgloss.Left,
		m.bigStat("wpm", fmt.Sprintf("%.0f", r.WPM)),
		pbLine,
		m.bigStat("acc", fmt.Sprintf("%.0f%%", r.Accuracy)),
		"",
		th.Faint.Render("test type"),
		th.Text.Render(typeSummary(m.cfg)),
		th.Text.Render(m.wordlistName()),
	)
```

(Note: `th.Accent` is a `lipgloss.Color`; if `Accent` is not a renderable style in `theme.Theme`, use `th.Main.Bold(true).Render("new pb")` instead — match whatever the theme exposes. Check `internal/theme/theme.go` for the field; `th.Main` is known to render.)

- [ ] **Step 3: Build + run UI tests**

Run: `go build ./... && go test ./internal/ui/`
Expected: PASS.

- [ ] **Step 4: Manual smoke (optional but recommended)**

Run: `go run . -mode words -words 10 -punctuation -seed 1`
Type the test; confirm the results screen shows `new pb` the first time and the type line reads `words 10 punctuation`.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/view.go
git commit -m "feat(ui): show personal-best badge and toggle summary on results"
```

---

## Task 8: main.go — merge prefs with flags + new flags + wire store

**Files:**
- Modify: `main.go`
- Test: `main_test.go` (create)

**Interfaces:**
- Consumes: `store.LoadPrefs`, `store.LoadHistory`, `Model.WithStore` (Tasks 1,2,6), `typing.Config` flags (Task 4).
- Produces: `func mergeConfig(p store.Prefs, set map[string]bool, mode string, t, count int, punct, nums bool) (typing.Config, error)` and `func resolveTheme(p store.Prefs, set map[string]bool, flagTheme string) string` — pure helpers for testing flag-over-pref precedence.

- [ ] **Step 1: Write the failing test**

```go
// main_test.go
package main

import (
	"testing"

	"github.com/ricardojparram/monkeytui/internal/store"
	"github.com/ricardojparram/monkeytui/internal/typing"
)

func TestMergeConfigPrecedence(t *testing.T) {
	p := store.Prefs{Mode: "words", TimeLimit: 60, WordCount: 50, Theme: "cyan", Punctuation: true}

	// No flags set on the command line: prefs win.
	cfg, err := mergeConfig(p, map[string]bool{}, "time", 30, 25, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != typing.ModeWords || cfg.WordCount != 50 || !cfg.Punctuation {
		t.Fatalf("unset flags must keep prefs, got %+v", cfg)
	}

	// Explicit -mode time -time 15 overrides the saved words pref.
	cfg, err = mergeConfig(p, map[string]bool{"mode": true, "time": true}, "time", 15, 25, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != typing.ModeTime || cfg.TimeLimit != 15 {
		t.Fatalf("explicit flags must override prefs, got %+v", cfg)
	}
	if !cfg.Punctuation {
		t.Fatalf("unset -punctuation must keep saved pref true, got %+v", cfg)
	}
}

func TestResolveThemePrecedence(t *testing.T) {
	p := store.Prefs{Theme: "cyan"}
	if got := resolveTheme(p, map[string]bool{}, "yellow"); got != "cyan" {
		t.Fatalf("unset -theme keeps pref: got %q", got)
	}
	if got := resolveTheme(p, map[string]bool{"theme": true}, "red"); got != "red" {
		t.Fatalf("explicit -theme wins: got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test . -run 'TestMergeConfig|TestResolveTheme' -v`
Expected: FAIL — `undefined: mergeConfig`.

- [ ] **Step 3: Implement the helpers**

Add to `main.go` (above `main`), importing `"github.com/ricardojparram/monkeytui/internal/store"`:

```go
// mergeConfig builds the starting test config from saved prefs, overriding a
// field only when its flag was explicitly set on the command line (set[name]).
func mergeConfig(p store.Prefs, set map[string]bool, mode string, t, count int, punct, nums bool) (typing.Config, error) {
	cfg := typing.Config{
		TimeLimit:   p.TimeLimit,
		WordCount:   p.WordCount,
		Punctuation: p.Punctuation,
		Numbers:     p.Numbers,
	}
	modeStr := p.Mode
	if set["mode"] {
		modeStr = mode
	}
	if set["time"] {
		cfg.TimeLimit = t
	}
	if set["words"] {
		cfg.WordCount = count
	}
	if set["punctuation"] {
		cfg.Punctuation = punct
	}
	if set["numbers"] {
		cfg.Numbers = nums
	}
	switch modeStr {
	case "words":
		cfg.Mode = typing.ModeWords
	case "quote":
		cfg.Mode = typing.ModeQuote
	case "time":
		cfg.Mode = typing.ModeTime
	default:
		return typing.Config{}, fmt.Errorf("unknown mode %q (use time|words|quote)", modeStr)
	}
	return cfg, nil
}

// resolveTheme returns the explicit -theme flag if set, else the saved pref.
func resolveTheme(p store.Prefs, set map[string]bool, flagTheme string) string {
	if set["theme"] {
		return flagTheme
	}
	if p.Theme != "" {
		return p.Theme
	}
	return flagTheme
}
```

- [ ] **Step 4: Run helper tests to verify they pass**

Run: `go test . -run 'TestMergeConfig|TestResolveTheme' -v`
Expected: PASS.

- [ ] **Step 5: Wire the helpers into `main`**

Add the new flags and replace the config-construction block in `main`:

```go
	mode := flag.String("mode", "time", "test mode: time | words | quote")
	t := flag.Int("time", 30, "seconds for time mode")
	count := flag.Int("words", 25, "word count for words mode")
	themeName := flag.String("theme", "yellow", "accent theme: yellow green cyan magenta blue red")
	seed := flag.Int64("seed", 0, "fixed RNG seed for reproducible words (0 = random)")
	punct := flag.Bool("punctuation", false, "mix in punctuation (time/words modes)")
	nums := flag.Bool("numbers", false, "mix in numbers (time/words modes)")
	// ... flag.Usage unchanged ...
	flag.Parse()

	if *seed != 0 {
		words.Seed(*seed)
	}

	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	prefs := store.LoadPrefs()
	cfg, err := mergeConfig(prefs, set, *mode, *t, *count, *punct, *nums)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	themeChoice := resolveTheme(prefs, set, *themeName)

	model := ui.New(cfg, themeChoice).WithStore(store.LoadHistory())
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
```

Remove the old `cfg := typing.Config{...}` + `switch *mode` block (replaced above). Keep the `typing` import only if still referenced — it is, via `mergeConfig`'s return type.

- [ ] **Step 6: Build + full test suite**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: merge saved prefs with explicit flags; add -punctuation/-numbers; load history"
```

---

## Task 9: `stats` subcommand

**Files:**
- Create: `internal/store/stats_render.go`
- Create: `internal/store/stats_render_test.go`
- Modify: `selfupdate.go`
- Modify: `main.go` (usage text)

**Interfaces:**
- Consumes: `Record`, `LoadHistory` (Tasks 1–2).
- Produces: `func FormatStats(h []Record) string`; `dispatchCommand` handles `stats`.

- [ ] **Step 1: Write the failing test**

```go
// internal/store/stats_render_test.go
package store

import (
	"strings"
	"testing"
)

func TestFormatStatsEmpty(t *testing.T) {
	out := FormatStats(nil)
	if !strings.Contains(out, "no tests recorded") {
		t.Fatalf("empty history message missing: %q", out)
	}
}

func TestFormatStatsSummarizesPerMode(t *testing.T) {
	h := []Record{
		{Mode: "time", TimeLimit: 30, WPM: 70, Accuracy: 95},
		{Mode: "time", TimeLimit: 30, WPM: 90, Accuracy: 97},
		{Mode: "words", WordCount: 25, WPM: 100, Accuracy: 99},
	}
	out := FormatStats(h)
	if !strings.Contains(out, "time") || !strings.Contains(out, "words") {
		t.Fatalf("missing mode rows: %q", out)
	}
	if !strings.Contains(out, "90") { // best time WPM
		t.Fatalf("missing best time wpm: %q", out)
	}
	if !strings.Contains(out, "3 tests") {
		t.Fatalf("missing total count: %q", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestFormatStats -v`
Expected: FAIL — `undefined: FormatStats`.

- [ ] **Step 3: Implement**

```go
// internal/store/stats_render.go

package store

import (
	"fmt"
	"sort"
	"strings"
)

// FormatStats renders a plain-text summary of recorded tests for the CLI
// `monkeytui stats` subcommand.
func FormatStats(h []Record) string {
	if len(h) == 0 {
		return "no tests recorded yet — run monkeytui to start typing.\n"
	}
	type agg struct {
		count   int
		best    float64
		sumWPM  float64
		sumAcc  float64
	}
	byMode := map[string]*agg{}
	order := []string{"time", "words", "quote"}
	for _, r := range h {
		a := byMode[r.Mode]
		if a == nil {
			a = &agg{}
			byMode[r.Mode] = a
		}
		a.count++
		a.sumWPM += r.WPM
		a.sumAcc += r.Accuracy
		if r.WPM > a.best {
			a.best = r.WPM
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "monkeytui — %d tests recorded\n\n", len(h))
	fmt.Fprintf(&b, "%-8s %6s %8s %8s %8s\n", "mode", "tests", "best", "avg wpm", "avg acc")
	for _, mode := range order {
		a := byMode[mode]
		if a == nil {
			continue
		}
		fmt.Fprintf(&b, "%-8s %6d %8.0f %8.0f %7.0f%%\n",
			mode, a.count, a.best, a.sumWPM/float64(a.count), a.sumAcc/float64(a.count))
	}

	// Recent runs (last 10, newest first).
	b.WriteString("\nrecent:\n")
	recent := append([]Record(nil), h...)
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].Time.After(recent[j].Time) })
	if len(recent) > 10 {
		recent = recent[:10]
	}
	for _, r := range recent {
		label := r.Mode
		switch r.Mode {
		case "time":
			label = fmt.Sprintf("time %d", r.TimeLimit)
		case "words":
			label = fmt.Sprintf("words %d", r.WordCount)
		}
		fmt.Fprintf(&b, "  %-12s %4.0f wpm  %3.0f%%\n", label, r.WPM, r.Accuracy)
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store/ -run TestFormatStats -v`
Expected: PASS.

- [ ] **Step 5: Wire the subcommand**

In `selfupdate.go`, add a case to the `switch args[0]` in `dispatchCommand`:

```go
	case "stats", "-stats", "--stats":
		fmt.Print(store.FormatStats(store.LoadHistory()))
```

Add the import `"github.com/ricardojparram/monkeytui/internal/store"` to `selfupdate.go`.

In `main.go`, add a usage line under the existing subcommands in `flag.Usage`:

```go
				"  monkeytui stats             show typing history summary\n"+
```

- [ ] **Step 6: Build + full suite + manual check**

Run: `go build ./... && go test ./...`
Expected: PASS.

Run: `go run . stats`
Expected: either the "no tests recorded yet" line, or a populated table if you ran tests earlier in Task 7's smoke step.

- [ ] **Step 7: Commit**

```bash
git add internal/store/stats_render.go internal/store/stats_render_test.go selfupdate.go main.go
git commit -m "feat: add monkeytui stats subcommand for history summary"
```

---

## Task 10: docs — README flags, keys, stats

**Files:**
- Modify: `README.md`

**Interfaces:** none (documentation).

- [ ] **Step 1: Update the flags table**

In `README.md`, add rows to the flags table:

```markdown
| `-punctuation` | `false` | mix in punctuation (time/words modes) |
| `-numbers`     | `false` | mix in numbers (time/words modes) |
```

- [ ] **Step 2: Document persistence + stats**

Add a short section after "Usage":

```markdown
### History & settings

monkeytui remembers your last mode, duration, word count, theme, and
punctuation/numbers toggles between runs (stored in your OS config dir, e.g.
`~/.config/monkeytui/`). Explicit flags always override the saved settings.

Every finished test is recorded; see a summary with:

```sh
monkeytui stats
```

The results screen shows `new pb` when you beat your best for the current mode.
```

Add the palette toggles to the Keys/command-palette note:

```markdown
In the command palette, `punctuation` and `numbers` toggle word decoration.
```

- [ ] **Step 3: Verify build still clean + commit**

Run: `go build ./... && go test ./...`
Expected: PASS.

```bash
git add README.md
git commit -m "docs: document persistence, stats subcommand, punctuation/numbers flags"
```

---

## Self-Review Notes

- **Spec coverage:** config persistence (Tasks 1,6,8) · history+records (Tasks 2,6,7,9) · punctuation/numbers (Tasks 3,4,5,6,7,8) · `stats` subcommand (Task 9) · resilience (Task 1/2 tests) · `flag.Visit` precedence (Task 8) · docs (Task 10). All spec sections mapped.
- **Type consistency:** `store.Prefs`/`Record`/`Bucket` field names identical across Tasks 1–9; `mergeConfig`/`resolveTheme` signatures match their tests; `WithStore`/`prefs`/`bucket` consistent in Tasks 6–8.
- **Known follow-up to verify during execution:** Task 7 references `th.Accent` — confirm against `internal/theme/theme.go`; fall back to `th.Main.Bold(true)` if `Accent` is a color, not a style. This is the only place needing a live check.
