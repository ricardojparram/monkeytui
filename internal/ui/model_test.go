package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestStatsScreenOpensAndClosesToPriorState(t *testing.T) {
	m := New(typing.Config{Mode: typing.ModeTime, TimeLimit: 30}, "yellow")
	m.state = stateResults // pretend the palette was opened from the results screen

	out, _ := m.apply(command{kind: cmdShowStats})
	opened := out.(Model)
	if opened.state != stateStats {
		t.Fatalf("apply cmdShowStats: state = %v want stateStats", opened.state)
	}
	if opened.prevState != stateResults {
		t.Fatalf("prevState = %v want stateResults", opened.prevState)
	}

	closed, _ := opened.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if got := closed.(Model).state; got != stateResults {
		t.Fatalf("any key should return to prior state, got %v", got)
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
