package ui

import "testing"

func TestScrollWindowFollowsSelectionToEnd(t *testing.T) {
	const maxRows = 10
	cases := []struct {
		name             string
		n, sel           int
		wantStart, wantEnd int
	}{
		{"short list no scroll", 5, 4, 0, 5},
		{"top of long list", 20, 0, 0, 10},
		{"last visible row", 20, 9, 0, 10},
		{"scroll past window", 20, 10, 1, 11},
		{"at the very end", 20, 19, 10, 20},
		{"exact window size", 10, 9, 0, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := scrollWindow(c.n, c.sel, maxRows)
			if start != c.wantStart || end != c.wantEnd {
				t.Fatalf("scrollWindow(%d,%d,%d) = (%d,%d) want (%d,%d)",
					c.n, c.sel, maxRows, start, end, c.wantStart, c.wantEnd)
			}
			// The selection must always fall inside the returned window.
			if c.n > 0 && (c.sel < start || c.sel >= end) {
				t.Fatalf("sel %d not in window [%d,%d)", c.sel, start, end)
			}
		})
	}
}
