A one-line label, value and optional meter, for a narrow column that holds every number.

- From draft 06, "sidebar" (38 cells wide), and draft 09's compact `system` box.
- You supply: label, value (a coloured line), percent for the meter or none, the column's inner width, and optionally the label width (7 by default; draft 09 uses 6).
- The label is `dim` and cut to its width. The meter uses `▰` on and `▱` off, fills the last inner width − 21 cells, and is coloured by `level()`.
- A value can be a strip of per-core blocks: one `▁…█` per core, each in its own `level()` colour.
- Break a column into groups with a blank row, and title a group with `rule()`: `── cloudflare ───…` in `dim`.
