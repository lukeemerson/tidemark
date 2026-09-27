A framed panel with a bold lowercase title on the top edge and an optional right-hand label.

- Use for every panel. The title is one lowercase noun; the right label is the panel's headline value (`28%`, `15.9 W`) or an age (`9h ago`).
- You supply: title, right label (coloured line or none), width in cells, height in rows (`0` fits the content), content lines, and optionally a frame.
- Frames: `heavy` (default, the adopted draft 03), `round` (01, 04, 06, 10), `square` (02, 08), `double` (07), `dashed` (09) and `rule` (05). Use one frame per screen.
- `rule` draws only the titled top edge. Its bottom row is blank, so pair it with a 3-cell gap (`rule-gap`) between columns.
- Content is padded to width − 4. A line that's too long is cut to plain text with `…`, losing its colour.
- Too many lines are dropped from the bottom. Too few are padded with blank rows.
- Don't nest boxes. Place them side by side with `hjoin(1, …)`. Draft 10 packs details into the right label instead of a content row: `E 2424 · P 3262 MHz   28%`.
