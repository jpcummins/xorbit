// Package tty implements application.Terminal and application.Display
// on the controlling terminal. The terminal stays in canonical mode with
// echo on; this package never changes its settings.
package tty

import (
	"errors"
	"io"
	"os"
	"runtime"
	"syscall"

	app "xorbit/internal/app"
)

// ErrNoTTY is returned when the controlling terminal is missing or not a TTY.
var ErrNoTTY = errors.New("no controlling terminal")

// ErrIO is returned when reading from or writing to the terminal fails.
var ErrIO = errors.New("terminal i/o failed")

// ErrShown is returned when Show is called a second time.
var ErrShown = errors.New("result already shown")

// The only escape sequences the program writes (AGENTS.md, The TTY interface).
const (
	enterAltScreen  = "\x1b[?1049h"
	leaveAltScreen  = "\x1b[?1049l"
	clearScreen     = "\x1b[H\x1b[2J"
	clearScrollback = "\x1b[3J"
)

var (
	_ app.Terminal = (*Terminal)(nil)
	_ app.Display  = (*Terminal)(nil)
)

// Terminal reads lines from and draws screens on /dev/tty.
type Terminal struct {
	rw     io.ReadWriter
	closer io.Closer // nil in tests
	shown  bool
	closed bool
}

// Open opens the controlling terminal read-write, verifies it is a real
// terminal, and switches to the alternate screen.
func Open() (*Terminal, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, ErrNoTTY
	}
	if !isTerminal(f) {
		f.Close()
		return nil, ErrNoTTY
	}
	if _, err := io.WriteString(f, enterAltScreen); err != nil {
		f.Close()
		return nil, ErrNoTTY
	}
	return &Terminal{rw: f, closer: f}, nil
}

// isTerminal checks the descriptor with ioctl(TCGETS). Passing a real termios
// buffer would require unsafe, which is forbidden, so a NULL pointer is passed
// instead: a terminal accepts the request and fails copying to the NULL
// address (EFAULT), while anything else rejects the request (ENOTTY/EINVAL).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, 0)
	return e == syscall.EFAULT
}

// Draw clears the screen and writes frame. The frame is written directly
// from the caller's locked buffer, never copied.
func (t *Terminal) Draw(frame []byte) error {
	if _, err := io.WriteString(t.rw, clearScreen); err != nil {
		return ErrIO
	}
	if _, err := t.rw.Write(frame); err != nil {
		return ErrIO
	}
	return nil
}

// Show draws the final result. It refuses to show it twice.
func (t *Terminal) Show(result []byte) error {
	if t.shown {
		return ErrShown
	}
	t.shown = true
	return t.Draw(result)
}

// ReadLine reads one line of operator input into dst and returns its length
// without the trailing newline.
//
// In canonical mode one Read returns at most one line. Three cases follow:
//   - the data ends in '\n': a complete line;
//   - the data is shorter than dst with no '\n': the operator pressed
//     Ctrl-D mid-line, which hands over the text typed so far; treat it as
//     a complete line;
//   - the data fills dst with no '\n': the line is too long. The rest of it
//     is read and discarded, dst is wiped, and ErrLineTooLong is returned.
//
// A zero-byte Read means end of input (Ctrl-D on an empty line).
func (t *Terminal) ReadLine(dst []byte) (int, error) {
	n, err := t.rw.Read(dst)
	if n == 0 {
		return 0, endOfInput(err)
	}
	if dst[n-1] == '\n' {
		return n - 1, nil
	}
	if n < len(dst) {
		return n, nil
	}
	for {
		wipe(dst)
		n, err = t.rw.Read(dst)
		if n == 0 {
			return 0, endOfInput(err)
		}
		if dst[n-1] == '\n' || n < len(dst) {
			wipe(dst)
			return 0, app.ErrLineTooLong
		}
	}
}

// endOfInput maps the error from a zero-byte Read.
func endOfInput(err error) error {
	if err == nil || errors.Is(err, io.EOF) {
		return app.ErrEOF
	}
	return ErrIO
}

// wipe clears the bytes of dst the terminal wrote into. It clears only
// len(dst), never beyond, because dst belongs to the caller.
func wipe(dst []byte) {
	clear(dst)
	runtime.KeepAlive(dst)
}

// Close clears the screen and scrollback, leaves the alternate screen, and
// closes the terminal. Clearing is a courtesy, not a security control.
// Calling Close more than once is harmless.
func (t *Terminal) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	_, err := io.WriteString(t.rw, clearScreen+clearScrollback+leaveAltScreen)
	if t.closer != nil {
		if cerr := t.closer.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		return ErrIO
	}
	return nil
}
