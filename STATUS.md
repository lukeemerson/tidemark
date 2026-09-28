# Status

Where tidemark stands, as of 2026-09-28. Everything is on `main`; the `3d` branch is merged and retired. Layout: `2d/` and `3d/`, each with `prod/` (shipped code) and `dev/` (specs, mockups, plans, prototypes); `3d/shared/` holds data both 3D halves use.

## Built

### 2D: tidemark (Go, `2d/prod/`)
- 10 layouts on `l`/`L`, each scaling down to `tiles` at 20×20.
- Colour roles: one ANSI slot per series. Upload uses slot 7 on dark, 0 on light. Built-in truecolor `tidemark` palette on `c`.
- Readable without colour: textured upload bars, and a labelled scale on every graph.
- **History** (SPEC.md): `-rec` / `-play` with mactop's raw stream; `space` pause, `[ ]` step, `{ }` ±30 through 400 samples; alert ticks `▲` in the track.
- **Diagnosis line**: throttle > swap > runaway (≥100% CPU for 10 samples); it follows the scrub cursor.
- **Disk store** (STORE-SPEC.md): 24 h on disk, with a raw hour at full detail plus 10 s summaries; `z` cycles the span 400 / 1h / 24h, with gaps and stored alert ticks.
- **Compare** (COMPARE-SPEC.md): `m` marks B. B draws as a dim ghost on braille graphs, auto-scaled to the larger of A and B, and values show the difference from B.
- **Settings** (SETTINGS-SPEC.md): `o` opens an overlay for layout, palette, interval and store. Changes apply live and save on close.
- Release packaging: `scripts/release.sh`, MIT.

### 3D: tidalrat (Rust, `3d/prod/tidalrat/`)
- `--cores`: a live 3D core chart in braille, a fixed size spinning in place.
- `--terrain`: per-core history as a braille landscape. Same scrub keys as 2D; `--play <file>` reads the same recordings.
- Terrain alert markers (TERRAIN-ALERTS-SPEC.md): tidalrat runs the three diagnosis rules, `▲` beside the floor, ticks and a count in the header.
- 18 tests pass.

### 3D: prototypes (`3d/dev/prototypes/`)
- A (VT room, three.js), B (silicon floor), C (history terrain), E (terminal 3D, Python). D was dropped.
- Evaluation, components and results are in `3d/dev/*.md`.

### Design
- `design-system/`: tokens and 11 components with HTML previews.
- `2d/dev/`: the 2D specs, mockups and gallery. `3d/dev/TERRAIN-ALERTS-SPEC.md` for 3D.

## Open
- `-play` process names come from this Mac's PID lookup. The fix is decided for v1 and is with build-2d (SPEC §1).
- A corrupt line in the middle of a `-play` file ends playback there (minor, known).
- The real non-`Nominal` thermal state names are unverified: record mactop under sustained load.

## Next
Proposed, not yet decided with Luke:

| area | item | source |
| --- | --- | --- |
| repo | **next:** a deeper config map | Luke |
| 2D | "what changed?" feed; processes grouped by project | README |
| 2D | process actions (kill/renice with a y/n confirm) | TODO-future |
| 3D | `--die` chip floorplan, then the throttle hotspot on it | PLAN row 3 |
| 3D | `--web`, `--wall` | PLAN rows 4, 7 |
| both | in-process sampling (vendored mactop IOReport), dropping the ~1–1.8 s startup | 2d/dev/PLAN.md |
| both | Rust rewrite: tidalrat is the pilot | 3d/dev/PLAN.md |

## Team
- design: specs, mockups, sign-off, and general manager.
- build-2d: Go, `main`.
- build-3d: Rust and prototypes, now also on `main`.
- review: reviews every commit, read-only.
