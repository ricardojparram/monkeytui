# monkeytui

A minimalist [monkeytype](https://monkeytype.com)-style typing test for the
terminal, built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
Adapts to your terminal's color theme (uses ANSI palette colors).

## Features

- **Modes**: `time` (15/30/60/120s), `words` (10/25/50/100), `quote`
- **Live test**: per-character coloring, caret, mistakes highlighted in red
- **Results**: WPM-over-time line chart, raw WPM, accuracy, consistency,
  character breakdown, and a colored replay of what you typed
- **Command palette** (monkeytype-style): press `Tab` or `Esc` to switch mode,
  time, word count, or accent theme — all without leaving the keyboard
- **Theme-adaptive**: colors come from your terminal's own ANSI palette;
  pick an accent: yellow, green, cyan, magenta, blue, red

## Install

One-line install (builds with Go and puts `monkeytui` on your PATH):

```sh
curl -fsSL https://raw.githubusercontent.com/ricardojparram/monkeytui/main/install.sh | bash
```

Or with Go directly:

```sh
go install github.com/ricardojparram/monkeytui@latest
```

Requires Go 1.21+. After installing, just run `monkeytui`.

## Run

```sh
go run .                      # default: 30s time mode, yellow accent
go run . -mode words -words 50
go run . -mode quote
go run . -mode time -time 60 -theme cyan
```

Or build a binary:

```sh
go build -o monkeytui .
./monkeytui
```

## Keys

| Key | Action |
|-----|--------|
| any letter | type |
| `space` | next word |
| `backspace` | delete (steps into previous word) |
| `Tab` / `Esc` | open command palette |
| `Enter` (results) | next test |
| `Ctrl+C` | quit |

### In the command palette

Type to filter, `↑`/`↓` to move, `Enter` to apply, `Esc` to close.

## Project layout

```
main.go              entry point, CLI flags
internal/words/      embedded English word list + random generator
internal/quotes/     bundled quotes for quote mode
internal/typing/     test engine: keystrokes, timing, metric capture
internal/stats/      WPM / accuracy / consistency + per-second samples
internal/theme/      lipgloss styles (ANSI, terminal-adaptive)
internal/ui/         Bubble Tea model, command palette, chart, views
```
