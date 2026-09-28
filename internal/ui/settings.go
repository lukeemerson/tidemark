package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lukeemerson/tidemark/internal/config"
	"github.com/lukeemerson/tidemark/internal/source"
)

// Settings menu (SETTINGS-SPEC.md): o opens an overlay for the settings tidemark already has.
// Changes apply live; the config is saved when it closes.

// Control restarts mactop and opens or closes the store for the menu.
type Control interface {
	Restart(intervalMs int) <-chan source.Sample // nil when mactop can't start
	Store(on bool) Recorder                      // nil when off or unavailable
}

// Setup is what the menu starts from.
type Setup struct {
	Ctl          Control  // nil on -play
	Interval     int      // mactop's interval now, ms
	Store        Recorder // the open store, nil when off
	IntervalFlag bool     // -i set the interval this run: its row is locked
	StoreFlag    bool     // -nostore this run: its row is locked
	Saved        config.Config
}

var intervals = []int{250, 500, 1000, 2000, 5000}

const (
	rowLayout = iota
	rowPalette
	rowInterval
	rowStore
	nRows
)

var rowNames = [nRows]string{"layout", "palette", "interval", "store"}

// Settings wires the menu to what main runs.
func (m Model) Settings(s Setup) Model {
	m.ctl, m.interval, m.saved = s.Ctl, s.Interval, s.Saved
	m.lockI, m.lockS = s.IntervalFlag, s.StoreFlag
	if s.Store != nil {
		m = m.Store(s.Store)
	}
	return m
}

// fixed is why a row can't change this run: "(-i)", "(-nostore)", "(replay)"; "" when it can.
func (m Model) fixed(row int) string {
	if row < rowInterval {
		return ""
	}
	switch {
	case m.replay != nil:
		return "(replay)"
	case row == rowInterval && m.lockI:
		return "(-i)"
	case row == rowStore && m.lockS:
		return "(-nostore)"
	}
	return ""
}

func (m Model) rowValue(row int) string {
	switch row {
	case rowLayout:
		return layouts[m.lay].name
	case rowPalette:
		return m.palette
	case rowInterval:
		if m.replay != nil {
			return "—"
		}
		return fmt.Sprintf("%d ms", m.interval)
	}
	if m.replay != nil {
		return "—"
	}
	if m.store != nil {
		return "on"
	}
	return "off"
}

// menuKey handles a key while the menu is open: only q (and ctrl+c) reach past it.
func (m Model) menuKey(k string) (Model, tea.Cmd) {
	switch k {
	case "q", "ctrl+c":
		m.menu = false
		return m, tea.Sequence(m.saveCmd(), tea.Quit)
	case "o", "esc":
		m.menu = false
		return m, m.saveCmd()
	case "up", "down":
		m.sel = (m.sel + map[string]int{"up": nRows - 1, "down": 1}[k]) % nRows
	case "left", "right":
		if m.fixed(m.sel) == "" && (m.sel < rowInterval || m.ctl != nil) {
			return m.change(map[string]int{"left": -1, "right": 1}[k])
		}
	}
	return m, nil
}

// change steps the selected row's value by one, wrapping, and applies it.
func (m Model) change(by int) (Model, tea.Cmd) {
	switch m.sel {
	case rowLayout:
		m.lay = (m.lay + by + len(layouts)) % len(layouts)
	case rowPalette:
		m.palette = map[string]string{"ansi": "tidemark", "tidemark": "ansi"}[m.palette]
		setPalette(m.palette, m.dark)
	case rowInterval:
		i := 2 // an interval off the steps (a hand-edited config) starts from 1000
		for j, v := range intervals {
			if v == m.interval {
				i = j
			}
		}
		m.interval = intervals[(i+by+len(intervals))%len(intervals)]
		m.saved.Interval = m.interval
		ch := m.ctl.Restart(m.interval)
		if ch == nil {
			return m, tea.Quit // main reports why
		}
		m.samples = ch
		return m, m.wait
	case rowStore:
		on := m.store == nil
		m.saved.Store = map[bool]string{true: "", false: "off"}[on]
		m.store, m.storeDir = nil, ""
		if r := m.ctl.Store(on); r != nil {
			m = m.Store(r)
		}
	}
	return m, nil
}

// menuBox draws the menu w columns wide: 36, or all of w below 38; short screens (under 11 rows)
// drop the blank line and the key hints.
func (m Model) menuBox(rows, w int) []string {
	bw := 36
	if w < 38 {
		bw = w
	}
	iw := bw - 2
	edge := func(s string) string { return dim.Render("║") + pad(s, max(iw, 0)) + dim.Render("║") }
	out := []string{dim.Render("╔═ ") + title.Render("settings") + dim.Render(" "+rep("═", max(iw-11, 0))+"╗")}
	for row := range nRows {
		cur, arrow := " ", dim
		if row == m.sel {
			cur, arrow = mid.Render("▸"), mid
		}
		name := fit(rowNames[row], 11)
		var l string
		if why := m.fixed(row); why != "" {
			l = dim.Render("   " + name + "  " + m.rowValue(row) + " " + why)
			if row == m.sel {
				l = " " + cur + dim.Render(" "+name+"  "+m.rowValue(row)+" "+why)
			}
		} else {
			l = " " + cur + " " + name + arrow.Render("◂ ") + m.rowValue(row) + arrow.Render(" ▸")
		}
		out = append(out, edge(l))
	}
	if rows >= 11 {
		out = append(out, edge(""), edge(dim.Render(" ↑↓ move  ←→ change  o close")))
	}
	out = append(out, dim.Render("╚"+rep("═", max(iw, 0))+"╝"))
	for i := range out {
		out[i] = ansi.Truncate(out[i], bw, "")
	}
	return out
}

// overlay draws the menu centred over the layout's lines, which stay visible around it.
func (m Model) overlay(out []string, w int) []string {
	box := m.menuBox(len(out), w)
	bw := ansi.StringWidth(box[0])
	x, y := max((w-bw)/2, 0), max((len(out)-len(box))/2, 0)
	for i, b := range box {
		if y+i >= len(out) {
			break
		}
		l := pad(out[y+i], w)
		out[y+i] = ansi.Truncate(l, x, "") + b + ansi.TruncateLeft(l, x+bw, "")
	}
	return out
}
