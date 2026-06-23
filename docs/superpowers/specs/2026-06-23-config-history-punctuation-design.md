# monkeytui — config persistence, history/records, punctuation & numbers

Date: 2026-06-23
Status: Approved design

## Goal

Close three gaps versus monkeytype, all sharing one on-disk data directory:

1. **Config persistence** — remember the last-used settings between launches.
2. **History + personal records** — store every finished test, surface bests.
3. **Punctuation & numbers** — optional decoration of generated words.

## Non-goals

- No cloud sync, no accounts.
- No live WPM / progress bar during the test (explicitly deferred).
- No new languages or custom-text mode.
- Quote mode is never decorated (fixed text) and is excluded from punctuation/numbers.

## Data directory

Resolved once via `os.UserConfigDir()` joined with `monkeytui/`
(e.g. `~/.config/monkeytui/` on Linux). Created lazily on first write.

Two files:

- `config.json` — persisted preferences (small, rewritten whole).
- `history.jsonl` — append-only, one JSON object per line per finished test.

### Resilience

All store I/O is best-effort and must never crash a test:

- Missing or unreadable `config.json` → in-memory defaults.
- Corrupt `config.json` (bad JSON) → defaults, file overwritten on next save.
- Unreadable `history.jsonl` → treated as empty history.
- A malformed line in `history.jsonl` → that line skipped, others still parsed.
- Any write error → logged to nothing (silently ignored); the test continues.

## Package: `internal/store`

New package owning all persistence. No dependency on `internal/ui`.

```go
type Prefs struct {
    Mode        string `json:"mode"`        // "time" | "words" | "quote"
    TimeLimit   int    `json:"time"`
    WordCount   int    `json:"words"`
    Theme       string `json:"theme"`
    Punctuation bool   `json:"punctuation"`
    Numbers     bool   `json:"numbers"`
}

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

func Dir() (string, error)              // resolve + ensure data dir
func LoadPrefs() Prefs                  // defaults on any failure
func SavePrefs(p Prefs) error           // atomic-ish whole-file write
func AppendRecord(r Record) error       // append one JSONL line
func LoadHistory() []Record             // skips malformed lines
func BestWPM(h []Record, bucket Bucket) (float64, bool) // personal best for a bucket
```

`Bucket` identifies a comparable group of tests: `(Mode, TimeLimit, WordCount,
Punctuation, Numbers)`. For time mode the WordCount field is ignored; for words
mode the TimeLimit is ignored; quote buckets ignore both numeric fields.

Default `Prefs`: mirror current CLI defaults — `time` mode, 30s, 25 words,
`yellow`, punctuation off, numbers off.

## Feature 1 — config persistence

Load/override precedence in `main.go`:

1. Start from `store.LoadPrefs()`.
2. Apply a CLI flag **only if the user explicitly set it** (detected via
   `flag.Visit`, which reports only flags present on the command line). This
   keeps unspecified flags from clobbering saved prefs with their defaults.
3. Build the initial `typing.Config` + theme from the merged result.

Persistence writes:

- After every palette `apply` in the UI (mode/time/words/theme/toggle change).
- After each finished test (captures the final settings).

The UI needs a way to persist. Inject a save callback (or a `store.Prefs`
snapshot built from `Model`) so `internal/ui` stays decoupled from disk: the UI
calls a `func(store.Prefs)` held on the `Model`, wired up in `main.go`. The
callback is a no-op in tests.

`-seed` is not persisted (it is a one-shot reproducibility flag).

## Feature 2 — history + records

On `finishToResults`, build a `store.Record` from the engine `Result` plus the
active config and call `store.AppendRecord`.

### Results screen

Compute the personal best for the current bucket from history loaded at startup
(plus the just-finished run). Display below/next to the headline WPM:

- `best 92` (faint) when a prior best exists.
- `NEW PB` badge (accent color) when this run's WPM exceeds the prior best, or
  on the first-ever run for the bucket.

History is loaded once at program start and the new record appended in memory so
the results screen reflects it without re-reading the file.

### `monkeytui stats` subcommand

Added to the existing `dispatchCommand` path in `selfupdate.go`/`main.go`
(runs before flag parsing, like `version`). Prints a plain-text table to stdout:

- Total tests, total time typed.
- Per mode (`time` / `words` / `quote`): test count, best WPM, average WPM,
  average accuracy.
- Last N (e.g. 10) runs: date, mode summary, wpm, acc.

Empty history → friendly "no tests recorded yet" line.

## Feature 3 — punctuation & numbers

`typing.Config` gains `Punctuation bool` and `Numbers bool`.

New generator in `internal/words`:

```go
func Decorate(in []string, punctuation, numbers bool, rng *rand.Rand) []string
```

Called from `typing.New` for `ModeTime` and `ModeWords` after `words.Random`,
never for `ModeQuote`. Uses the package `rng` (so `-seed` keeps decoration
reproducible); `Decorate` takes an explicit `*rand.Rand` for testability.

### Punctuation rules (approximate monkeytype feel)

Applied per word with independent probabilities:

- Capitalize the first letter of the first word, and of any word following
  sentence-ending punctuation, plus ~10% of other words.
- ~15% of words get a trailing mark, weighted toward `.` then `,` then
  `?!;:`.
- ~3% of words get wrapped: `"word"`, `(word)`, `[word]`, `-word-`.
- Order: wrapping applies to the bare word; trailing punctuation goes outside
  any closing wrap; capitalization applies to the leading letter.

### Numbers rules

- ~15% of words replaced by a random integer string of 1–4 digits.
- Replacement happens before punctuation so a number can still get a trailing
  mark.
- Never two number tokens in a row.

Decoration changes target text only; keystroke handling, scoring, and rendering
are unaffected (they already operate on arbitrary runes).

### Surfacing

- Palette: two new toggle commands, `punctuation` and `numbers`, that flip the
  bool and restart the test. Reflected in the test-type summary on the results
  screen (e.g. `time 30 punctuation numbers`).
- Flags: `-punctuation` and `-numbers` (bool). Persisted in `config.json`.

## Testing

- `store`: prefs save/load roundtrip; corrupt-file → defaults; history append +
  load skips a malformed line; `BestWPM` bucketing (time vs words vs flags).
- `main` precedence: explicitly-set flag overrides saved pref; unset flag keeps
  saved pref (via `flag.Visit` logic, unit-tested as a pure helper).
- `words.Decorate`: deterministic output for a fixed seed; punctuation only adds
  expected marks; numbers tokens are digits; no two numbers adjacent; empty
  input safe.
- `BestWPM`/record building from a `stats.Result` stays consistent with existing
  stat tests.

## Rollout / files touched

- New: `internal/store/store.go`, `internal/store/store_test.go`.
- New: `internal/words` decorator + tests (extend `words.go` / new file).
- Edit: `internal/typing/typing.go` (`Config` fields, `New` decoration).
- Edit: `internal/ui/model.go` (toggle commands, save callback, PB on results).
- Edit: `internal/ui/view.go` (PB badge, type-summary string).
- Edit: `main.go` (flag merge with prefs, `-punctuation`/`-numbers`, wire save
  callback) and the `dispatchCommand` path (`stats` subcommand).
