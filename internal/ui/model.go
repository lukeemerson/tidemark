// Package ui is the Bubble Tea model for monitor's draft-03 tile layout.
package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/sys/unix"

	"github.com/lukeemerson/mac-monitor/internal/source"
)

const histLen = 400

type sampleMsg source.Sample
type doneMsg struct{}

type Model struct {
	samples <-chan source.Sample
	runs    []source.Run
	names   source.ProcNames
	w, h    int

	name     string
	ne, np   int
	memTotal float64

	have              bool
	s                 source.Sample
	hcpu, hgpu, hpow  []float64
	hmem, htc         []float64
	cfdl, cful, cflat []float64
}

// New builds the model from what is known instantly (sysctl, saved cloudy runs);
// mactop's samples arrive on samples later.
func New(samples <-chan source.Sample, runs []source.Run) Model {
	m := Model{samples: samples, runs: runs, names: source.ProcNames{}}
	m.name, _ = unix.Sysctl("machdep.cpu.brand_string")
	e, _ := unix.SysctlUint32("hw.perflevel1.physicalcpu")
	p, _ := unix.SysctlUint32("hw.perflevel0.physicalcpu")
	mt, _ := unix.SysctlUint64("hw.memsize")
	m.ne, m.np, m.memTotal = int(e), int(p), float64(mt)
	for _, r := range runs {
		m.cfdl = append(m.cfdl, r.Download.Mbps)
		m.cful = append(m.cful, r.Upload.Mbps)
		m.cflat = append(m.cflat, r.IdleLatency.MedianMs)
	}
	return m
}

func (m Model) wait() tea.Msg {
	s, ok := <-m.samples
	if !ok {
		return doneMsg{}
	}
	return sampleMsg(s)
}

func (m Model) Init() tea.Cmd { return m.wait }

func push(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > histLen {
		h = h[1:]
	}
	return h
}

func (m Model) memPct() float64 {
	if m.s.Memory.Total == 0 {
		return 0
	}
	return m.s.Memory.Used * 100 / m.s.Memory.Total
}

func (m Model) ngc() string {
	if !m.have {
		return "?"
	}
	return fmt.Sprint(m.s.SystemInfo.GPUCoreCount)
}

func (m Model) batt() string {
	b := m.s.Battery
	if !b.Present {
		return ""
	}
	st := ""
	if b.Charging {
		st = " charging"
	} else if b.OnACPower {
		st = " on AC"
	}
	return fmt.Sprintf("%.0f%%%s", b.Percent, st)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyPressMsg:
		if k := msg.String(); k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
	case sampleMsg:
		m.s, m.have = source.Sample(msg), true
		sort.SliceStable(m.s.Processes, func(i, j int) bool {
			return m.s.Processes[i].CPUPercent > m.s.Processes[j].CPUPercent
		})
		m.hcpu = push(m.hcpu, m.s.CPUUsage)
		m.hgpu = push(m.hgpu, m.s.GPUUsage)
		m.hpow = push(m.hpow, m.s.SoC.TotalPower)
		m.hmem = push(m.hmem, m.memPct())
		m.htc = push(m.htc, m.s.SoC.CPUTemp)
		return m, m.wait
	case doneMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	if m.w == 0 {
		return v
	}
	out := m.layout(m.h, m.w-2)
	if len(out) > m.h {
		out = out[:m.h]
	}
	for i := range out {
		out[i] = " " + out[i]
	}
	v.SetContent(strings.Join(out, "\n"))
	return v
}

func (m Model) layout(rows, w int) []string {
	out := []string{m.head(w)}
	out = append(out, m.tiles(w)...)

	cw := (w - 1) / 2
	out = append(out, hjoin(
		panel("cpu", m.hCPU(), cw, 9, func(iw int) []string { return m.pCPU(iw, 6) }),
		panel("gpu", m.hGPU(), w-cw-1, 9, func(iw int) []string { return m.pGPU(iw, 6) }),
	)...)
	out = append(out, hjoin(
		panel("cores", "", cw, 7, func(iw int) []string { return m.pCores(iw, 2) }),
		panel("power", m.hPow(), w-cw-1, 7, func(iw int) []string { return m.pPow(iw, 4) }),
	)...)

	ph := max(rows-len(out), 8)
	cw = w * 2 / 3
	rw := w - cw - 1
	right := panel("memory", m.hMem(), rw, 5, m.pMem)
	right = append(right, panel("sensors", "", rw, 5, m.pSens)...)
	right = append(right, panel("io", "", rw, 5, m.pIO)...)
	age := ""
	if len(m.runs) > 0 {
		age = dim.Render(m.cfAge())
	}
	right = append(right, panel("cloudflare", age, rw, ph-15, m.pCF)...)
	return append(out, hjoin(
		panel("processes", "", cw, ph, func(iw int) []string { return m.pProc(iw, ph-3) }),
		right,
	)...)
}

func (m Model) head(w int) string {
	l := title.Render(m.name) + dim.Render(fmt.Sprintf("  ·  %dE + %dP CPU  ·  %s-core GPU", m.ne, m.np, m.ngc()))
	r := mid.Render("● starting mactop…")
	if m.have {
		r = dim.Render(time.Now().Format("15:04:05"))
		if b := m.batt(); b != "" {
			r = dim.Render("battery "+b+"   ") + r
		}
	}
	return spread(l, r, w)
}

func tile(name, val string, hist []float64, top float64, col *lipgloss.Style, w int) []string {
	iw := w - 4
	sp := spark(hist, iw, top)
	if col != nil {
		sp = col.Render(sp)
	}
	return box(name, "", w, 4, center(val, iw), sp)
}

func (m Model) tiles(w int) []string {
	tw := (w - 7) / 8
	last := w - 7*tw - 7
	temp := num(m.have, "%.0f°", m.s.SoC.CPUTemp)
	if m.have {
		temp = title.Inherit(level(m.s.SoC.CPUTemp)).Render(temp)
	}
	var dl, ul, lat float64
	if len(m.runs) > 0 {
		r := m.runs[len(m.runs)-1]
		dl, ul, lat = r.Download.Mbps, r.Upload.Mbps, r.IdleLatency.MedianMs
	}
	return hjoin(
		tile("cpu", m.hCPU(), m.hcpu, 100, nil, tw),
		tile("gpu", m.hGPU(), m.hgpu, 100, &gpu, tw),
		tile("power", m.hPow(), m.hpow, hmax(m.hpow, 1), &power, tw),
		tile("mem", m.hMem(), m.hmem, 100, &mid, tw),
		tile("temp", temp, m.htc, 110, &high, tw),
		tile("↓ cf", title.Inherit(net).Render(fmt.Sprintf("%.0f Mbps", dl)), m.cfdl, hmax(m.cfdl, 1), &net, tw),
		tile("↑ cf", title.Inherit(power).Render(fmt.Sprintf("%.0f Mbps", ul)), m.cful, hmax(m.cful, 1), &power, tw),
		tile("ping", title.Render(fmt.Sprintf("%.0f ms", lat)), m.cflat, hmax(m.cflat, 1), &dim, last),
	)
}
