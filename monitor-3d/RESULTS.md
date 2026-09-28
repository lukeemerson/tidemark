# Results

All four prototypes run. Nothing is committed.

## How to view

```
cd <worktree> && python3 -m http.server 8743
```

Then open http://localhost:8743/monitor-3d/. I only tested the browser prototypes over HTTP. They use relative paths and CDN scripts, so opening them from `file://` should also work, but I didn't confirm it.

To run the terminal prototype:

```
python3 monitor-3d/prototypes/e-terminal-3d/monitor3d.py
```

Add `--live` for real mactop data.

## What I checked

| | prototype | loads | console errors | interaction I saw work |
| --- | --- | --- | --- | --- |
| A | `prototypes/a-vt-room/` | yes | none | warp, pause, legend, hover tooltip, click-to-fly (agent); full-history graphs after the replay fix (me) |
| B | `prototypes/b-silicon-floor/` | yes | none | hover tooltip, click isolate, pause and step, `?`, preset 2 (agent) |
| C | `prototypes/c-history-terrain/` | yes | none | hover, ridge isolate, scrubber click, `l` layers (agent); full 90-sample terrain after the fix (me) |
| E | `prototypes/e-terminal-3d/monitor3d.py` | yes, run under a pty | exits 0, restores the terminal | braille chart, keys, `--live`, SIGWINCH redraw, and RGP messages under `--force-ratty` (agent); clean exit at 90×24 (me) |

**Fix I made after the builds:** `shared/replay.js` now fills the history with all 90 samples at load. Before that, C drew a flat, pale plane for the missing 60 seconds, and A's graphs started two-thirds empty.

**Not checked:**
- Loading from `file://`.
- Dragging to orbit, the `1`–`4` tweens, and reduced motion in any browser prototype.
- E inside a real Ratty terminal. The detection reply was never received, and I don't know whether RGP rows and columns count from 0 or 1.
- E in your actual Alacritty window.

## Evaluation of the prototypes

| | strongest point | weakest point | verdict |
| --- | --- | --- | --- |
| A | It's recognisably your monitor. The bendable slab and the columns anchored to cells show Ratty's idea working with your own TUI. | Tilting it costs legibility, and the columns cover the percent readouts. It shows no new information. | Best showpiece. Next step: a real Ratty build with `ratatui-ratty`. |
| B | Shows at a glance that the E cluster is pinned while the P cores idle. | The floorplan is invented. The memory block is huge and dominates the scene. The temperature glow barely varies (61–69°). | Keep the idea, shrink memory, and use a colour ramp for temperature. |
| C | The most new information: you can see load moving between cores over time. | Vertex colours blend green into lilac into red, so the level colours stop reading as bands. It's busy from the default angle. | Strongest data view. Next step: flat-shaded level bands and a higher default camera. |
| E | The only one that ships: stdlib Python, ANSI slots, runs today, and emits RGP inside Ratty. | Braille is low resolution, and the chart only fills the lower half of the frame at 90×24. | Best candidate for the Go/Bubble Tea port. |

**Recommendation:** take E into the Go port as a `cores · 3d` panel option. Keep C as the companion web view, after fixing its colour banding.

## Deviations the agents reported
- **A:**
  - Processes' GPU ms/s shows `0.0`, because the samples have no per-process GPU field.
  - Fields the recording doesn't have (fan, thermal, volume, cloudflare grades) show as a dim `—`.
- **B/C:**
  - The time-slice highlight uses `text-bright`.
  - `t−Ns` counts replayed samples. In the recording they're about 1.2 s apart.
  - `Replay.step(-1)` appends an older sample rather than rewinding.
- **E:**
  - `cube.obj` is registered once per core (ids 1–10).
  - Added a dim `paused` before the clock.
  - Auto-spin is on at start.
