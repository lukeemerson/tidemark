# Plan: a separate 3D interactive monitor component

Goal: a 3D view of the same mactop data, built as its own component beside `bin/monitor` and drawn from the Monitor TUI design system.

1. **Review** what exists (`REVIEW.md`). ✔
2. **Mock up** five concepts (`mockups/index.html`). ✔
3. **Evaluate** them against signal, fit, interaction, Ratty, ship and cost (`EVALUATION.md`). The picks are A, B, C and E; D is dropped. ✔
4. **Design the components**: TermSurface, CellAnchor, AnchoredObject, LoadColumn, DieBlock, Terrain, TuiHud, Scrubber and CameraRig (`COMPONENTS.md`). ✔
5. **Build prototypes** in `prototypes/`, split across three agents:
   - Agent 1: `a-vt-room/` (A, the Ratty presentation layer in three.js)
   - Agent 2: `b-silicon-floor/` and `c-history-terrain/` (B and C, which share LoadColumn, TuiHud and CameraRig)
   - Agent 3: `e-terminal-3d/` (E, Python stdlib, braille 3D, RGP when inside Ratty)
6. **Verify** each prototype: load it, screenshot it, check the console, run E in a PTY. Record the results in `RESULTS.md`, with a gallery at `index.html`.
7. **Next, if one wins:** port it into the Go/Bubble Tea plan (E), or run the web view as a companion window fed by the same mactop stream (A, B, C).

Committed on branch `3d`; `main` is untouched.

## tidalrat (Rust CLI, branch `3d`)

`main` is production (tidemark, Go). All 3D work lives on `3d` until you merge it. tidalrat is Rust + Ratatui, as a pilot for a Rust rewrite.

| # | command | status |
| --- | --- | --- |
| 1 | `--cores`: live 3D core chart in braille, real 3D cubes inside Ratty | **built** (`tidalrat/`) |
| 2 | `--terrain`: C in the terminal, a braille history landscape you can rotate and scrub | option |
| 3 | `--die`: B in the terminal, an isometric chip floorplan | option |
| 4 | `--web`: serve live mactop to the browser prototypes A, B and C | option |
| 5 | `--ratty`: tidemark's screen with 3D objects pinned to panels (needs Ratty) | option |
| 6 | `--record` / `--replay`: save and play back sessions for any view | option |
| 7 | `--wall`: a tidemark-style layout with a switchable 3D panel | option |
