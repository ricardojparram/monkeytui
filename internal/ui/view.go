package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ricardojparram/monkeytui/internal/typing"
)

// View renders the current screen.
func (m Model) View() string {
	w, h := m.width, m.height
	if w == 0 {
		w, h = 80, 24
	}
	if m.pal.open {
		return m.pal.view(m.th, w, h)
	}
	var body string
	if m.state == stateResults {
		body = m.renderResults(w)
	} else {
		body = m.renderTyping(w)
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, body)
}

// caretActive reports whether the caret should be drawn on word i right now.
func (m Model) caretActive(i int) bool {
	return m.state == stateTyping && !m.eng.Finished() &&
		m.caretShown && i == m.eng.Cur()
}

// wordLen returns the display width of word i (max of target and typed).
func (m Model) wordLen(i int) int {
	target := len([]rune(m.eng.TargetWords()[i]))
	typed := 0
	if i < m.eng.Cur() {
		typed = len([]rune(m.eng.TypedWords()[i]))
	} else if i == m.eng.Cur() {
		typed = len(m.eng.CurInput())
	}
	return max(target, typed)
}

// renderWordCells renders word i's characters. It returns the styled string and
// whether the caret belongs on the separator *after* this word (because the
// word is already fully typed).
func (m Model) renderWordCells(i int) (string, bool) {
	th := m.th
	target := []rune(m.eng.TargetWords()[i])
	var typed []rune
	if i < m.eng.Cur() {
		typed = []rune(m.eng.TypedWords()[i])
	} else if i == m.eng.Cur() {
		typed = m.eng.CurInput()
	}

	caretPos := -1
	if m.caretActive(i) {
		caretPos = len(m.eng.CurInput())
	}

	L := max(len(target), len(typed))
	var sb strings.Builder
	for j := 0; j < L; j++ {
		var ch rune
		var stl lipgloss.Style
		switch {
		case j < len(typed) && j < len(target):
			ch = target[j]
			if typed[j] == target[j] {
				stl = th.Text
			} else {
				stl = th.Error
			}
		case j < len(typed):
			ch = typed[j]
			stl = th.Extra
		default:
			ch = target[j]
			stl = th.Sub
		}
		if j == caretPos {
			stl = th.Caret // caret tints the upcoming (untyped) char
		}
		sb.WriteString(stl.Render(string(ch)))
	}
	trailing := caretPos >= 0 && caretPos >= L
	return sb.String(), trailing
}

// renderLine renders a line of words with exactly one space between them. When a
// word's caret is trailing, that separating space becomes the caret instead of
// adding an extra cell.
func (m Model) renderLine(idxs []int) string {
	th := m.th
	var sb strings.Builder
	for k, i := range idxs {
		cells, trailing := m.renderWordCells(i)
		sb.WriteString(cells)
		last := k == len(idxs)-1
		switch {
		case trailing:
			sb.WriteString(th.Caret.Render(" ")) // caret sits in the gap
		case !last:
			sb.WriteString(th.Sub.Render(" "))
		}
	}
	return sb.String()
}

// layout greedily wraps words into lines of index lists fitting maxW cells.
func (m Model) layout(maxW, cap int) [][]int {
	var lines [][]int
	var line []int
	lineW := 0
	n := len(m.eng.TargetWords())
	if cap < n {
		n = cap
	}
	for i := 0; i < n; i++ {
		wl := m.wordLen(i)
		add := wl
		if len(line) > 0 {
			add++
		}
		if lineW+add > maxW && len(line) > 0 {
			lines = append(lines, line)
			line = nil
			lineW = 0
			add = wl
		}
		line = append(line, i)
		lineW += add
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

// renderTyping draws the status counter and a 3-line scrolling word window.
func (m Model) renderTyping(w int) string {
	th := m.th
	maxW := min(w-4, 72)
	if maxW < 20 {
		maxW = 20
	}

	var counter string
	switch m.cfg.Mode {
	case typing.ModeTime:
		counter = fmt.Sprintf("%.0f", m.eng.Remaining(m.now))
	default:
		counter = fmt.Sprintf("%d/%d", m.eng.Cur(), len(m.eng.TargetWords()))
	}
	status := th.Main.Render(counter)

	lines := m.layout(maxW, m.eng.Cur()+90)
	curLine := 0
	for li, ln := range lines {
		for _, wi := range ln {
			if wi == m.eng.Cur() {
				curLine = li
			}
		}
	}
	start := curLine
	if start > len(lines)-3 {
		start = len(lines) - 3
	}
	if start < 0 {
		start = 0
	}
	end := min(start+3, len(lines))

	var rows []string
	for li := start; li < end; li++ {
		rows = append(rows, m.renderLine(lines[li]))
	}
	words := lipgloss.NewStyle().Width(maxW).Render(strings.Join(rows, "\n"))

	hint := th.Faint.Render("tab / esc  command line     ctrl+c  quit")
	block := lipgloss.JoinVertical(lipgloss.Left, status, "", words, "", hint)
	return lipgloss.NewStyle().Width(maxW).Render(block)
}

// statCol renders a monkeytype-style stat: a small faint label above a value.
func (m Model) statCol(label, value string) string {
	th := m.th
	return lipgloss.JoinVertical(lipgloss.Left,
		th.Faint.Render(label),
		th.Main.Render(value))
}

// bigStat renders a headline metric (wpm / acc): a faint label above a bold,
// accent-colored value — prominent but not oversized.
func (m Model) bigStat(label, value string) string {
	th := m.th
	return lipgloss.JoinVertical(lipgloss.Left,
		th.Faint.Render(label),
		th.Main.Bold(true).Render(value))
}

// typeSummary describes the test on the results screen: mode plus any active
// decoration toggles, e.g. "time 30 punctuation numbers".
func typeSummary(cfg typing.Config) string {
	s := modeLabel(cfg)
	if cfg.Mode != typing.ModeQuote {
		if cfg.Punctuation {
			s += " punctuation"
		}
		if cfg.Numbers {
			s += " numbers"
		}
	}
	return s
}

// wordlistName labels the active content source, like monkeytype's "english".
func (m Model) wordlistName() string {
	if m.cfg.Mode == typing.ModeQuote {
		return "quote"
	}
	return "english"
}

// renderResults reproduces the monkeytype results screen: a left headline block
// (wpm / acc / test type) beside the chart, a stat row underneath, then a
// compact replay and key hints.
func (m Model) renderResults(w int) string {
	th := m.th
	r := m.result
	maxW := min(w-4, 100)
	if maxW < 40 {
		maxW = 40
	}

	// Personal-best line under the wpm headline.
	var pbLine string
	if m.isPB {
		pbLine = th.Main.Bold(true).Render("new pb")
	} else if m.priorBest > 0 {
		pbLine = th.Faint.Render(fmt.Sprintf("best %.0f", m.priorBest))
	}

	// Left headline column: wpm, pb/best, acc, and the test-type summary.
	left := lipgloss.JoinVertical(lipgloss.Left,
		m.bigStat("wpm", fmt.Sprintf("%.0f", r.WPM)),
		pbLine,
		m.bigStat("acc", fmt.Sprintf("%.0f%%", r.Accuracy)),
		"",
		th.Faint.Render("test type"),
		th.Text.Render(typeSummary(m.cfg)),
		th.Text.Render(m.wordlistName()),
	)
	leftW := lipgloss.Width(left)

	chartW := maxW - leftW - 3
	if chartW < 30 {
		chartW = 30
	}
	chart := renderChart(r.Samples, th, chartW, 9)

	top := lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", chart)

	// Stat row beneath, monkeytype-style label-over-value columns.
	cols := []string{
		m.statCol("raw", fmt.Sprintf("%.0f", r.Raw)),
		m.statCol("characters", fmt.Sprintf("%d/%d/%d/%d", r.Correct, r.Incorrect, r.Extra, r.Missed)),
		m.statCol("consistency", fmt.Sprintf("%.0f%%", r.Consistency)),
		m.statCol("time", fmt.Sprintf("%.0fs", r.Seconds)),
	}
	var rowParts []string
	for i, c := range cols {
		if i > 0 {
			rowParts = append(rowParts, "     ")
		}
		rowParts = append(rowParts, c)
	}
	statRow := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)

	// Compact replay (kept for the TUI; monkeytype shows the words inline).
	lines := m.layout(maxW, len(m.eng.TargetWords()))
	limit := min(len(lines), 3)
	var replayRows []string
	for li := 0; li < limit; li++ {
		var idxs []int
		for _, wi := range lines[li] {
			if wi <= m.eng.Cur() {
				idxs = append(idxs, wi)
			}
		}
		if len(idxs) > 0 {
			replayRows = append(replayRows, m.renderLine(idxs))
		}
	}
	replay := lipgloss.NewStyle().Width(maxW).Render(strings.Join(replayRows, "\n"))

	hint := th.Faint.Render("⏎  next test     esc  command line     ctrl+c  quit")
	block := lipgloss.JoinVertical(lipgloss.Left,
		top, "", statRow, "",
		th.Faint.Render("replay"), replay, "", hint)
	return lipgloss.NewStyle().Width(maxW).Render(block)
}
