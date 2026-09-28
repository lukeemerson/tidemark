package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/lukeemerson/tidemark/internal/source"
)

// Scrub (SPEC.md §2): pause live data and step back through the last histLen samples, or watch
// a -play recording. Only the header changes; every layout draws the sample under the cursor.

// replayInfo describes a -play recording.
type replayInfo struct {
	total    int
	recorded time.Time
}

// Replay marks the model as playing a recording of total samples, starting with first. The chip,
// core counts and memory total come from the recording, not this Mac. Load and memory pressure
// come from sysctl, not mactop, so they aren't in the recording and draw as —.
func (m Model) Replay(total int, first source.Sample) Model {
	m.replay = &replayInfo{total, first.Timestamp}
	m.noSys = true
	// each field only when the recording has it, so an older recording keeps this Mac's value
	// rather than, say, zero cores
	si := first.SystemInfo
	if si.Name != "" {
		m.name = si.Name
	}
	if si.ECoreCount > 0 {
		m.ne = si.ECoreCount
	}
	if si.PCoreCount > 0 {
		m.np = si.PCoreCount
	}
	if first.Memory.Total > 0 {
		m.memTotal = first.Memory.Total
	}
	return m
}

// scrubbing is true when the header shows the scrubber instead of the clock.
func (m Model) scrubbing() bool { return m.paused || m.replay != nil }

// shown is the seq index of the sample on screen.
func (m Model) shown() int {
	if m.paused {
		return m.cursor
	}
	return m.seq - 1
}

// moveCursor steps the paused cursor, staying inside the samples still held.
func (m *Model) moveCursor(by int) {
	m.cursor = min(max(m.cursor+by, m.seq-len(m.past)), m.seq-1)
}

// at is the model as it was at the cursor: that sample, and every history cut off there.
func (m Model) at() Model {
	if m.paused && m.span != spanMem && len(m.tier) > 0 {
		return m.atTier()
	}
	if !m.paused || len(m.past) == 0 {
		return m
	}
	k := min(max(m.seq-1-m.cursor, 0), len(m.past)-1) // samples back from the newest
	cut := func(h []float64) []float64 { return h[:max(len(h)-k, 0)] }
	n := len(m.past)
	m.s, m.sys = m.past[n-1-k], m.psys[n-1-k]
	m.diag = m.diag[:n-k]
	m.hcpu, m.hgpu, m.hpow, m.hmem = cut(m.hcpu), cut(m.hgpu), cut(m.hpow), cut(m.hmem)
	m.htc, m.htg, m.hload, m.hswap = cut(m.htc), cut(m.htg), cut(m.hload), cut(m.hswap)
	m.hnin, m.hnout, m.hdr, m.hdw, m.hdram = cut(m.hnin), cut(m.hnout), cut(m.hdr), cut(m.hdw), cut(m.hdram)
	return m
}

// track draws cells for n samples with the cursor at pos: ▮ up to the cursor, ▯ after, and ▲
// wherever an alert fired (alert holds positions on the same 0..n-1 scale). The cursor's own cell
// always reads as the cursor: a ▲ there is drawn in the cursor's colour, since the cell spans
// several samples and the one under the cursor may come before the alert.
func track(pos, n, cells int, alert func(lo, hi int) bool) string {
	if n <= 0 || cells <= 0 {
		return ""
	}
	cur := pos * cells / n
	var b strings.Builder
	for i := 0; i < cells; i++ {
		lo, hi := i*n/cells, max((i+1)*n/cells, i*n/cells+1)
		switch {
		case alert(lo, hi) && i == cur:
			b.WriteString(mid.Render("▲"))
		case alert(lo, hi):
			b.WriteString(high.Render("▲"))
		case i <= cur:
			b.WriteString(mid.Render("▮"))
		default:
			b.WriteString(dim.Render("▯"))
		}
	}
	return b.String()
}

// scrubHead is the header while paused or replaying:
//
//	Apple M2 Pro        ‖ paused   ◀ ▮▮▮▮▲▮▯▯▯ ▶  t−34s   [ ] step  { } ±30  space live   tiles 1/10
//	Apple M2 Pro        ▶ replay 34/90 · 1× · recorded 2026-09-26 19:44   ◀ ▮▮▮▯▯▯ ▶   instrument 3/10
//
// Pieces drop as the width shrinks (key hints, then the layout indicator), then it folds to the
// compact form: ‖ ◀ ▮▮▯▯ ▶ t−34s.
func (m Model) scrubHead(w int) string {
	name := title.Render(m.name)
	idx := m.shown()

	// the track spans the whole recording on replay, and the held samples live
	oldest, n := m.seq-len(m.past), len(m.past)
	if m.replay != nil {
		oldest, n = 0, max(m.replay.total, m.seq)
	}
	pos := idx - oldest
	nAlerts := 0
	for _, a := range m.alerts {
		if a >= oldest && a < oldest+n {
			nAlerts++
		}
	}
	alertIn := func(lo, hi int) bool {
		for _, a := range m.alerts {
			if a-oldest >= lo && a-oldest < hi {
				return true
			}
		}
		return false
	}

	drawTrack := func(cells int) string { return track(pos, n, cells, alertIn) }
	var status, when, hints string
	if m.span != spanMem && len(m.tier) > 0 { // a stored tier: time-based track, span label
		drawTrack = m.tierTrack
		nAlerts = m.tierAlerts()
	}
	if m.replay != nil {
		sym := cPower.Render("▶ replay")
		if m.paused {
			sym = mid.Render("‖ paused")
		}
		status = sym + dim.Render(fmt.Sprintf(" %d/%d · 1× · recorded %s", idx+1, m.replay.total, m.replay.recorded.Format("2006-01-02 15:04")))
	} else {
		status = mid.Render("‖ paused")
		back := m.seq - 1 - idx
		if len(m.past) > 0 {
			back = seconds(m.past[pos].Timestamp, m.past[len(m.past)-1].Timestamp, back)
		}
		when = dim.Render(fmt.Sprintf("t−%ds", back))
		if m.span != spanMem && len(m.tier) > 0 {
			when = dim.Render("t−" + ago(m.now().Sub(m.tier[m.tcur].t)) + "  ·  " + spanNames[m.span])
		}
		hints = dim.Render("   [ ] step  { } ±30  space live")
	}
	if nAlerts > 0 {
		hints = dim.Render(fmt.Sprintf("  ·  %d alert%s", nAlerts, map[bool]string{true: "", false: "s"}[nAlerts == 1]))
	}
	ind := dim.Render("   " + m.indicator())

	full := func(cells int, withHints, withInd bool) string {
		r := status + "   " + dim.Render("◀ ") + drawTrack(cells) + dim.Render(" ▶")
		if when != "" {
			r += "  " + when
		}
		if withHints {
			r += hints
		}
		if withInd {
			r += ind
		}
		return r
	}
	fits := func(r string) bool { return lipgloss.Width(name)+1+lipgloss.Width(r) <= w }
	for _, v := range []struct{ hints, ind bool }{{true, true}, {false, true}, {false, false}} {
		if r := full(30, v.hints, v.ind); fits(r) {
			return spread(name, r, w)
		}
	}

	// compact: ‖ ◀ track ▶ t−34s, the track taking whatever room is left (at least 4 cells)
	sym := mid.Render("‖")
	tail := ""
	if m.replay != nil {
		if !m.paused {
			sym = cPower.Render("▶")
		}
		tail = dim.Render(fmt.Sprintf(" %d/%d", idx+1, m.replay.total))
	} else {
		tail = " " + when
		if m.span != spanMem && len(m.tier) > 0 {
			tail = dim.Render(" " + spanNames[m.span] + " t−" + ago(m.now().Sub(m.tier[m.tcur].t)))
		}
	}
	fixed := lipgloss.Width(name) + 1 + lipgloss.Width(sym) + lipgloss.Width(" ◀ ") + lipgloss.Width(" ▶") + lipgloss.Width(tail)
	cells := min(max(w-fixed, 4), 30)
	r := sym + dim.Render(" ◀ ") + drawTrack(cells) + dim.Render(" ▶") + tail
	return pad(spread(name, r, w), w)
}
