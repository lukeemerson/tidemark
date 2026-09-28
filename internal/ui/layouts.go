package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// layoutDef is one l/L layout. fits says whether it can draw at rows × w; if not, tiles does.
type layoutDef struct {
	name string
	fits func(m Model, rows, w int) bool
	draw func(m Model, rows, w int) []string
}

var layouts []layoutDef

func init() { // in init: the draw functions reach layouts through head()
	layouts = []layoutDef{
		{"tiles", func(Model, int, int) bool { return true }, Model.tilesLayout},
		{"sidebar", func(_ Model, rows, w int) bool { return w >= 80 && rows >= 28 }, Model.sidebarLayout},
		{"instrument", func(m Model, rows, w int) bool {
			return w >= 70 && rows >= 17+max(3, (m.ne+m.np+1)/2)
		}, Model.instrumentLayout},
		{"console", func(m Model, rows, w int) bool {
			return w >= 90 && rows >= 21+max(3, (m.ne+m.np+1)/2)
		}, Model.consoleLayout},
		{"compute", func(m Model, rows, w int) bool {
			return w >= 80 && rows >= 16+max((m.ne+m.np+1)/2+2, 7)
		}, Model.computeLayout},
		{"memory", func(_ Model, rows, w int) bool { return w >= 70 && rows >= 23 }, Model.memoryLayout},
		{"io", func(_ Model, rows, w int) bool { return w >= 70 && rows >= 24 }, Model.ioLayout},
		{"glance", func(_ Model, rows, w int) bool { return w >= 40 && rows >= 16 }, Model.glanceLayout},
		{"wall", func(_ Model, rows, w int) bool { return w >= 60 && rows >= 25 }, Model.wallLayout},
		{"thermal", func(_ Model, rows, w int) bool { return w >= 70 && rows >= 24 }, Model.thermalLayout},
	}
}

// ---------------------------------------------------------------------------
// sidebar: rounded light frames. Every number in one column at the left; the rest is graphs
// over processes. For reading values at a glance while watching trends.

const sideW = 38

func (m Model) sidebarLayout(rows, w int) []string {
	gw := w - sideW - 1
	left := rounded.box(m.name, "", sideW, rows-1, m.side(sideW-4, rows-3)...)

	soc := m.s.SoC
	freq := dim.Render(fmt.Sprintf("E %s · P %s MHz", num(m.have, "%.0f", soc.EFreqMHz), num(m.have, "%.0f", soc.PFreqMHz)))
	ch, gh, pwh := 8, 7, 7
	ph := rows - 1 - ch - gh - pwh
	if extra := ph - procMax; extra > 0 { // spare height goes to the three graphs
		ch, gh, pwh, ph = ch+extra-2*(extra/3), gh+extra/3, pwh+extra/3, procMax
	}
	right := rounded.box("cpu", freq+"  "+m.hCPU(), gw, ch, axisGraph(m.hcpu, gw-4, ch-2, 100, "100%", cCPU)...)
	right = append(right, rounded.box("gpu", m.hGPU(), gw, gh, m.pGPU(gw-4, gh-3)...)...)
	right = append(right, rounded.box("power", m.hPow(), gw, pwh, m.pPow(gw-4, pwh-3)...)...)
	right = append(right, rounded.box("processes", "", gw, ph, m.pProc(gw-4, ph-3)...)...)
	return append([]string{m.head(w)}, hjoin(left, right)...)
}

// side is the sidebar's column of readouts, iw wide and at most h lines.
func (m Model) side(iw, h int) []string {
	row := func(label, val string, p ...float64) string {
		l := dim.Render(fit(label, 7)) + val
		if len(p) == 0 {
			return l
		}
		bw := iw - 21
		return pad(l, iw-bw) + barG(p[0], bw, nil, "▰", "▱")
	}
	soc, nd := m.s.SoC, m.s.NetDisk
	strip := ""
	for i := 0; i < m.ne+m.np; i++ {
		p := 0.0
		if i < len(m.s.CoreUsages) {
			p = m.s.CoreUsages[i]
		}
		strip += level(p).Render(string(spk[min(int(p/100*7+0.5), 7)]))
	}
	thermal, fan := m.s.ThermalState, 0.0
	if thermal == "" {
		thermal = "—"
	}
	if len(m.s.Fans) > 0 {
		fan = m.s.Fans[0].RPM
	}
	s := []string{
		row("cpu", m.hCPU(), m.s.CPUUsage),
		row("cores", strip),
		row("gpu", m.hGPU(), m.s.GPUUsage),
		row("mem", m.hMem(), m.memPct()),
		row("power", m.hPow()),
		"",
		row("temp", num(m.have, "%.0f°", soc.CPUTemp)+dim.Render(" cpu  ")+
			num(m.have, "%.0f°", soc.GPUTemp)+dim.Render(" gpu")),
		row("fan", num(m.have, "%.0f", fan)+dim.Render(" rpm · "+thermal)),
		row("net", "↓ "+rate(m.have, nd.InBytes)+" ↑ "+rate(m.have, nd.OutBytes)),
		row("disk", "r "+rate(m.have, nd.ReadKBytes*1024)+" w "+rate(m.have, nd.WriteKB*1024)),
	}
	if len(m.s.Volumes) > 0 {
		v := m.s.Volumes[0]
		s = append(s, row(fit(v.Name, 6), fmt.Sprintf("%.0f%%", v.UsedPercent), v.UsedPercent))
	}
	s = append(s, "", dim.Render("── cloudflare "+rep("─", iw-14)))
	if len(m.runs) == 0 {
		return append(s, dim.Render(" no saved runs — run: cloudy"))
	}
	r := m.runs[len(m.runs)-1]
	where := m.cfAge() + " · " + orQ(r.Meta.Colo.IATA)
	if r.Wireless {
		where += " · wi-fi"
	}
	if m.cloudy != nil {
		where = "testing…"
	}
	s = append(s,
		row("speed", title.Render(fmt.Sprintf("↓ %.0f", r.Download.Mbps))+dim.Render(" / ")+
			title.Render(fmt.Sprintf("↑ %.0f", r.Upload.Mbps))+dim.Render(" Mbps")),
		row("ping", fmt.Sprintf("%.0f ms", r.IdleLatency.MedianMs)+dim.Render("  jitter ")+
			fmt.Sprintf("%.0f ms", r.IdleLatency.JitterMs)),
		row("grade", dim.Render("bloat ")+orQ(r.Quality.Bufferbloat)+dim.Render("  stable ")+orQ(r.Quality.Stability)),
		row("last", dim.Render(where)),
	)
	// leftover height: one ↓/↑ bar pair per saved run
	ch := h - len(s) - 3
	if ch < 3 {
		return s
	}
	n := min(len(m.runs), (iw-5)/3)
	dl, ul := m.cfdl[len(m.cfdl)-n:], m.cful[len(m.cful)-n:]
	top := max(hmax(dl, 1), hmax(ul, 1))
	s = append(s, "")
	for i, l := range vbar2(dl, ul, ch, top, cDown, cUp) {
		lbl := "     "
		if i == 0 {
			lbl = fmt.Sprintf("%4.0f ", top)
		}
		s = append(s, dim.Render(lbl)+l)
	}
	return append(s, dim.Render(fmt.Sprintf("     %d runs · ", len(m.runs)))+cDown.Render("█")+
		dim.Render(" down ")+cUp.Render("▓")+dim.Render(" up"))
}

// ---------------------------------------------------------------------------
// instrument: one double-line frame, sections split by shared ╠═╬═╣ dividers, bars and numbers
// instead of graphs. The process table takes everything below. For working with processes.

// seg is a divider segment span wide: ═ title ═══ right label ═
func seg(t, rt string, span int) string {
	tl, tr := " "+t+" ", ""
	if rt != "" {
		tr = " " + rt + " "
	}
	if lipgloss.Width(tl)+lipgloss.Width(tr) > span-2 {
		tr = ""
	}
	if lipgloss.Width(tl) > span-2 {
		tl = " " + fit(t, span-4) + " "
	}
	return dim.Render("═") + title.Render(tl) + dim.Render(rep("═", span-2-lipgloss.Width(tl)-lipgloss.Width(tr))) + tr + dim.Render("═")
}

// band is a two-column section under a divider whose junctions are j (left, middle, right).
func band(j [3]string, lt, lrt string, lw int, rt, rrt string, rw int, L, R []string) []string {
	out := []string{dim.Render(j[0]) + seg(lt, lrt, lw) + dim.Render(j[1]) + seg(rt, rrt, rw) + dim.Render(j[2])}
	v := dim.Render("║")
	for i := 0; i < max(len(L), len(R)); i++ {
		l, r := "", ""
		if i < len(L) {
			l = L[i]
		}
		if i < len(R) {
			r = R[i]
		}
		out = append(out, v+" "+pad(l, lw-2)+" "+v+" "+pad(r, rw-2)+" "+v)
	}
	return out
}

func (m Model) instrumentLayout(rows, w int) []string {
	out := m.instrumentBands(w)
	pr := rows - len(out) - 1
	out = append(out, m.historyBand(&pr, w)...)
	out = append(out, m.procBand(pr, w)...)
	return append(out, dim.Render("╚"+rep("═", w-2)+"╝"))
}

// consoleLayout is instrument with the tiles layout's 8 small panels across the bottom,
// as columns of the same frame.
func (m Model) consoleLayout(rows, w int) []string {
	out := m.instrumentBands(w)
	pr := rows - len(out) - 4
	out = append(out, m.historyBand(&pr, w)...)
	out = append(out, m.procBand(pr, w)...)
	return append(out, tileBand(m.statList(), w)...)
}

// historyBand takes the rows the process table can't fill (it never has more than 20
// processes) and draws cpu and power history there; nil when there aren't at least 3.
func (m Model) historyBand(pr *int, w int) []string {
	extra := *pr - (procMax - 1) // procBand is a divider, a header and up to 20 rows
	if extra < 3 {
		return nil
	}
	*pr -= extra
	lw := (w - 3) * 3 / 5
	rw := w - 3 - lw
	return band([3]string{"╠", "╬", "╣"}, "cpu history", m.hCPU(), lw, "power history", m.hPow(), rw,
		axisGraph(m.hcpu, lw-2, extra-1, 100, "100%", cCPU),
		axisGraph(m.hpow, rw-2, extra-1, hmax(m.hpow, 1), fmt.Sprintf("%.1fW", hmax(m.hpow, 1)), cPower))
}

// procBand is the full-width process table under the last two-column band, rows lines tall.
func (m Model) procBand(rows, w int) []string {
	lw := (w - 3) * 3 / 5
	out := []string{dim.Render("╠") + seg("processes", "", lw) + dim.Render("╩"+rep("═", w-3-lw)+"╣")}
	v := dim.Render("║")
	for _, l := range m.pProc(w-4, rows-2) {
		out = append(out, v+" "+pad(l, w-4)+" "+v)
	}
	return out
}

// tileBand closes the frame with one column per stat: divider, value, sparkline, bottom edge.
func tileBand(ss []stat, w int) []string {
	n := len(ss)
	cw := (w - 1 - n) / n
	top, val, sp, bot := dim.Render("╠"), "", "", dim.Render("╚")
	v := dim.Render("║")
	for i, st := range ss {
		c := cw
		if i == n-1 {
			c = w - 1 - n - (n-1)*cw
		}
		j, jb := "╦", "╩"
		if i == n-1 {
			j, jb = "╣", "╝"
		}
		top += seg(st.name, "", c) + dim.Render(j)
		val += v + " " + pad(center(st.val, c-2), c-2) + " "
		sp += v + " " + st.spark(c-2) + " "
		bot += dim.Render(rep("═", c) + jb)
	}
	return []string{top, val + v, sp + v, bot}
}

// instrumentBands is the header and the three two-column bands shared by instrument and console.
func (m Model) instrumentBands(w int) []string {
	lw := (w - 3) * 3 / 5
	rw := w - 3 - lw
	soc := m.s.SoC
	thermal, fan := m.s.ThermalState, 0.0
	if thermal == "" {
		thermal = "—"
	}
	if len(m.s.Fans) > 0 {
		fan = m.s.Fans[0].RPM
	}
	out := []string{m.head(w)}

	cpuL := []string{
		m.pCPU(lw-2, 0)[0],
		bar(m.s.CPUUsage, lw-2, nil),
		dim.Render(" temp ") + num(m.have, "%.0f°", soc.CPUTemp) + dim.Render(" cpu  ") +
			num(m.have, "%.0f°", soc.GPUTemp) + dim.Render(" gpu  ·  ") + thermal +
			dim.Render("  ·  ") + num(m.have, "%.0f", fan) + dim.Render(" rpm"),
	}
	out = append(out, band([3]string{"╔", "╦", "╗"}, "cpu", m.hCPU(), lw, "cores", "", rw, cpuL, m.pCores(rw-2, 2))...)

	peak := hmax(m.hpow, 1)
	powR := []string{
		m.pPow(rw-2, 0)[0],
		bar(soc.TotalPower/peak*100, rw-2, &cPower),
		dim.Render(fmt.Sprintf(" of %.1f W peak", peak)),
	}
	gpuL := []string{m.pGPU(lw-2, 0)[0], bar(m.s.GPUUsage, lw-2, &cGPU)}
	out = append(out, band([3]string{"╠", "╬", "╣"}, "gpu", m.hGPU(), lw, "power", m.hPow(), rw, gpuL, powR)...)

	memL := append(m.pMem(lw-2), m.pIO(lw-2)...)
	return append(out, band([3]string{"╠", "╬", "╣"}, "memory · io", m.hMem(), lw, "cloudflare", m.cfLabel(), rw, memL, m.pCF(rw-2))...)
}
