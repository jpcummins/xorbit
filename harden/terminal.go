// Terminal and debugger checks (AGENTS.md "Process hardening", steps 10
// and 11). Both read /proc and nothing else. They fail closed: any
// doubt is an error, and the error carries no data.
package harden

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Sentinel errors for the checks. They carry no data.
var (
	// ErrTraced means a debugger is attached to the process.
	ErrTraced = errors.New("debugger attached")
	// ErrTerminal means the controlling terminal is held by a process
	// outside the program's own ancestry, or the check could not be
	// completed.
	ErrTerminal = errors.New("terminal held by another process")
)

// CheckTraced implements hardening step 10: read TracerPid from
// /proc/self/status and fail if it is not 0. Defense in depth behind
// PR_SET_DUMPABLE (step 3), which already blocks same-user tracers.
func CheckTraced() error {
	traced, err := tracerPid("/proc/self/status")
	if err != nil || traced {
		return ErrTraced
	}
	return nil
}

// CheckTerminal implements hardening step 11: fail if any process
// outside this process's own ancestry holds the controlling terminal
// open.
//
// A pty slave offers no exclusive-open mechanism, so a same-user
// process could otherwise race the program for terminal input (the
// operator's rolls are the entropy preimage) or inject input with
// TIOCSTI. Detection is the available control.
//
// Scope, stated honestly:
//   - Covers pty terminals (the supported Tails desktop case). On a
//     Linux virtual console (the controlling terminal is not a pty) the
//     scan is a no-op.
//   - Processes in the program's own ancestry (its shell and the shell's
//     parents) legitimately hold the terminal; a multiplexer or logger
//     there is invisible to this check, which is why the operator
//     guidance against tmux, screen, and script stands.
//   - If a process's fd table cannot be read, its controlling terminal is
//     checked. It is treated as a holder only when /proc/<pid>/stat reports
//     this same pty. This avoids rejecting unrelated protected desktop
//     processes, but cannot detect a process that opened this pty without
//     making it the process's controlling terminal. A process that vanished
//     during the scan is ignored.
func CheckTerminal() error {
	return checkTerminal("/proc", os.Getpid())
}

// checkTerminal is CheckTerminal with an injectable /proc root and pid,
// so tests can drive it against a fabricated process tree.
func checkTerminal(proc string, self int) error {
	pty, ok, err := controllingPTY(proc, self)
	if err != nil {
		return ErrTerminal
	}
	if !ok {
		// The controlling terminal is not a pty (for example a Linux
		// virtual console): the scan does not apply. The TTY checks in
		// main still require a real terminal.
		return nil
	}
	allowed, err := ancestry(proc, self)
	if err != nil {
		return ErrTerminal
	}
	if holderOutsideAncestry(proc, pty, allowed) {
		return ErrTerminal
	}
	return nil
}

// statFields parses the fields of /proc/<pid>/stat that follow the comm
// field. comm is parenthesized and may itself contain spaces and
// parentheses, so the fields are taken relative to the last ')' in the
// line; everything after comm is numbers and single letters. The
// returned slice starts at the state field, so ppid is index 1 and
// tty_nr is index 4.
func statFields(stat string) ([]string, bool) {
	i := strings.LastIndexByte(stat, ')')
	if i < 0 || i+2 >= len(stat) {
		return nil, false
	}
	fields := strings.Fields(stat[i+2:])
	if len(fields) < 5 {
		return nil, false
	}
	return fields, true
}

// controllingPTY returns the pty index of the process's controlling
// terminal (from the tty_nr field of /proc/<pid>/stat), and whether the
// controlling terminal is a pty at all. tty_nr and st_rdev share the
// kernel's dev_t encoding: major 136 is a Unix98 pty slave, and the
// minor is the pty index N of /dev/pts/N. A tty_nr of 0 means there is
// no controlling terminal, which is an error here.
func controllingPTY(proc string, pid int) (int, bool, error) {
	b, err := os.ReadFile(proc + "/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false, err
	}
	fields, ok := statFields(string(b))
	if !ok {
		return 0, false, errors.New("bad stat")
	}
	dev, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil || dev == 0 {
		return 0, false, errors.New("no controlling terminal")
	}
	pty, ok := ptyNumber(dev)
	return pty, ok, nil
}

// ptyNumber decodes a Linux new_encode_dev value. The major is the 12-bit
// field above bit 7; the minor combines the low 8 bits with the remaining
// high bits. It returns the Unix98 pty number only for major 136. For example,
// device value 34823 decodes to /dev/pts/7. The values are process metadata,
// not secrets, so constant-time handling is unnecessary.
func ptyNumber(dev uint64) (int, bool) {
	major := (dev >> 8) & 0xfff
	minor := (dev & 0xff) | ((dev >> 12) &^ 0xff)
	if major != 136 {
		return 0, false
	}
	return int(minor), true
}

// ancestry returns the set of pids from pid up through its parent
// chain, excluding pid 1 and anything above it. Those processes may
// legitimately hold the terminal (inherited fds), so the scan ignores
// them. Failing to read an ancestor means the chain cannot be proven,
// so it is an error (fail closed).
func ancestry(proc string, pid int) (map[int]bool, error) {
	allowed := map[int]bool{}
	for pid > 1 {
		if allowed[pid] {
			break // cycle: should not happen, but stop walking
		}
		allowed[pid] = true
		b, err := os.ReadFile(proc + "/" + strconv.Itoa(pid) + "/stat")
		if err != nil {
			return nil, err
		}
		fields, ok := statFields(string(b))
		if !ok {
			return nil, errors.New("bad stat")
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		pid = ppid
	}
	return allowed, nil
}

// holderOutsideAncestry reports whether any process outside allowed has
// an open fd for /dev/pts/<pty>, or has an unreadable fd table and reports
// that same pty as its controlling terminal. A process that vanishes during
// the scan is ignored.
func holderOutsideAncestry(proc string, pty int, allowed map[int]bool) bool {
	target := "/dev/pts/" + strconv.Itoa(pty)
	entries, err := os.ReadDir(proc)
	if err != nil {
		return true // cannot scan at all: fail closed
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || allowed[pid] {
			continue
		}
		fdDir := proc + "/" + e.Name() + "/fd"
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			samePTY, statErr := processHasPTY(proc, pid, pty)
			if errors.Is(statErr, os.ErrNotExist) {
				continue // process vanished during the scan
			}
			if statErr != nil || samePTY {
				return true // cannot rule out a process attached to this pty
			}
			continue // a different or absent controlling pty is not a match
		}
		for _, fd := range fds {
			link, err := os.Readlink(fdDir + "/" + fd.Name())
			if err == nil && link == target {
				return true
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				samePTY, statErr := processHasPTY(proc, pid, pty)
				if errors.Is(statErr, os.ErrNotExist) {
					break // process vanished during the scan
				}
				if statErr != nil || samePTY {
					return true // cannot rule out a holder on this pty
				}
			}
		}
	}
	return false
}

// processHasPTY reports whether pid's controlling terminal is /dev/pts/pty,
// as encoded by the tty_nr field in /proc/<pid>/stat. The controlling terminal
// is used only as a filter when the process's fd table cannot be inspected.
func processHasPTY(proc string, pid, pty int) (bool, error) {
	b, err := os.ReadFile(proc + "/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false, err
	}
	fields, ok := statFields(string(b))
	if !ok {
		return false, errors.New("bad stat")
	}
	dev, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil {
		return false, err
	}
	if dev == 0 {
		return false, nil
	}
	processPTY, isPTY := ptyNumber(dev)
	return isPTY && processPTY == pty, nil
}

// tracerPid reads /proc/<pid>/status and reports whether TracerPid is
// not 0.
func tracerPid(status string) (bool, error) {
	b, err := os.ReadFile(status)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "TracerPid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return false, errors.New("bad status")
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			return false, err
		}
		return pid != 0, nil
	}
	return false, errors.New("no TracerPid line")
}
