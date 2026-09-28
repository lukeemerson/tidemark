# Status

Where tidemark stands, as of 2026-09-28. Everything is on `main`; the `3d` branch is merged and retired.

## Built

### 2D: tidemark (Go, `cmd/` + `internal/`)
- 10 layouts on `l`/`L`, each scaling down to `tiles` at 20×20.
- Colour roles: one ANSI slot per series. Upload uses slot 7 on dark, 0 on light. Built-in truecolor `tidemark` palette on `c`.
- Readable without colour: textured upload bars, and a labelled scale on every graph.
- **History** (SPEC.md): `-rec` / `-play` with mactop's raw stream; `space` pause, `[ ]` step, `{ }` ±30 through 400 samples; alert ticks `▲` in the track.
- **Diagnosis line**: throttle > swap > runaway (≥100% CPU for 10 samples); it follows the scrub cursor.
- Release packaging: `scripts/release.sh`, MIT.

### 3D: tidalrat (Rust, `monitor-3d/tidalrat/`)
- `--cores`: a live 3D core chart in braille, a fixed size spinning in place; real cubes inside Ratty.
- `--terrain`: per-core history as a braille landscape. Same scrub keys as 2D; `--play <file>` reads the same recordings.
- 18 tests pass.

### 3D: prototypes (`monitor-3d/prototypes/`)
- A (VT room, three.js), B (silicon floor), C (history terrain), E (terminal 3D, Python). D was dropped.
- Evaluation, components and results are in `monitor-3d/*.md`.

### Design
- `design-system/`: tokens and 11 components with HTML previews.
- `monitor-3d/prod/`: the history/diagnosis spec, mockups and gallery.

## Open
- `-play` of another Mac's recording shows this Mac's chip and core split. The fix is decided for v1 and is with build-2d (SPEC §1).
- A corrupt line in the middle of a `-play` file ends playback there (minor, known).
- The tidalrat `--terrain` header cuts its left side mid-token at 110 columns, and its key hints are duplicated in the footer (design finding, sent to build-3d).
- The real non-`Nominal` thermal state names are unverified: record mactop under sustained load.
- `monitor-3d/PLAN.md` still says 3D lives on branch `3d`.

## Next
Proposed, not yet decided with Luke:

| area | item | source |
| --- | --- | --- |
| 2D | on-demand settings menu | README |
| 2D | "what changed?" feed; processes grouped by project | README |
| 2D | disk store, which unlocks compare-two-windows and history longer than 400 samples | TODO-future |
| 2D | process actions (kill/renice with a y/n confirm) | TODO-future |
| 3D | alert `▲` markers on the terrain's time axis | TODO-future |
| 3D | `--die` chip floorplan, then the throttle hotspot on it | PLAN row 3 |
| 3D | `--web`, `--ratty`, `--wall` | PLAN rows 4, 5, 7 |
| both | in-process sampling (vendored mactop IOReport), dropping the ~1–1.8 s startup | PLAN.md |
| both | Rust rewrite: tidalrat is the pilot | monitor-3d/PLAN.md |
| repo | reorganize into 2d/3d × prod/dev | Luke, earlier |

## Team
- design: specs, mockups, sign-off, and general manager.
- build-2d: Go, `main`.
- build-3d: Rust and prototypes, now also on `main`.
- review: reviews every commit, read-only.
