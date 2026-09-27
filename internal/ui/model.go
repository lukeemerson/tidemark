// Package ui is the Bubble Tea model for monitor's draft-03 tile layout.
package ui

import (
	"fmt"
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
	w, h    int

	name   string
	ne, np int

	have              bool
	s                 source.Sample
	hcpu, hgpu, hpow  []float64
	hmem, htc         []float64
	cfdl, cful, cflat []float64
}

// New builds the model from what is known instantly (sysctl, saved cloudy runs);
// mactop's samples arrive on samples later.
func New(samples <-chan source.Sample, runs []source.Run) Model {
	m := Model{samples: samples, runs: runs}
	m.name, _ = unix.Sysctl("machdep.cpu.brand_string")
	e, _ := unix.SysctlUint32("hw.perflevel1.physicalcpu")
	p, _ := unix.SysctlUint32("hw.perflevel0.physicalcpu")
	m.ne, m.np = int(e), int(p)
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
	w := m.w - 2
	out := []string{m.head(w)}
	out = append(out, m.tiles(w)...)
	for i := range out {
		out[i] = " " + out[i]
	}
	v.SetContent(strings.Join(out, "\n"))
	return v
}

func (m Model) head(w int) string {
	ngc := "?"
	if m.have {
		ngc = fmt.Sprint(m.s.SystemInfo.GPUCoreCount)
	}
	l := title.Render(m.name) + dim.Render(fmt.Sprintf("  ·  %dE + %dP CPU  ·  %s-core GPU", m.ne, m.np, ngc))
	r := mid.Render("● starting mactop…")
	if m.have {
		r = dim.Render(time.Now().Format("15:04:05"))
		if b := m.s.Battery; b.Present {
			st := ""
			if b.Charging {
				st = " charging"
			} else if b.OnACPower {
				st = " on AC"
			}
			r = dim.Render(fmt.Sprintf("battery %.0f%%%s   ", b.Percent, st)) + r
		}
	}
	return spread(l, r, w)
}

// value formats a live reading, or a dim dash before mactop's first sample.
func (m Model) value(live bool, st lipgloss.Style, format string, v float64) string {
	if !live {
		return dim.Render("—")
	}
	return st.Render(fmt.Sprintf(format, v))
}

func tile(name, val string, hist []float64, top float64, sparkStyle func(...string) string, w int) []string {
	iw := w - 4
	return box(name, w, 4, center(val, iw), sparkStyle(spark(hist, iw, top)))
}

func (m Model) tiles(w int) []string {
	tw := (w - 7) / 8
	last := w - 7*tw - 7
	lvl := func(v float64) lipgloss.Style { return title.Inherit(level(v)) }
	plain := func(s ...string) string { return strings.Join(s, "") }
	cf := len(m.runs) > 0
	var dl, ul, lat float64
	if cf {
		r := m.runs[len(m.runs)-1]
		dl, ul, lat = r.Download.Mbps, r.Upload.Mbps, r.IdleLatency.MedianMs
	}
	return hjoin(
		tile("cpu", m.value(m.have, lvl(m.s.CPUUsage), "%3.0f%%", m.s.CPUUsage), m.hcpu, 100, plain, tw),
		tile("gpu", m.value(m.have, title.Inherit(gpu), "%3.0f%%", m.s.GPUUsage), m.hgpu, 100, gpu.Render, tw),
		tile("power", m.value(m.have, title.Inherit(power), "%.1f W", m.s.SoC.TotalPower), m.hpow, hmax(m.hpow, 1), power.Render, tw),
		tile("mem", m.value(m.have, lvl(m.memPct()), "%3.0f%%", m.memPct()), m.hmem, 100, mid.Render, tw),
		tile("temp", m.value(m.have, lvl(m.s.SoC.CPUTemp), "%.0f°", m.s.SoC.CPUTemp), m.htc, 110, high.Render, tw),
		tile("↓ cf", m.value(cf, title.Inherit(net), "%.0f Mbps", dl), m.cfdl, hmax(m.cfdl, 1), net.Render, tw),
		tile("↑ cf", m.value(cf, title.Inherit(power), "%.0f Mbps", ul), m.cful, hmax(m.cful, 1), power.Render, tw),
		tile("ping", m.value(cf, title, "%.0f ms", lat), m.cflat, hmax(m.cflat, 1), dim.Render, last),
	)
}
