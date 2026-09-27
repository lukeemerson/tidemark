The ultimate Master monitor TUI for mac,  the best of cloudflare, btop, vtop, mactop with a customizable gui

A terminal system monitor for Apple silicon, drawn in character cells: heavy box frames, braille graphs, block sparklines and square-cell meters, coloured by your Alacritty palette. Source: `bin/monitor` (draft 03, "tiles", adopted) and the ten framings in `drafts/`.

## Voice and copy

- Lowercase labels: box titles are `cpu`, `gpu`, `power`, `mem`, `processes`, `cloudflare`. Only proper names and the process table header (`PID`, `COMMAND`, `CPU%`) keep their case.
- Short nouns and units, no sentences. Write `E 2424 · P 3262 MHz`, `swap 0.0 / 16.0 GB`, `9h ago`.
- Separate facts with a middle dot and two spaces either side: `4E + 6P CPU  ·  16-core GPU`.
- Arrows name direction: `↓` in, read or download; `↑` out, write or upload.
- Status notes end in an ellipsis: `● starting mactop…`, `loading…`. Empty states say what to run: `no saved runs — run: cloudy`, `run cloudy for a fresh test`.
- Counts and legends are dim prose: `11 saved runs  ·  ▌ download  ▌ upload`.
- Two subjects in one frame join with a middle dot: `memory · sensors`. Two-word titles stay lowercase: `network in`.
- No emoji. The only glyphs are the ones in Glyphs below.

## Colour

Colours are ANSI slots, not hex, so the terminal's palette decides the final hue. Write `ESC[33m`, not a 24-bit colour. The hex values here are your Alacritty palette. Two slots are remapped there: yellow (`ansi-yellow`) renders lilac and magenta (`ansi-magenta`) renders amber.

- `text` on `bg` for readings and names. Bold `text` for titles and headline values.
- `dim` for everything structural: frames, labels, units, separators, the off cells of a bar, the clock.
- Load uses three levels from one rule, `level()`: under 50 is `level-low`, 50 to 79 is `level-mid`, 80 and up is `level-high`. Apply it to CPU, memory, per-core load, temperature and process CPU%.
- Each series without a natural threshold has a fixed colour: `series-gpu` for GPU, `series-power` for watts and upload, `series-net` for download.
- A braille graph with no fixed colour is shaded by row height with `level()`, so the top of a tall graph turns red.
- `accent` belongs to the cursor and the preview page. Don't use it for data.
- Every text colour is at least 5.3:1 on `bg`. `ansi-black` is 1.25:1 and only draws page hairlines.

## Type

One face at one size: `mono` (JetBrains Mono NL Nerd Font, no ligatures) as `cell`. Hierarchy comes from `cell-bold` and colour only. The preview page sets its own chrome in `sans` with `page-title`, `card-title`, `body`, and `mono-label` and `draft-num` for mono details.

## Layout

- Count in cells. One blank column each side (`cell-xpad`), one column between boxes (`cell-gap`), and a box's content is its width minus 4 (`cell-inset` each side).
- Screen order: header line, a row of 8 tiles (`tile-rows` tall), cpu and gpu graphs, cores and power, then processes on the left two thirds with memory, sensors, io and cloudflare stacked on the right.
- Tiles split the width evenly; the last tile takes the remainder.
- The process table takes whatever height is left and drops columns as it narrows (`ProcessTable`).
- The adopted frame is heavy: `┏ ┓ ┗ ┛ ━ ┃`. Use one frame style per screen (see Frames).

## Frames

| frame | characters | drafts | reads as |
| --- | --- | --- | --- |
| heavy | `┏ ┓ ┗ ┛ ━ ┃` | 03 (adopted) | solid tiles, the loudest frame |
| round | `╭ ╮ ╰ ╯ ─ │` | 01, 04, 06, 10 | btop-style, soft |
| square | `┌ ┐ └ ┘ ─ │` | 02, 08 | plain light boxes |
| double | `╔ ╗ ╚ ╝ ═ ║` | 07 | formal, busy at small sizes |
| dashed | `┌ ┐ └ ┘ ┄ ┆` | 09 | light enough for a small pane |
| rule | `── title ──`, no sides | 05 | the quietest; columns need `rule-gap` |

Draft 04 is one round frame divided by `├ ┼ ┤ ┬ ┴` junctions (`SectionedFrame`).

## Layouts

Each draft arranges the same panels differently. All start with the header line and give the process table whatever height is left.

- **01 · btop classic:** cpu graph and cores across the top, then mem, gpu and power in thirds; processes on the left three fifths, sensors, io and cloudflare stacked on the right.
- **02 · two stacks:** cpu, cores and memory on the left; gpu, power and cloudflare on the right; sensors and io under them; processes full width.
- **03 · tiles (adopted):** a row of 8 tiles, cpu and gpu graphs, cores and power, then processes on the left two thirds with memory, sensors, io and cloudflare on the right.
- **04 · instrument panel:** one frame in two columns (cpu | cores and temps, gpu | power, memory and io | cloudflare), processes in the full-width bottom section.
- **05 · rules:** two columns under rule-only frames, 3 cells apart; processes full width.
- **06 · sidebar:** every number in a 38-cell column titled with the machine name (`StatRow`, `RunBars`); the rest is cpu, gpu and power graphs over processes.
- **07 · three columns:** compute (cpu, cores, sensors) | graphics and power (gpu, power, memory) | network (io, cloudflare bars); processes full width.
- **08 · network first:** a full-width cloudflare box with stats beside 2-cell run bars, then cpu, gpu, power and memory in quarters, then processes with cores, sensors and io.
- **09 · compact:** sized for a 90×26 zellij pane. Short graphs, one `system` box of label rows, a small cloudflare box and processes.
- **10 · graph wall:** full-width cpu, gpu, power and network-in graphs with their numbers in the frame titles, then cores, memory · sensors and cloudflare in thirds.

## Data states

- Frames draw first, before mactop's first sample (about 2.8 s). Until then every number is a `dim` `—` at the width the number will take, so nothing shifts when data lands.
- The header shows `● starting mactop…` in `level-mid` until the first sample, then the clock in `dim`.
- History keeps `history` samples per series, newest on the right. Graphs and sparklines stay blank where there is no history yet.

## Glyphs

| glyph | use |
| --- | --- |
| `┏ ┓ ┗ ┛ ━ ┃` | heavy frame (`Box`, `Tile`) |
| `╭ ╮ ╰ ╯ ─ │` `┌ ┐ └ ┘` `╔ ╗ ╚ ╝ ═ ║` `┄ ┆` | other `Box` frames |
| `├ ┼ ┤ ┬ ┴` | shared dividers (`SectionedFrame`) |
| `■` / `·` | meter on / off cell (`Bar`) |
| `▁▂▃▄▅▆▇█` | 8-level sparkline (`Sparkline`), per-core strip (`StatRow`), eighth-steps in `RunBars` |
| `▰` / `▱` | sidebar meter on / off (`StatRow`) |
| `▌` | legend swatch before a series name |
| `U+2800–28FF` | braille graph, 2 samples × 4 dots per cell (`BrailleGraph`) |
| `↓ ↑` | in / out |
| `·` | fact separator |
| `●` | status dot |
| `…` | truncation and pending status |

## Changed since the drafts

`bin/monitor` fixed three things the drafts share through `drafts/core.zsh`. Follow the monitor:

- Bar off-cells were a dim `■`. They're now `·`, so a bar's length reads without colour.
- The battery label was mactop's state and percent (`Battery Power 80%`). It's now `battery 80%`, plus ` charging` or ` on AC`.
- A process mactop names only by version (`2.1.283`) now shows argv[0]'s basename.

## Preview page

Rendered drafts sit on `bg-page` in cards filled with `bg`, bordered in `ansi-black`, with `radius-card` corners. The live / before-data toggle is a segmented control with `radius-seg` corners; the selected segment is filled with `ansi-black`.
