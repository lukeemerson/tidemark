// Package store keeps 24 hours of history on disk (STORE-SPEC.md): mactop's raw lines in one
// file per clock hour (the current and previous hour are kept), and 10 s summary buckets in one
// file per day (24 h kept). A second tidemark finds the directory locked and doesn't write.
package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/lukeemerson/tidemark/internal/source"
)

const (
	BucketLen   = 10 * time.Second
	SummarySpan = 24 * time.Hour
	rawLayout   = "20060102-15"
	sumLayout   = "20060102"
)

// ErrLocked means another tidemark owns the store; this one runs without writing.
var ErrLocked = errors.New("history store is in use by another tidemark")

// Dir is where the store lives.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "tidemark", "history")
}

// Store writes both tiers. Write takes mactop's raw stream (it is teed next to -rec); Add takes
// each decoded sample for the summary. Neither ever returns an error to the caller: a full disk
// or a vanished directory stops storing, never monitoring.
type Store struct {
	dir  string
	now  func() time.Time
	lock *os.File

	raw     *os.File
	rawHour string
	partial []byte // raw bytes after the last newline, until the line completes

	b         *bucket
	lastPrune string // the hour of the last prune, as rawLayout
	broken    bool
}

// Open takes the store's lock and prunes it. now is time.Now in the app.
func Open(dir string, now func() time.Time) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}
	s := &Store{dir: dir, now: now, lock: lock}
	s.prune(now())
	return s, nil
}

// Write appends each complete line of mactop's stream to the file for the current clock hour.
func (s *Store) Write(p []byte) (int, error) {
	if s.broken {
		return len(p), nil
	}
	data := append(s.partial, p...)
	i := bytes.LastIndexByte(data, '\n')
	if i < 0 {
		s.partial = data
		return len(p), nil
	}
	lines, rest := data[:i+1], data[i+1:]
	s.partial = append([]byte(nil), rest...)

	now := s.now()
	if hour := now.Format(rawLayout); hour != s.rawHour || s.raw == nil {
		if s.raw != nil {
			s.raw.Close()
		}
		f, err := os.OpenFile(filepath.Join(s.dir, "raw-"+hour+".raw"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			s.broken = true
			return len(p), nil
		}
		s.raw, s.rawHour = f, hour
		if hour != s.lastPrune {
			s.prune(now)
		}
	}
	if _, err := s.raw.Write(lines); err != nil {
		s.broken = true
	}
	return len(p), nil
}

// Add folds one sample into its 10 s bucket, writing the previous bucket when a new one starts.
// fired says a diagnosis rule started firing on this sample; name resolves process names.
func (s *Store) Add(smp source.Sample, sys source.Sys, sysOK, fired bool, name func(int, string) string) {
	start := smp.Timestamp.Truncate(BucketLen)
	if s.b != nil && !s.b.t.Equal(start) {
		s.flush()
	}
	if s.b == nil {
		s.b = newBucket(start)
	}
	s.b.add(smp, sys, sysOK, fired, name)
}

// Close writes the open bucket and releases the lock.
func (s *Store) Close() {
	s.flush()
	if s.raw != nil {
		s.raw.Close()
	}
	s.lock.Close()
}

func (s *Store) flush() {
	if s.b == nil || s.broken {
		s.b = nil
		return
	}
	day := s.b.t.Local().Format(sumLayout) // the bucket's own day, not today's
	line, err := json.Marshal(s.b.close())
	s.b = nil
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(s.dir, "sum-"+day+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		s.broken = true
		return
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		s.broken = true
	}
}

// prune keeps the current and previous raw hours, and summary lines from the last 24 h.
func (s *Store) prune(now time.Time) {
	s.lastPrune = now.Format(rawLayout)
	keep := map[string]bool{
		"raw-" + now.Format(rawLayout) + ".raw":                     true,
		"raw-" + now.Add(-time.Hour).Format(rawLayout) + ".raw":     true,
		"sum-" + now.Format(sumLayout) + ".jsonl":                   true,
		"sum-" + now.AddDate(0, 0, -1).Format(sumLayout) + ".jsonl": true,
	}
	entries, _ := os.ReadDir(s.dir)
	for _, e := range entries {
		n := e.Name()
		isRaw := strings.HasPrefix(n, "raw-") && strings.HasSuffix(n, ".raw")
		isSum := strings.HasPrefix(n, "sum-") && strings.HasSuffix(n, ".jsonl")
		switch {
		case (isRaw || isSum) && !keep[n]:
			os.Remove(filepath.Join(s.dir, n))
		case isSum:
			dropOlder(filepath.Join(s.dir, n), now.Add(-SummarySpan))
		}
	}
}

// dropOlder rewrites a summary file without lines older than cutoff (or unreadable ones, such
// as a line cut off mid-write). The rewrite goes through a temp file and a rename.
func dropOlder(path string, cutoff time.Time) {
	lines, dropped := readLines(path, cutoff)
	if !dropped {
		return
	}
	tmp := path + ".tmp"
	var buf bytes.Buffer
	for _, l := range lines {
		buf.Write(l)
		buf.WriteByte('\n')
	}
	if os.WriteFile(tmp, buf.Bytes(), 0o644) == nil {
		os.Rename(tmp, path)
	}
}

// readLines returns a summary file's parseable lines at or after cutoff, and whether any were
// dropped.
func readLines(path string, cutoff time.Time) (keep [][]byte, dropped bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		var b struct {
			T time.Time `json:"t"`
		}
		if json.Unmarshal(sc.Bytes(), &b) != nil || b.T.Before(cutoff) {
			dropped = true
			continue
		}
		keep = append(keep, append([]byte(nil), sc.Bytes()...))
	}
	return keep, dropped
}

// Dir is the store's directory, for reading the tiers back.
func (s *Store) Dir() string { return s.dir }

// Raw reads the full tier: every sample from the hour before now, oldest first, from the current
// and previous hour files. Unreadable lines (a line cut off mid-write) are skipped.
func Raw(dir string, now time.Time) []source.Sample {
	var out []source.Sample
	for _, h := range []time.Time{now.Add(-time.Hour), now} {
		f, err := os.Open(filepath.Join(dir, "raw-"+h.Format(rawLayout)+".raw"))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
		for sc.Scan() {
			line := bytes.TrimRight(bytes.TrimLeft(sc.Bytes(), "[,"), ",]")
			var s source.Sample
			if len(line) == 0 || line[0] != '{' || json.Unmarshal(line, &s) != nil {
				continue
			}
			if !s.Timestamp.Before(now.Add(-time.Hour)) && !s.Timestamp.After(now) {
				out = append(out, s)
			}
		}
		f.Close()
	}
	return out
}

// Summary reads the summary tier from dir: buckets from the last 24 h before now, oldest first.
// Unreadable lines (a line cut off mid-write) are skipped.
func Summary(dir string, now time.Time) []Bucket {
	var out []Bucket
	for _, day := range []time.Time{now.AddDate(0, 0, -1), now} {
		lines, _ := readLines(filepath.Join(dir, "sum-"+day.Format(sumLayout)+".jsonl"), now.Add(-SummarySpan))
		for _, l := range lines {
			var b Bucket
			if json.Unmarshal(l, &b) == nil {
				out = append(out, b)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].T.Before(out[j].T) })
	return out
}
