package source

import (
	"os"
	"path/filepath"
	"testing"
)

// testdata/mactop.raw is 90s of real mactop --headless -i 1000 output.
func TestDecodeRecording(t *testing.T) {
	f, err := os.Open("testdata/mactop.raw")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ch := make(chan Sample)
	errc := make(chan error, 1)
	go func() { errc <- Decode(f, ch); close(ch) }()
	var got []Sample
	for s := range ch {
		got = append(got, s)
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	if len(got) < 80 {
		t.Fatalf("decoded %d samples, want ~90", len(got))
	}
	s := got[0]
	if s.SystemInfo.Name != "Apple M2 Pro" || s.SystemInfo.ECoreCount != 4 || s.SystemInfo.PCoreCount != 6 {
		t.Errorf("system_info = %+v", s.SystemInfo)
	}
	if len(s.CoreUsages) != 10 || s.Memory.Total == 0 || s.SoC.TotalPower == 0 || len(s.Processes) == 0 {
		t.Errorf("missing fields: cores=%d mem=%v power=%v procs=%d",
			len(s.CoreUsages), s.Memory.Total, s.SoC.TotalPower, len(s.Processes))
	}
	if !s.Battery.Present || s.Battery.Percent != 80 {
		t.Errorf("battery = %+v", s.Battery)
	}
}

func TestCloudyRunsOrder(t *testing.T) {
	runs := CloudyRuns(CloudyDir(), 40)
	if len(runs) == 0 {
		t.Skip("no saved cloudy runs on this machine")
	}
	for i := 1; i < len(runs); i++ {
		if runs[i].Timestamp.Before(runs[i-1].Timestamp) {
			t.Fatalf("runs not oldest-first at %d", i)
		}
	}
	if last := runs[len(runs)-1]; last.Download.Mbps == 0 || last.IdleLatency.MedianMs == 0 {
		t.Errorf("latest run missing speeds: %+v", last)
	}
}

func TestProcNames(t *testing.T) {
	n := ProcNames{}
	if got := n.Name(1, "launchd"); got != "launchd" {
		t.Errorf("non-version command changed: %q", got)
	}
	// this test binary's own argv[0]
	if got, want := n.Name(os.Getpid(), "1.2.3"), filepath.Base(os.Args[0]); got != want {
		t.Errorf("argv0 = %q, want %q", got, want)
	}
	if got := n.Name(999999, "2.1.283"); got != "2.1.283" {
		t.Errorf("dead pid should fall back, got %q", got)
	}
}
