# Spec: settings menu (2D, tidemark)

One overlay for the settings tidemark already has. Nothing new becomes configurable.

## 1. Contents

| row | values | today |
| --- | --- | --- |
| layout | the 10 layouts | `l` / `L`, saved |
| palette | `ansi`, `tidemark` | `c`, saved |
| interval | 250, 500, 1000, 2000, 5000 ms | `-i`, not saved |
| store | on, off | `-nostore`, not saved |

- `interval` and `store` are now saved in `config.json` as well. Their flags still override for that one run.
- A row set by a flag this run is dim and reads `2000 ms (-i)` or `off (-nostore)`. It can't be changed from the menu.
- During `-play`, `interval` and `store` are dim and read `— (replay)`.

## 2. Opening and using it

- `o` opens it (btop's key; `m` is taken by compare). `o` or `esc` closes it.
- `↑` `↓` move, and `←` `→` change the selected row. Values wrap around.
- Changes apply live: the layout redraws, the palette swaps, the interval restarts mactop at the new rate (a gap of about a second in the graphs), and the store opens or closes.
- The config is saved once, when the menu closes. `l` and `c` keep saving as they do today.
- While the menu is open, other keys do nothing except `q`, which still quits. Samples keep arriving, and pause and scrub state are kept.

## 3. Look

```
╔═ settings ═══════════════════════╗
║ ▸ layout     ◂ sidebar ▸         ║
║   palette    ◂ ansi ▸            ║
║   interval   ◂ 1000 ms ▸         ║
║   store      ◂ on ▸              ║
║                                  ║
║ ↑↓ move  ←→ change  o close      ║
╚══════════════════════════════════╝
```

- A double frame in `dim`, with the title in bold text and centred over the layout. The layout stays visible around it.
- The selected row has `▸` and its `◂ ▸` in `level-mid` (the cursor colour, as on the scrubber). Other rows' arrows are `dim`, and values are in text colour.
- 36 × 8 cells. Below 38 columns it spans the full width, and below 11 rows it drops the blank line and the key hints.

## 4. Out of scope

- New settings: diagnosis thresholds, store retention and location, and colours per series.
- Mouse support.
- A settings file format beyond the two new keys in `config.json`.

## Judgment calls (design)

- **Flags win and lock their row:** a one-run flag shouldn't be silently overwritten by the menu or saved over.
- **Save on close, not on each change:** stepping through the ten layouts doesn't write the file ten times.
- **Fixed interval steps:** five values cover mactop's useful range without a number input.
