# 3D components

The contract every prototype builds on. Browser prototypes use three.js 0.147.0, the UMD build and `examples/js` OrbitControls from jsDelivr, so they open straight from `file://`. The terminal prototype uses Python 3 with the standard library only.

## Shared inputs (`shared/`)
- `samples.js` → `window.SAMPLES = {meta, samples[90]}`. Each sample has `cpu gpu cores[10] efreq pfreq gfreq pw pwg tc tg dr dw mu mt nin nout dkr dkw procs[12]`.
- `replay.js` → `window.Replay`: plays at 1 Hz and loops. It exposes `onSample(fn(sample, hist, i))`, `step(±1)`, `toggle()`, `setSpeed(x)`, and `hist.{cpu,gpu,pw,mem,tc,nin,cores}`, each holding up to 400 samples.
- `theme.js` → `window.THEME[token] = '#hex'`. `tokens.css` defines the same tokens as CSS variables.
- `../../design-system/components/bundle.js` → `window.MonitorTUI`, which draws the real TUI boxes, bars, graphs and process table as line segments.

## Components

| component | job | Ratty source |
| --- | --- | --- |
| **TermSurface** | Draws `MonitorTUI` lines onto a canvas cell grid (JetBrains Mono, 11.5px/1.18 metrics × devicePixelRatio). The canvas is a `CanvasTexture` on a plane, which can bend with a `curvature` value from 0 to 1. Only re-uploads when a sample lands. | The Ratatui buffer drawn to a texture, then Bevy puts it in a scene. |
| **CellAnchor** | `cellToScene(row, col, rows, cols, plane)` returns the centre of that cell in scene space, using the blog's `cell_width`/`cell_height` mapping. | The cursor mapping. |
| **AnchoredObject** | A mesh placed by `{id, row, col, w, h, depth, scale, color, brightness, animate}`. It follows its anchor when the surface moves or bends. | The RGP `r`/`p`/`d` verbs. |
| **LoadColumn** | A box whose height eases to `load/100 × maxH` over about 400 ms. It's coloured by `level()`, the flat colour unlit plus a little emissive, with a dim wireframe cap. | |
| **DieBlock** | A floorplan tile (E core, P core, GPU core, memory). It extrudes with load, and its emissive intensity follows temperature. | |
| **Terrain** | A cores × time grid mesh with vertex colours from `level()`, per-core ridge lines, and a floor grid in `dim`. | |
| **TuiHud** | A DOM `<pre class="mt-screen">` drawn with `MonitorTUI.box`: the tooltip on hover, the corner status line, and the key legend. Never a CSS card. | |
| **Scrubber** | A time cursor drawn in the TUI's glyphs: `◀ ▮▮▮▯▯ ▶  t−34s`. | |
| **CameraRig** | OrbitControls with damping, plus presets on `1`–`4` that tween over 600 ms. Respects `prefers-reduced-motion`, which cuts the tween. | |

## Interaction contract (all prototypes)
- **Mouse:** drag to orbit, wheel to zoom, hover to see a TUI tooltip, click to focus or isolate.
- **Keys:**
  - `space`: pause
  - `←` / `→`: step one sample while paused
  - `1`–`4`: camera presets
  - `?`: show the key legend
- The page opens in a working state: data already replaying and the camera already framed. No splash screen.
- **Status line:** bottom-left, dim, e.g. `replay 34/90 · 1× · recorded 2026-09-26 19:44`.

## Visual rules
- **Ground:** `bg` `#1d1d1d`. No sky gradients, no bloom that washes the palette out, no purple-to-blue.
- **Lighting:** one soft key light and one ambient. Colours must still read as their tokens.
- **Type:** mono only for anything the viewer reads. Titles are lowercase, per the brand book.
- **Chip layout:** the M2 Pro floorplan in B is schematic, and the page labels it that way.
