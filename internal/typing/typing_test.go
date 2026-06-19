package typing

import (
	"testing"
	"time"
)

// typeString feeds a full string (spaces commit words) into the engine.
func typeString(e *Engine, s string, base time.Time) {
	for _, r := range s {
		if r == ' ' {
			e.Space(base)
		} else {
			e.Type(base, r)
		}
	}
}

func TestPerfectWordsRun(t *testing.T) {
	e := New(Config{Mode: ModeWords, WordCount: 3})
	target := e.TargetWords()
	base := time.Now()
	full := target[0] + " " + target[1] + " " + target[2]
	typeString(e, full, base)

	if !e.Finished() {
		t.Fatalf("expected finished after typing all 3 words")
	}
	now := base.Add(2 * time.Second)
	r := e.Result(now)
	if r.Incorrect != 0 || r.Extra != 0 || r.Missed != 0 {
		t.Errorf("perfect run should have no errors, got %+v", r)
	}
	if r.Accuracy != 100 {
		t.Errorf("accuracy = %v, want 100", r.Accuracy)
	}
}

func TestBackspaceIntoPreviousWord(t *testing.T) {
	e := New(Config{Mode: ModeWords, WordCount: 2})
	base := time.Now()
	e.Type(base, 'x') // wrong-ish char into first word
	e.Space(base)
	if e.Cur() != 1 {
		t.Fatalf("cur = %d, want 1", e.Cur())
	}
	e.Backspace() // curInput empty -> step back into word 0
	if e.Cur() != 0 {
		t.Fatalf("after backspace cur = %d, want 0", e.Cur())
	}
	if string(e.CurInput()) != "x" {
		t.Fatalf("curInput = %q, want \"x\"", string(e.CurInput()))
	}
}

func TestTimeModeFinishesOnTick(t *testing.T) {
	e := New(Config{Mode: ModeTime, TimeLimit: 5})
	base := time.Now()
	e.Type(base, []rune(e.TargetWords()[0])[0]) // start the clock
	if e.Tick(base.Add(2 * time.Second)) {
		t.Fatal("should not finish before limit")
	}
	if !e.Tick(base.Add(6 * time.Second)) {
		t.Fatal("should finish after limit")
	}
	if !e.Finished() {
		t.Fatal("engine not marked finished")
	}
}
