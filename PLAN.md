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
cmd/monitor/main.go      flags (-i interval), starts mactop, tea.NewProgram(model)
internal/source/         mactop.go (exec + json.Decoder → chan Sample), cloudy.go, procname.go
internal/ui/             model.go (Init/Update/View, alt screen), draw.go (box/graph/spark/bar/hjoin), panels.go
```

- `Sample` is a struct matching mactop's JSON, with only the fields the zsh `$JQ` read.
- History is a slice per series capped at 400 (`histLen`), the same as `HIST`.

## Bubble Tea mapping

| zsh                         | Bubble Tea                                                                                     |
| --------------------------- | ---------------------------------------------------------------------------------------------- |
| `draw` before mactop starts | `View()` with `have=false` renders dashes; `Init()` returns `m.wait`                           |
| `zselect` on the mactop fd  | `Model.wait` receives one `Sample` from the channel as `sampleMsg`, re-issued in `Update`      |
| `TRAPWINCH`                 | `tea.WindowSizeMsg`                                                                            |
| `q` key                     | `tea.KeyPressMsg` "q" / ctrl+c → `tea.Quit`; main.go stops mactop                              |
| `box`/`hjoin`/`padv`/`vis`  | hand-ported `box`/`hjoin` in draw.go; `lipgloss.Width` measures width                          |
| `graph` / `spark` / `bar`   | hand-written pure functions in draw.go                                                         |
| ANSI palette names          | lipgloss ANSI colours 1–6, 8, so the Alacritty palette still applies                           |

Border titles: lipgloss has no titled border, so `box()` in draw.go builds the top line itself.

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
5. ✅ `l`/`L` layouts (tiles · sidebar · instrument), each with its own line weight and skeleton;
   choice saved to `~/.config/mac-monitor/config.json`.
6. On-demand config UI (btop-style, closer to mactop's): more settings behind one menu.

## Expansion: why this one is better

Identity: **the monitor that explains what changed, and lets you rewind to see it.**
The baseline is high: [iStat Menus](https://bjango.com/mac/istatmenus/) has history, sensors,
per-app stats and alert rules; [mactop](https://github.com/metaspartan/mactop) owns Apple Silicon
terminal monitoring. More metrics alone won't stand out. The edge is connecting signals already
here: Apple Silicon metrics, process activity, power, thermals, saved network-quality tests.

Strongest combination: rewind + change detection + project grouping. Together they answer
*when did it happen, what changed, and which piece of my work was involved?*

1. **Rewind the whole machine view** (build first). Press a key when something stutters, then
   scrub back through synchronized CPU, GPU, memory, thermals and process snapshots; every
   panel follows one cursor. Needs timestamped samples plus process history (the 400-sample
   graph histories are a start). Success: identify what was active during a vanished spike
   within 30 seconds.
2. **"What changed?" feed.** Meaningful events beside the graphs ("compilers became the top CPU
   consumers", "swap grew 800 MB in two minutes", "power stayed high after the build").
   Select one to see its measurements. Start with explicit rules; keep observations separate
   from suspected causes.
3. **Organize activity by work.** Group processes into projects/sessions ("mac-monitor build",
   "editor + language servers", "browser", "local model"), expandable to the process tree.
   Useful for agent-heavy work: which session spawned the busy compiler or left a server
   running. Needs process ancestry and session metadata.
4. **Before-and-after mode.** Mark a baseline, change something, compare: idle power after
   closing an app, peak memory of a new build, loaded latency after switching networks. Show
   duration and workload context so comparisons stay fair.
5. **"My Mac or my connection?"** Combine local pressure with network observations as separate
   findings ("local load normal", "latency rose during upload"). Keep saved test results
   visibly dated; live diagnosis needs extra lightweight probes.
6. **Learn what's normal for this machine.** Flag deviations under comparable conditions
   (plugged in vs battery, idle vs building): "idle power above your usual range" beats a
   universal red threshold. Show the baseline and its sample count.
7. **Compact view with drill-down.** In a small pane: current condition, the biggest change, a
   few trends; one key expands to the full investigation view. Stable panel positions; bright
   accents only for the selection and meaningful changes.
8. **Tiny incident recordings.** Export a time window (graphs, events, process summaries);
   reopen it in the monitor or render a readable report, for intermittent problems and
   sharing evidence.
