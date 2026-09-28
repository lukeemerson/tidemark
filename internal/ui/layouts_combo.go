package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// Combination layouts: each mixes areas for one way of watching the machine.

// ---------------------------------------------------------------------------
// glance (rules: titled lines, no sides): for a small pane. Every current value on one line
// each, sensors and io, then the busiest processes.

func (m Model) glanceLayout(rows, w int) []string {
	out := []string{m.head(w)}
	out = append(out, rules.box("now", "", w, 10, statLines(m.statList(), w-4)...)...)
	if rows-len(out)-8 >= minProc {
		out = append(out, rules.box("machine", "", w, 8, append(m.pSens(w-4), m.pIO(w-4)...)...)...)
	}
	ph := rows - len(out)
	if extra := ph - procMax; extra >= 4 {
		out = append(out, rules.box("cpu", m.hCPU(), w, extra, axisGraph(m.hcpu, m.ghost(sCPU), w-4, extra-2, 100, "100%", cCPU)...)...)
		ph = procMax
	}
	return append(out, rules.box(m.procTitle("processes"), "", w, ph, m.pProc(w-4, ph-3)...)...)
}

// ---------------------------------------------------------------------------
// wall (block frames): every history as a full-width graph, numbers in the titles, no tables.
// For watching trends from across the room.

func (m Model) wallLayout(rows, w int) []string {
	out := []string{m.head(w)}
	hw := (w - 1) / 2
	rw := w - hw - 1
	nd, soc := m.s.NetDisk, m.s.SoC
	h := (rows - 1) / 6
	first := rows - 1 - 5*h // the cpu graph takes the remainder
	g := func(t, rt string, hist, ghost []float64, bw, bh int, top float64, label string, col lipgloss.Style) []string {
		return block.box(t, rt, bw, bh, axisGraph(hist, ghost, bw-4, bh-2, top, label, col)...)
	}
	freq := dim.Render(fmt.Sprintf("E %s · P %s MHz", num(m.have, "%.0f", soc.EFreqMHz), num(m.have, "%.0f", soc.PFreqMHz)))
	out = append(out, g("cpu", freq+"  "+m.hCPU(), m.hcpu, m.ghost(sCPU), w, first, 100, "100%", cCPU)...)
	out = append(out, g("gpu", m.hGPU(), m.hgpu, m.ghost(sGPU), w, h, 100, "100%", cGPU)...)
	out = append(out, g("power", m.hPow(), m.hpow, m.ghost(sPow), w, h, hmax(m.hpow, 1), fmt.Sprintf("%.1fW", hmax(m.hpow, 1)), cPower)...)
	out = append(out, g("memory", m.hMem(), m.hmem, m.ghost(sMem), w, h, 100, "100%", cMem)...)
	out = append(out, hjoin(
		g("net ↓", rate(m.have, nd.InBytes), m.hnin, m.ghost(sNin), hw, h, hmax(m.hnin, 1), shortRate(hmax(m.hnin, 1)), cDown),
		g("net ↑", rate(m.have, nd.OutBytes), m.hnout, m.ghost(sNout), rw, h, hmax(m.hnout, 1), shortRate(hmax(m.hnout, 1)), cUp),
	)...)
	return append(out, hjoin(
		g("disk read", rate(m.have, nd.ReadKBytes*1024), m.hdr, m.ghost(sDR), hw, h, hmax(m.hdr, 1), shortRate(hmax(m.hdr, 1)), cDown),
		g("disk write", rate(m.have, nd.WriteKB*1024), m.hdw, m.ghost(sDW), rw, h, hmax(m.hdw, 1), shortRate(hmax(m.hdw, 1)), cUp),
	)...)
}

// ---------------------------------------------------------------------------
// thermal (ascii frames): power and heat. Power history beside a readout of everything that
// draws or sheds energy, both temperatures over time, then the processes doing the work.

func (m Model) thermalLayout(rows, w int) []string {
	lw := w * 2 / 3
	rw := w - lw - 1
	hw := (w - 1) / 2
	soc := m.s.SoC
	out := []string{m.head(w)}

	ph, th, tg := rows-1-10-8, 10, 8
	if extra := ph - procMax; extra > 0 { // process list is full; the temperature graphs take the rest
		tg, ph = tg+extra, procMax
	}
	peak := hmax(m.hpow, 1)
	thermal, fan := m.s.ThermalState, 0.0
	if thermal == "" {
		thermal = "—"
	}
	if len(m.s.Fans) > 0 {
		fan = m.s.Fans[0].RPM
	}
	batt := m.batt()
	if batt == "" {
		batt = "none"
	}
	readout := []string{
		kv("total", m.hPow()),
		kv("system", num(m.have, "%.1f W", soc.SystemPower)),
		kv("gpu", num(m.have, "%.1f W", soc.GPUPower)),
		kv("peak", fmt.Sprintf("%.1f W", peak)),
		"",
		kv("thermal", thermal),
		kv("fan", num(m.have, "%.0f rpm", fan)),
		kv("battery", batt),
	}
	out = append(out, hjoin(
		ascii.box("power", m.hPow(), lw, th, axisGraph(m.hpow, m.ghost(sPow), lw-4, th-2, peak, fmt.Sprintf("%.1fW", peak), cPower)...),
		ascii.box("energy", "", rw, th, readout...),
	)...)
	cpuT := withB(num(m.have, "%.0f°", soc.CPUTemp), m.vsB(sTC))
	gpuT := num(m.have, "%.0f°", soc.GPUTemp)
	out = append(out, hjoin(
		ascii.box("cpu temp", cpuT, hw, tg, axisGraph(m.htc, m.ghost(sTC), hw-4, tg-2, 110, "110°", cTemp)...),
		ascii.box("gpu temp", gpuT, w-hw-1, tg, axisGraph(m.htg, m.ghost(sTG), w-hw-5, tg-2, 110, "110°", cTemp)...),
	)...)
	return append(out, ascii.box(m.procTitle("processes"), "", w, ph, m.pProc(w-4, ph-3)...)...)
}
