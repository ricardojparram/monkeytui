package store

import (
	"strings"
	"testing"
)

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
