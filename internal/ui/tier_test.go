package ui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lukeemerson/tidemark/internal/source"
	"github.com/lukeemerson/tidemark/internal/store"
)

type dirStub struct{ dir string }

func (d dirStub) Add(source.Sample, source.Sys, bool, bool, func(int, string) string) {}
func (d dirStub) Dir() string                                                         { return d.dir }

var tierNow = time.Date(2026, 9, 27, 15, 0, 0, 0, time.Local)

// storeWith writes a temp store: 60 s of summary 3 h ago and 60 s ending 9 min ago (a gap in
// between, one alert in the recent part), and 30 s of raw samples 30 min ago.
func storeWith(t *testing.T) string {
	dir := t.TempDir()
	clock := tierNow.Add(-31 * time.Minute)
	st, err := store.Open(dir, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	all := recording(t)
	name := func(_ int, c string) string { return c }
	for i := 0; i < 60; i++ {
		s := all[i%len(all)]
		s.Timestamp = tierNow.Add(-3*time.Hour + time.Duration(i)*time.Second)
		st.Add(s, source.Sys{Load: [3]float64{2, 2, 2}, Pressure: 1, FreePct: 50}, true, false, name)
	}
	for i := 0; i < 60; i++ {
		s := all[i%len(all)]
		s.Timestamp = tierNow.Add(-10*time.Minute + time.Duration(i)*time.Second)
		st.Add(s, source.Sys{Load: [3]float64{2, 2, 2}, Pressure: 1, FreePct: 50}, true, i == 20, name)
	}
	for i := 0; i < 30; i++ {
		s := all[i%len(all)]
		s.Timestamp = tierNow.Add(-30*time.Minute + time.Duration(i)*time.Second)
		line, _ := json.Marshal(s)
		st.Write(append(append([]byte(","), line...), '\n'))
	}
	st.Close()
	return dir
}

func tierModel(t *testing.T, dir string) Model {
	m := New(nil, nil, "", "", nil).Store(dirStub{dir})
	m.now = func() time.Time { return tierNow }
	return feed(m, recording(t)[:5])
}

func TestDaySpan(t *testing.T) {
	m := tierModel(t, storeWith(t))
	m = key(key(m, "z"), "z")
	if m.span != spanDay || len(m.tier) != 12 {
		t.Fatalf("after z z: span=%d points=%d, want 24h and 12 buckets", m.span, len(m.tier))
	}
	head := ansi.Strip(m.layout(42, 138)[0])
	for _, want := range []string{"‖ paused", "t−9m  ·  24h", "1 alert", "·", "▲"} {
		if !strings.Contains(head, want) {
			t.Errorf("24h header %q missing %q", head, want)
		}
	}
	for i := 0; i < 6; i++ {
		m = key(m, "[")
	}
	if head := ansi.Strip(m.layout(42, 138)[0]); !strings.Contains(head, "t−2h59m  ·  24h") {
		t.Errorf("stepping should cross the gap to 3 h ago: %q", head)
	}
	frame := ansi.Strip(strings.Join(m.layout(42, 138), "\n"))
	if !strings.Contains(frame, procSummaryTitle) {
		t.Errorf("summary tier's process box isn't titled %q", procSummaryTitle)
	}
	m.lay = 5 // memory: top by memory isn't stored, so it's the top 5 by cpu too
	if frame := ansi.Strip(strings.Join(m.layout(42, 138), "\n")); !strings.Contains(frame, procSummaryTitle) {
		t.Errorf("memory layout on the summary tier should show %q", procSummaryTitle)
	}
	if small := ansi.Strip(m.layout(20, 58)[0]); !strings.Contains(small, "24h t−2h59m") {
		t.Errorf("compact 24h header = %q", small)
	}
	for li := range layouts {
		for _, sz := range [][2]int{{140, 42}, {60, 20}, {20, 20}} {
			m.lay = li
			out := m.layout(sz[1], sz[0]-2)
			if len(out) > sz[1] {
				t.Errorf("%s %dx%d: %d rows", layouts[li].name, sz[0], sz[1], len(out))
			}
			for _, l := range out {
				if ansi.StringWidth(l) > sz[0]-2 {
					t.Errorf("%s %dx%d: line too wide: %q", layouts[li].name, sz[0], sz[1], ansi.Strip(l))
				}
			}
		}
	}
	if m = key(m, "space"); m.paused || m.span != spanMem {
		t.Errorf("space should return to live and the 400 span")
	}
}

func TestHourSpan(t *testing.T) {
	m := key(tierModel(t, storeWith(t)), "z")
	if m.span != spanHour || len(m.tier) != 30 {
		t.Fatalf("after z: span=%d points=%d, want 1h and 30 raw samples", m.span, len(m.tier))
	}
	if head := ansi.Strip(m.layout(42, 138)[0]); !strings.Contains(head, "t−29m  ·  1h") {
		t.Errorf("1h header = %q", head)
	}
	if !m.at().recorded || m.at().sysOK() {
		t.Errorf("the raw tier is recorded data without sysctl readings")
	}
	if m = key(m, "z"); m.span != spanDay {
		t.Errorf("z from 1h should go to 24h")
	}
	if m = key(m, "z"); m.span != spanMem || m.tier != nil {
		t.Errorf("z from 24h should come back to 400")
	}
}

// Samples taken every 3 s (-i 3000) are the tier's normal cadence, not gaps (review on 6b71c35).
func TestTierGapsFollowTheCadence(t *testing.T) {
	m := New(nil, nil, "", "", nil)
	m.span = spanHour
	for i := 0; i < 6; i++ {
		s := source.Sample{Timestamp: tierNow.Add(time.Duration(i*3) * time.Second), CPUUsage: 50}
		m.tier = append(m.tier, tierPoint{t: s.Timestamp, s: s})
	}
	m.tcur, m.tierStep = 5, medianGap(m.tier, spanStep(spanHour))
	if got := m.atTier().hcpu; len(got) != 6 {
		t.Errorf("3 s cadence read as gaps: hcpu = %v", got)
	}
	m.tier[5].t = m.tier[4].t.Add(time.Minute) // a real gap: 20 missed steps
	m.tierStep = medianGap(m.tier, spanStep(spanHour))
	if got := m.atTier().hcpu; len(got) <= 6 {
		t.Errorf("a minute's gap should be drawn blank: hcpu = %v", got)
	}
}

// z passes over spans with nothing stored yet.
func TestZSkipsEmptySpans(t *testing.T) {
	m := tierModel(t, t.TempDir()) // a store with nothing in it
	if m = key(m, "z"); m.span != spanMem || m.tier != nil {
		t.Errorf("z with an empty store: span=%d tier=%d points, want to stay on 400", m.span, len(m.tier))
	}
}
