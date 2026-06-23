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
