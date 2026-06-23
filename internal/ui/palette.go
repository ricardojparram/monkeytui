package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ricardojparram/monkeytui/internal/theme"
	"github.com/ricardojparram/monkeytui/internal/typing"
)

// cmdKind identifies what applying a command does.
type cmdKind int

const (
	cmdModeTime cmdKind = iota
	cmdModeWords
	cmdModeQuote
	cmdTheme
	cmdRestart
	cmdTogglePunct
	cmdToggleNumbers
	cmdShowStats
	cmdQuit
)

// command is a single selectable palette entry.
type command struct {
	title string
	group string
	kind  cmdKind
	arg   int    // seconds or word count
	sarg  string // theme name
}

// palette is the monkeytype-style command line: a filterable command list.
type palette struct {
	open     bool
	query    string
	items    []command
	filtered []command
	sel      int
}

// newPalette builds the full command set.
func newPalette() palette {
	items := []command{
		{title: "restart test", group: "action", kind: cmdRestart},
		{title: "time 15", group: "mode", kind: cmdModeTime, arg: 15},
		{title: "time 30", group: "mode", kind: cmdModeTime, arg: 30},
		{title: "time 60", group: "mode", kind: cmdModeTime, arg: 60},
		{title: "time 120", group: "mode", kind: cmdModeTime, arg: 120},
		{title: "words 10", group: "mode", kind: cmdModeWords, arg: 10},
		{title: "words 25", group: "mode", kind: cmdModeWords, arg: 25},
		{title: "words 50", group: "mode", kind: cmdModeWords, arg: 50},
		{title: "words 100", group: "mode", kind: cmdModeWords, arg: 100},
		{title: "quote", group: "mode", kind: cmdModeQuote},
		{title: "punctuation", group: "toggle", kind: cmdTogglePunct},
		{title: "numbers", group: "toggle", kind: cmdToggleNumbers},
	}
	for _, name := range theme.Names() {
		items = append(items, command{title: "theme " + name, group: "theme", kind: cmdTheme, sarg: name})
	}
	items = append(items, command{title: "stats", group: "action", kind: cmdShowStats})
	items = append(items, command{title: "quit", group: "action", kind: cmdQuit})

	p := palette{items: items}
	p.refilter()
	return p
}

func (p *palette) openPalette() {
	p.open = true
	p.query = ""
	p.sel = 0
	p.refilter()
}

func (p *palette) close() { p.open = false }

// refilter recomputes the visible list from the current query (substring match).
func (p *palette) refilter() {
	q := strings.ToLower(strings.TrimSpace(p.query))
	p.filtered = p.filtered[:0]
	for _, it := range p.items {
		if q == "" || strings.Contains(strings.ToLower(it.title), q) {
			p.filtered = append(p.filtered, it)
		}
	}
	if p.sel >= len(p.filtered) {
		p.sel = len(p.filtered) - 1
	}
	if p.sel < 0 {
		p.sel = 0
	}
}

func (p *palette) move(d int) {
	if len(p.filtered) == 0 {
		return
	}
	p.sel = (p.sel + d + len(p.filtered)) % len(p.filtered)
}

func (p *palette) typeRune(r rune) {
	p.query += string(r)
	p.sel = 0
	p.refilter()
}

func (p *palette) backspace() {
	if p.query != "" {
		p.query = p.query[:len(p.query)-1]
		p.refilter()
	}
}

// selected returns the highlighted command and whether one exists.
func (p *palette) selected() (command, bool) {
	if p.sel >= 0 && p.sel < len(p.filtered) {
		return p.filtered[p.sel], true
	}
	return command{}, false
}

// scrollWindow returns the [start,end) slice of items to render so that sel
// stays visible within a window of at most maxRows, including when sel reaches
// the end of a list longer than the window.
func scrollWindow(n, sel, maxRows int) (int, int) {
	start := 0
	if n > maxRows && sel >= maxRows {
		start = sel - maxRows + 1
		if start > n-maxRows {
			start = n - maxRows
		}
	}
	return start, min(start+maxRows, n)
}

// view renders the palette as a centered box.
func (p *palette) view(th theme.Theme, width, height int) string {
	boxW := min(50, width-4)
	if boxW < 20 {
		boxW = max(width-2, 20)
	}

	prompt := th.Main.Render("> ") + th.Text.Render(p.query) + th.Caret.Render(" ")
	rows := []string{prompt, ""}

	maxRows := 10
	n := len(p.filtered)
	start, end := scrollWindow(n, p.sel, maxRows)
	if start > 0 {
		rows = append(rows, th.Faint.Render("  ↑ …"))
	}
	for i := start; i < end; i++ {
		it := p.filtered[i]
		line := "  " + it.title
		grp := th.Faint.Render(" " + it.group)
		if i == p.sel {
			line = th.Main.Render("› " + it.title)
		} else {
			line = th.Text.Render(line)
		}
		rows = append(rows, line+grp)
	}
	if end < n {
		rows = append(rows, th.Faint.Render("  ↓ …"))
	}
	if n == 0 {
		rows = append(rows, th.Faint.Render("  no matches"))
	}
	rows = append(rows, "", th.Faint.Render("enter select · esc close · ↑↓ move"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(0, 1).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// modeLabel renders the active mode/limit for the status bar.
func modeLabel(cfg typing.Config) string {
	switch cfg.Mode {
	case typing.ModeTime:
		return "time " + itoa(cfg.TimeLimit)
	case typing.ModeWords:
		return "words " + itoa(cfg.WordCount)
	default:
		return "quote"
	}
}
