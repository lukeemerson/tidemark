package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lukeemerson/tidemark/internal/config"
	"github.com/lukeemerson/tidemark/internal/source"
)

type ctlStub struct {
	restarts []int
	ch       chan source.Sample
	dir      string
}

func (c *ctlStub) Restart(ms int) <-chan source.Sample {
	c.restarts = append(c.restarts, ms)
	return c.ch
}

func (c *ctlStub) Store(on bool) Recorder {
	if on {
		return dirStub{c.dir}
	}
	return nil
}

func press(m Model, k string) (Model, tea.Cmd) {
	code := map[string]rune{"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight, "esc": tea.KeyEscape}[k]
	msg := tea.KeyPressMsg{Code: code}
	if code == 0 {
		msg = tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
	}
	mm, cmd := m.Update(msg)
	return mm.(Model), cmd
}

func menuModel(t *testing.T, s Setup, saved *config.Config) Model {
	m := New(nil, nil, "", "", func(c config.Config) { *saved = c })
	return feed(m.Settings(s), recording(t)[:5])
}

func TestMenuChangesAndSavesOnClose(t *testing.T) {
	var saved config.Config
	ctl := &ctlStub{ch: make(chan source.Sample), dir: "x"}
	m := menuModel(t, Setup{Ctl: ctl, Interval: 1000}, &saved)
	m, _ = press(m, "o")
	if !m.menu {
		t.Fatal("o didn't open the menu")
	}
	m, _ = press(m, "l") // other keys do nothing while it's open
	m, _ = press(m, "left")
	if m.lay != len(layouts)-1 {
		t.Errorf("layout ← from tiles = %d, want it to wrap to the last", m.lay)
	}
	m, _ = press(m, "down")
	m, _ = press(m, "right")
	if m.palette != "tidemark" {
		t.Errorf("palette = %q", m.palette)
	}
	m, _ = press(m, "down")
	m, cmd := press(m, "right")
	if m.interval != 2000 || len(ctl.restarts) != 1 || ctl.restarts[0] != 2000 || m.samples != (<-chan source.Sample)(ctl.ch) || cmd == nil {
		t.Errorf("interval → : %d ms, restarts %v, new channel %v", m.interval, ctl.restarts, m.samples == (<-chan source.Sample)(ctl.ch))
	}
	m, _ = press(m, "down")
	if m, _ = press(m, "right"); m.store == nil || m.storeDir != "x" {
		t.Error("store → on didn't open the store")
	}
	if m, _ = press(m, "right"); m.store != nil || m.storeDir != "" {
		t.Error("store → off didn't close it")
	}
	if saved != (config.Config{}) {
		t.Errorf("saved before closing: %+v", saved)
	}
	m, cmd = press(m, "esc")
	if m.menu || cmd == nil {
		t.Fatal("esc didn't close and save")
	}
	cmd()
	want := config.Config{Layout: layouts[len(layouts)-1].name, Palette: "tidemark", Interval: 2000, Store: "off"}
	if saved != want {
		t.Errorf("saved %+v, want %+v", saved, want)
	}
}

func TestMenuLocks(t *testing.T) {
	var saved config.Config
	ctl := &ctlStub{}
	m := menuModel(t, Setup{Ctl: ctl, Interval: 2000, IntervalFlag: true, StoreFlag: true}, &saved)
	m, _ = press(m, "o")
	m.sel = rowInterval
	m, _ = press(m, "right")
	m.sel = rowStore
	m, _ = press(m, "right")
	if len(ctl.restarts) != 0 || m.store != nil {
		t.Error("locked rows changed")
	}
	box := ansi.Strip(strings.Join(m.menuBox(42, 138), "\n"))
	for _, want := range []string{"interval     2000 ms (-i)", "store        off (-nostore)"} {
		if !strings.Contains(box, want) {
			t.Errorf("menu missing %q:\n%s", want, box)
		}
	}

	all := recording(t)
	r := feed(New(nil, nil, "", "", nil).Replay(len(all), all[0]), all[:5]).Settings(Setup{})
	if box := ansi.Strip(strings.Join(r.menuBox(42, 138), "\n")); strings.Count(box, "— (replay)") != 2 {
		t.Errorf("replay menu:\n%s", box)
	}
}

func TestMenuBox(t *testing.T) {
	m := menuModel(t, Setup{Interval: 1000}, new(config.Config))
	for _, c := range []struct{ rows, w, h, bw int }{{42, 138, 8, 36}, {10, 138, 6, 36}, {20, 30, 8, 30}} {
		box := m.menuBox(c.rows, c.w)
		if len(box) != c.h {
			t.Errorf("%dx%d: %d lines, want %d", c.w, c.rows, len(box), c.h)
		}
		for _, l := range box {
			if n := ansi.StringWidth(l); n != c.bw {
				t.Errorf("%dx%d: line %q is %d wide, want %d", c.w, c.rows, ansi.Strip(l), n, c.bw)
			}
		}
	}
	if got := ansi.Strip(m.menuBox(42, 138)[1]); got != "║ ▸ layout     ◂ tiles ▸           ║" {
		t.Errorf("first row %q", got)
	}
}

// After a restart, the old mactop's last sample and its closing channel are dropped, not taken as
// the end of the program.
func TestStaleChannelIgnored(t *testing.T) {
	old := make(chan source.Sample)
	m := New(old, nil, "", "", nil)
	m.samples = make(chan source.Sample)
	s := recording(t)[0]
	mm, _ := m.Update(fromMsg{old, sampleMsg(s)})
	if mm.(Model).seq != 0 {
		t.Error("a stale sample was taken")
	}
	if _, cmd := m.Update(fromMsg{old, doneMsg{}}); cmd != nil {
		t.Error("the old channel closing ended the program")
	}
	if mm, _ := m.Update(fromMsg{m.samples, sampleMsg(s)}); mm.(Model).seq != 1 {
		t.Error("a current sample was dropped")
	}
}
