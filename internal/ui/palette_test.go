package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The tidemark palette is validated per variant; make sure each background gets its own steps
// and that ansi switches back to terminal colours.
func TestPaletteVariants(t *testing.T) {
	defer setPalette("ansi", true)
	hex := func() string {
		r, g, b, _ := cCPU.GetForeground().RGBA()
		return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	setPalette("tidemark", true)
	if got := hex(); got != "#3987e5" {
		t.Errorf("dark cpu = %s, want #3987e5", got)
	}
	setPalette("tidemark", false)
	if got := hex(); got != "#2a78d6" {
		t.Errorf("light cpu = %s, want #2a78d6", got)
	}
	setPalette("ansi", false)
	if got := cCPU.GetForeground(); got != lipgloss.ANSIColor(6) {
		t.Errorf("ansi cpu = %v, want ANSI slot 6", got)
	}
}

// Review findings on f9257d4: a small upload must still show, and a graph's width must not
// depend on how wide its peak label is.
func TestSmallUploadVisible(t *testing.T) {
	rows := vbar2([]float64{500, 480}, []float64{20, 35}, 4, 500, dim, dim)
	if !strings.Contains(ansi.Strip(rows[len(rows)-1]), "▓") {
		t.Errorf("upload of 20 beside 500 vanished: bottom row %q", ansi.Strip(rows[len(rows)-1]))
	}
}

func TestAxisGraphStableWidth(t *testing.T) {
	// where the graph starts: the gutter's width on a row without a label
	edge := func(label string) int {
		row := ansi.Strip(axisGraph([]float64{1, 2, 3}, 40, 2, 3, label, dim)[1])
		return len(row) - len(strings.TrimLeft(row, " "))
	}
	for _, pair := range [][2]string{{"9.9K/s", "10K/s"}, {"9.9W", "10.0W"}, {"100%", "100K/s"}} {
		if a, b := edge(pair[0]), edge(pair[1]); a != b {
			t.Errorf("graph starts at column %d with %q but %d with %q", a, pair[0], b, pair[1])
		}
	}
}
