package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/lukeemerson/mac-monitor/internal/source"
	"github.com/lukeemerson/mac-monitor/internal/ui"
)

func main() {
	interval := flag.Int("i", 1000, "mactop sample interval (ms)")
	flag.Parse()

	samples, stop, err := source.Mactop(*interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, "monitor:", err)
		os.Exit(1)
	}
	m := ui.New(samples, source.CloudyRuns(source.CloudyDir(), 40))
	_, err = tea.NewProgram(m).Run()
	stop()
	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		fmt.Fprintln(os.Stderr, "monitor:", err)
		os.Exit(1)
	}
}
