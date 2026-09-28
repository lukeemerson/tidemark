package ui

import (
	"fmt"
	"math"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/lukeemerson/tidemark/internal/source"
)

// Diagnosis (SPEC.md §3): one line under the header while a rule fires. Rules in priority order.
const (
	ruleNone = iota
	ruleThrottle
	ruleSwap
	ruleRunaway
	nRules
)

const (
	clearAfter  = 5                 // samples below threshold before a rule clears
	swapWindow  = 10                // samples the swap rise is measured over
	swapRise    = 256 * 1024 * 1024 // bytes of swap growth that counts as a storm
	runawayCPU  = 100               // one full core, in mactop's per-core CPUPercent
	runawayRuns = 10                // samples the same top process must stay at or above it
)

// diagState is what the line shows at one sample: the winning rule, its text, and a shorter
// form for narrow screens.
type diagState struct {
	rule        int
	text, short string
}

func (d diagState) style() lipgloss.Style {
	if d.rule == ruleThrottle {
		return high
	}
	return mid
}

// diagnoser carries each rule's state from sample to sample.
type diagnoser struct {
	on     [nRules]bool
	quiet  [nRules]int
	shown  [nRules]diagState
	swaps  []float64
	stamps []time.Time
	runPID int
	runN   int
	runT0  time.Time
}

// step folds one sample into the rules. It returns the updated state, what the line shows now,
// and whether any rule started firing on this sample (an alert tick).
func (d diagnoser) step(s source.Sample, sys source.Sys, sysOK bool, name func(int, string) string) (diagnoser, diagState, bool) {
	var match [nRules]bool
	var now [nRules]diagState

	if st := s.ThermalState; st != "" && st != "Nominal" {
		match[ruleThrottle] = true
		t := fmt.Sprintf("▲ throttling: thermal %s · cpu %.0f°C", st, s.SoC.CPUTemp)
		now[ruleThrottle] = diagState{ruleThrottle, t, t}
	}

	d.swaps = append(append([]float64(nil), d.swaps...), s.Memory.SwapUsed)
	d.stamps = append(append([]time.Time(nil), d.stamps...), s.Timestamp)
	if len(d.swaps) > swapWindow+1 {
		d.swaps, d.stamps = d.swaps[1:], d.stamps[1:]
	}
	if rise := d.swaps[len(d.swaps)-1] - d.swaps[0]; sysOK && sys.Pressure >= 2 && len(d.swaps) == swapWindow+1 && rise > swapRise {
		match[ruleSwap] = true
		p := "warn"
		if sys.Pressure >= 4 {
			p = "critical"
		}
		secs := seconds(d.stamps[0], d.stamps[len(d.stamps)-1], swapWindow)
		t := fmt.Sprintf("▲ swapping: +%.1f GB in %ds · pressure %s", rise/1073741824, secs, p)
		now[ruleSwap] = diagState{ruleSwap, t, t}
	}

	if len(s.Processes) > 0 && s.Processes[0].CPUPercent >= runawayCPU {
		top := s.Processes[0]
		if top.PID == d.runPID {
			d.runN++
		} else {
			d.runPID, d.runN, d.runT0 = top.PID, 1, s.Timestamp
		}
		if d.runN >= runawayRuns {
			match[ruleRunaway] = true
			short := fmt.Sprintf("▲ runaway: %s (%d) %.0f%% cpu", name(top.PID, top.Command), top.PID, top.CPUPercent)
			t := fmt.Sprintf("%s for %ds", short, seconds(d.runT0, s.Timestamp, d.runN-1)+1)
			now[ruleRunaway] = diagState{ruleRunaway, t, short}
		}
	} else {
		d.runPID, d.runN = 0, 0
	}

	fired := false
	shown := diagState{}
	for r := ruleThrottle; r < nRules; r++ {
		switch {
		case match[r]:
			if !d.on[r] {
				fired = true
			}
			d.on[r], d.quiet[r], d.shown[r] = true, 0, now[r]
		case d.on[r]:
			if d.quiet[r]++; d.quiet[r] >= clearAfter {
				d.on[r] = false
			}
		}
		if d.on[r] && shown.rule == ruleNone {
			shown = d.shown[r]
		}
	}
	return d, shown, fired
}

// seconds between two sample times; without timestamps, assume 1 s per sample.
func seconds(from, to time.Time, samples int) int {
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return samples
	}
	return int(math.Round(to.Sub(from).Seconds()))
}
