package store

import (
	"sort"
	"time"

	"github.com/lukeemerson/tidemark/internal/source"
)

// Bucket is one 10 s summary line: the averaged sample the panels draw, the peak of each graphed
// series, the sysctl readings, the top 5 processes by CPU and whether a rule fired.
type Bucket struct {
	T        time.Time          `json:"t"` // bucket start
	N        int                `json:"n"` // samples in the bucket
	Avg      source.Sample      `json:"avg"`
	Peak     map[string]float64 `json:"peak"`
	Load     [3]float64         `json:"load"`
	Pressure int                `json:"pressure"` // the worst level seen
	Free     int                `json:"free"`     // the lowest available % seen
	SysOK    bool               `json:"sys"`      // load/pressure/free were read
	Top      []Proc             `json:"top"`      // top 5 by peak CPU
	Alert    bool               `json:"alert"`    // a diagnosis rule started firing
}

type Proc struct {
	PID  int     `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"` // peak % in the bucket
}

// Series names the graphed values a bucket keeps a peak for, and how to read each from a sample.
var Series = map[string]func(s source.Sample, sys source.Sys) float64{
	"cpu":    func(s source.Sample, _ source.Sys) float64 { return s.CPUUsage },
	"gpu":    func(s source.Sample, _ source.Sys) float64 { return s.GPUUsage },
	"power":  func(s source.Sample, _ source.Sys) float64 { return s.SoC.TotalPower },
	"mem":    func(s source.Sample, _ source.Sys) float64 { return memPct(s) },
	"ctemp":  func(s source.Sample, _ source.Sys) float64 { return s.SoC.CPUTemp },
	"gtemp":  func(s source.Sample, _ source.Sys) float64 { return s.SoC.GPUTemp },
	"swap":   func(s source.Sample, _ source.Sys) float64 { return s.Memory.SwapUsed },
	"netin":  func(s source.Sample, _ source.Sys) float64 { return s.NetDisk.InBytes },
	"netout": func(s source.Sample, _ source.Sys) float64 { return s.NetDisk.OutBytes },
	"diskr":  func(s source.Sample, _ source.Sys) float64 { return s.NetDisk.ReadKBytes * 1024 },
	"diskw":  func(s source.Sample, _ source.Sys) float64 { return s.NetDisk.WriteKB * 1024 },
	"dram":   func(s source.Sample, _ source.Sys) float64 { return s.SoC.DRAMRead + s.SoC.DRAMWrite },
	"load":   func(_ source.Sample, sys source.Sys) float64 { return sys.Load[0] },
}

func memPct(s source.Sample) float64 {
	if s.Memory.Total == 0 {
		return 0
	}
	return s.Memory.Used * 100 / s.Memory.Total
}

type bucket struct {
	t     time.Time
	n     int
	sum   source.Sample
	peak  map[string]float64
	load  [3]float64
	sysN  int
	press int
	free  int
	procs map[int]Proc
	alert bool
	last  source.Sample
}

func newBucket(t time.Time) *bucket {
	return &bucket{t: t, peak: map[string]float64{}, procs: map[int]Proc{}, free: 101}
}

func (b *bucket) add(s source.Sample, sys source.Sys, sysOK, fired bool, name func(int, string) string) {
	b.n++
	b.last = s
	a := &b.sum
	a.CPUUsage += s.CPUUsage
	a.GPUUsage += s.GPUUsage
	for i, c := range s.CoreUsages {
		if i >= len(a.CoreUsages) {
			a.CoreUsages = append(a.CoreUsages, 0)
		}
		a.CoreUsages[i] += c
	}
	x, y := &a.SoC, s.SoC
	x.TotalPower += y.TotalPower
	x.SystemPower += y.SystemPower
	x.GPUPower += y.GPUPower
	x.EFreqMHz += y.EFreqMHz
	x.PFreqMHz += y.PFreqMHz
	x.GPUFreqMHz += y.GPUFreqMHz
	x.ANEActive += y.ANEActive
	x.CPUTemp += y.CPUTemp
	x.GPUTemp += y.GPUTemp
	x.DRAMRead += y.DRAMRead
	x.DRAMWrite += y.DRAMWrite
	a.Memory.Used += s.Memory.Used
	a.Memory.SwapUsed += s.Memory.SwapUsed
	a.NetDisk.InBytes += s.NetDisk.InBytes
	a.NetDisk.OutBytes += s.NetDisk.OutBytes
	a.NetDisk.ReadKBytes += s.NetDisk.ReadKBytes
	a.NetDisk.WriteKB += s.NetDisk.WriteKB
	if len(s.Fans) > 0 {
		if len(a.Fans) == 0 {
			a.Fans = append(a.Fans, s.Fans[0]) // a copy: the sample's own slice must not change
			a.Fans[0].RPM = 0
		}
		a.Fans[0].RPM += s.Fans[0].RPM
	}
	if s.ThermalState != "" && s.ThermalState != "Nominal" {
		a.ThermalState = s.ThermalState // the bucket keeps any throttling it saw
	}
	for k, f := range Series {
		b.peak[k] = max(b.peak[k], f(s, sys))
	}
	if sysOK {
		b.sysN++
		for i := range b.load {
			b.load[i] += sys.Load[i]
		}
		b.press = max(b.press, sys.Pressure)
		b.free = min(b.free, sys.FreePct)
	}
	for _, p := range s.Processes {
		if q, ok := b.procs[p.PID]; !ok || p.CPUPercent > q.CPU {
			b.procs[p.PID] = Proc{p.PID, name(p.PID, p.Command), p.CPUPercent}
		}
	}
	b.alert = b.alert || fired
}

func (b *bucket) close() Bucket {
	out := Bucket{T: b.t, N: b.n, Peak: b.peak, Alert: b.alert, SysOK: b.sysN > 0, Pressure: b.press}
	n := float64(b.n)
	a := b.sum
	a.Timestamp = b.t
	a.CPUUsage /= n
	a.GPUUsage /= n
	for i := range a.CoreUsages {
		a.CoreUsages[i] /= n
	}
	x := &a.SoC
	for _, v := range []*float64{&x.TotalPower, &x.SystemPower, &x.GPUPower, &x.EFreqMHz, &x.PFreqMHz,
		&x.GPUFreqMHz, &x.ANEActive, &x.CPUTemp, &x.GPUTemp, &x.DRAMRead, &x.DRAMWrite,
		&a.Memory.Used, &a.Memory.SwapUsed, &a.NetDisk.InBytes, &a.NetDisk.OutBytes,
		&a.NetDisk.ReadKBytes, &a.NetDisk.WriteKB} {
		*v /= n
	}
	if len(a.Fans) > 0 {
		a.Fans[0].RPM /= n
	}
	if a.ThermalState == "" {
		a.ThermalState = b.last.ThermalState
	}
	a.Memory.Total, a.Memory.SwapTotal = b.last.Memory.Total, b.last.Memory.SwapTotal
	a.SystemInfo, a.Battery, a.Volumes = b.last.SystemInfo, b.last.Battery, b.last.Volumes
	out.Avg = a
	if b.sysN > 0 {
		for i := range b.load {
			out.Load[i] = b.load[i] / float64(b.sysN)
		}
		out.Free = b.free
	}
	for _, p := range b.procs {
		out.Top = append(out.Top, p)
	}
	sort.Slice(out.Top, func(i, j int) bool { return out.Top[i].CPU > out.Top[j].CPU })
	if len(out.Top) > 5 {
		out.Top = out.Top[:5]
	}
	return out
}
