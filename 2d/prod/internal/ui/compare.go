package ui

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/lukeemerson/tidemark/internal/source"
)

// Compare (COMPARE-SPEC.md): m marks the moment on screen as B. Graphs draw B's window as a dim
// ghost behind A (what's on screen), values gain their difference from B, and the header and
// track say where B is.

// The history series, in atTier's order.
const (
	sCPU = iota
	sGPU
	sPow
	sMem
	sTC
	sTG
	sLoad
	sSwap
	sNin
	sNout
	sDR
	sDW
	sDRAM
	nSeries
)

// markB is the marked moment: its sample, and every history up to it.
type markB struct {
	s    source.Sample
	hs   [nSeries][]float64
	step time.Duration // time per history value when marked
}

func (m Model) series() [nSeries][]float64 {
	return [nSeries][]float64{m.hcpu, m.hgpu, m.hpow, m.hmem, m.htc, m.htg, m.hload, m.hswap,
		m.hnin, m.hnout, m.hdr, m.hdw, m.hdram}
}

// toggleMark marks the sample on screen as B, or clears the mark.
func (m Model) toggleMark() Model {
	if m.mark != nil || !m.have {
		m.mark = nil
		return m
	}
	v := m.at()
	b := &markB{s: v.s, step: m.step()}
	for i, h := range v.series() {
		b.hs[i] = slices.Clone(h)
	}
	m.mark = b
	return m
}

// step is the time per history value on screen: the stored tier's, or the samples' interval.
func (m Model) step() time.Duration {
	if m.paused && m.span != spanMem && len(m.tier) > 0 {
		return m.tierStep
	}
	if n := len(m.past); n >= 2 {
		if d := m.past[n-1].Timestamp.Sub(m.past[0].Timestamp) / time.Duration(n-1); d > 0 {
			return d
		}
	}
	return time.Second
}

// ghost is B's history for series i at A's resolution, nil with no mark.
func (m Model) ghost(i int) []float64 {
	if m.mark == nil {
		return nil
	}
	return resample(m.mark.hs[i], m.mark.step, m.step())
}

// resample turns h, one value per from, into one value per to, ending at the newest: a coarser
// h repeats each value over the columns its time spans (a 10 s bucket fills ten 1 s columns), a
// finer one is averaged.
func resample(h []float64, from, to time.Duration) []float64 {
	r := float64(from) / float64(to)
	switch {
	case r >= 1.5:
		k := int(r + 0.5)
		out := make([]float64, 0, len(h)*k)
		for _, v := range h {
			for range k {
				out = append(out, v)
			}
		}
		return out
	case r > 1/1.5:
		return h
	}
	k := int(1/r + 0.5)
	var out []float64
	for end := len(h); end > 0; end -= k {
		lo, sum := max(end-k, 0), 0.0
		for _, v := range h[lo:end] {
			sum += v
		}
		out = append(out, sum/float64(end-lo))
	}
	slices.Reverse(out)
	return out
}

// value is series i (cpu, gpu, power, mem or cpu temp) of s, and whether s has it.
func value(s source.Sample, i int) (float64, bool) {
	switch i {
	case sCPU:
		return s.CPUUsage, true
	case sGPU:
		return s.GPUUsage, true
	case sPow:
		return s.SoC.TotalPower, true
	case sMem:
		return memPctOf(s), s.Memory.Total > 0
	}
	return s.SoC.CPUTemp, true
}

// how each value is shown, and its decimals
var valFormat = map[int]string{sCPU: "%3.0f%%", sGPU: "%3.0f%%", sPow: "%.1f W", sMem: "%3.0f%%", sTC: "%.0f°"}
var valPrec = map[int]int{sPow: 1}

// delta is a−b at prec decimals, compared as shown: ▲+12, ▼−3.1, ±0.
func delta(a, b float64, prec int) string {
	p := math.Pow(10, float64(prec))
	d := (math.Round(a*p) - math.Round(b*p)) / p
	switch {
	case d > 0:
		return fmt.Sprintf("▲+%.*f", prec, d)
	case d < 0:
		return fmt.Sprintf("▼−%.*f", prec, -d)
	}
	return "±0"
}

// vsB is series i's dim difference from B, "Δ —" when B lacks it, "" with no mark.
func (m Model) vsB(i int) string {
	if m.mark == nil || !m.have {
		return ""
	}
	a, _ := value(m.s, i)
	b, ok := value(m.mark.s, i)
	if !ok {
		return dim.Render("Δ —")
	}
	return dim.Render(delta(a, b, valPrec[i]))
}

// withB joins a value and its difference.
func withB(v, d string) string {
	if d == "" {
		return v
	}
	return v + " " + d
}

// vsText is how far B is from A: "vs B t−3h12m", or compact "vs −3h12m" (t+ when B is later).
func (m Model) vsText(compact bool) string {
	d, sign := m.s.Timestamp.Sub(m.mark.s.Timestamp), "−"
	if d < 0 {
		d, sign = -d, "+"
	}
	if compact {
		return "vs " + sign + ago(d)
	}
	return "vs B t" + sign + ago(d)
}
