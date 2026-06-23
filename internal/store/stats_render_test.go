package store

import (
	"strings"
	"testing"
	"time"
)

// timeSeq returns a function mapping index i to a strictly increasing time,
// so later indices are "newer".
func timeSeq() func(i int) time.Time {
	base := time.Unix(1_700_000_000, 0)
	return func(i int) time.Time { return base.Add(time.Duration(i) * time.Minute) }
}

func TestFormatStatsEmpty(t *testing.T) {
	out := FormatStats(nil)
	if !strings.Contains(out, "no tests recorded") {
		t.Fatalf("empty history message missing: %q", out)
	}
}

func TestFormatStatsSummarizesPerMode(t *testing.T) {
	h := []Record{
		{Mode: "time", TimeLimit: 30, WPM: 70, Accuracy: 95},
		{Mode: "time", TimeLimit: 30, WPM: 90, Accuracy: 97},
		{Mode: "words", WordCount: 25, WPM: 100, Accuracy: 99},
	}
	out := FormatStats(h)
	if !strings.Contains(out, "time") || !strings.Contains(out, "words") {
		t.Fatalf("missing mode rows: %q", out)
	}
	if !strings.Contains(out, "90") { // best time WPM
		t.Fatalf("missing best time wpm: %q", out)
	}
	if !strings.Contains(out, "3 tests") {
		t.Fatalf("missing total count: %q", out)
	}
	if !strings.Contains(out, "80") { // avg time wpm = (70+90)/2
		t.Fatalf("missing avg time wpm: %q", out)
	}
	if !strings.Contains(out, "96") { // avg time acc = (95+97)/2
		t.Fatalf("missing avg time acc: %q", out)
	}
}

func TestSummarizeAggregatesAndOrders(t *testing.T) {
	h := []Record{
		{Mode: "words", WordCount: 25, WPM: 100, Accuracy: 99},
		{Mode: "time", TimeLimit: 30, WPM: 70, Accuracy: 95},
		{Mode: "time", TimeLimit: 30, WPM: 90, Accuracy: 97},
	}
	s := Summarize(h)
	if s.Total != 3 {
		t.Fatalf("total: got %d want 3", s.Total)
	}
	// Modes ordered time, words, quote — quote absent.
	if len(s.Modes) != 2 || s.Modes[0].Mode != "time" || s.Modes[1].Mode != "words" {
		t.Fatalf("mode order/presence wrong: %+v", s.Modes)
	}
	tm := s.Modes[0]
	if tm.Count != 2 || tm.Best != 90 || tm.AvgWPM != 80 || tm.AvgAcc != 96 {
		t.Fatalf("time agg wrong: %+v", tm)
	}
}

func TestSummarizeRecentNewestFirstCapped(t *testing.T) {
	h := make([]Record, 12)
	base := timeSeq()
	for i := range h {
		h[i] = Record{Mode: "time", TimeLimit: 30, WPM: float64(i), Time: base(i)}
	}
	s := Summarize(h)
	if len(s.Recent) != 10 {
		t.Fatalf("recent cap: got %d want 10", len(s.Recent))
	}
	// Newest (highest i, latest Time) first.
	if s.Recent[0].WPM != 11 {
		t.Fatalf("recent not newest-first: got WPM %v want 11", s.Recent[0].WPM)
	}
}
