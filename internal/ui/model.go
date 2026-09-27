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

	have              bool
	s                 source.Sample
	hcpu, hgpu, hpow  []float64
	hmem, htc         []float64
	cfdl, cful, cflat []float64
}

// New builds the model from what is known instantly (sysctl, saved cloudy runs);
// mactop's samples arrive on samples later.
func New(samples <-chan source.Sample, runs []source.Run) Model {
	m := Model{samples: samples, names: source.ProcNames{}}
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

// layout fits the screen to rows × w: panels drop out, narrowest first, rather than overflow.
func (m Model) layout(rows, w int) []string {
	if w < 8 || rows < 1 {
		return nil
	}
	out := []string{m.head(w)}
	out = append(out, m.stats(w)...)

	if w >= 61 { // two-column graph panels
		cw := (w - 1) / 2
		if rows-len(out)-9 >= minProc {
			out = append(out, hjoin(
				panel("cpu", m.hCPU(), cw, 9, func(iw int) []string { return m.pCPU(iw, 6) }),
				panel("gpu", m.hGPU(), w-cw-1, 9, func(iw int) []string { return m.pGPU(iw, 6) }),
			)...)
		}
		ch := (m.ne+m.np+1)/2 + 2 // two cores per row
		if rows-len(out)-ch >= minProc {
			out = append(out, hjoin(
				panel("cores", "", cw, ch, func(iw int) []string { return m.pCores(iw, 2) }),
				panel("power", m.hPow(), w-cw-1, ch, func(iw int) []string { return m.pPow(iw, ch-3) }),
			)...)
		}
	}

	ph := rows - len(out)
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
	if cf := ph - len(right); cf >= 4 {
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
	var out []string
	for _, st := range ss {
		sw := w - 16
		out = append(out, pad(dim.Render(fit(st.name, 6))+pad(st.val, 9)+" "+st.spark(max(sw, 0)), w))
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
