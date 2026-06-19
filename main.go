// monkeytui is a minimalist monkeytype-style typing test for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ricardojparram/monkeytui/internal/typing"
	"github.com/ricardojparram/monkeytui/internal/ui"
	"github.com/ricardojparram/monkeytui/internal/words"
)

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

	cfg := typing.Config{TimeLimit: *t, WordCount: *count}
	switch *mode {
	case "words":
		cfg.Mode = typing.ModeWords
	case "quote":
		cfg.Mode = typing.ModeQuote
	case "time":
		cfg.Mode = typing.ModeTime
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (use time|words|quote)\n", *mode)
		os.Exit(2)
	}

	p := tea.NewProgram(ui.New(cfg, *themeName), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
