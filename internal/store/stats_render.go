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
		count  int
		best   float64
		sumWPM float64
		sumAcc float64
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
