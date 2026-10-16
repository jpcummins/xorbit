package harden

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// ttyNr encodes a device number the way /proc/<pid>/stat's tty_nr field
// does (the kernel's new_encode_dev): major in the high bits, minor
// split across two fields.
func ttyNr(major, minor uint64) uint64 {
	return (major << 8) | (minor & 0xff) | ((minor &^ 0xff) << 12)
}

// statLine fabricates a /proc/<pid>/stat line.
func statLine(pid int, comm string, ppid int, tty uint64) string {
	return strconv.Itoa(pid) + " (" + comm + ") R " + strconv.Itoa(ppid) +
		" 100 100 " + strconv.FormatUint(tty, 10) + " 0 0 0"
}

// writeStat creates <root>/<pid>/stat with the given contents.
func writeStat(t *testing.T, root string, pid int, contents string) {
	t.Helper()
	dir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

// linkFd creates <root>/<pid>/fd/<fd> as a symlink to target. The target
// need not exist: the scan compares link text, never the target.
func linkFd(t *testing.T, root string, pid, fd int, target string) {
	t.Helper()
	dir := filepath.Join(root, strconv.Itoa(pid), "fd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, strconv.Itoa(fd))); err != nil {
		t.Fatal(err)
	}
}

func TestStatFields(t *testing.T) {
	// comm may contain spaces and parentheses; fields are taken from the
	// last ')'.
	line := statLine(123, "we ) ird ( name", 45, 34823)
	fields, ok := statFields(line)
	if !ok {
		t.Fatal("parse failed")
	}
	if fields[1] != "45" {
		t.Errorf("ppid = %q, want 45", fields[1])
	}
	if fields[4] != "34823" {
		t.Errorf("tty_nr = %q, want 34823", fields[4])
	}
	if _, ok := statFields("garbage with no parens"); ok {
		t.Error("a line without comm parens must fail to parse")
	}
}

func TestTracerPid(t *testing.T) {
	dir := t.TempDir()
	status := filepath.Join(dir, "status")
	if err := os.WriteFile(status, []byte("Name:\ttest\nTracerPid:\t0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	traced, err := tracerPid(status)
	if err != nil || traced {
		t.Errorf("TracerPid 0: traced = %v, err = %v", traced, err)
	}
	if err := os.WriteFile(status, []byte("Name:\ttest\nTracerPid:\t4242\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	traced, err = tracerPid(status)
	if err != nil || !traced {
		t.Errorf("TracerPid 4242: traced = %v, err = %v", traced, err)
	}
	if _, err := tracerPid(filepath.Join(dir, "missing")); err == nil {
		t.Error("a missing status file must error")
	}
}

// The real /proc/self/status reports no tracer under `go test`.
func TestCheckTracedSelf(t *testing.T) {
	if err := CheckTraced(); err != nil {
		t.Errorf("the test process must not be traced: %v", err)
	}
}

// newProcTree builds a fake /proc with a process `self` on /dev/pts/7
// whose parent is `parent`.
func newProcTree(t *testing.T, self, parent int) string {
	t.Helper()
	root := t.TempDir()
	writeStat(t, root, self, statLine(self, "xorbit", parent, ttyNr(136, 7)))
	writeStat(t, root, parent, statLine(parent, "bash", 1, ttyNr(136, 7)))
	return root
}

func TestCheckTerminalClean(t *testing.T) {
	root := newProcTree(t, 100, 99)
	// The parent (an ancestor) holds the same pty: that is legitimate.
	linkFd(t, root, 99, 0, "/dev/pts/7")
	if err := checkTerminal(root, 100); err != nil {
		t.Errorf("clean tree must pass, got %v", err)
	}
}

func TestCheckTerminalSnooper(t *testing.T) {
	root := newProcTree(t, 100, 99)
	// A non-ancestral process holds the same pty.
	linkFd(t, root, 200, 3, "/dev/pts/7")
	if err := checkTerminal(root, 100); !errors.Is(err, ErrTerminal) {
		t.Errorf("a snooper must fail the check, got %v", err)
	}
}

func TestCheckTerminalOtherPtyIsFine(t *testing.T) {
	root := newProcTree(t, 100, 99)
	// A non-ancestral process holds a *different* pty.
	linkFd(t, root, 200, 3, "/dev/pts/8")
	if err := checkTerminal(root, 100); err != nil {
		t.Errorf("another pty must not be flagged, got %v", err)
	}
}

func TestCheckTerminalUnreadableFdDir(t *testing.T) {
	tests := []struct {
		name     string
		stat     string
		wantFail bool
	}{
		{
			name:     "same controlling pty",
			stat:     statLine(200, "unreadable", 1, ttyNr(136, 7)),
			wantFail: true,
		},
		{
			name: "different controlling pty",
			stat: statLine(200, "unreadable", 1, ttyNr(136, 8)),
		},
		{
			name: "no controlling pty",
			stat: statLine(200, "unreadable", 1, 0),
		},
		{
			name:     "malformed stat",
			stat:     "malformed",
			wantFail: true,
		},
		{
			name: "process vanished",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newProcTree(t, 100, 99)
			if test.stat != "" {
				processDir := filepath.Join(root, "200")
				if err := os.MkdirAll(processDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// A regular file at fd makes ReadDir fail without relying on
				// filesystem permissions (which behave differently as root).
				if err := os.WriteFile(filepath.Join(processDir, "fd"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
				if test.stat != "" {
					if err := os.WriteFile(filepath.Join(processDir, "stat"), []byte(test.stat), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}

			err := checkTerminal(root, 100)
			if test.wantFail && !errors.Is(err, ErrTerminal) {
				t.Errorf("unreadable fd table with matching or unknown pty must fail, got %v", err)
			}
			if !test.wantFail && err != nil {
				t.Errorf("unreadable fd table with a different, absent, or vanished pty must pass, got %v", err)
			}
		})
	}
}

func TestCheckTerminalUnreadableFDLinkUsesTTYNr(t *testing.T) {
	for _, test := range []struct {
		name     string
		tty      uint64
		wantFail bool
	}{
		{name: "same controlling pty", tty: ttyNr(136, 7), wantFail: true},
		{name: "different controlling pty", tty: ttyNr(136, 8)},
		{name: "no controlling pty", tty: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := newProcTree(t, 100, 99)
			writeStat(t, root, 200, statLine(200, "unreadable", 1, test.tty))
			fdDir := filepath.Join(root, "200", "fd")
			if err := os.MkdirAll(fdDir, 0o755); err != nil {
				t.Fatal(err)
			}
			// Readlink on a regular file fails without relying on permissions.
			if err := os.WriteFile(filepath.Join(fdDir, "3"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			err := checkTerminal(root, 100)
			if test.wantFail && !errors.Is(err, ErrTerminal) {
				t.Errorf("unreadable fd link on the same pty must fail, got %v", err)
			}
			if !test.wantFail && err != nil {
				t.Errorf("unreadable fd link on another pty must pass, got %v", err)
			}
		})
	}
}

func TestCheckTerminalNotAPty(t *testing.T) {
	root := t.TempDir()
	// Controlling terminal is a Linux VT (major 4): the scan is a no-op.
	writeStat(t, root, 100, statLine(100, "xorbit", 1, ttyNr(4, 4)))
	linkFd(t, root, 200, 3, "/dev/tty4")
	if err := checkTerminal(root, 100); err != nil {
		t.Errorf("a non-pty terminal skips the scan, got %v", err)
	}
}

func TestCheckTerminalNoControllingTerminal(t *testing.T) {
	root := t.TempDir()
	writeStat(t, root, 100, statLine(100, "xorbit", 1, 0))
	if err := checkTerminal(root, 100); !errors.Is(err, ErrTerminal) {
		t.Errorf("no controlling terminal must fail closed, got %v", err)
	}
}

func TestCheckTerminalBrokenAncestry(t *testing.T) {
	root := t.TempDir()
	// The parent's stat is missing: the chain cannot be proven.
	writeStat(t, root, 100, statLine(100, "xorbit", 99, ttyNr(136, 7)))
	if err := checkTerminal(root, 100); !errors.Is(err, ErrTerminal) {
		t.Errorf("an unreadable ancestor must fail closed, got %v", err)
	}
}

func TestCheckTerminalMissingSelf(t *testing.T) {
	if err := checkTerminal(t.TempDir(), 100); !errors.Is(err, ErrTerminal) {
		t.Errorf("a missing self stat must fail closed, got %v", err)
	}
}

// A large pty index (minor > 255) must decode correctly.
func TestControllingPTYLargeIndex(t *testing.T) {
	root := t.TempDir()
	writeStat(t, root, 100, statLine(100, "test", 1, ttyNr(136, 300)))
	pty, ok, err := controllingPTY(root, 100)
	if err != nil || !ok {
		t.Fatalf("pty = %d, ok = %v, err = %v", pty, ok, err)
	}
	if pty != 300 {
		t.Errorf("pty index = %d, want 300", pty)
	}
}

func TestPTYNumberExample(t *testing.T) {
	pty, ok := ptyNumber(34823)
	if !ok || pty != 7 {
		t.Errorf("ptyNumber(34823) = (%d, %v), want (7, true)", pty, ok)
	}
	if pty, ok := ptyNumber(ttyNr(4, 4)); ok || pty != 0 {
		t.Errorf("virtual-console device decoded as pty: (%d, %v)", pty, ok)
	}
}
