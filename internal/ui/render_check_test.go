package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"monkeytui/internal/stats"
	"monkeytui/internal/typing"
)

// TestRenderSmoke drives a words test to completion and prints both the typing
// and results frames so a human can eyeball layout. Run with -v.
func TestRenderSmoke(t *testing.T) {
	var model tea.Model = New(typing.Config{Mode: typing.ModeWords, WordCount: 3}, "yellow")
	model, _ = model.Update(tea.WindowSizeMsg{Width: 90, Height: 28})

	cur := model.(Model)
	tw := cur.eng.TargetWords()
	full := tw[0] + " " + tw[1] + " " + tw[2]

	for i, r := range full {
		var km tea.KeyMsg
		if r == ' ' {
			km = tea.KeyMsg{Type: tea.KeySpace}
		} else {
			km = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		}
		model, _ = model.Update(km)
		if i == len(full)/2 {
			fmt.Println("=== TYPING (mid) ===")
			fmt.Println(model.View())
		}
	}

	// Seed a few fake samples so the chart renders, then show results.
	mv := model.(Model)
	mv.result.Samples = []stats.Sample{
		{T: 1, WPM: 40, Raw: 45, Errors: 0},
		{T: 2, WPM: 55, Raw: 60, Errors: 1},
		{T: 3, WPM: 50, Raw: 52, Errors: 0},
		{T: 4, WPM: 62, Raw: 66, Errors: 0},
	}
	fmt.Println("=== RESULTS ===")
	fmt.Println(mv.View())
}
