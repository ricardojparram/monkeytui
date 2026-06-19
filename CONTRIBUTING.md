# Contributing to monkeytui

Thanks for your interest in improving monkeytui! This is a small, focused project,
so the process is light.

## Getting started

```sh
git clone https://github.com/ricardojparram/monkeytui
cd monkeytui
go build -o monkeytui .
go test ./...
./monkeytui
```

Requires Go 1.21+.

## Workflow

1. **Open an issue first** for anything non-trivial — a bug, a feature, a behavior
   change — so we can agree on the approach before you write code.
2. Fork, branch from `main`, and keep your branch focused on one thing.
3. Make your change with tests where it makes sense (the engine and UI rendering are
   covered by unit tests — see `internal/typing` and `internal/ui`).
4. Run the checks below.
5. Open a pull request describing **what** changed and **why**.

## Checks before opening a PR

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .      # should print nothing
```

Please keep commits clean and use [Conventional Commits](https://www.conventionalcommits.org)
for the subject line (e.g. `feat:`, `fix:`, `docs:`, `refactor:`).

## Project layout

```
main.go              entry point, CLI flags, subcommands (update/uninstall)
selfupdate.go        self-update and uninstall logic
install.sh           one-line installer (prebuilt binary, Go fallback)
demo.tape            VHS script for the README demo GIF
internal/words/      embedded English word list + random generator
internal/quotes/     bundled quotes for quote mode
internal/typing/     test engine: keystrokes, timing, metric capture
internal/stats/      WPM / accuracy / consistency + per-second samples
internal/theme/      terminal-adaptive lipgloss styles
internal/ui/         Bubble Tea model, command palette, chart, views
```

## Conventions

- **Keep packages focused.** Each one has a single clear job and a small public API.
- **Colors come from the ANSI palette** (`internal/theme`), never hardcoded RGB — this
  is what makes monkeytui adapt to the user's terminal theme. Typed text uses the
  terminal default foreground; untyped uses ANSI 8. Preserve that contrast contract.
- **Match the surrounding style** — comment density, naming, and idioms.

## Ideas that would be welcome

- More word lists (e.g. `english-1k`, punctuation, numbers) selectable via the palette.
- Persisting results / personal bests locally.
- Additional accent themes.

## Releasing (maintainers)

```sh
git tag -a vX.Y.Z -m "monkeytui vX.Y.Z" && git push origin main vX.Y.Z
# cross-compile to dist/ with -ldflags "-s -w -X main.version=vX.Y.Z"
# sha256sum monkeytui_* > checksums.txt
gh release create vX.Y.Z --title "monkeytui vX.Y.Z" --notes "…" dist/*
```
