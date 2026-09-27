package source

import (
	"bytes"
	"path/filepath"
	"regexp"

	"golang.org/x/sys/unix"
)

var bareVersion = regexp.MustCompile(`^[0-9.]+$`)

// ProcNames names processes that mactop reports as a bare version number
// (claude runs as …/versions/2.1.283) by argv[0]'s basename. Lookups are cached per pid.
type ProcNames map[int]string

func (n ProcNames) Name(pid int, command string) string {
	if !bareVersion.MatchString(command) {
		return command
	}
	name, ok := n[pid]
	if !ok {
		name = argv0(pid)
		n[pid] = name
	}
	if name == "" {
		return command
	}
	return name
}

// argv0 reads kern.procargs2: int32 argc, exec path, NUL padding, then argv[0].
func argv0(pid int) string {
	b, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(b) < 4 {
		return ""
	}
	b = b[4:]
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return ""
	}
	b = bytes.TrimLeft(b[i:], "\x00")
	if i = bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	// first word only, like ps's args column split by the shell: some processes retitle argv[0]
	f := bytes.Fields(b)
	if len(f) == 0 {
		return ""
	}
	return filepath.Base(string(f[0]))
}
