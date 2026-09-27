import html, sys, pyte

# alacritty palette (normal, bright)
PAL = {
    "black": "#302e2b", "red": "#f06459", "green": "#a3be8c", "brown": "#c898ca",
    "yellow": "#c898ca", "blue": "#81b0e0", "magenta": "#f4b83f", "cyan": "#65b9ce",
    "white": "#bdb3a7", "brightblack": "#9b9288", "brightred": "#ff897d",
    "brightgreen": "#bcd8a5", "brightyellow": "#e2b1e4", "brightbrown": "#e2b1e4",
    "brightblue": "#9acafb", "brightmagenta": "#ffda62", "brightcyan": "#7fd3e8",
    "brightwhite": "#ede3d6", "default": "#d6cbbd",
}

class Screen(pyte.Screen):
    # pyte lacks CBT (ESC[Z); Bubble Tea moves the cursor with it
    def cursor_back_tab(self, count=None, *a, **k):
        for _ in range(count or 1):
            stops = [t for t in sorted(self.tabstops) if t < self.cursor.x]
            self.cursor.x = stops[-1] if stops else 0

pyte.Stream.csi = dict(pyte.Stream.csi, Z="cursor_back_tab")

src, cols, rows = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
screen = Screen(cols, rows)
stream = pyte.ByteStream(screen)
data = open(src, "rb").read()
# stop at the last full frame (cleanup exits the alt screen)
stream.feed(data)

out = []
for y in range(rows):
    line = screen.buffer[y]
    run_style, run_text, parts = None, "", []
    for x in range(cols):
        ch = line[x]
        fg = PAL.get(ch.fg, "#" + ch.fg if len(ch.fg) == 6 else PAL["default"])
        style = (fg, ch.bold)
        if style != run_style and run_text:
            parts.append((run_style, run_text)); run_text = ""
        run_style = style
        run_text += ch.data
    parts.append((run_style, run_text))
    s = ""
    for (fg, bold), t in parts:
        w = ";font-weight:700" if bold else ""
        s += f'<span style="color:{fg}{w}">{html.escape(t)}</span>'
    out.append(s)
print("\n".join(out))
