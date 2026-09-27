One outer frame split into two columns by shared dividers, with a full-width section at the bottom.

- From draft 04, "instrument panel". Use it instead of separate boxes when the screen should read as one instrument.
- You supply: the left and right column widths, a list of bands (each a left and right `[title, right label, lines]`), and a bottom `[title, lines]` that spans both columns. Total width is left + right + 3.
- The first band opens with `╭ ┬ ╮`. Later bands use `├ ┼ ┤`. The bottom section closes the column with `┴` and ends in `╰ ╯`.
- A band is as tall as its taller side. The shorter side is padded with blank rows.
- Always the round, light frame. Don't mix it with boxes on the same screen.
