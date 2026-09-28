package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lukeemerson/tidemark/internal/source"
	"github.com/lukeemerson/tidemark/internal/store"
)

// Spans (STORE-SPEC.md §3): z cycles the scrubber between the 400 samples in memory, the last
// hour of raw samples on disk and the last 24 h of 10 s summaries.
const (
	spanMem = iota
	spanHour
	spanDay
	nSpans
)

var spanNames = [nSpans]string{"400", "1h", "24h"}

// procSummaryTitle names the process box on the summary tier, where it holds each bucket's top 5
// processes by peak CPU (the memory layout included: top by memory isn't kept).
const procSummaryTitle = "top 5 by cpu · 10 s peak"

// tierPoint is one step of a stored tier: a sample (1h) or a bucket's averages (24h).
type tierPoint struct {
	t     time.Time
	s     source.Sample
	sys   source.Sys
	sysOK bool
	alert bool
	diag  diagState // 1h: the diagnosis at that sample, rerun from the raw lines
}

// spanLen and spanStep are a tier's window and its time per step.
func spanLen(span int) time.Duration {
	if span == spanDay {
		return store.SummarySpan
	}
	return time.Hour
}

func spanStep(span int) time.Duration {
	if span == spanDay {
		return store.BucketLen
	}
	return time.Second
}

// loadTier reads the chosen span's points from the store: the raw hour files for 1h, the
// summary for 24h. Alert ticks come from the summary's stored flags in both.
func (m *Model) loadTier() {
	now := m.now()
	sums := store.Summary(m.storeDir, now)
	m.tier = nil
	switch m.span {
	case spanHour:
		var dg diagnoser
		for _, s := range store.Raw(m.storeDir, now) {
			var d diagState
			dg, d, _ = dg.step(s, source.Sys{}, false, func(_ int, c string) string { return c })
			m.tier = append(m.tier, tierPoint{t: s.Timestamp, s: s, diag: d})
		}
		// a bucket's flag marks the first raw sample inside it
		i := 0
		for _, b := range sums {
			if !b.Alert {
				continue
			}
			for i < len(m.tier) && m.tier[i].t.Before(b.T) {
				i++
			}
			if i < len(m.tier) && m.tier[i].t.Before(b.T.Add(store.BucketLen)) {
				m.tier[i].alert = true
			}
		}
	case spanDay:
		for _, b := range sums {
			s := b.Avg
			s.Processes = nil
			for _, p := range b.Top {
				s.Processes = append(s.Processes, source.Process{PID: p.PID, Command: p.Name,
					CPUPercent: p.CPU, GPUMsPerS: p.GPU, MemPercent: p.Mem, RSSKB: p.RSS})
			}
			sys := source.Sys{Load: b.Load, Pressure: b.Pressure, FreePct: b.Free}
			m.tier = append(m.tier, tierPoint{t: b.T, s: s, sys: sys, sysOK: b.SysOK, alert: b.Alert})
		}
	}
	m.tcur = max(len(m.tier)-1, 0)
}

// atTier is at() for a stored tier: the point under the cursor, with every history rebuilt from
// the tier up to it. Times with no data become zeros, so graphs show them blank.
func (m Model) atTier() Model {
	p := m.tier[m.tcur]
	m.s, m.sys, m.have, m.noSys, m.recorded = p.s, p.sys, true, !p.sysOK, true
	m.diag = []diagState{p.diag}
	step := spanStep(m.span)
	var hs [13][]float64
	for i, q := range m.tier[:m.tcur+1] {
		if i > 0 {
			if gap := q.t.Sub(m.tier[i-1].t); gap > step*5/2 {
				for k := 0; k < min(int(gap/step)-1, histLen); k++ {
					for j := range hs {
						hs[j] = append(hs[j], 0)
					}
				}
			}
		}
		mp := 0.0
		if q.s.Memory.Total > 0 {
			mp = q.s.Memory.Used * 100 / q.s.Memory.Total
		}
		for j, v := range []float64{q.s.CPUUsage, q.s.GPUUsage, q.s.SoC.TotalPower, mp, q.s.SoC.CPUTemp,
			q.s.SoC.GPUTemp, q.sys.Load[0], q.s.Memory.SwapUsed, q.s.NetDisk.InBytes, q.s.NetDisk.OutBytes,
			q.s.NetDisk.ReadKBytes * 1024, q.s.NetDisk.WriteKB * 1024, q.s.SoC.DRAMRead + q.s.SoC.DRAMWrite} {
			hs[j] = append(hs[j], v)
		}
	}
	m.hcpu, m.hgpu, m.hpow, m.hmem, m.htc, m.htg = hs[0], hs[1], hs[2], hs[3], hs[4], hs[5]
	m.hload, m.hswap, m.hnin, m.hnout, m.hdr, m.hdw, m.hdram = hs[6], hs[7], hs[8], hs[9], hs[10], hs[11], hs[12]
	return m
}

// tierTrack draws a stored tier over its time window: · where there's no data, ▮ up to the
// cursor, ▯ after, ▲ where a stored alert flag is set (in the cursor's colour on its own cell).
func (m Model) tierTrack(cells int) string {
	if cells <= 0 || len(m.tier) == 0 {
		return ""
	}
	end := m.now()
	start := end.Add(-spanLen(m.span))
	cellOf := func(t time.Time) int {
		c := int(float64(t.Sub(start)) / float64(end.Sub(start)) * float64(cells))
		return min(max(c, 0), cells-1)
	}
	has, alert := make([]bool, cells), make([]bool, cells)
	for _, p := range m.tier {
		c := cellOf(p.t)
		has[c] = true
		alert[c] = alert[c] || p.alert
	}
	cur := cellOf(m.tier[m.tcur].t)
	var b strings.Builder
	for i := 0; i < cells; i++ {
		switch {
		case !has[i]:
			b.WriteString(dim.Render("·"))
		case alert[i] && i == cur:
			b.WriteString(mid.Render("▲"))
		case alert[i]:
			b.WriteString(high.Render("▲"))
		case i <= cur:
			b.WriteString(mid.Render("▮"))
		default:
			b.WriteString(dim.Render("▯"))
		}
	}
	return b.String()
}

// ago formats how far back the cursor is: 34s, 23m, 3h12m.
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// tierAlerts counts the stored alert flags in the tier.
func (m Model) tierAlerts() int {
	n := 0
	for _, p := range m.tier {
		if p.alert {
			n++
		}
	}
	return n
}
