package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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

func main() {
	interval := flag.Int("i", 1000, "mactop sample interval (ms)")
	rec := flag.String("rec", "", "also record mactop's samples to `file`")
	play := flag.String("play", "", "replay a recording from `file` instead of running mactop")
	nostore := flag.Bool("nostore", false, "don't keep history in ~/Library/Application Support/tidemark")
	flag.Parse()
	if *rec != "" && *play != "" {
		fail(errors.New("-rec and -play can't be used together"))
	}

	cfg := config.Load()
	save := func(layout, palette string) {
		cfg.Layout, cfg.Palette = layout, palette
		config.Save(cfg)
	}
	runs := source.CloudyRuns(source.CloudyDir(), 40)

	var m ui.Model
	var col *source.Collector
	var st *store.Store
	var recf *os.File
	if *play != "" {
		samples, total, first, err := source.Play(*play, time.Sleep)
		if err != nil {
			fail(err)
		}
		m = ui.New(samples, runs, cfg.Layout, cfg.Palette, save).Replay(total, first)
	} else {
		var tees []io.Writer
		if *rec != "" {
			var err error
			if recf, err = os.Create(*rec); err != nil {
				fail(err)
			}
			tees = append(tees, recf)
		}
		if !*nostore {
			// another tidemark owning the store (ErrLocked), or an unwritable directory, just
			// means this one doesn't store
			if s, err := store.Open(store.Dir(), time.Now); err == nil {
				st = s
				tees = append(tees, st)
			}
		}
		var tee io.Writer
		if len(tees) > 0 {
			tee = io.MultiWriter(tees...)
		}
		var err error
		if col, err = source.Mactop(*interval, tee); err != nil {
			fail(err)
		}
		m = ui.New(col.Samples, runs, cfg.Layout, cfg.Palette, save)
		if st != nil {
			m = m.Store(st)
		}
	}

	final, err := tea.NewProgram(m).Run()
	var cerr error
	if col != nil {
		cerr = col.Err() // read before Stop: only set if mactop ended on its own
		col.Stop()
	}
	if fm, ok := final.(ui.Model); ok {
		fm.Stop()
	}
	if st != nil {
		st.Close()
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
