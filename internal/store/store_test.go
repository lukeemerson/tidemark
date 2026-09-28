package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lukeemerson/tidemark/internal/source"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

var t0 = time.Date(2026, 9, 27, 13, 59, 58, 0, time.Local)

func TestSecondInstanceIsLockedOut(t *testing.T) {
	dir := t.TempDir()
	c := &clock{t0}
	a, err := Open(dir, c.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir, c.now); !errors.Is(err, ErrLocked) {
		t.Errorf("second Open: err = %v, want ErrLocked", err)
	}
	a.Close()
	b, err := Open(dir, c.now)
	if err != nil {
		t.Errorf("after the first closes, Open should succeed: %v", err)
	}
	b.Close()
}

// Lines arrive in arbitrary chunks; each lands whole in its clock hour's file, and each hour
// file plays back as a recording.
func TestRawHourFiles(t *testing.T) {
	dir := t.TempDir()
	c := &clock{t0}
	s, _ := Open(dir, c.now)
	raw, _ := os.ReadFile("../source/testdata/mactop.raw")
	lines := strings.SplitAfter(string(raw), "\n")[:6]
	s.Write([]byte(lines[0] + lines[1] + lines[2][:40])) // a line split across writes
	s.Write([]byte(lines[2][40:]))
	c.t = t0.Add(3 * time.Second) // 14:00:01, the next hour
	s.Write([]byte(lines[3] + lines[4] + lines[5]))
	s.Close()

	for file, want := range map[string]int{"raw-20260927-13.raw": 3, "raw-20260927-14.raw": 3} {
		ch, total, _, err := source.Play(filepath.Join(dir, file), func(time.Duration) {})
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for range ch {
		}
		if total != want {
			t.Errorf("%s: %d samples, want %d", file, total, want)
		}
	}
}

func sample(ts time.Time, cpu float64, procs ...source.Process) source.Sample {
	s := source.Sample{Timestamp: ts, CPUUsage: cpu, CoreUsages: []float64{cpu, cpu / 2}, Processes: procs}
	s.Memory.Total, s.Memory.Used = 100, 40
	return s
}

func TestSummaryBuckets(t *testing.T) {
	dir := t.TempDir()
	c := &clock{t0}
	s, _ := Open(dir, c.now)
	name := func(_ int, cmd string) string { return cmd }
	base := time.Date(2026, 9, 27, 14, 0, 0, 0, time.Local)
	for i := 0; i < 25; i++ { // buckets 14:00:00, :10 and :20 (the last still open until Close)
		p := source.Process{PID: 100 + i%7, Command: "p", CPUPercent: float64(i)}
		s.Add(sample(base.Add(time.Duration(i)*time.Second), float64(i), p), source.Sys{Load: [3]float64{2, 1, 1}, Pressure: 1, FreePct: 60}, true, i == 12, name)
	}
	s.Close()
	bs := Summary(dir, base.Add(time.Minute))
	if len(bs) != 3 {
		t.Fatalf("%d buckets, want 3", len(bs))
	}
	b := bs[0]
	if b.N != 10 || b.Avg.CPUUsage != 4.5 || b.Peak["cpu"] != 9 || b.Avg.CoreUsages[1] != 2.25 || b.Peak["mem"] != 40 {
		t.Errorf("first bucket: n=%d avg=%v peak=%v cores=%v memPeak=%v", b.N, b.Avg.CPUUsage, b.Peak["cpu"], b.Avg.CoreUsages, b.Peak["mem"])
	}
	if !b.SysOK || b.Load[0] != 2 || b.Pressure != 1 || b.Free != 60 {
		t.Errorf("first bucket sys: ok=%v load=%v pressure=%d free=%d", b.SysOK, b.Load, b.Pressure, b.Free)
	}
	if len(b.Top) != 5 || b.Top[0].CPU != 9 || b.Top[0].PID != 102 {
		t.Errorf("top 5 = %+v", b.Top)
	}
	if bs[0].Alert || !bs[1].Alert || bs[2].Alert {
		t.Errorf("alert flags = %v %v %v, want only the second (sample 12)", bs[0].Alert, bs[1].Alert, bs[2].Alert)
	}
	if bs[2].N != 5 {
		t.Errorf("Close should write the open bucket: n=%d", bs[2].N)
	}
}

// Startup pruning keeps the current and previous raw hour and 24 h of summary; a line cut off
// mid-write is dropped.
func TestPrune(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 27, 14, 30, 0, 0, time.Local)
	for _, h := range []string{"10", "12", "13", "14"} {
		os.WriteFile(filepath.Join(dir, "raw-20260927-"+h+".raw"), []byte("x\n"), 0o644)
	}
	line := func(t time.Time) string { return `{"t":"` + t.Format(time.RFC3339) + `","n":1}` + "\n" }
	os.WriteFile(filepath.Join(dir, "sum-20260925.jsonl"), []byte(line(now.Add(-50*time.Hour))), 0o644)
	os.WriteFile(filepath.Join(dir, "sum-20260926.jsonl"),
		[]byte(line(now.Add(-25*time.Hour))+line(now.Add(-23*time.Hour))), 0o644)
	os.WriteFile(filepath.Join(dir, "sum-20260927.jsonl"), []byte(line(now.Add(-time.Hour))+`{"t":"2026-09-27T14:2`), 0o644)

	s, err := Open(dir, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	var names []string
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	if got, want := strings.Join(names, " "), "raw-20260927-13.raw raw-20260927-14.raw sum-20260926.jsonl sum-20260927.jsonl"; got != want {
		t.Errorf("after prune: %s\nwant: %s", got, want)
	}
	if bs := Summary(dir, now); len(bs) != 2 {
		t.Errorf("%d buckets in the last 24 h, want 2 (the 25 h one and the cut-off line dropped)", len(bs))
	}
}
