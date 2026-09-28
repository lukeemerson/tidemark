package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lukeemerson/tidemark/internal/source"
)

func replayed(t *testing.T) Model {
	f, err := os.Open("../source/testdata/mactop.raw")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan source.Sample)
	go func() { source.Decode(f, ch); close(ch) }()
	m := New(nil, source.CloudyRuns(source.CloudyDir(), 40), "", "", nil)
	for s := range ch {
		mm, _ := m.Update(sampleMsg(s))
		m = mm.(Model)
	}
	return m
}

// TestSweep checks every layout at sizes from 1×1 up, before and after data, paused and replaying: lines fit the width, the frame
// fits the rows, and boxes are never cut open at the bottom.
func TestSweep(t *testing.T) {
	live := replayed(t)
	lazy := New(nil, source.CloudyRuns(source.CloudyDir(), 40), "", "", nil)
	sizes := [][2]int{{1, 1}, {5, 3}, {20, 20}, {30, 15}, {40, 24}, {60, 20}, {66, 27}, {80, 24}, {100, 30}, {140, 42}}
	paused := live
	paused.paused, paused.cursor = true, paused.seq-1-34
	all := recording(t)
	compare := key(key(paused, "m"), "space") // B 34 samples back, A live
	replay := feed(New(nil, source.CloudyRuns(source.CloudyDir(), 40), "", "", nil).Replay(len(all), all[0]), all[:34])
	for _, st := range []struct {
		name string
		m    Model
	}{{"lazy", lazy}, {"live", live}, {"paused", paused}, {"replay", replay}, {"compare", compare}} {
		for li := range layouts {
			for _, sz := range sizes {
				cols, rows := sz[0], sz[1]
				m := st.m
				m.lay = li
				out := m.layout(rows, cols-2)
				if len(out) > rows {
					t.Errorf("%s/%s %dx%d: %d lines > %d rows", st.name, layouts[li].name, cols, rows, len(out), rows)
				}
				for i, l := range out {
					if n := ansi.StringWidth(l); n > cols-2 {
						t.Errorf("%s/%s %dx%d line %d: width %d > %d: %q", st.name, layouts[li].name, cols, rows, i, n, cols-2, ansi.Strip(l))
					}
				}
				if len(out) > 0 && strings.ContainsRune(ansi.Strip(strings.Join(out, "")), '┏') &&
					!strings.ContainsRune(ansi.Strip(out[len(out)-1]), '┗') {
					t.Errorf("%s/%s %dx%d: last line isn't a bottom border: %q", st.name, layouts[li].name, cols, rows, ansi.Strip(out[len(out)-1]))
				}
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

// TestNoEmptyBoxes: on tall screens every layout hands spare height to something that can use
// it, so no box interior stays blank for more than 3 rows running.
func TestNoEmptyBoxes(t *testing.T) {
	m := replayed(t)
	for li := range layouts {
		for _, sz := range [][2]int{{140, 42}, {90, 30}, {100, 60}, {140, 70}, {200, 80}, {40, 80}} {
			cols, rows := sz[0], sz[1]
			m.lay = li
			if !layouts[li].fits(m, rows, cols-2) {
				continue
			}
			run := map[[2]int]int{}
			for y, l := range m.layout(rows, cols-2) {
				r := []rune(ansi.Strip(l))
				var vs []int
				for x, c := range r {
					if strings.ContainsRune("│┃║╎╏|▌▐", c) {
						vs = append(vs, x)
					}
				}
				seen := map[[2]int]bool{}
				for i := 0; i+1 < len(vs); i++ {
					a, b := vs[i], vs[i+1]
					if b-a <= 2 { // the gap between two neighbouring boxes
						continue
					}
					k := [2]int{a, b}
					if strings.TrimSpace(string(r[a+1:b])) != "" {
						continue
					}
					seen[k] = true
					if run[k]++; run[k] == 4 {
						t.Errorf("%s %dx%d: columns %d–%d blank for 4+ rows ending at row %d",
							layouts[li].name, cols, rows, a, b, y)
					}
				}
				for k := range run {
					if !seen[k] {
						delete(run, k)
					}
				}
			}
		}
	}
}
