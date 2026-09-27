# mac-monitor

A terminal system monitor for Apple Silicon Macs: CPU, GPU, power, memory pressure, thermals,
network and your saved Cloudflare speed tests, with ten layouts on the `l` key. Written in Go
with [Bubble Tea](https://github.com/charmbracelet/bubbletea); data comes from
[mactop](https://github.com/metaspartan/mactop).

<table>
  <tr>
    <th></th>
    <th>Ghostty</th>
    <th>Alacritty</th>
  </tr>
  <tr>
    <th>light<br><sub>Gruvbox Material Light</sub><br><sub><code>sidebar</code></sub></th>
    <td><img src="docs/screenshots/light-ghostty.png" alt="sidebar layout, Gruvbox Material Light, Ghostty"></td>
    <td><img src="docs/screenshots/light-alacritty.png" alt="sidebar layout, Gruvbox Material Light, Alacritty"></td>
  </tr>
  <tr>
    <th>warm dark<br><sub>Earthsong-based</sub><br><sub><code>console</code></sub></th>
    <td><img src="docs/screenshots/yours-ghostty.png" alt="console layout, warm dark theme, Ghostty"></td>
    <td><img src="docs/screenshots/yours-alacritty.png" alt="console layout, warm dark theme, Alacritty"></td>
  </tr>
  <tr>
    <th>soft dark<br><sub>Gruvbox Material Dark</sub><br><sub><code>compute</code></sub></th>
    <td><img src="docs/screenshots/soft-ghostty.png" alt="compute layout, Gruvbox Material Dark, Ghostty"></td>
    <td><img src="docs/screenshots/soft-alacritty.png" alt="compute layout, Gruvbox Material Dark, Alacritty"></td>
  </tr>
</table>

<sub>Real 140×42 terminal windows running the recorded mactop sample from the tests. The monitor
only uses the terminal's 16 ANSI colours, so it takes on whatever theme you run.</sub>

## Run it

Needs macOS on Apple Silicon, Go 1.27+, and mactop.

```sh
brew install mactop
git clone https://github.com/lukeemerson/mac-monitor && cd mac-monitor
go build -o ~/.local/bin/monitor ./cmd/monitor
monitor
```

`r` needs [cloudflare-speed-cli](https://github.com/kavehtehrani/cloudflare-speed-cli). Without it
the cloudflare panels still show any runs it saved earlier.

| key | does |
| --- | --- |
| `l` / `L` | next / previous layout (remembered in `~/.config/mac-monitor/config.json`) |
| `r` | run a Cloudflare speed test (~30 s) and reload the cloudflare panels |
| `q` | quit |

`monitor -i 500` samples every 500 ms (default 1000).

## Layouts

Each layout has its own frame style. Every one scales down: below its minimum size it hands
over to `tiles`, which fits down to 20×20, and the header says so (`sidebar → tiles`).

| # | layout | frame | for |
| --- | --- | --- | --- |
| 1 | `tiles` | heavy `┏━┓` | the overview: 8 summary tiles, graphs, processes and a side column |
| 2 | `sidebar` | rounded `╭─╮` | every number in one column, graphs over processes |
| 3 | `instrument` | double `╔═╦═╗` | one frame split by shared dividers; bars and numbers, full-width processes |
| 4 | `console` | double + tile band | instrument with the summary tiles closing the frame |
| 5 | `compute` | square `┌─┐` | CPU and GPU side by side, down to the top CPU and GPU processes |
| 6 | `memory` | dashed `┌╌┐` | usage, kernel pressure, swap and load; processes by memory |
| 7 | `io` | heavy dashed `┏╍┓` | network and disk throughput, the full speed-test history |
| 8 | `glance` | rules `── ──` | a small pane: one line per stat, sensors, top processes |
| 9 | `wall` | block `▛▀▜` | every history as a full-width graph, no tables |
| 10 | `thermal` | ascii `+-+` | power and heat: energy readout, CPU and GPU temperature graphs |

## Components

Everything is drawn from a few pieces in [`internal/ui/draw.go`](internal/ui/draw.go). These
examples are printed by that code from the test sample. Their design tokens and HTML previews
are in [`design-system/`](design-system/).

**Frames:** one `border` set per layout. Titles sit in the top edge and values in its right end.

```
┏━ heavy ━━━━━━┓ ╭─ rounded ────╮ ╔═ double ═════╗ ┌─ square ─────┐ ┌╌ dashed ╌╌╌╌╌┐
┃              ┃ │              │ ║              ║ │              │ ╎              ╎
┗━━━━━━━━━━━━━━┛ ╰──────────────╯ ╚══════════════╝ └──────────────┘ └╌╌╌╌╌╌╌╌╌╌╌╌╌╌┘

┏╍ heavy dash ╍┓ +- ascii ------+ ▛▀ block ▀▀▀▀▀▀▜ ── rules ───────
╏              ╏ |              | ▌              ▐
┗╍╍╍╍╍╍╍╍╍╍╍╍╍╍┛ +--------------+ ▙▄▄▄▄▄▄▄▄▄▄▄▄▄▄▟
```

**Tile:** a value and its sparkline.

```
┏━ cpu ━━━━━━━┓ ┏━ gpu ━━━━━━━┓ ┏━ power ━━━━━┓ ┏━ mem ━━━━━━━━┓
┃     28%     ┃ ┃      2%     ┃ ┃   15.9 W    ┃ ┃      75%     ┃
┃ ▅▄▄▃▃▃▃▃▃▃▃ ┃ ┃ ▃▁▁▁▁▁▁▁▁▁▁ ┃ ┃ ▅▇▆▅▅▅▄▄▄▄▄ ┃ ┃ ▆▆▆▆▆▆▆▆▆▆▆▆ ┃
┗━━━━━━━━━━━━━┛ ┗━━━━━━━━━━━━━┛ ┗━━━━━━━━━━━━━┛ ┗━━━━━━━━━━━━━━┛
```

**Braille graph:** two samples per cell, four dots high, coloured by height (green, amber, red)
or in one fixed colour.

```
┏━ cpu ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  28% ━┓
┃  E 2424 · P 3262 MHz                     ┃
┃ ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⣄⣄⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠀⠀⠀⠀⠀⠀⠀ ┃
┃ ⣀⣀⣀⢀⣄⢀⣠⢠⣀⣀⣾⣿⣶⣄⣀⣀⢠⣿⣿⣼⣿⣦⣤⣀⣀⢀⡀⢀⣀⣄⡀⣀⣿⣶⣼⣦⣀⣠⣀⠀ ┃
┃ ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿ ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

**Bars:** `■·` in most layouts and `▰▱` in the sidebar. Cores get a bar each.

```
■■■■■■■■■■■■■■■·········
▰▰▰▰▰▰▰▰▰▰▰▰▰▰▰▱▱▱▱▱▱▱▱▱

E1 ■■■■■■············  36%   E2 ■■■■··············  22%
E3 ■■■···············  14%   E4 ■■················   9%
```

**Stat lines:** the tile row folded to one line each for narrow terminals.

```
cpu    28%      ▃▃▃▃▃▃▃▃▅▅▄▄▄▅▄▄▃▃▃▃▃▃▃▃
gpu     2%      ▁▁▂▂▁▁▁▂▁▂▁▂▂▃▁▁▁▁▁▁▁▁▁▁
power 15.9 W    ▅▅▅▅▆▅▅▅▆▅█▇▅▅▇▆▅▅▅▄▄▄▄▄
```

**Run bars:** download and upload for each saved speed test, oldest on the left.

```
   ▅  ▄  ▅  ▂  █  ▆  ▆
█  █  █  █  █  █  █  █
█▁ █▆ █▅ █▃ █▁ █▅ █▆ █▅
██ ██ ██ ██ ██ ██ ██ ██
```

**Bands:** sections sharing one frame's dividers (`instrument`, `console`).

```
╔═ gpu ═════════════════════   2% ═╦═ power ═════════ 15.9 W ═╗
║  16 cores  ·   444 MHz  ·  ANE … ║ system 15.8 W            ║
║ ■······························· ║ ■■■■■■■■■■·············· ║
╚══════════════════════════════════╩══════════════════════════╝
```

## Where the numbers come from

- **mactop** (`--headless`, streamed): CPU, GPU, ANE, per-core use, frequencies, power,
  temperatures, fans, memory, DRAM bandwidth, network and disk rates, battery and the top 20
  processes. `monitor` draws its frames before the first sample, then fills them in.
- **sysctl:** load averages (`vm.loadavg`), kernel memory pressure and the available %
  (`kern.memorystatus_*`), and the names of processes that mactop reports only as a version number.
- **cloudflare-speed-cli:** the saved runs in `~/Library/Application Support/cloudflare-speed-cli/runs`.
  They're read from disk; a test only runs when you press `r`.

If mactop exits or sends something unreadable, `monitor` quits with the reason and exit status 1.

## Development

```sh
go test ./...
```

- **Decoding:** a 90-second mactop recording (`internal/source/testdata`) drives the tests.
- **Sweep:** every layout is drawn at sizes from 1×1 to 200×80, before and after data. The test
  fails if a line overflows, the frame doesn't fit, a box is left open at the bottom, or a box
  interior stays blank for 4+ rows.

[`PLAN.md`](PLAN.md) has the build notes and what comes next: an on-demand settings menu, then
rewind, a "what changed?" feed, and processes grouped by project.
