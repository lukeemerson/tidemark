package source

import (
	"encoding/binary"

	"golang.org/x/sys/unix"
)

// Sys is what mactop doesn't report: load averages and the kernel's memory-pressure view.
type Sys struct {
	Load     [3]float64 // 1, 5 and 15 minute load averages
	Pressure int        // kern.memorystatus_vm_pressure_level: 1 normal, 2 warn, 4 critical
	FreePct  int        // kern.memorystatus_level: % of memory the kernel considers available
}

func ReadSys() Sys {
	var s Sys
	// struct loadavg { uint32 ldavg[3]; long fscale; } — fscale at offset 16 after padding
	if b, err := unix.SysctlRaw("vm.loadavg"); err == nil && len(b) >= 24 {
		if scale := float64(binary.LittleEndian.Uint64(b[16:])); scale > 0 {
			for i := range s.Load {
				s.Load[i] = float64(binary.LittleEndian.Uint32(b[i*4:])) / scale
			}
		}
	}
	if v, err := unix.SysctlUint32("kern.memorystatus_vm_pressure_level"); err == nil {
		s.Pressure = int(v)
	}
	if v, err := unix.SysctlUint32("kern.memorystatus_level"); err == nil {
		s.FreePct = int(v)
	}
	return s
}
