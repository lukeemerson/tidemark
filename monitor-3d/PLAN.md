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

Nothing in `monitor-3d/` is committed. It stays in the working tree.
