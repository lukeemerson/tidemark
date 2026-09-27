package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/lukeemerson/tidemark/internal/config"
	"github.com/lukeemerson/tidemark/internal/source"
	"github.com/lukeemerson/tidemark/internal/ui"
)

func main() {
	interval := flag.Int("i", 1000, "mactop sample interval (ms)")
	flag.Parse()

	col, err := source.Mactop(*interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tidemark:", err)
		os.Exit(1)
	}
	cfg := config.Load()
	save := func(layout string) {
		cfg.Layout = layout
		config.Save(cfg)
	}
	m := ui.New(col.Samples, source.CloudyRuns(source.CloudyDir(), 40), cfg.Layout, save)
	final, err := tea.NewProgram(m).Run()
	cerr := col.Err() // read before Stop: only set if mactop ended on its own
	col.Stop()
	if fm, ok := final.(ui.Model); ok {
		fm.Stop()
	}
	if err == nil || errors.Is(err, tea.ErrProgramKilled) {
		err = cerr
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tidemark:", err)
		os.Exit(1)
	}
}
