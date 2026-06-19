// monkeytui is a minimalist monkeytype-style typing test for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ricardojparram/monkeytui/internal/typing"
	"github.com/ricardojparram/monkeytui/internal/ui"
)

func main() {
	mode := flag.String("mode", "time", "test mode: time | words | quote")
	t := flag.Int("time", 30, "seconds for time mode")
	count := flag.Int("words", 25, "word count for words mode")
	themeName := flag.String("theme", "yellow", "accent theme: yellow green cyan magenta blue red")
	flag.Parse()

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
