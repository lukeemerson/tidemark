# Evaluation of mockups A–E

Each concept is scored 1–5 on six criteria. **Signal** means it shows something the flat TUI can't. **Fit** means it stays inside the Monitor TUI system (tokens, cells, copy). **Interaction** means depth is something you operate, not decoration. **Ratty** means it uses the Ratty mechanics. **Ship** means there's a path into the real tool. **Cost** means prototype effort; higher is cheaper.

| | concept | signal | fit | interaction | ratty | ship | cost | total |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| A | VT in the room | 3 | 5 | 4 | 5 | 3 | 3 | **23** |
| B | Silicon floor | 5 | 4 | 4 | 3 | 3 | 4 | **23** |
| C | History terrain | 5 | 4 | 5 | 2 | 3 | 4 | **23** |
| D | Panel constellation | 1 | 4 | 3 | 3 | 2 | 4 | 17 |
| E | Terminal-native 3D | 3 | 5 | 3 | 4 | 5 | 4 | **24** |

## Decisions
- **Build A, B, C and E.** Each answers a different question:
  - A: what does the real TUI look like as a 3D object?
  - B: where on the chip is the work?
  - C: how did load move over time?
  - E: can any of this run in a terminal?
- **Drop D.** Spreading panels through space adds camera travel but no new information. Its one good idea, click a panel to focus it, moves into A.
- **Shared rules for all four:**
  - Draw every colour from `level-*`, `series-*` and `dim`, over `bg`.
  - Set all text in the mono face and follow the TUI's copy rules.
  - Replay `shared/samples.js` so all four show the same 90 seconds.
- **Anchoring:** A and E place 3D objects by terminal cell (row, col, width, height, depth), following Ratty's graphics protocol. B and C are free 3D scenes, but their overlays are Monitor TUI boxes.

## Scoring notes
- A scores highest on Ratty and fit because it renders the actual `bundle.js` frames. Its signal is lower because the numbers are the same ones the TUI already shows.
- E has the best ship score because it's the only one that fits `bin/monitor` or the Go port. Its signal is capped by braille resolution.
