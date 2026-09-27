# Go + Bubble Tea port of `monitor`

Goal: one static binary that renders the draft-03 layout, installed with `go install` or Homebrew.
Go 1.27.1 is installed (Homebrew). Bubble Tea v2 (`charm.land/bubbletea/v2`) and lipgloss v2.

## Data source

mactop's collectors live in `internal/app`, so another module can't import them.

1. **Start: spawn `mactop --headless --count 0 -i 1000`** and decode its JSON stream. This is
   the same contract as the zsh script, needs no cgo, and makes mactop a runtime dependency.
2. **Later, if the ~1–1.8 s mactop startup matters:** vendor `ioreport.go/.m`, `native_stats.go`,
   and `battery.go` (MIT, keep the notice) and sample in-process. This needs cgo, and a
   `darwin/arm64` build only.

cloudy: read `~/Library/Application Support/cloudflare-speed-cli/runs/run-*.json` directly.
Process names: when mactop reports a bare version (`2.1.283`), resolve argv[0] with
`sysctl kern.procargs2` (`golang.org/x/sys/unix`) instead of shelling out to `ps`.

## Layout

```
cmd/monitor/main.go      flags (-i interval), tea.NewProgram(model, tea.WithAltScreen())
internal/source/         mactop.go (exec + json.Decoder → chan Sample), cloudy.go, procname.go
internal/ui/             model.go (Init/Update/View), tiles.go, panels.go, graph.go (braille), style.go
```

- `Sample` is a struct matching mactop's JSON, with only the fields the zsh `$JQ` reads.
- History is a fixed ring buffer (400) per series, the same as `HIST`.

## Bubble Tea mapping

| zsh                         | Bubble Tea                                                                                     |
| --------------------------- | ---------------------------------------------------------------------------------------------- |
| `draw` before mactop starts | `View()` with `have=false` renders dashes; `Init()` returns the start-mactop cmd               |
| `zselect` on the mactop fd  | a `tea.Cmd` that reads one line and returns `sampleMsg`, re-issued in `Update`                 |
| `TRAPWINCH`                 | `tea.WindowSizeMsg`                                                                            |
| `q` key                     | `tea.KeyMsg` "q" / ctrl+c → kill mactop, `tea.Quit`                                            |
| `box`/`hjoin`/`padv`/`vis`  | lipgloss `Border(lipgloss.ThickBorder())` + `JoinHorizontal/Vertical`; lipgloss measures width |
| `graph` / `spark` / `bar`   | hand-written (small, pure functions; unit-test them)                                           |
| ANSI palette names          | lipgloss ANSI colours 1–6, 8, so the Alacritty palette still applies                           |

Border titles: lipgloss has no titled border, so each panel builds its top line itself
(port `box()`'s top-line logic).

## Packaging

- `go build -trimpath -ldflags "-s -w" ./cmd/monitor`, target `darwin/arm64`.
- `go install github.com/<you>/mac-monitor/cmd/monitor@latest` once it's pushed.
- Optional: a Homebrew tap formula with `depends_on "mactop"` (and `depends_on arch: :arm64`).
- goreleaser only if you want tagged binary releases.

## Order

1. ✅ `go mod init`, source package + a test that decodes `internal/source/testdata/mactop.raw`.
2. ✅ Model, rendering the header and tiles only. Check it against the zsh version side by side.
3. ✅ Remaining panels, then resize and quit handling.
4. ✅ Replace `bin/monitor`, and remove `drafts/` and `render/` once the Go version matches.
