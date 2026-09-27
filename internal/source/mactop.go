// Package source reads the data monitor draws: mactop's headless JSON stream and cloudy's saved runs.
package source

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strconv"
)

// Sample is one mactop --headless sample, limited to the fields monitor draws.
type Sample struct {
	CPUUsage     float64   `json:"cpu_usage"`
	GPUUsage     float64   `json:"gpu_usage"`
	CoreUsages   []float64 `json:"core_usages"`
	ThermalState string    `json:"thermal_state"`
	SoC          struct {
		TotalPower  float64 `json:"total_power"`
		SystemPower float64 `json:"system_power"`
		GPUPower    float64 `json:"gpu_power"`
		EFreqMHz    float64 `json:"e_cluster_freq_mhz"`
		PFreqMHz    float64 `json:"p_cluster_freq_mhz"`
		GPUFreqMHz  float64 `json:"gpu_freq_mhz"`
		ANEActive   float64 `json:"ane_active"`
		CPUTemp     float64 `json:"cpu_temp"`
		GPUTemp     float64 `json:"gpu_temp"`
		DRAMRead    float64 `json:"dram_read_bw_gbs"`
		DRAMWrite   float64 `json:"dram_write_bw_gbs"`
	} `json:"soc_metrics"`
	Memory struct {
		Total     float64 `json:"total"`
		Used      float64 `json:"used"`
		SwapTotal float64 `json:"swap_total"`
		SwapUsed  float64 `json:"swap_used"`
	} `json:"memory"`
	NetDisk struct {
		InBytes    float64 `json:"in_bytes_per_sec"`
		OutBytes   float64 `json:"out_bytes_per_sec"`
		ReadKBytes float64 `json:"read_kbytes_per_sec"`
		WriteKB    float64 `json:"write_kbytes_per_sec"`
	} `json:"net_disk"`
	SystemInfo struct {
		Name         string `json:"name"`
		ECoreCount   int    `json:"e_core_count"`
		PCoreCount   int    `json:"p_core_count"`
		GPUCoreCount int    `json:"gpu_core_count"`
	} `json:"system_info"`
	Processes []Process `json:"processes"`
	Fans      []struct {
		RPM float64 `json:"rpm"`
	} `json:"fans"`
	Volumes []struct {
		Name        string  `json:"name"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"volumes"`
	Battery struct {
		Present   bool    `json:"present"`
		Percent   float64 `json:"percent"`
		Charging  bool    `json:"charging"`
		OnACPower bool    `json:"on_ac_power"`
	} `json:"battery"`
}

type Process struct {
	PID        int     `json:"pid"`
	Command    string  `json:"command"`
	CPUPercent float64 `json:"cpu_percent"`
	GPUMsPerS  float64 `json:"gpu_ms_per_sec"`
	MemPercent float64 `json:"memory_percent"`
	RSSKB      float64 `json:"rss_kb"`
}

// Decode reads mactop's stream — a JSON array written one element per line ("[{…}", ",{…}") —
// and sends each sample until r ends.
func Decode(r io.Reader, out chan<- Sample) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for sc.Scan() {
		line := bytes.TrimLeft(sc.Bytes(), "[,")
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var s Sample
		if err := json.Unmarshal(bytes.TrimRight(line, ",]"), &s); err != nil {
			return err
		}
		out <- s
	}
	return sc.Err()
}

// Mactop starts mactop --headless at the given interval and streams its samples.
// The channel closes when mactop exits or ctx is cancelled.
func Mactop(ctx context.Context, intervalMs int) (<-chan Sample, error) {
	cmd := exec.CommandContext(ctx, "mactop", "--headless", "--count", "0", "-i", strconv.Itoa(intervalMs))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	ch := make(chan Sample)
	go func() {
		defer close(ch)
		Decode(stdout, ch)
		cmd.Wait()
	}()
	return ch, nil
}
