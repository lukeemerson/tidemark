package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ANSI palette indices, so the terminal's own theme (Alacritty) supplies the colours.
var (
	dim   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	title = lipgloss.NewStyle().Bold(true)
	low   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2))
	mid   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(3))
	high  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))
	// low/mid/high are for state only (meters, pressure, errors); each series has its own slot.
	// Neighbouring tiles differ even in themes where bright slots equal normal ones.
	cCPU   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))  // cpu, load
	cGPU   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(4))  // gpu
	cPower = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(5))  // power
	cMem   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(14)) // memory, swap
	cTemp  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(13)) // temperatures
	cDown  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12)) // download, disk read
	cUp    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))  // upload, disk write
	cPing  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))  // latency
)

func level(v float64) lipgloss.Style {
	switch {
	case v >= 80:
		return high
	case v >= 50:
		return mid
	}
	return low
}

func rep(s string, n int) string { return strings.Repeat(s, max(n, 0)) }

// border is a frame's line set; each layout draws with its own weight.
// Most sets repeat one horizontal and one vertical; block frames use different glyphs per side,
// and rules has no sides at all (open drops the bottom edge too).
type border struct {
	tl, tr, bl, br string
	h, hb          string // top and bottom horizontals
	vl, vr         string // left and right verticals
	open           bool
}

func lines(tl, tr, bl, br, h, v string) border { return border{tl, tr, bl, br, h, h, v, v, false} }

var (
	heavy   = lines("┏", "┓", "┗", "┛", "━", "┃")
	rounded = lines("╭", "╮", "╰", "╯", "─", "│")
	double  = lines("╔", "╗", "╚", "╝", "═", "║")
	square  = lines("┌", "┐", "└", "┘", "─", "│")
	dashed  = lines("┌", "┐", "└", "┘", "╌", "╎")
	hdashed = lines("┏", "┓", "┗", "┛", "╍", "╏")
	ascii   = lines("+", "+", "+", "+", "-", "|")
	block   = border{"▛", "▜", "▙", "▟", "▀", "▄", "▌", "▐", false}
	rules   = border{"─", "─", " ", " ", "─", " ", " ", " ", true}
)

// box draws a heavy frame; see border.box.
func box(t, rt string, w, h int, lines ...string) []string { return heavy.box(t, rt, w, h, lines...) }

// box draws a frame w wide and h tall: title and optional right label set into the top
// border, lines padded or cut to the inner width.
func (b border) box(t, rt string, w, h int, lines ...string) []string {
	iw := w - 4
	tl, tr := " "+t+" ", ""
	if rt != "" {
		tr = " " + rt + " "
	}
	// narrow boxes drop the right label first, then shorten the title
	if lipgloss.Width(tl)+lipgloss.Width(tr) > w-4 {
		tr = ""
	}
	if lipgloss.Width(tl) > w-4 {
		tl = " " + fit(t, w-6) + " "
	}
	n := w - 4 - lipgloss.Width(tl) - lipgloss.Width(tr)
	out := []string{dim.Render(b.tl+b.h) + title.Render(tl) + dim.Render(rep(b.h, n)) + tr + dim.Render(b.h+b.tr)}
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, dim.Render(b.vl)+" "+pad(l, iw)+" "+dim.Render(b.vr))
	}
	if b.open {
		return append(out, rep(" ", w))
	}
	return append(out, dim.Render(b.bl+rep(b.hb, w-2)+b.br))
}

// fit cuts s to n characters with a trailing … or pads it with spaces to n.
func fit(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		if n < 1 {
			return ""
		}
		return string(r[:n-1]) + "…"
	}
	return s + rep(" ", n-len(r))
}

// pad fits a styled string to w visible columns; too-long strings lose their colour and are cut.
func pad(s string, w int) string {
	n := lipgloss.Width(s)
	if n > w {
		return fit(ansi.Strip(s), w)
	}
	return s + rep(" ", w-n)
}

func center(s string, w int) string { return rep(" ", (w-lipgloss.Width(s))/2) + s }

// spread puts l at the left and r at the right of w columns.
func spread(l, r string, w int) string {
	return l + rep(" ", max(w-lipgloss.Width(l)-lipgloss.Width(r), 1)) + r
}

// num formats v, or before mactop's first sample a dim dash right-aligned in the same width.
func num(have bool, format string, v float64) string {
	s := fmt.Sprintf(format, v)
	if !have {
		return dim.Render(rep(" ", len([]rune(s))-1) + "—")
	}
	return s
}

func rate(have bool, b float64) string {
	if !have {
		return dim.Render("—")
	}
	u := []string{"B", "K", "M", "G"}
	i := 0
	for b >= 1024 && i < 3 {
		b /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s/s", b, u[i])
}

func gb(v float64) string { return fmt.Sprintf("%.1f", v/1073741824) }

// bar is a p% meter w wide: ■ for the filled part in col (or the level colour), dim · after.
func bar(p float64, w int, col *lipgloss.Style) string { return barG(p, w, col, "■", "·") }

func barG(p float64, w int, col *lipgloss.Style, on, off string) string {
	p = min(max(p, 0), 100)
	n := int(p/100*float64(w) + 0.5)
	st := level(p)
	if col != nil {
		st = *col
	}
	return st.Render(rep(on, n)) + dim.Render(rep(off, w-n))
}

var vb = []rune(" ▁▂▃▄▅▆▇█")

// vbar2 draws paired vertical bars, a[i] then b[i], h rows tall (newest pair at the right).
func vbar2(a, b []float64, h int, top float64, ca, cb lipgloss.Style) []string {
	if top <= 0 {
		top = 1
	}
	cell := func(v float64, r int) string {
		e := min(max(int(v/top*float64(h*8)+0.5)-(r-1)*8, 0), 8)
		return string(vb[e])
	}
	var out []string
	for r := h; r >= 1; r-- {
		var l strings.Builder
		for i := range a {
			l.WriteString(ca.Render(cell(a[i], r)) + cb.Render(cell(b[i], r)) + " ")
		}
		out = append(out, l.String())
	}
	return out
}

var spk = []rune("▁▂▃▄▅▆▇█")

// spark renders the newest w values of hist, newest at the right, blank where there is no data.
func spark(hist []float64, w int, top float64) string {
	if top <= 0 {
		top = 1
	}
	var b strings.Builder
	for i := len(hist) - w; i < len(hist); i++ {
		if i < 0 {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(spk[min(int(hist[i]/top*7+0.5), 7)])
	}
	return b.String()
}

func hmax(hist []float64, floor float64) float64 {
	m := floor
	for _, v := range hist {
		m = max(m, v)
	}
	return m
}

// braille dot bits per fill level (0–4 dots from the bottom) for the left and right columns
var fillL = []int{0, 64, 68, 70, 71}
var fillR = []int{0, 128, 160, 176, 184}

// graph draws the newest 2w values of hist as a w×h braille area chart (two samples per cell)
// in the series colour.
func graph(hist []float64, w, h int, top float64, col lipgloss.Style) []string {
	if top <= 0 {
		top = 1
	}
	w = max(w, 0) // narrow terminals give negative widths
	cells := make([]int, w*h)
	n := len(hist)
	for c := 0; c < w; c++ {
		for side := 0; side < 2; side++ {
			idx := n - 2*w + c*2 + side
			if idx < 0 {
				continue
			}
			v := hist[idx]
			dots := int(min(v/top, 1)*float64(h*4) + 0.5)
			if v > 0 && dots == 0 {
				dots = 1
			}
			for r := h - 1; r >= 0 && dots > 0; r-- {
				k := min(dots, 4)
				if side == 0 {
					cells[r*w+c] += fillL[k]
				} else {
					cells[r*w+c] += fillR[k]
				}
				dots -= k
			}
		}
	}
	out := make([]string, h)
	for r := 0; r < h; r++ {
		var b strings.Builder
		for c := 0; c < w; c++ {
			b.WriteRune(rune(0x2800 + cells[r*w+c]))
		}
		out[r] = col.Render(b.String())
	}
	return out
}

// hjoin places blocks side by side with a one-column gap, each padded to its widest line.
func hjoin(blocks ...[]string) []string {
	h := 0
	ws := make([]int, len(blocks))
	for j, b := range blocks {
		h = max(h, len(b))
		for _, l := range b {
			ws[j] = max(ws[j], lipgloss.Width(l))
		}
	}
	out := make([]string, h)
	for i := range out {
		row := make([]string, len(blocks))
		for j, b := range blocks {
			l := ""
			if i < len(b) {
				l = b[i]
			}
			row[j] = pad(l, ws[j])
		}
		out[i] = strings.Join(row, " ")
	}
	return out
}
