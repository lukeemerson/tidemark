package source

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

// Run is one saved cloudflare-speed-cli run, limited to the fields monitor draws.
type Run struct {
	Timestamp time.Time `json:"timestamp_utc"`
	Download  struct {
		Mbps float64 `json:"mbps"`
	} `json:"download"`
	Upload struct {
		Mbps float64 `json:"mbps"`
	} `json:"upload"`
	IdleLatency struct {
		MedianMs float64 `json:"median_ms"`
		JitterMs float64 `json:"jitter_ms"`
		Loss     float64 `json:"loss"`
	} `json:"idle_latency"`
	LoadedDownload struct {
		MedianMs float64 `json:"median_ms"`
	} `json:"loaded_latency_download"`
	LoadedUpload struct {
		MedianMs float64 `json:"median_ms"`
	} `json:"loaded_latency_upload"`
	Quality struct {
		Bufferbloat string `json:"bufferbloat_grade"`
		Stability   string `json:"stability_grade"`
	} `json:"connection_quality"`
	Meta struct {
		Colo struct {
			IATA string `json:"iata"`
		} `json:"colo"`
		ASOrg string `json:"asOrganization"`
	} `json:"meta"`
	Colo     string `json:"colo"`   // fallback for meta.colo.iata
	ASOrg    string `json:"as_org"` // fallback for meta.asOrganization
	Wireless bool   `json:"is_wireless"`
}

// CloudyRuns reads the newest (by mtime) max saved runs from dir, oldest first. No test is run.
func CloudyRuns(dir string, max int) []Run {
	files, _ := filepath.Glob(filepath.Join(dir, "run-*.json"))
	mtime := map[string]time.Time{}
	for _, f := range files {
		if st, err := os.Stat(f); err == nil {
			mtime[f] = st.ModTime()
		}
	}
	sort.Slice(files, func(i, j int) bool { return mtime[files[i]].After(mtime[files[j]]) })
	if len(files) > max {
		files = files[:max]
	}
	var runs []Run
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var r Run
		if json.Unmarshal(b, &r) == nil {
			runs = append(runs, r)
		}
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].Timestamp.Before(runs[j].Timestamp) })
	return runs
}

// CloudyDir is where cloudflare-speed-cli saves runs.
func CloudyDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "cloudflare-speed-cli", "runs")
}

// StartCloudy starts one cloudflare-speed-cli test (~30s); it saves its run to CloudyDir itself.
func StartCloudy() (*exec.Cmd, error) {
	cmd := exec.Command("cloudflare-speed-cli", "--silent", "--json") // --silent requires --json; the run is read back from disk
	return cmd, cmd.Start()
}
