package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lukeemerson/tidemark/internal/store"
)

func TestDelta(t *testing.T) {
	for _, c := range []struct {
		a, b float64
		prec int
		want string
	}{{28.4, 28.2, 0, "±0"}, {28, 57, 0, "▼−29"}, {40, 28, 0, "▲+12"}, {15.9, 23.5, 1, "▼−7.6"}, {0.04, 0, 1, "±0"}} {
		if got := delta(c.a, c.b, c.prec); got != c.want {
			t.Errorf("delta(%v, %v, %d) = %q, want %q", c.a, c.b, c.prec, got, c.want)
		}
	}
}

func TestResample(t *testing.T) {
	up := resample([]float64{1, 2}, 10*time.Second, time.Second)
	if len(up) != 20 || up[9] != 1 || up[10] != 2 {
		t.Errorf("10 s → 1 s: %v", up)
	}
	if down := resample([]float64{1, 2, 3, 4, 5}, time.Second, 2*time.Second); len(down) != 3 || down[0] != 1 || down[2] != 4.5 {
		t.Errorf("1 s → 2 s: %v, want [1 2.5 4.5]", down)
	}
}

// A wins any cell holding one of its dots; B's ghost shows, dim, only where A has none.
func TestGhostUnderA(t *testing.T) {
	if dim.Render("⣿") == cCPU.Render("⣿") {
		t.Fatal("styles don't render distinctly here")
	}
	row := graph([]float64{0, 0, 10, 10}, []float64{100, 100, 100, 100}, 2, 1, 100, cCPU)[0]
	if got := ansi.Strip(row); got != "⣿⣀" {
		t.Errorf("cells = %q, want the ghost's full cell then A's low one", got)
	}
	if !strings.Contains(row, dim.Render("⣿")) || !strings.Contains(row, cCPU.Render("⣀")) {
		t.Errorf("ghost should be dim and A in its colour: %q", row)
	}
}

func TestMarkLive(t *testing.T) {
	all := recording(t)
	m := feed(New(nil, nil, "", "", nil), all[:40])
	m = key(m, "space")
	for range 30 {
		m = key(m, "[")
	}
	m = key(key(m, "m"), "space") // B 30 samples back, A live again
	if m.mark == nil {
		t.Fatal("m set no mark")
	}
	want := "vs B t−" + ago(all[39].Timestamp.Sub(all[9].Timestamp))
	head := ansi.Strip(m.layout(42, 138)[0])
	if !strings.Contains(head, "·  "+want) || strings.Contains(head, "battery") {
		t.Errorf("header %q: want %q and no battery", head, want)
	}
	frame := ansi.Strip(strings.Join(m.layout(42, 138), "\n"))
	if !strings.Contains(frame, "±0") || !strings.ContainsAny(frame, "▲▼") {
		t.Errorf("tiles missing their differences:\n%s", frame)
	}
	m.lay = 7 // glance, narrow: the compact form
	if head := ansi.Strip(m.layout(30, 56)[0]); !strings.Contains(head, "vs −") || strings.Contains(head, "vs B") {
		t.Errorf("narrow header %q: want the compact vs", head)
	}

	m.lay = 0
	p := key(m, "space")
	for w, want := range map[int]string{138: "  ·  vs B t−", 56: " vs −"} {
		if head := ansi.Strip(p.layout(42, w)[0]); !strings.Contains(head, want) || strings.Count(head, "vs ") != 1 {
			t.Errorf("paused header at %d: %q, want one %q", w, head, want)
		}
	}

	m.mark.s.Memory.Total = 0
	if got := ansi.Strip(m.vsB(sMem)); got != "Δ —" {
		t.Errorf("B without memory: %q, want Δ —", got)
	}
	if m = key(m, "m"); m.mark != nil || strings.Contains(ansi.Strip(m.layout(42, 138)[0]), "vs B") {
		t.Error("m again should clear the mark")
	}
}

func TestMarkTrack(t *testing.T) {
	if got := ansi.Strip(track(80, 90, 30, func(int, int) bool { return false }, 10)); string([]rune(got)[3]) != "B" {
		t.Errorf("track = %q, want B in cell 3", got)
	}
	if got := ansi.Strip(track(10, 90, 30, func(int, int) bool { return false }, 10)); strings.Contains(got, "B") {
		t.Errorf("track = %q: the cursor's own cell should read as the cursor", got)
	}
}

// B marked 3 h back on the 24h tier: the ghost is widened to A's resolution once live, and the
// tier track shows B once the cursor moves off it.
func TestMarkDaySpan(t *testing.T) {
	m := tierModel(t, storeWith(t))
	m = key(key(m, "z"), "z")
	for range 6 {
		m = key(m, "[")
	}
	m = key(m, "m")
	if m.mark == nil || m.mark.step != store.BucketLen {
		t.Fatalf("mark on 24h: %+v", m.mark)
	}
	for range 6 {
		m = key(m, "]")
	}
	head := ansi.Strip(m.layout(42, 138)[0])
	if !strings.Contains(head, "B") || !strings.Contains(head, "vs B t−2h5") {
		t.Errorf("24h header %q: want B on the track and vs B t−2h5…", head)
	}
	m = key(m, "space")
	k := int(float64(store.BucketLen)/float64(m.step()) + 0.5)
	if got, want := len(m.ghost(sCPU)), len(m.mark.hs[sCPU])*k; k < 2 || got != want {
		t.Errorf("ghost at %v per column: %d values, want %d", m.step(), got, want)
	}
}

// Review finding on 2deda05: an auto-scaled graph takes B's peak when it's higher, so B at 40 W
// doesn't draw as tall as A's 10 W peak.
func TestGhostSetsTop(t *testing.T) {
	m := New(nil, nil, "", "", nil)
	m.have, m.hpow = true, []float64{5, 10}
	m.mark = &markB{step: time.Second}
	m.mark.hs[sPow] = []float64{40, 40}
	if got := ansi.Strip(m.pPow(40, 4)[1]); !strings.Contains(got, "40.0W") {
		t.Errorf("power graph's top row %q, want the 40.0W label", got)
	}
}
