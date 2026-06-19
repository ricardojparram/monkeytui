package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"monkeytui/internal/typing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func strip(s string) string { return ansiRE.ReplaceAllString(s, "") }

// TestNoDoubleSpaceAfterWord types a complete word (caret now trailing) and
// asserts there is exactly one space before the next word, not two.
func TestNoDoubleSpaceAfterWord(t *testing.T) {
	var model tea.Model = New(typing.Config{Mode: typing.ModeWords, WordCount: 3}, "yellow")
	model, _ = model.Update(tea.WindowSizeMsg{Width: 90, Height: 28})

	w0 := model.(Model).eng.TargetWords()[0]
	w1 := model.(Model).eng.TargetWords()[1]
	for _, r := range w0 { // type first word fully, no space -> caret trailing
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	plain := strip(model.View())
	needle := w0 + "  " + w1 // two spaces would be the bug
	if strings.Contains(plain, needle) {
		t.Fatalf("found double space between words: %q", needle)
	}
	if !strings.Contains(plain, w0+" "+w1) {
		t.Fatalf("expected single space between %q and %q; frame:\n%s", w0, w1,
			firstWordLine(plain, w0))
	}
}

func firstWordLine(s, w string) string {
	for _, ln := range strings.Split(s, "\n") {
		if strings.Contains(ln, w) {
			return strings.TrimRight(ln, " ")
		}
	}
	return "(not found)"
}
