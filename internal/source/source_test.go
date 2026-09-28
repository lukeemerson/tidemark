package source

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestReadSys(t *testing.T) {
	s := ReadSys()
	if s.Load[0] <= 0 || s.Load[0] > 1000 {
		t.Errorf("load = %v", s.Load)
	}
	if s.Pressure != 1 && s.Pressure != 2 && s.Pressure != 4 {
		t.Errorf("pressure level = %d", s.Pressure)
	}
	if s.FreePct <= 0 || s.FreePct > 100 {
		t.Errorf("free %% = %d", s.FreePct)
	}
	t.Logf("%+v", s)
}

// -rec copies mactop's stream byte for byte, and Play reads it back at the recorded spacing.
func TestRecordThenPlay(t *testing.T) {
	dir := t.TempDir()
	raw, _ := filepath.Abs("testdata/mactop.raw")
	fake := filepath.Join(dir, "mactop")
	os.WriteFile(fake, []byte("#!/bin/sh\ncat '"+raw+"'\n"), 0o755)
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	rec := filepath.Join(dir, "rec.raw")
	f, err := os.Create(rec)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Mactop(1000, f)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for range c.Samples {
		n++
	}
	f.Close()
	want, _ := os.ReadFile(raw)
	got, _ := os.ReadFile(rec)
	if n != 90 || string(got) != string(want) {
		t.Fatalf("recorded %d samples, %d of %d bytes match", n, len(got), len(want))
	}
	var waits []time.Duration
	ch, total, first, err := Play(rec, func(d time.Duration) { waits = append(waits, d) })
	if err != nil {
		t.Fatal(err)
	}
	played := 0
	for s := range ch {
		if s.Timestamp.IsZero() {
			t.Fatalf("sample %d has no timestamp", played)
		}
		played++
	}
	if total != 90 || played != 90 || len(waits) != 89 || first.Timestamp.Format("2006-01-02 15:04") != "2026-09-26 19:44" {
		t.Errorf("total=%d played=%d waits=%d recorded=%v", total, played, len(waits), first.Timestamp)
	}
	for _, d := range waits {
		if d <= 0 || d > 10*time.Second {
			t.Errorf("odd gap %v between recorded samples", d)
			break
		}
	}
}

// A recording cut off mid-sample (quit or kill during a write) still plays everything before
// the cut (review's finding on bb97cde).
func TestPlayTruncatedRecording(t *testing.T) {
	raw, _ := os.ReadFile("testdata/mactop.raw")
	cut := filepath.Join(t.TempDir(), "cut.raw")
	os.WriteFile(cut, raw[:len(raw)-700], 0o644)
	ch, total, _, err := Play(cut, func(time.Duration) {})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for range ch {
		n++
	}
	if total != 89 || n != 89 {
		t.Errorf("total=%d played=%d, want 89", total, n)
	}
}
