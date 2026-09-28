# Spec: compare view (2D, tidemark)

Compare what's on screen now (A) against a marked moment (B) as an overlay on the existing layouts. Needs the disk store (STORE-SPEC.md) for any B older than 400 samples.

## 1. Marking B

- `m` marks the sample under the cursor as B: the paused cursor, or the newest sample when live. `m` again clears it.
- B can be anywhere in the store (400, 1h or 24h span). A keeps following live, or the cursor while paused.
- Marking B in a gap (dim `·`) snaps to the nearest stored sample.
- B is session-only. It isn't saved and doesn't survive a restart.

## 2. What changes on screen

- **Graphs:** B's window draws as a dim ghost behind A, in `dim`. A keeps its series colour on top, and A wins where both have a dot. Both windows are end-aligned: A's last column is A's sample, and B's last column is B's sample. Auto-scaled graphs (power, rates, swap, load) scale to the larger of A and B, and their top label follows. Fixed-scale graphs (%, °) keep their scale.
- **Tile and box values:** A's value, then its difference from B: `37%  ▲+12` or `24.9 W  ▼−3.1`. The difference is in `dim` with ▲/▼. It is never in a state colour, since colour roles keep green, amber and red for state. When the two match after rounding it reads `±0`.
- **Header:** after the time, `·  vs B t−3h12m` (how far back B is from A). The compact form is `vs −3h12m`.
- **Track:** B's position shows as a dim `B` in the cell holding it.
- Unchanged: process lists, the diagnosis line, bars, meters and block sparklines (tiles, glance stat lines) all show A only. Block characters can't overlay, so only braille graphs ghost.

## 3. Resolution

- The ghost covers the same duration as A's graph. When B sits in the summary tier (10 s buckets) and A is at full resolution, each bucket fills the columns its 10 s spans. The ghost is coarser, not shorter.
- The difference compares A's sample against B's sample or bucket average.
- A tile says `Δ —` when B has no value for that series (for example, memory pressure in a `-play` file).

## 4. Out of scope

- Comparing two recordings, or a recording against live.
- Fixed-offset shortcuts (yesterday, an hour ago).
- Ghosts for process lists, bars or the terrain in 3D.
- Saving a mark.

## Judgment calls (design)

- **End-aligned windows:** "the minute before A" against "the minute before B" is the natural reading of a moment compare.
- **The difference in dim, not coloured:** CPU going up isn't good or bad by itself, so the arrows carry direction without implying state.
- **`m` for mark:** it's free. `b` was considered, but `b` and `B` would be ambiguous next to `l`/`L`.
