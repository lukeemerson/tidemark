# Spec: history, replay and diagnosis (2D, tidemark)

Terminal features for tidemark (Go). The 3D versions are in `TODO-future.md`.

## 1. Record and replay

- **Format**: mactop's own `--headless` stream, one JSON sample per line, each with a `timestamp`. `source.Decode` already reads it, and `internal/source/testdata/mactop.raw` is already a recording.
- `tidemark -rec <file>`: tee mactop's stdout to `<file>` while running as normal.
- `tidemark -play <file>`: feed `<file>` to `Decode` instead of starting mactop. Samples arrive at the recorded spacing.
- The same format settles tidalrat's `--record` / `--replay` (PLAN row 6), since `source.rs` parses the same stream.
- On `-play`, the chip name, E/P core counts and memory total come from the recording (`system_info`, `memory.total`), not this Mac.
- On `-play`, process names are the recorded ones. There is no local PID lookup, since the PIDs belong to the recording Mac.
- **v1 gap**: `Sys` (load, memory pressure, FreePct) comes from sysctl, not mactop, so it isn't in the recording. On replay those fields draw as `–`. Cloudflare panels show the runs saved on disk, as they do now.

## 2. Scrub

- A mode over every layout, not a new layout. Only the header changes.
- `space`: pause/resume. `[` / `]`: step one sample while paused. `{` / `}`: jump 30 samples.
- The header cursor reuses the 3D Scrubber glyphs: `◀ ▮▮▮▯▯ ▶  t−34s`.
- The status line reuses replay.js's format: `replay 34/90 · 1× · recorded 2026-09-26 19:44`.
- **Depth**: bounded by the in-memory `histLen = 400` (about 6.7 min at 1 s). No disk store in v1.
- Paused live data keeps collecting into the ring buffer. `space` resumes at live.

## 3. Diagnosis line

One line under the header, shown only while a rule fires. The highest-priority rule wins; no config.

| priority | rule | fields | text |
| --- | --- | --- | --- |
| 1 | thermal throttle | `ThermalState` ≠ `Nominal` | `▲ throttling: thermal ‹STATE› · cpu 94°C` |
| 2 | swap storm | `Memory.SwapUsed` rising > 256 MB over 10 samples and `Sys.Pressure` ≥ 2 | `▲ swapping: +1.2 GB in 10s · pressure warn` |
| 3 | runaway process | top `Process.CPUPercent` ≥ 100 (one full core) for 10 samples | `▲ runaway: replayd (900) 176% cpu for 45s` |

- `‹STATE›` is a placeholder in the mockups only. The build shows mactop's actual `thermal_state` string. Only `Nominal` has been seen in a recording (see TODO-future.md).
- While scrubbing, the line shows the rule state at the cursor's sample, so stepping back to a `▲` shows that alert again.
- `CPUPercent` is per core and goes past 100. The threshold is one full core: a spinning thread pegs exactly one, and it means the same on any chip, unlike a share of total cores. The recording has replayd at ~175% for 45 samples, so rule 3 fires on it.
- A rule clears after 5 samples below its threshold.
- Coloured with the existing tokens: `level-high` for throttle, `level-mid` for swap and runaway. Nothing new in the design system except the line itself.

## 4. Alerts

- A fired rule also marks its time on the scrubber (`▲` above the cursor track), so you can pause and step back to it.
- No sound and no macOS notification in v1.

## Decided

- Process actions (kill/renice): not in v1. Moved to TODO-future.md.
- Compare two windows and the disk store: v2, in TODO-future.md.
- Location: stays here for now. Reorganize into 2d/3d × prod/dev once the other agents finish.

## Mockup phase (after sign-off)

- Text frames at 140×42: scrub paused, diagnosis line (each rule), replay header, `tiles → glance` at small size.
- There is no text-to-PNG pipeline: `render.py` was removed in `3cdda90`, and the README screenshots come from real terminals. Plan: the HTML gallery renders the `.txt` frames in the design-system mono stack, and that stands in for PNGs.
- Then: hand the new elements (DiagnosisLine, Scrubber header, alert tick) to the prod agents and review what they build.
