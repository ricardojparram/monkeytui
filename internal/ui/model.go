// Package ui wires the typing engine, theme, command palette and views into a
// single bubbletea program.
package ui

import (
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ricardojparram/monkeytui/internal/stats"
	"github.com/ricardojparram/monkeytui/internal/store"
	"github.com/ricardojparram/monkeytui/internal/theme"
	"github.com/ricardojparram/monkeytui/internal/typing"
)

type state int

const (
	stateTyping state = iota
	stateResults
)

// tickMsg fires once per second to sample metrics and drive the countdown.
type tickMsg time.Time

// blinkMsg toggles the caret to give it a soft monkeytype-style pulse.
type blinkMsg time.Time

// blinkInterval is how often the caret toggles while idle.
const blinkInterval = 530 * time.Millisecond

// Model is the root bubbletea model.
type Model struct {
	eng        *typing.Engine
	cfg        typing.Config
	th         theme.Theme
	themeName  string
	state      state
	result     stats.Result
	pal        palette
	now        time.Time
	width      int
	height     int
	caretShown bool

	history   []store.Record
	persist   bool
	priorBest float64
	isPB      bool
}

// New creates the initial model from a starting config and theme name.
func New(cfg typing.Config, themeName string) Model {
	return Model{
		eng:        typing.New(cfg),
		cfg:        cfg,
		th:         theme.ByName(themeName),
		themeName:  themeName,
		state:      stateTyping,
		pal:        newPalette(),
		now:        time.Now(),
		caretShown: true,
	}
}

// WithStore enables disk persistence and seeds the in-memory history used for
// personal-best comparisons. main.go calls this; tests leave persistence off.
func (m Model) WithStore(history []store.Record) Model {
	m.history = history
	m.persist = true
	return m
}

// prefs snapshots the current settings for persistence.
func (m Model) prefs() store.Prefs {
	return store.Prefs{
		Mode:        m.cfg.Mode.String(),
		TimeLimit:   m.cfg.TimeLimit,
		WordCount:   m.cfg.WordCount,
		Theme:       m.themeName,
		Punctuation: m.cfg.Punctuation,
		Numbers:     m.cfg.Numbers,
	}
}

// savePrefs persists the current settings when persistence is enabled.
func (m *Model) savePrefs() {
	if m.persist {
		_ = store.SavePrefs(m.prefs())
	}
}

// bucket identifies the current test's personal-best group.
func (m Model) bucket() store.Bucket {
	return store.Bucket{
		Mode: m.cfg.Mode.String(), TimeLimit: m.cfg.TimeLimit, WordCount: m.cfg.WordCount,
		Punctuation: m.cfg.Punctuation, Numbers: m.cfg.Numbers,
	}
}

func (m Model) Init() tea.Cmd { return tea.Batch(tick(), blink()) }

// tick schedules the next one-second sampling tick.
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// blink schedules the next caret toggle.
func blink() tea.Cmd {
	return tea.Tick(blinkInterval, func(t time.Time) tea.Msg { return blinkMsg(t) })
}

// solidCaret keeps the caret visible immediately after a keystroke, so it never
// blinks out mid-typing.
func (m *Model) solidCaret() { m.caretShown = true }

// restart begins a fresh test with the current config.
func (m *Model) restart() {
	m.eng = typing.New(m.cfg)
	m.state = stateTyping
	m.now = time.Now()
	m.caretShown = true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if m.state == stateTyping {
			m.eng.Sample(m.now)
			if m.eng.Tick(m.now) {
				m.finishToResults()
			}
		}
		return m, tick()

	case blinkMsg:
		m.caretShown = !m.caretShown
		return m, blink()

	case tea.KeyMsg:
		m.now = time.Now()
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) finishToResults() {
	m.result = m.eng.Result(m.now)
	m.state = stateResults

	prev, ok := store.BestWPM(m.history, m.bucket())
	m.priorBest = prev
	m.isPB = !ok || m.result.WPM > prev

	rec := store.Record{
		Time:        m.now,
		Mode:        m.cfg.Mode.String(),
		TimeLimit:   m.cfg.TimeLimit,
		WordCount:   m.cfg.WordCount,
		Punctuation: m.cfg.Punctuation,
		Numbers:     m.cfg.Numbers,
		WPM:         m.result.WPM,
		Raw:         m.result.Raw,
		Accuracy:    m.result.Accuracy,
		Consistency: m.result.Consistency,
	}
	m.history = append(m.history, rec)
	if m.persist {
		_ = store.AppendRecord(rec)
		m.savePrefs()
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit.
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Palette overlay swallows input while open.
	if m.pal.open {
		return m.handlePaletteKey(msg)
	}

	switch m.state {
	case stateTyping:
		return m.handleTypingKey(msg)
	case stateResults:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyTab:
			m.restart()
			return m, tick()
		case tea.KeyEsc:
			m.pal.openPalette()
		}
	}
	return m, nil
}

func (m Model) handleTypingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyEsc:
		m.pal.openPalette()
		return m, nil
	case tea.KeyBackspace:
		m.solidCaret()
		m.eng.Backspace()
		return m, nil
	case tea.KeySpace:
		m.solidCaret()
		m.eng.Space(m.now)
		if m.eng.Finished() {
			m.finishToResults()
		}
		return m, nil
	case tea.KeyRunes:
		m.solidCaret()
		for _, r := range msg.Runes {
			m.eng.Type(m.now, r)
		}
		if m.eng.Finished() {
			m.finishToResults()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.pal.close()
	case tea.KeyUp, tea.KeyCtrlP:
		m.pal.move(-1)
	case tea.KeyDown, tea.KeyCtrlN:
		m.pal.move(1)
	case tea.KeyBackspace:
		m.pal.backspace()
	case tea.KeyEnter:
		if cmd, ok := m.pal.selected(); ok {
			m.pal.close()
			return m.apply(cmd)
		}
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.pal.typeRune(r)
		}
	case tea.KeySpace:
		m.pal.typeRune(' ')
	}
	return m, nil
}

// apply runs a chosen command.
func (m Model) apply(cmd command) (tea.Model, tea.Cmd) {
	switch cmd.kind {
	case cmdModeTime:
		m.cfg.Mode = typing.ModeTime
		m.cfg.TimeLimit = cmd.arg
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdModeWords:
		m.cfg.Mode = typing.ModeWords
		m.cfg.WordCount = cmd.arg
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdModeQuote:
		m.cfg.Mode = typing.ModeQuote
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdTogglePunct:
		m.cfg.Punctuation = !m.cfg.Punctuation
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdToggleNumbers:
		m.cfg.Numbers = !m.cfg.Numbers
		m.restart()
		m.savePrefs()
		return m, tick()
	case cmdTheme:
		m.th = theme.ByName(cmd.sarg)
		m.themeName = cmd.sarg
		m.savePrefs()
		return m, nil
	case cmdRestart:
		m.restart()
		return m, tick()
	case cmdQuit:
		return m, tea.Quit
	}
	return m, nil
}

// small int->string helpers shared by views.
func itoa(n int) string { return strconv.Itoa(n) }
