package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lukeemerson/tidemark/internal/source"
)

func recording(t *testing.T) []source.Sample {
	f, err := os.Open("../source/testdata/mactop.raw")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ch := make(chan source.Sample)
	go func() { source.Decode(f, ch); close(ch) }()
	var all []source.Sample
	for s := range ch {
		all = append(all, s)
	}
	return all
}

func feed(m Model, ss []source.Sample) Model {
	for _, s := range ss {
		mm, _ := m.Update(sampleMsg(s))
		m = mm.(Model)
	}
	return m
}

func key(m Model, k string) Model {
	msg := tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
	if k == "space" {
		msg = tea.KeyPressMsg{Code: tea.KeySpace}
	}
	mm, _ := m.Update(msg)
	return mm.(Model)
}

// The recording has replayd above one core for most of its 90 s, so the runaway rule fires.
func TestRunawayFiresOnRecording(t *testing.T) {
	m := feed(New(nil, nil, "", "", nil), recording(t))
	if len(m.alerts) == 0 {
		t.Fatal("no alert ticks from the recording")
	}
	d := m.diag[len(m.diag)-1]
	if d.rule != ruleRunaway || !strings.HasPrefix(d.text, "▲ runaway: replayd (900) ") || !strings.Contains(d.text, "% cpu for ") {
		t.Errorf("last diagnosis = %+v", d)
	}
	if got := ansi.Strip(m.layout(42, 138)[1]); !strings.HasPrefix(got, "▲ runaway: replayd (900)") {
		t.Errorf("line under the header = %q", got)
	}
}

func TestRulesPriorityAndClear(t *testing.T) {
	var d diagnoser
	name := func(_ int, c string) string { return c }
	t0 := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	s := func(i int, thermal string, swapGB float64, topCPU float64) source.Sample {
		x := source.Sample{Timestamp: t0.Add(time.Duration(i) * time.Second), ThermalState: thermal}
		x.Memory.SwapUsed = swapGB * 1073741824
		x.SoC.CPUTemp = 94
		x.Processes = []source.Process{{PID: 42, Command: "spin", CPUPercent: topCPU}}
		return x
	}
	warn := source.Sys{Pressure: 2}
	var st diagState
	var fired bool
	alerts := 0
	for i := 0; i < 12; i++ { // swap rises 1.2 GB over 10 s at pressure warn, a process spins
		d, st, fired = d.step(s(i, "Nominal", float64(i)*0.12, 150), warn, true, name)
		if fired {
			alerts++
		}
	}
	if st.rule != ruleSwap || st.text != "▲ swapping: +1.2 GB in 10s · pressure warn" {
		t.Errorf("swap beats runaway: got %+v", st)
	}
	d, st, _ = d.step(s(12, "Heavy", 1.44, 150), warn, true, name)
	if st.rule != ruleThrottle || st.text != "▲ throttling: thermal Heavy · cpu 94°C" {
		t.Errorf("throttle wins: got %+v", st)
	}
	for i := 13; i < 17; i++ { // 4 quiet samples: throttle still holds
		d, st, _ = d.step(s(i, "Nominal", 1.44, 150), warn, true, name)
	}
	if st.rule != ruleThrottle {
		t.Errorf("throttle cleared before 5 quiet samples: %+v", st)
	}
	d, st, _ = d.step(s(17, "Nominal", 1.44, 150), warn, true, name)
	if st.rule == ruleThrottle {
		t.Errorf("throttle still on after 5 quiet samples")
	}
	if alerts < 2 {
		t.Errorf("want alert ticks for runaway and swap, got %d", alerts)
	}
}

func TestScrubKeys(t *testing.T) {
	m := feed(New(nil, nil, "", "", nil), recording(t))
	m = key(m, "space")
	if !m.paused || m.cursor != m.seq-1 {
		t.Fatalf("space: paused=%v cursor=%d seq=%d", m.paused, m.cursor, m.seq)
	}
	m = key(key(m, "{"), "[")
	if m.cursor != m.seq-1-31 {
		t.Errorf("{ then [: cursor %d, want %d", m.cursor, m.seq-1-31)
	}
	for i := 0; i < 10; i++ {
		m = key(m, "{")
	}
	if m.cursor != m.seq-len(m.past) {
		t.Errorf("{ past the start: cursor %d, want oldest %d", m.cursor, m.seq-len(m.past))
	}
	head := ansi.Strip(m.layout(42, 138)[0])
	for _, want := range []string{"‖ paused", "◀ ", " ▶", "t−", "tiles 1/10"} {
		if !strings.Contains(head, want) {
			t.Errorf("paused header %q missing %q", head, want)
		}
	}
	m = feed(m, recording(t)[:5]) // samples keep arriving while paused
	if !m.paused || m.at().s.Timestamp.IsZero() {
		t.Errorf("paused screen lost its sample")
	}
	if m = key(m, "space"); m.paused {
		t.Errorf("space didn't resume live")
	}
}

func TestReplayHeader(t *testing.T) {
	all := recording(t)
	m := feed(New(nil, nil, "", "", nil).Replay(len(all), all[0].Timestamp), all[:34])
	m.lay = 2
	head := ansi.Strip(m.layout(42, 138)[0])
	if !strings.Contains(head, "▶ replay 34/90 · 1× · recorded 2026-09-26 19:44") || !strings.Contains(head, "instrument 3/10") {
		t.Errorf("replay header = %q", head)
	}
	if m.sysOK() {
		t.Errorf("replay shouldn't claim sysctl readings")
	}
	mm, _ := m.Update(doneMsg{})
	if m = mm.(Model); !m.ended {
		t.Errorf("end of a recording should stay on screen, not quit")
	}
}

// A tick in the cursor's own cell keeps the cursor visible: ▲ in the cursor colour, not the
// alert colour (design's finding on bb97cde: cursor on sample 55, alert at 56, same cell).
func TestTrackCursorOnTick(t *testing.T) {
	alertAt := func(a int) func(lo, hi int) bool { return func(lo, hi int) bool { return a >= lo && a < hi } }
	onCursor := track(55, 90, 30, alertAt(56))
	if !strings.Contains(onCursor, mid.Render("▲")) || strings.Contains(onCursor, high.Render("▲")) {
		t.Errorf("tick in the cursor's cell should be drawn as the cursor")
	}
	elsewhere := track(30, 90, 30, alertAt(56))
	if !strings.Contains(elsewhere, high.Render("▲")) {
		t.Errorf("tick away from the cursor should stay in the alert colour")
	}
}
