package tty

import (
	"bytes"
	"errors"
	"io"
	"testing"

	app "xorbit/internal/app"
)

// fakeTTY stands in for /dev/tty. Each entry in reads is what one Read
// returns, like one canonical-mode line from the kernel.
type fakeTTY struct {
	reads    []string
	readErr  error // returned once reads is empty; io.EOF if nil
	out      bytes.Buffer
	writeErr error
	closed   bool
}

func (f *fakeTTY) Read(p []byte) (int, error) {
	if len(f.reads) == 0 {
		if f.readErr != nil {
			return 0, f.readErr
		}
		return 0, io.EOF
	}
	r := f.reads[0]
	f.reads = f.reads[1:]
	if len(r) > len(p) {
		panic("test read chunk is larger than the buffer")
	}
	return copy(p, r), nil
}

func (f *fakeTTY) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.out.Write(p)
}

func (f *fakeTTY) Close() error {
	f.closed = true
	return nil
}

func newFake(reads ...string) (*Terminal, *fakeTTY) {
	f := &fakeTTY{reads: reads}
	return &Terminal{rw: f, closer: f}, f
}

func TestReadLine(t *testing.T) {
	term, _ := newFake("c\n")
	dst := make([]byte, 8)
	n, err := term.ReadLine(dst)
	if err != nil || string(dst[:n]) != "c" {
		t.Fatalf("got %q, %v", dst[:n], err)
	}
}

func TestReadLineExactFit(t *testing.T) {
	term, _ := newFake("abc\n")
	dst := make([]byte, 4)
	n, err := term.ReadLine(dst)
	if err != nil || string(dst[:n]) != "abc" {
		t.Fatalf("got %q, %v", dst[:n], err)
	}
}

func TestReadLineWithoutNewline(t *testing.T) {
	term, _ := newFake("ab")
	dst := make([]byte, 8)
	n, err := term.ReadLine(dst)
	if err != nil || string(dst[:n]) != "ab" {
		t.Fatalf("Ctrl-D mid-line must hand over the line, got %q, %v", dst[:n], err)
	}
}

func TestReadLineTooLong(t *testing.T) {
	term, f := newFake("abcd", "efgh", "i\n", "next\n")
	dst := make([]byte, 4)
	if _, err := term.ReadLine(dst); !errors.Is(err, app.ErrLineTooLong) {
		t.Fatal("want ErrLineTooLong")
	}
	if !bytes.Equal(dst, make([]byte, 4)) {
		t.Fatal("dst must be wiped")
	}
	if len(f.reads) != 1 {
		t.Fatal("the rest of the long line must be drained, and nothing more")
	}
}

func TestReadLineTooLongThenEOF(t *testing.T) {
	term, _ := newFake("abcd")
	dst := make([]byte, 4)
	if _, err := term.ReadLine(dst); !errors.Is(err, app.ErrEOF) {
		t.Fatal("want ErrEOF")
	}
	if !bytes.Equal(dst, make([]byte, 4)) {
		t.Fatal("dst must be wiped")
	}
}

func TestReadLineEOF(t *testing.T) {
	term, _ := newFake()
	if _, err := term.ReadLine(make([]byte, 8)); !errors.Is(err, app.ErrEOF) {
		t.Fatal("want ErrEOF")
	}
}

func TestReadLineError(t *testing.T) {
	term, f := newFake()
	f.readErr = errors.New("read failed")
	if _, err := term.ReadLine(make([]byte, 8)); !errors.Is(err, ErrIO) {
		t.Fatal("want ErrIO")
	}
}

func TestDraw(t *testing.T) {
	term, f := newFake()
	if err := term.Draw([]byte("frame")); err != nil {
		t.Fatal("unexpected error")
	}
	if got := f.out.String(); got != clearScreen+"frame" {
		t.Fatalf("got %q", got)
	}
}

func TestDrawWriteError(t *testing.T) {
	term, f := newFake()
	f.writeErr = errors.New("write failed")
	if err := term.Draw([]byte("frame")); !errors.Is(err, ErrIO) {
		t.Fatal("want ErrIO")
	}
}

func TestShowOnce(t *testing.T) {
	term, f := newFake()
	if err := term.Show([]byte("result")); err != nil {
		t.Fatal("unexpected error")
	}
	if err := term.Show([]byte("result")); !errors.Is(err, ErrShown) {
		t.Fatal("second Show must fail")
	}
	if got := f.out.String(); got != clearScreen+"result" {
		t.Fatalf("result must be written once, got %q", got)
	}
}

func TestClose(t *testing.T) {
	term, f := newFake()
	if err := term.Close(); err != nil {
		t.Fatal("unexpected error")
	}
	if got := f.out.String(); got != clearScreen+clearScrollback+leaveAltScreen {
		t.Fatalf("got %q", got)
	}
	if !f.closed {
		t.Fatal("file must be closed")
	}
	if err := term.Close(); err != nil || f.out.Len() != len(clearScreen+clearScrollback+leaveAltScreen) {
		t.Fatal("second Close must do nothing")
	}
}
