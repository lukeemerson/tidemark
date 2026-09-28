package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// Focus layouts: each gives the whole screen to one area of the machine.

// peakGraph is a graph whose first line names its scale: "peak <label>".
func peakGraph(hist []float64, w, h int, top float64, label string, col lipgloss.Style) []string {
	return append([]string{dim.Render(" peak " + label)}, graph(hist, w, h-1, top, col)...)
}

// kv is a readout line: a dim label in a fixed column, then the value.
func kv(label, val string) string { return dim.Render(" "+fit(label, 10)) + val }

// ---------------------------------------------------------------------------
// compute (square light): cpu and gpu side by side, all the way down to who is using them.

func (m Model) computeLayout(rows, w int) []string {
	hw := (w - 1) / 2
	rw := w - hw - 1
	soc := m.s.SoC
	out := []string{m.head(w)}

	cpuT := dim.Render(fmt.Sprintf("E %s · P %s MHz", num(m.have, "%.0f", soc.EFreqMHz), num(m.have, "%.0f", soc.PFreqMHz))) + "  " + m.hCPU()
	gpuT := dim.Render(fmt.Sprintf("%s MHz · ANE %s", num(m.have, "%.0f", soc.GPUFreqMHz), num(m.have, "%.0f%%", soc.ANEActive))) + "  " + m.hGPU()
	ch := max((m.ne+m.np+1)/2+2, 7)
	gh := 10
	if extra := rows - 1 - gh - ch - procMax; extra > 0 { // process lists are full; graphs take the rest
		gh += extra
	}
	out = append(out, hjoin(
		square.box("cpu", cpuT, hw, gh, graph(m.hcpu, hw-4, gh-2, 100, cCPU)...),
		square.box("gpu", gpuT, rw, gh, graph(m.hgpu, rw-4, gh-2, 100, cGPU)...),
	)...)

	out = append(out, hjoin(
		square.box("cores", "", hw, ch, m.pCores(hw-4, 2)...),
		square.box("power", m.hPow(), rw, ch, m.pPow(rw-4, ch-3)...),
	)...)

	ph := rows - len(out)
	return append(out, hjoin(
		square.box("top cpu", "", hw, ph, m.pProc(hw-4, ph-3)...),
		square.box("top gpu", "", rw, ph, m.pProcBy(rw-4, ph-3, "gpu")...),
	)...)
}

// ---------------------------------------------------------------------------
// memory (dashed light): usage, pressure, swap and load, then who holds the memory.

func (m Model) pressure() string {
	if !m.have {
		return dim.Render("—")
	}
	switch m.sys.Pressure {
	case 2:
		return mid.Render("warn")
	case 4:
		return high.Render("critical")
	}
	return low.Render("normal")
}

func (m Model) memoryLayout(rows, w int) []string {
	lw := w * 2 / 3
	rw := w - lw - 1
	mem, soc := m.s.Memory, m.s.SoC
	out := []string{m.head(w)}

	memL := append([]string{m.pMem(lw - 4)[0]}, graph(m.hmem, lw-4, 7, 100, cMem)...)
	free := float64(m.sys.FreePct)
	usedCol := level(100 - free)
	load := dim.Render("—")
	if m.have {
		load = fmt.Sprintf("%.2f  %.2f  %.2f", m.sys.Load[0], m.sys.Load[1], m.sys.Load[2])
	}
	readout := []string{
		kv("pressure", m.pressure()),
		kv("available", num(m.have, "%.0f%%", free)),
		" " + bar(free, rw-6, &usedCol),
		kv("used", num(m.have, "%.1f", mem.Used/1073741824)+dim.Render(" / "+gb(m.memTotal)+" GB")),
		kv("swap", num(m.have, "%.2f", mem.SwapUsed/1073741824)+dim.Render(" / "+gb(mem.SwapTotal)+" GB")),
		kv("dram", "↓ "+num(m.have, "%.1f", soc.DRAMRead)+"  ↑ "+num(m.have, "%.1f", soc.DRAMWrite)+dim.Render(" GB/s")),
		kv("load", load+dim.Render(fmt.Sprintf("  /%d cores", m.ne+m.np))),
	}
	out = append(out, hjoin(
		dashed.box("memory", m.hMem(), lw, 10, memL...),
		dashed.box("pressure", "", rw, 10, readout...),
	)...)

	hw := (w - 1) / 2
	sh := 7
	if extra := rows - len(out) - sh - procMax; extra > 0 { // process list is full; graphs take the rest
		sh += extra
	}
	swapTop := max(hmax(m.hswap, 1), mem.SwapTotal)
	loadTop := max(hmax(m.hload, 1), float64(m.ne+m.np))
	out = append(out, hjoin(
		dashed.box("swap", num(m.have, "%.2f GB", mem.SwapUsed/1073741824), hw, sh,
			peakGraph(m.hswap, hw-4, sh-3, swapTop, gb(swapTop)+" GB", cMem)...),
		dashed.box("load", num(m.have, "%.2f", m.sys.Load[0]), w-hw-1, sh,
			peakGraph(m.hload, w-hw-5, sh-3, loadTop, fmt.Sprintf("%.0f", loadTop), cCPU)...),
	)...)

	ph := rows - len(out)
	return append(out, dashed.box("processes by memory", "", w, ph, m.pProcBy(w-4, ph-3, "mem")...)...)
}

// ---------------------------------------------------------------------------
// io (heavy dashed): network and disk throughput, then the connection's saved speed tests.

func (m Model) ioLayout(rows, w int) []string {
	hw := (w - 1) / 2
	rw := w - hw - 1
	nd := m.s.NetDisk
	out := []string{m.head(w)}

	// the cloudflare row stops at 10 (its stats are 6 lines); net and disk graphs take the rest
	nh, dh := 8, 8
	if extra := rows - 1 - nh - dh - 10; extra > 0 {
		nh, dh = nh+extra-extra/2, dh+extra/2
	}
	rg := func(t string, v float64, hist []float64, bw, bh int, col lipgloss.Style) []string {
		top := hmax(hist, 1)
		return hdashed.box(t, rate(m.have, v), bw, bh, peakGraph(hist, bw-4, bh-2, top, rate(true, top), col)...)
	}
	out = append(out, hjoin(
		rg("net ↓", nd.InBytes, m.hnin, hw, nh, cDown),
		rg("net ↑", nd.OutBytes, m.hnout, rw, nh, cUp),
	)...)
	out = append(out, hjoin(
		rg("disk read", nd.ReadKBytes*1024, m.hdr, hw, dh, cDown),
		rg("disk write", nd.WriteKB*1024, m.hdw, rw, dh, cUp),
	)...)

	ph := rows - len(out)
	var hist []string
	if len(m.runs) > 0 {
		iw := rw - 4
		n := min(len(m.runs), (iw-5)/3)
		dl, ul := m.cfdl[len(m.cfdl)-n:], m.cful[len(m.cful)-n:]
		top := max(hmax(dl, 1), hmax(ul, 1))
		for i, l := range vbar2(dl, ul, max(ph-4, 1), top, cDown, cUp) {
			lbl := "     "
			if i == 0 {
				lbl = fmt.Sprintf("%4.0f ", top)
			}
			hist = append(hist, dim.Render(lbl)+l)
		}
		hist = append(hist, dim.Render(fmt.Sprintf("     Mbps · %d runs · ", len(m.runs)))+cDown.Render("▌")+
			dim.Render(" down ")+cUp.Render("▌")+dim.Render(" up"))
	}
	return append(out, hjoin(
		hdashed.box("cloudflare", m.cfLabel(), hw, ph, m.pCF(hw-4)...),
		hdashed.box("speed tests", "", rw, ph, hist...),
	)...)
}
