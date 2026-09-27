package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// ANSI palette indices, so the terminal's own theme (Alacritty) supplies the colours.
var (
	dim   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	title = lipgloss.NewStyle().Bold(true)
	low   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2))
	mid   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(3))
	high  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))
	gpu   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(4))
	power = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))
	net   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(5))
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

// box draws a heavy frame w wide and h tall with the title set into the top border.
// Lines are padded or cut to the inner width.
func box(t string, w, h int, lines ...string) []string {
	iw := w - 4
	tl := " " + t + " "
	n := max(w-4-lipgloss.Width(tl), 0)
	out := []string{dim.Render("┏━") + title.Render(tl) + dim.Render(strings.Repeat("━", n)+"━┓")}
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, dim.Render("┃")+" "+pad(l, iw)+" "+dim.Render("┃"))
	}
	return append(out, dim.Render("┗"+strings.Repeat("━", w-2)+"┛"))
}

// pad fits a styled string to exactly w visible columns.
func pad(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func center(s string, w int) string {
	return strings.Repeat(" ", max((w-lipgloss.Width(s))/2, 0)) + s
}

// spread puts l at the left and r at the right of w columns.
func spread(l, r string, w int) string {
	return l + strings.Repeat(" ", max(w-lipgloss.Width(l)-lipgloss.Width(r), 1)) + r
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

// hjoin places blocks side by side with a one-column gap.
func hjoin(blocks ...[]string) []string {
	var out []string
	for i := range blocks[0] {
		row := make([]string, len(blocks))
		for j, b := range blocks {
			row[j] = b[i]
		}
		out = append(out, strings.Join(row, " "))
	}
	return out
}
