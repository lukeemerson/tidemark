package ui

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/lukeemerson/tidemark/internal/source"
)

// Panel contents take the inner width (panel width - 4) and return lines.

// v is series i's value on screen (compare.go), bold once mactop has reported.
func (m Model) v(i int) string {
	x, _ := value(m.s, i)
	s := num(m.have, valFormat[i], x)
	if m.have {
		s = title.Render(s)
	}
	return s
}

// hCPU etc. are box values: the value, then its difference from a marked B.
func (m Model) hCPU() string { return withB(m.v(sCPU), m.vsB(sCPU)) }
func (m Model) hGPU() string { return withB(m.v(sGPU), m.vsB(sGPU)) }
func (m Model) hPow() string { return withB(m.v(sPow), m.vsB(sPow)) }
func (m Model) hMem() string { return withB(m.v(sMem), m.vsB(sMem)) }

func (m Model) pCPU(w, gh int) []string {
	soc := m.s.SoC
	l := dim.Render(" E ") + num(m.have, "%4.0f", soc.EFreqMHz) + dim.Render(" · P ") +
		num(m.have, "%4.0f", soc.PFreqMHz) + dim.Render(" MHz")
	return append([]string{l}, axisGraph(m.hcpu, m.ghost(sCPU), w, gh, 100, "100%", cCPU)...)
}

func (m Model) pGPU(w, gh int) []string {
	soc := m.s.SoC
	l := dim.Render(fmt.Sprintf(" %s cores  ·  ", m.ngc())) + num(m.have, "%4.0f", soc.GPUFreqMHz) + " MHz" +
		dim.Render("  ·  ANE ") + num(m.have, "%3.0f%%", soc.ANEActive)
	return append([]string{l}, axisGraph(m.hgpu, m.ghost(sGPU), w, gh, 100, "100%", cGPU)...)
}

func (m Model) pCores(w, per int) []string {
	nc := m.ne + m.np
	cellW := (w - 3*(per-1)) / per
	lw := len(fmt.Sprint(max(m.ne, m.np))) + 2 // "E"/"P", digits, a space
	bw := cellW - 5 - lw
	var out []string
	for i := 0; i*per < nc; i++ {
		l := ""
		for j := 1; j <= per; j++ {
			idx := i*per + j
			if idx > nc {
				continue
			}
			p := 0.0
			if idx <= len(m.s.CoreUsages) {
				p = m.s.CoreUsages[idx-1]
			}
			label := fmt.Sprintf("P%d", idx-m.ne)
			if idx <= m.ne {
				label = fmt.Sprintf("E%d", idx)
			}
			l += dim.Render(fit(label, lw)) + bar(p, bw, nil) + num(m.have, "%4.0f%%", p)
			if j < per {
				l += "   "
			}
		}
		out = append(out, l)
	}
	return out
}

func (m Model) pPow(w, h int) []string {
	soc := m.s.SoC
	out := []string{dim.Render(" system ") + num(m.have, "%.1f", soc.SystemPower) + dim.Render(" W  ·  GPU ") +
		num(m.have, "%.1f", soc.GPUPower) + dim.Render(" W")}
	top := hmax(m.hpow, 1)
	return append(out, axisGraph(m.hpow, m.ghost(sPow), w, h, top, fmt.Sprintf("%.1fW", top), cPower)...)
}

func (m Model) pMem(w int) []string {
	mem := m.s.Memory
	mp := m.memPct()
	used, swap := gb(mem.Used), gb(mem.SwapUsed)
	if !m.have {
		used, swap = "—", "—"
	}
	total := mem.Total
	if !m.have {
		total = m.memTotal // sysctl, until mactop reports
	}
	l := used + dim.Render(" / "+gb(total)+" GB")
	r := dim.Render("swap " + swap + " / " + gb(mem.SwapTotal) + " GB")
	return []string{
		spread(l, r, w),
		bar(mp, w, nil),
		dim.Render(" DRAM") + "  ↓ " + num(m.have, "%.1f", m.s.SoC.DRAMRead) + "  ↑ " +
			num(m.have, "%.1f", m.s.SoC.DRAMWrite) + " GB/s",
	}
}

func (m Model) pSens(w int) []string {
	soc := m.s.SoC
	out := []string{
		dim.Render(" CPU ") + num(m.have, "%.0f°", soc.CPUTemp) +
			dim.Render("   GPU ") + num(m.have, "%.0f°", soc.GPUTemp),
	}
	thermal, fan := m.s.ThermalState, 0.0
	if thermal == "" {
		thermal = "—"
	}
	if len(m.s.Fans) > 0 {
		fan = m.s.Fans[0].RPM
	}
	out = append(out, dim.Render(" thermal ")+thermal+dim.Render("   fan ")+num(m.have, "%.0f", fan)+" rpm")
	if b := m.batt(); b != "" {
		out = append(out, dim.Render(" battery ")+b)
	}
	return out
}

func (m Model) pIO(w int) []string {
	nd := m.s.NetDisk
	out := []string{
		dim.Render(" NET  ") + "↓ " + rate(m.have, nd.InBytes) + "  ↑ " + rate(m.have, nd.OutBytes),
		dim.Render(" DISK ") + "r " + rate(m.have, nd.ReadKBytes*1024) + "  w " + rate(m.have, nd.WriteKB*1024),
	}
	if len(m.s.Volumes) == 0 {
		return append(out, dim.Render(" volume —"))
	}
	v := m.s.Volumes[0]
	return append(out, dim.Render(" "+fit(v.Name, 12))+bar(v.UsedPercent, w-20, nil)+fmt.Sprintf("%4.0f%%", v.UsedPercent))
}

func (m Model) cfAge() string {
	s := int(time.Since(m.runs[len(m.runs)-1].Timestamp).Seconds())
	switch {
	case s < 3600:
		return fmt.Sprintf("%dm ago", s/60)
	case s < 172800:
		return fmt.Sprintf("%dh ago", s/3600)
	}
	return fmt.Sprintf("%dd ago", s/86400)
}

func (m Model) pCF(w int) []string {
	if len(m.runs) == 0 {
		return []string{dim.Render(" no saved runs — run: cloudy")}
	}
	r := m.runs[len(m.runs)-1]
	bn := title
	lat := r.IdleLatency
	out := []string{
		bn.Render(fmt.Sprintf("↓ %.0f", r.Download.Mbps)) + dim.Render(" Mbps") + "   " +
			bn.Render(fmt.Sprintf("↑ %.0f", r.Upload.Mbps)) + dim.Render(" Mbps"),
		dim.Render(" ping ") + fmt.Sprintf("%.0f ms", lat.MedianMs) + dim.Render("  jitter ") +
			fmt.Sprintf("%.0f ms", lat.JitterMs) + dim.Render("  loss ") + fmt.Sprintf("%d%%", int(lat.Loss*100)),
		dim.Render(" loaded ↓ ") + fmt.Sprintf("%.0f", r.LoadedDownload.MedianMs) + dim.Render(" ↑ ") +
			fmt.Sprintf("%.0f ms", r.LoadedUpload.MedianMs) + dim.Render("   bloat ") + orQ(r.Quality.Bufferbloat) +
			dim.Render("  stable ") + orQ(r.Quality.Stability),
	}
	sw := min(len(m.runs), w-9)
	out = append(out,
		dim.Render(" ↓ hist ")+cDown.Render(spark(m.cfdl, sw, hmax(m.cfdl, 1))),
		dim.Render(" ↑ hist ")+cUp.Render(spark(m.cful, sw, hmax(m.cful, 1))),
	)
	colo, isp := r.Meta.Colo.IATA, r.Meta.ASOrg
	if colo == "" {
		colo = orQ(r.Colo)
	}
	if isp == "" {
		isp = r.ASOrg
	}
	isp, _, _ = strings.Cut(isp, ",")
	isp = strings.TrimSuffix(isp, " Communications")
	where := " " + colo + " · " + isp
	if r.Wireless {
		where += " · wi-fi"
	}
	return append(out, dim.Render(where+"  ·  "+m.cfAge()))
}

func orQ(s string) string {
	if s == "" {
		return "?"
	}
	return s
}

func (m Model) pProc(w, n int) []string { return m.procTable(w, n, "cpu") }

// procTable lists processes in their current order; sorted by "gpu", the mid-width table
// shows GPU ms/s where it otherwise shows MEM%.
func (m Model) procTable(w, n int, by string) []string {
	cols := "full"
	if w < 72 {
		cols = "mid"
	}
	if w < 48 {
		cols = "min"
	}
	nameW := map[string]int{"full": w - 50, "mid": w - 24, "min": w - 8}[cols]
	hd := fit("COMMAND", nameW)
	var out []string
	switch cols {
	case "full":
		out = []string{title.Render(" PID     " + hd + fmt.Sprintf("%7s %10s %7s %10s", "CPU%", "GPU ms/s", "MEM%", "RSS"))}
	case "mid":
		second := "MEM%"
		if by == "gpu" {
			second = "GPU ms"
		}
		out = []string{title.Render(" PID     " + hd + fmt.Sprintf("%7s %7s", "CPU%", second))}
	default:
		out = []string{title.Render(" " + hd + fmt.Sprintf("%7s", "CPU%"))}
	}
	for i := 0; i < n; i++ {
		if i >= len(m.s.Processes) {
			if !m.have && i < 3 {
				out = append(out, dim.Render(" loading…"))
			} else {
				out = append(out, "")
			}
			continue
		}
		p := m.s.Processes[i]
		pid := dim.Render(" " + fit(fmt.Sprint(p.PID), 8))
		name := fit(m.procName(p.PID, p.Command), nameW)
		cpu := fmt.Sprintf("%7.1f", p.CPUPercent)
		memp := dim.Render(fmt.Sprintf("%8.1f", p.MemPercent))
		if by == "gpu" {
			memp = fmt.Sprintf("%8.1f", p.GPUMsPerS)
		}
		switch cols {
		case "full":
			out = append(out, pid+name+cpu+fmt.Sprintf("%11.1f", p.GPUMsPerS)+
				dim.Render(fmt.Sprintf("%8.1f%8.0f MB", p.MemPercent, p.RSSKB/1024)))
		case "mid":
			out = append(out, pid+name+cpu+memp)
		default:
			out = append(out, " "+name+cpu)
		}
	}
	return out
}

// pProcBy is pProc with the table sorted by "gpu" (ms/s) or "mem" (RSS) instead of CPU.
func (m Model) pProcBy(w, n int, by string) []string {
	ps := slices.Clone(m.s.Processes)
	key := func(p source.Process) float64 { return p.GPUMsPerS }
	if by == "mem" {
		key = func(p source.Process) float64 { return p.RSSKB }
	}
	sort.SliceStable(ps, func(i, j int) bool { return key(ps[i]) > key(ps[j]) })
	m.s.Processes = ps
	return m.procTable(w, n, by)
}

// panel boxes a panel's contents: the content function gets the inner width.
func panel(t, rt string, w, h int, content func(iw int) []string) []string {
	return box(t, rt, w, h, content(w-4)...)
}
