// Package typing implements the core typing-test engine: target generation,
// keystroke handling (per-word, monkeytype-style), timing and metric capture.
package typing

import (
	"strings"
	"time"

	"github.com/ricardojparram/monkeytui/internal/quotes"
	"github.com/ricardojparram/monkeytui/internal/stats"
	"github.com/ricardojparram/monkeytui/internal/words"
)

// Mode selects how a test is bounded.
type Mode int

const (
	ModeTime Mode = iota
	ModeWords
	ModeQuote
)

func (m Mode) String() string {
	switch m {
	case ModeTime:
		return "time"
	case ModeWords:
		return "words"
	case ModeQuote:
		return "quote"
	}
	return "?"
}

// Config describes the parameters of a single test.
type Config struct {
	Mode      Mode
	TimeLimit int // seconds, for ModeTime
	WordCount int // words, for ModeWords
}

// Engine holds the mutable state of an in-progress test.
type Engine struct {
	cfg Config

	targetWords []string
	typedWords  []string // committed words (index < cur)
	cur         int
	curInput    []rune

	quoteSource string

	started   bool
	startTime time.Time
	finished  bool

	keystrokes        int
	correctKeystrokes int

	samples []stats.Sample
	lastErr int // committed errors at last sample (for per-second delta)
}

// New builds a fresh engine for the given config.
func New(cfg Config) *Engine {
	e := &Engine{cfg: cfg, typedWords: []string{}}
	switch cfg.Mode {
	case ModeWords:
		e.targetWords = words.Random(max(cfg.WordCount, 1))
	case ModeQuote:
		q := quotes.Random()
		e.targetWords = strings.Fields(q.Text)
		e.quoteSource = q.Source
	default: // ModeTime
		e.targetWords = words.Random(60)
	}
	return e
}

// --- accessors for rendering ---

func (e *Engine) TargetWords() []string { return e.targetWords }
func (e *Engine) TypedWords() []string  { return e.typedWords }
func (e *Engine) Cur() int              { return e.cur }
func (e *Engine) CurInput() []rune      { return e.curInput }
func (e *Engine) Config() Config        { return e.cfg }
func (e *Engine) QuoteSource() string   { return e.quoteSource }
func (e *Engine) Started() bool         { return e.started }
func (e *Engine) StartTime() time.Time  { return e.startTime }
func (e *Engine) Finished() bool        { return e.finished }

// Elapsed returns seconds since the first keystroke (0 before start).
func (e *Engine) Elapsed(now time.Time) float64 {
	if !e.started {
		return 0
	}
	return now.Sub(e.startTime).Seconds()
}

// Remaining returns seconds left in a timed test (0 for other modes).
func (e *Engine) Remaining(now time.Time) float64 {
	if e.cfg.Mode != ModeTime {
		return 0
	}
	r := float64(e.cfg.TimeLimit) - e.Elapsed(now)
	if r < 0 {
		return 0
	}
	return r
}

func (e *Engine) start(now time.Time) {
	if !e.started {
		e.started = true
		e.startTime = now
	}
}

// Type registers a printable rune keystroke.
func (e *Engine) Type(now time.Time, r rune) {
	if e.finished {
		return
	}
	e.start(now)
	target := []rune(e.targetWords[e.cur])
	if len(e.curInput) < len(target) && r == target[len(e.curInput)] {
		e.correctKeystrokes++
	}
	e.keystrokes++
	e.curInput = append(e.curInput, r)
	e.ensureBuffer()
	e.checkFinish(now)
}

// Space commits the current word and advances. Leading spaces are ignored.
func (e *Engine) Space(now time.Time) {
	if e.finished || len(e.curInput) == 0 {
		return
	}
	e.start(now)
	e.keystrokes++
	e.correctKeystrokes++ // a word-separating space is always valid

	e.typedWords = append(e.typedWords, string(e.curInput))
	e.cur++
	e.curInput = nil
	e.ensureBuffer()
	e.checkFinish(now)
}

// Backspace deletes the last rune, stepping into the previous word if empty.
func (e *Engine) Backspace() {
	if e.finished {
		return
	}
	if len(e.curInput) > 0 {
		e.curInput = e.curInput[:len(e.curInput)-1]
		return
	}
	if e.cur > 0 {
		e.cur--
		e.curInput = []rune(e.typedWords[e.cur])
		e.typedWords = e.typedWords[:e.cur]
	}
}

// ensureBuffer keeps timed tests stocked with upcoming words.
func (e *Engine) ensureBuffer() {
	if e.cfg.Mode == ModeTime && e.cur >= len(e.targetWords)-5 {
		e.targetWords = append(e.targetWords, words.Random(30)...)
	}
}

// checkFinish marks word/quote tests complete once the final word is filled.
func (e *Engine) checkFinish(now time.Time) {
	if e.cfg.Mode == ModeTime {
		return
	}
	last := len(e.targetWords) - 1
	if e.cur == last && len(e.curInput) >= len([]rune(e.targetWords[last])) {
		e.finish(now)
	}
}

func (e *Engine) finish(now time.Time) {
	if !e.finished {
		e.finished = true
	}
}

// Tick advances a timed test and returns true if it just finished.
func (e *Engine) Tick(now time.Time) bool {
	if e.cfg.Mode == ModeTime && e.started && !e.finished && e.Elapsed(now) >= float64(e.cfg.TimeLimit) {
		e.finish(now)
		return true
	}
	return false
}

// counts scans committed and current input for character-level metrics.
func (e *Engine) counts() (correct, incorrect, extra, missed, typed int) {
	scan := func(in, tgt []rune, committed bool) {
		n := min(len(in), len(tgt))
		for i := 0; i < n; i++ {
			if in[i] == tgt[i] {
				correct++
			} else {
				incorrect++
			}
		}
		if len(in) > len(tgt) {
			extra += len(in) - len(tgt)
		} else if committed {
			missed += len(tgt) - len(in)
		}
		typed += len(in)
	}
	for i := 0; i < e.cur; i++ {
		scan([]rune(e.typedWords[i]), []rune(e.targetWords[i]), true)
	}
	if e.cur < len(e.targetWords) {
		scan(e.curInput, []rune(e.targetWords[e.cur]), false)
	}
	correct += e.cur // committed spaces
	typed += e.cur
	return
}

// Sample captures a per-second data point for the results chart.
func (e *Engine) Sample(now time.Time) {
	if !e.started || e.finished {
		return
	}
	sec := e.Elapsed(now)
	if sec <= 0 {
		return
	}
	correct, incorrect, extra, _, typed := e.counts()
	errs := incorrect + extra
	e.samples = append(e.samples, stats.Sample{
		T:      sec,
		WPM:    stats.WPM(correct, sec),
		Raw:    stats.Raw(typed, sec),
		Errors: errs - e.lastErr,
	})
	e.lastErr = errs
}

// Result computes the final immutable summary of the test.
func (e *Engine) Result(now time.Time) stats.Result {
	sec := e.Elapsed(now)
	if e.cfg.Mode == ModeTime {
		sec = float64(e.cfg.TimeLimit)
	}
	correct, incorrect, extra, missed, typed := e.counts()
	return stats.Result{
		WPM:         stats.WPM(correct, sec),
		Raw:         stats.Raw(typed, sec),
		Accuracy:    stats.Accuracy(e.correctKeystrokes, e.keystrokes),
		Consistency: stats.Consistency(e.samples),
		Seconds:     sec,
		Correct:     correct,
		Incorrect:   incorrect,
		Extra:       extra,
		Missed:      missed,
		Keystrokes:  e.keystrokes,
		Samples:     e.samples,
	}
}
