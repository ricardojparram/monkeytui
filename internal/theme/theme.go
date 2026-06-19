// Package theme defines lipgloss styles built from ANSI 0-15 colors so the UI
// follows whatever palette the user's terminal is configured with. Only the
// accent color varies between named themes.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme is a set of pre-built styles plus the raw accent color (for the chart).
type Theme struct {
	Name   string
	Accent lipgloss.Color

	Sub    lipgloss.Style // untyped target text (dim)
	Text   lipgloss.Style // correctly typed text
	Error  lipgloss.Style // mistyped target characters
	Extra  lipgloss.Style // characters typed past a word's length
	Caret  lipgloss.Style // the cursor block
	Main   lipgloss.Style // accent-colored text
	Faint  lipgloss.Style // hints and chrome
}

// ANSI color slots (0-15) map to the terminal's own palette.
const (
	ansiRed     = lipgloss.Color("1")
	ansiBrBlack = lipgloss.Color("8")
	ansiWhite   = lipgloss.Color("7")
	ansiBrRed   = lipgloss.Color("9")
)

// build assembles a theme from a single accent color.
func build(name string, accent lipgloss.Color) Theme {
	return Theme{
		Name:   name,
		Accent: accent,
		Sub:    lipgloss.NewStyle().Foreground(ansiBrBlack),
		Text:   lipgloss.NewStyle().Foreground(ansiWhite),
		Error:  lipgloss.NewStyle().Foreground(ansiRed).Underline(true),
		Extra:  lipgloss.NewStyle().Foreground(ansiBrRed),
		// Caret is a monkeytype-style line: the upcoming character is tinted the
		// accent color and underlined, so it never hides text or jumps over it.
		Caret:  lipgloss.NewStyle().Foreground(accent).Underline(true).Bold(true),
		Main:   lipgloss.NewStyle().Foreground(accent),
		Faint:  lipgloss.NewStyle().Foreground(ansiBrBlack),
	}
}

// registry of named accent variants; all share the terminal-adaptive base.
var order = []string{"yellow", "green", "cyan", "magenta", "blue", "red"}

var accents = map[string]lipgloss.Color{
	"yellow":  lipgloss.Color("3"),
	"green":   lipgloss.Color("2"),
	"cyan":    lipgloss.Color("6"),
	"magenta": lipgloss.Color("5"),
	"blue":    lipgloss.Color("4"),
	"red":     lipgloss.Color("1"),
}

// Default returns the monkeytype-like yellow-accent theme.
func Default() Theme { return build("yellow", accents["yellow"]) }

// ByName returns the named theme, or the default if unknown.
func ByName(name string) Theme {
	if c, ok := accents[name]; ok {
		return build(name, c)
	}
	return Default()
}

// Names lists the available theme names in display order.
func Names() []string { return order }
