package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lukeemerson/tidemark/internal/config"
	"github.com/lukeemerson/tidemark/internal/source"
	"github.com/lukeemerson/tidemark/internal/store"
	"github.com/lukeemerson/tidemark/internal/ui"
)

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tidemark:", err)
	os.Exit(1)
}

// tee takes mactop's raw stream for the -rec file and the store; the store can be swapped while
// mactop runs (the settings menu), so writes and swaps share a lock.
type tee struct {
	mu  sync.Mutex
	rec io.Writer
	st  *store.Store
}

func (t *tee) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.rec != nil {
		if _, err := t.rec.Write(p); err != nil {
			return 0, err
		}
	}
	if t.st != nil {
		t.st.Write(p)
	}
	return len(p), nil
}

// runner owns mactop and the store for the settings menu (ui.Control).
type runner struct {
	col *source.Collector
	tee *tee
	err error // why a restart failed
}

func (r *runner) Restart(ms int) <-chan source.Sample {
	r.col.Stop()
	for range r.col.Samples { // until the old mactop's last raw bytes are through the tee
	}
	col, err := source.Mactop(ms, r.tee)
	if err != nil {
		r.err = err
		return nil
	}
	r.col = col
	return col.Samples
}

func (r *runner) Store(on bool) ui.Recorder {
	r.tee.mu.Lock()
	defer r.tee.mu.Unlock()
	if !on {
		if r.tee.st != nil {
			r.tee.st.Close()
			r.tee.st = nil
		}
		return nil
	}
	if r.tee.st == nil {
		// another tidemark owning the store (ErrLocked), or an unwritable directory, just
		// means this one doesn't store
		st, err := store.Open(store.Dir(), time.Now)
		if err != nil {
			return nil
		}
		r.tee.st = st
	}
	return r.tee.st
}

func main() {
	interval := flag.Int("i", 1000, "mactop sample interval (ms); overrides the saved one")
	rec := flag.String("rec", "", "also record mactop's samples to `file`")
	play := flag.String("play", "", "replay a recording from `file` instead of running mactop")
	nostore := flag.Bool("nostore", false, "don't keep history in ~/Library/Application Support/tidemark")
	flag.Parse()
	if *rec != "" && *play != "" {
		fail(errors.New("-rec and -play can't be used together"))
	}
	iflag := false
	flag.Visit(func(f *flag.Flag) { iflag = iflag || f.Name == "i" })

	cfg := config.Load()
	save := func(c config.Config) { config.Save(c) }
	runs := source.CloudyRuns(source.CloudyDir(), 40)

	var m ui.Model
	var r *runner
	var recf *os.File
	if *play != "" {
		samples, total, first, err := source.Play(*play, time.Sleep)
		if err != nil {
			fail(err)
		}
		m = ui.New(samples, runs, cfg.Layout, cfg.Palette, save).Replay(total, first)
		m = m.Settings(ui.Setup{Saved: cfg})
	} else {
		ms := *interval
		if !iflag && cfg.Interval > 0 {
			ms = cfg.Interval
		}
		r = &runner{tee: &tee{}}
		if *rec != "" {
			var err error
			if recf, err = os.Create(*rec); err != nil {
				fail(err)
			}
			r.tee.rec = recf
		}
		var st ui.Recorder
		if !*nostore && cfg.Store != "off" {
			st = r.Store(true)
		}
		var err error
		if r.col, err = source.Mactop(ms, r.tee); err != nil {
			fail(err)
		}
		m = ui.New(r.col.Samples, runs, cfg.Layout, cfg.Palette, save)
		m = m.Settings(ui.Setup{Ctl: r, Interval: ms, Store: st, IntervalFlag: iflag, StoreFlag: *nostore, Saved: cfg})
	}

	final, err := tea.NewProgram(m).Run()
	var cerr error
	if r != nil {
		cerr = r.col.Err() // read before Stop: only set if mactop ended on its own
		if r.err != nil {
			cerr = r.err
		}
		r.col.Stop()
		r.Store(false)
	}
	if fm, ok := final.(ui.Model); ok {
		fm.Stop()
	}
	if recf != nil {
		recf.Close()
	}
	if err == nil || errors.Is(err, tea.ErrProgramKilled) {
		err = cerr
	}
	if err != nil {
		fail(err)
	}
}
