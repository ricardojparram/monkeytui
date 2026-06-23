package store

import (
	"fmt"
	"sort"
	"strings"
)

// ModeStat is the aggregated performance for one test mode.
type ModeStat struct {
	Mode   string
	Count  int
	Best   float64
	AvgWPM float64
	AvgAcc float64
}

// Summary is the structured view of recorded history shared by the CLI
// `stats` subcommand and the in-TUI stats screen.
type Summary struct {
	Total  int
	Modes  []ModeStat // ordered time, words, quote; only modes with records
	Recent []Record   // newest first, capped at 10
}

// RecordLabel describes a single run, e.g. "time 30" / "words 25" / "quote".
func RecordLabel(r Record) string {
	switch r.Mode {
	case "time":
		return fmt.Sprintf("time %d", r.TimeLimit)
	case "words":
		return fmt.Sprintf("words %d", r.WordCount)
	default:
		return r.Mode
	}
}

// Summarize aggregates history into per-mode stats and the most recent runs.
func Summarize(h []Record) Summary {
	type agg struct {
		count  int
		best   float64
		sumWPM float64
		sumAcc float64
	}
	byMode := map[string]*agg{}
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

	var modes []ModeStat
	for _, mode := range []string{"time", "words", "quote"} {
		a := byMode[mode]
		if a == nil {
			continue
		}
		modes = append(modes, ModeStat{
			Mode:   mode,
			Count:  a.count,
			Best:   a.best,
			AvgWPM: a.sumWPM / float64(a.count),
			AvgAcc: a.sumAcc / float64(a.count),
		})
	}

	recent := append([]Record(nil), h...)
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].Time.After(recent[j].Time) })
	if len(recent) > 10 {
		recent = recent[:10]
	}

	return Summary{Total: len(h), Modes: modes, Recent: recent}
}

// FormatStats renders a plain-text summary of recorded tests for the CLI
// `monkeytui stats` subcommand.
func FormatStats(h []Record) string {
	if len(h) == 0 {
		return "no tests recorded yet — run monkeytui to start typing.\n"
	}
	s := Summarize(h)

	var b strings.Builder
	fmt.Fprintf(&b, "monkeytui — %d tests recorded\n\n", s.Total)
	fmt.Fprintf(&b, "%-8s %6s %8s %8s %8s\n", "mode", "tests", "best", "avg wpm", "avg acc")
	for _, m := range s.Modes {
		fmt.Fprintf(&b, "%-8s %6d %8.0f %8.0f %7.0f%%\n",
			m.Mode, m.Count, m.Best, m.AvgWPM, m.AvgAcc)
	}

	b.WriteString("\nrecent:\n")
	for _, r := range s.Recent {
		fmt.Fprintf(&b, "  %-12s %4.0f wpm  %3.0f%%\n", RecordLabel(r), r.WPM, r.Accuracy)
	}
	return b.String()
}
