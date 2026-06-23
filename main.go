// monkeytui is a minimalist monkeytype-style typing test for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ricardojparram/monkeytui/internal/store"
	"github.com/ricardojparram/monkeytui/internal/typing"
	"github.com/ricardojparram/monkeytui/internal/ui"
	"github.com/ricardojparram/monkeytui/internal/words"
)

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

func main() {
	// Subcommands (update / uninstall / version) run before flag parsing.
	if dispatchCommand(os.Args[1:]) {
		return
	}

	mode := flag.String("mode", "time", "test mode: time | words | quote")
	t := flag.Int("time", 30, "seconds for time mode")
	count := flag.Int("words", 25, "word count for words mode")
	themeName := flag.String("theme", "yellow", "accent theme: yellow green cyan magenta blue red")
	seed := flag.Int64("seed", 0, "fixed RNG seed for reproducible words (0 = random)")
	punct := flag.Bool("punctuation", false, "mix in punctuation (time/words modes)")
	nums := flag.Bool("numbers", false, "mix in numbers (time/words modes)")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			"monkeytui — minimalist terminal typing test\n\n"+
				"Usage:\n"+
				"  monkeytui [flags]            start a typing test\n"+
				"  monkeytui update            update to the latest release\n"+
				"  monkeytui uninstall         remove the installed binary\n"+
				"  monkeytui version           print the version\n\n"+
				"Flags:\n")
		flag.PrintDefaults()
	}
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
}
