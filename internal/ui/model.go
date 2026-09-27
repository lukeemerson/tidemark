// Package ui is the Bubble Tea model for monitor's draft-03 tile layout.
package ui

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/sys/unix"

	"github.com/lukeemerson/mac-monitor/internal/source"
)

const histLen = 400

type sampleMsg source.Sample
type doneMsg struct{}
type cloudyMsg struct{ err error }

type Model struct {
	samples <-chan source.Sample
	runs    []source.Run
	names   source.ProcNames
	w, h    int

	name     string
	ne, np   int
	memTotal float64

	cloudy    *exec.Cmd // running speed test, if any
	cloudyErr error

	lay      int               // chosen layout (index into layouts)
	fallback bool              // set per frame when the chosen layout doesn't fit
	save     func(name string) // persists the chosen layout; nil in tests

	have              bool
	s                 source.Sample
	sys               source.Sys
	hcpu, hgpu, hpow  []float64
	hmem, htc, htg    []float64 // htc/htg: cpu and gpu temperature
	hload, hswap      []float64
	hnin, hnout       []float64 // network bytes/s
	hdr, hdw          []float64 // disk bytes/s
	hdram             []float64 // DRAM read+write GB/s
	cfdl, cful, cflat []float64
}

// New builds the model from what is known instantly (sysctl, saved cloudy runs);
// mactop's samples arrive on samples later. layout names the starting layout ("" = tiles).
func New(samples <-chan source.Sample, runs []source.Run, layout string, save func(string)) Model {
	m := Model{samples: samples, names: source.ProcNames{}, save: save}
	for i, l := range layouts {
		if l.name == layout {
			m.lay = i
		}
	}
	m.setRuns(runs)
	m.name, _ = unix.Sysctl("machdep.cpu.brand_string")
	e, _ := unix.SysctlUint32("hw.perflevel1.physicalcpu")
	p, _ := unix.SysctlUint32("hw.perflevel0.physicalcpu")
	mt, _ := unix.SysctlUint64("hw.memsize")
	m.ne, m.np, m.memTotal = int(e), int(p), float64(mt)
	return m
}

func (m *Model) setRuns(runs []source.Run) {
	m.runs, m.cfdl, m.cful, m.cflat = runs, nil, nil, nil
	for _, r := range runs {
		m.cfdl = append(m.cfdl, r.Download.Mbps)
		m.cful = append(m.cful, r.Upload.Mbps)
		m.cflat = append(m.cflat, r.IdleLatency.MedianMs)
	}
}

// Stop kills a speed test still running when the program exits.
func (m Model) Stop() {
	if m.cloudy != nil {
		m.cloudy.Process.Kill()
	}
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
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l", "L":
			step := 1
			if msg.String() == "L" {
				step = len(layouts) - 1
			}
			m.lay = (m.lay + step) % len(layouts)
			if m.save != nil {
				name, save := layouts[m.lay].name, m.save
				return m, func() tea.Msg { save(name); return nil }
			}
		case "r":
			if m.cloudy != nil {
				break
			}
			cmd, err := source.StartCloudy()
			if err != nil {
				m.cloudyErr = err
				break
			}
			m.cloudy, m.cloudyErr = cmd, nil
			return m, func() tea.Msg { return cloudyMsg{cmd.Wait()} }
		}
	case cloudyMsg:
		m.cloudy, m.cloudyErr = nil, msg.err
		m.setRuns(source.CloudyRuns(source.CloudyDir(), 40))
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
		m.htg = push(m.htg, m.s.SoC.GPUTemp)
		m.sys = source.ReadSys()
		m.hload = push(m.hload, m.sys.Load[0])
		m.hswap = push(m.hswap, m.s.Memory.SwapUsed)
		m.hnin = push(m.hnin, m.s.NetDisk.InBytes)
		m.hnout = push(m.hnout, m.s.NetDisk.OutBytes)
		m.hdr = push(m.hdr, m.s.NetDisk.ReadKBytes*1024)
		m.hdw = push(m.hdw, m.s.NetDisk.WriteKB*1024)
		m.hdram = push(m.hdram, m.s.SoC.DRAMRead+m.s.SoC.DRAMWrite)
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
	for i := range out {
		// backstop only: layout already fits its lines to w
		out[i] = ansi.Truncate(" "+out[i], m.w, "")
	}
	v.SetContent(strings.Join(out, "\n"))
	return v
}

// minProc is the smallest processes box worth drawing: borders, header and two rows.
const minProc = 5

// procMax is the tallest processes box with anything in it: mactop --headless sends at most
// 20 processes. Height beyond it goes to graphs, never to empty rows.
const procMax = 2 + 1 + 20

// layout draws the chosen layout, or tiles when the chosen one doesn't fit rows × w.
func (m Model) layout(rows, w int) []string {
	if w < 8 || rows < 1 {
		return nil
	}
	l := layouts[m.lay]
	if !l.fits(m, rows, w) {
		m.fallback, l = true, layouts[0]
	}
	return l.draw(m, rows, w)
}

// tilesLayout fits the screen to rows × w: panels drop out, narrowest first, rather than overflow.
func (m Model) tilesLayout(rows, w int) []string {
	out := []string{m.head(w)}
	out = append(out, m.stats(w)...)

	g, c, ng := 0, 0, 0       // graph row, cores row, narrow full-width cpu graph
	ch := (m.ne+m.np+1)/2 + 2 // two cores per row
	if w >= 61 {
		if rows-len(out)-9 >= minProc {
			g = 9
		}
		if rows-len(out)-g-ch >= minProc {
			c = ch
		}
	}
	ph := rows - len(out) - g - c
	if extra := ph - procMax; extra > 0 {
		switch {
		case g > 0:
			g, ph = g+extra, procMax
		case w < 61 && extra >= 4:
			ng, ph = extra, procMax
		}
	}
	if g > 0 {
		cw := (w - 1) / 2
		out = append(out, hjoin(
			panel("cpu", m.hCPU(), cw, g, func(iw int) []string { return m.pCPU(iw, g-3) }),
			panel("gpu", m.hGPU(), w-cw-1, g, func(iw int) []string { return m.pGPU(iw, g-3) }),
		)...)
	}
	if c > 0 {
		cw := (w - 1) / 2
		out = append(out, hjoin(
			panel("cores", "", cw, c, func(iw int) []string { return m.pCores(iw, 2) }),
			panel("power", m.hPow(), w-cw-1, c, func(iw int) []string { return m.pPow(iw, c-3) }),
		)...)
	}
	if ng > 0 {
		out = append(out, panel("cpu", m.hCPU(), w, ng, func(iw int) []string { return m.pCPU(iw, ng-3) })...)
	}

	if ph < 3 {
		return out[:min(len(out), rows)]
	}
	if w < 61 {
		return append(out, panel("processes", "", w, ph, func(iw int) []string { return m.pProc(iw, ph-3) })...)
	}
	cw := w * 2 / 3
	rw := w - cw - 1
	var right []string
	add := func(t, rt string, h int, content func(int) []string) {
		if len(right)+h <= ph {
			right = append(right, panel(t, rt, rw, h, content)...)
		}
	}
	add("memory", m.hMem(), 5, m.pMem)
	add("sensors", "", 5, m.pSens)
	add("io", "", 5, m.pIO)
	if cf := min(ph-len(right), 8); cf >= 4 {
		add("cloudflare", m.cfLabel(), cf, m.pCF)
	}
	return append(out, hjoin(
		panel("processes", "", cw, ph, func(iw int) []string { return m.pProc(iw, ph-3) }),
		right,
	)...)
}

func (m Model) cfLabel() string {
	switch {
	case m.cloudy != nil:
		return mid.Render("testing…")
	case m.cloudyErr != nil:
		return high.Render("test failed")
	case len(m.runs) > 0:
		return dim.Render(m.cfAge())
	}
	return ""
}

func (m Model) head(w int) string {
	l := title.Render(m.name) + dim.Render(fmt.Sprintf("  ·  %dE + %dP CPU  ·  %s-core GPU", m.ne, m.np, m.ngc()))
	r := mid.Render("● starting mactop…")
	if m.have {
		r = dim.Render(time.Now().Format("15:04:05"))
		if b := m.batt(); b != "" && lipgloss.Width(l)+lipgloss.Width(b)+20 <= w {
			r = dim.Render("battery "+b+"   ") + r
		}
	}
	ind := fmt.Sprintf("%s %d/%d", layouts[m.lay].name, m.lay+1, len(layouts))
	if m.fallback {
		ind = layouts[m.lay].name + " → tiles"
	}
	if lipgloss.Width(l)+lipgloss.Width(r)+lipgloss.Width(ind)+4 <= w {
		r = dim.Render(ind+"   ") + r
	}
	if lipgloss.Width(l)+lipgloss.Width(r)+1 > w {
		l = title.Render(m.name) // drop the core counts before the clock
	}
	return pad(spread(l, r, w), w)
}

type stat struct {
	name, val string
	hist      []float64
	top       float64
	col       *lipgloss.Style
}

func (m Model) statList() []stat {
	temp := num(m.have, "%.0f°", m.s.SoC.CPUTemp)
	if m.have {
		temp = title.Inherit(level(m.s.SoC.CPUTemp)).Render(temp)
	}
	var dl, ul, lat float64
	if len(m.runs) > 0 {
		r := m.runs[len(m.runs)-1]
		dl, ul, lat = r.Download.Mbps, r.Upload.Mbps, r.IdleLatency.MedianMs
	}
	return []stat{
		{"cpu", m.hCPU(), m.hcpu, 100, nil},
		{"gpu", m.hGPU(), m.hgpu, 100, &gpu},
		{"power", m.hPow(), m.hpow, hmax(m.hpow, 1), &power},
		{"mem", m.hMem(), m.hmem, 100, &mid},
		{"temp", temp, m.htc, 110, &high},
		{"↓ cf", title.Inherit(net).Render(fmt.Sprintf("%.0f Mbps", dl)), m.cfdl, hmax(m.cfdl, 1), &net},
		{"↑ cf", title.Inherit(power).Render(fmt.Sprintf("%.0f Mbps", ul)), m.cful, hmax(m.cful, 1), &power},
		{"ping", title.Render(fmt.Sprintf("%.0f ms", lat)), m.cflat, hmax(m.cflat, 1), &dim},
	}
}

func (st stat) spark(w int) string {
	sp := spark(st.hist, w, st.top)
	if st.col != nil {
		sp = st.col.Render(sp)
	}
	return sp
}

// stats shows the summary: 8 tiles in a row when wide, two rows of 4 when medium,
// one line per stat when narrow.
func (m Model) stats(w int) []string {
	ss := m.statList()
	switch {
	case w >= 103:
		return tileRow(ss, w)
	case w >= 51:
		return append(tileRow(ss[:4], w), tileRow(ss[4:], w)...)
	}
	return statLines(ss, w)
}

// statLines is one line per stat: name, value, sparkline filling the rest of w.
func statLines(ss []stat, w int) []string {
	var out []string
	for _, st := range ss {
		out = append(out, pad(dim.Render(fit(st.name, 6))+pad(st.val, 9)+" "+st.spark(max(w-16, 0)), w))
	}
	return out
}

// tileRow splits w evenly between boxed tiles; the last takes the remainder.
func tileRow(ss []stat, w int) []string {
	n := len(ss)
	tw := (w - (n - 1)) / n
	var blocks [][]string
	for i, st := range ss {
		bw := tw
		if i == n-1 {
			bw = w - (n-1)*(tw+1)
		}
		iw := bw - 4
		blocks = append(blocks, box(st.name, "", bw, 4, center(st.val, iw), st.spark(iw)))
	}
	return hjoin(blocks...)
}
