package source

import (
	"bytes"
	"fmt"
	"os"
	"time"
)

// Play replays a recording made with -rec (mactop's own --headless stream) at its recorded
// pace: wait is called with the gap between consecutive timestamps (time.Sleep in the app, a
// no-op in tests). It also returns how many samples the file holds and the first one, which says
// when the recording was made and on which machine.
func Play(path string, wait func(time.Duration)) (samples <-chan Sample, total int, first Sample, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, Sample{}, err
	}
	all := make(chan Sample)
	errc := make(chan error, 1)
	go func() { errc <- Decode(bytes.NewReader(b), all); close(all) }()
	var list []Sample
	for s := range all {
		list = append(list, s)
	}
	// A recording can end mid-sample (tidemark quit or was killed while mactop was writing a
	// line), so a bad line only matters if nothing before it decoded.
	if err := <-errc; err != nil && len(list) == 0 {
		return nil, 0, Sample{}, fmt.Errorf("%s: %w", path, err)
	}
	if len(list) == 0 {
		return nil, 0, Sample{}, fmt.Errorf("%s: no mactop samples", path)
	}
	out := make(chan Sample)
	go func() {
		defer close(out)
		for i, s := range list {
			if i > 0 {
				d := s.Timestamp.Sub(list[i-1].Timestamp)
				if d <= 0 || d > 10*time.Second { // missing or odd timestamps: fall back to 1 s
					d = time.Second
				}
				wait(d)
			}
			out <- s
		}
	}()
	return out, len(list), list[0], nil
}
