# Spec: disk store (2D, tidemark)

24 hours of scrubbable history on disk, on by default. Builds on SPEC.md (record, replay, scrub, diagnosis).

## 1. What's kept

Two tiers under `~/Library/Application Support/tidemark/history/`:

| tier | span | resolution | content | file |
| --- | --- | --- | --- | --- |
| full | last 1 h | every sample | mactop's raw `--headless` line, the same as `-rec` | `raw-YYYYMMDD-HH.raw`, one per clock hour |
| summary | last 24 h | 10 s buckets | every graphed series (average and peak) and the averaged panel values, load and memory pressure, the top 5 processes by CPU (name, PID, peak %), and whether a diagnosis rule fired in the bucket | `sum-YYYYMMDD.jsonl`, one line per bucket |

- A raw hour file is a valid `-play` recording, so an hour can be replayed or shared as-is.
- The summary lines carry `Sys` (load, pressure), so the summary tier has none of the replay gap.
- Sizes: raw is about 28 MB an hour, and the current and previous hour files are both kept, so up to about 56 MB. Summary lines also carry the averaged panel values (per-core loads, frequencies, temperatures, net and disk), about 2 KB each, so about 17 MB for 24 h. Budget: about 75 MB total.

## 2. Writing and pruning

- On by default whenever tidemark runs live. `-nostore` turns it off. `-play` never writes.
- Every sample is appended to the current raw hour file, and each 10 s bucket to the summary when it closes.
- At startup and on the hour: delete raw files except the current and previous hour's and summary lines older than 24 h. A partially written last line is skipped, as `-play` already does.
- A second tidemark running at the same time doesn't write: an advisory lock on the directory. The first one owns the store.

## 3. Scrubbing it

- `z` cycles the track's span: `400` (today's in-memory behaviour) → `1h` (raw tier) → `24h` (summary tier).
- The header shows the span next to the track: `‖ paused   ◀ ▮▮▮▲▮▮▯▯ ▶  t−3h12m  ·  24h  ·  2 alerts`.
- `[ ]` step one unit of the tier (a sample, or 10 s), and `{ }` jump 30 units.
- On the summary tier, panels draw the bucket's averages, and graphs end at the cursor. The process list shows the bucket's top 5 with their peak %.
- Times when tidemark wasn't running are blank in graphs and `·` in the track (dim). Stepping skips over them.
- Alert ticks come from the stored diagnosis flags, so a `▲` from this morning stays visible.
- Live samples keep being stored while paused, as in memory today.

## 4. Out of scope

- The compare-two-windows view: its own spec, next, built on this store.
- A background recorder that runs while tidemark is closed (launchd).
- tidalrat reading the store. That goes in TODO-future.md; the raw hour files already work with `--play`.
- Settings for retention or location. Values are fixed as above.

## Judgment calls (design)

- **`z` for span, rather than more keys:** one key cycles three spans and matches the l/L pattern.
- **Clock-hour raw files, not one rolling file:** pruning is a file delete, never a rewrite, and each file is a ready `-play` recording.
- **The summary keeps the top 5 by CPU only.** Top by memory isn't kept, so the memory layout's process list on the summary tier shows the CPU top 5, labelled as such.
