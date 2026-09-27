package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lukeemerson/mac-monitor/internal/source"
)

func replayed(t *testing.T) Model {
	f, err := os.Open("../source/testdata/mactop.raw")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan source.Sample)
	go func() { source.Decode(f, ch); close(ch) }()
	m := New(nil, source.CloudyRuns(source.CloudyDir(), 40))
	for s := range ch {
		mm, _ := m.Update(sampleMsg(s))
		m = mm.(Model)
	}
	return m
}

// TestSweep checks sizes from 1×1 up, before and after data: lines fit the width, the frame
// fits the rows, and boxes are never cut open at the bottom.
func TestSweep(t *testing.T) {
	live := replayed(t)
	lazy := New(nil, source.CloudyRuns(source.CloudyDir(), 40))
	sizes := [][2]int{{1, 1}, {5, 3}, {20, 20}, {30, 15}, {40, 24}, {60, 20}, {66, 27}, {80, 24}, {100, 30}, {140, 42}}
	for _, st := range []struct {
		name string
		m    Model
	}{{"lazy", lazy}, {"live", live}} {
		for _, sz := range sizes {
			cols, rows := sz[0], sz[1]
			out := st.m.layout(rows, cols-2)
			if len(out) > rows {
				t.Errorf("%s %dx%d: %d lines > %d rows", st.name, cols, rows, len(out), rows)
			}
			for i, l := range out {
				if n := ansi.StringWidth(l); n > cols-2 {
					t.Errorf("%s %dx%d line %d: width %d > %d: %q", st.name, cols, rows, i, n, cols-2, ansi.Strip(l))
				}
			}
			if len(out) > 0 && strings.ContainsRune(ansi.Strip(strings.Join(out, "")), '┏') &&
				!strings.ContainsRune(ansi.Strip(out[len(out)-1]), '┗') {
				t.Errorf("%s %dx%d: last line isn't a bottom border: %q", st.name, cols, rows, ansi.Strip(out[len(out)-1]))
			}
		}
	}
}

func TestCoreCounts(t *testing.T) {
	m := replayed(t)
	for _, c := range [][2]int{{4, 8}, {4, 12}} {
		m.ne, m.np = c[0], c[1]
		m.s.CoreUsages = make([]float64, c[0]+c[1])
		frame := ansi.Strip(strings.Join(m.layout(60, 138), "\n"))
		if want := fmt.Sprintf("P%d ", c[1]); !strings.Contains(frame, want) {
			t.Errorf("%dE+%dP: %q missing from cores panel", c[0], c[1], want)
		}
	}
}
