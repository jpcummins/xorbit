package application_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"xorbit/bip39"
	application "xorbit/internal/app"
)

func TestFrameText(t *testing.T) {
	f := application.NewFrame(make([]byte, 64), 3)
	f.Text("ab\ncd")
	f.NewLine()
	f.Text("> ")
	if f.Err() != nil {
		t.Fatal("unexpected error")
	}
	if got := string(f.Out()); got != "ab\ncd\n> " {
		t.Fatalf("got %q", got)
	}
}

func TestFrameWidth(t *testing.T) {
	f := application.NewFrame(make([]byte, 256), 2)
	f.Text(strings.Repeat("x", application.FrameCols))
	if f.Err() != nil {
		t.Fatal("80 columns must fit")
	}
	f.Text("x")
	if !errors.Is(f.Err(), application.ErrFrameOverflow) {
		t.Fatal("81 columns must overflow")
	}
}

func TestFrameRows(t *testing.T) {
	f := application.NewFrame(make([]byte, 64), 2)
	f.Text("a\nb\n")
	if f.Err() != nil {
		t.Fatal("2 rows must fit")
	}
	f.Text("c")
	if !errors.Is(f.Err(), application.ErrFrameOverflow) {
		t.Fatal("a third row must overflow")
	}
}

func TestFrameEmptyRowsCount(t *testing.T) {
	f := application.NewFrame(make([]byte, 64), 2)
	f.Text("\n\n")
	if f.Err() != nil {
		t.Fatal("2 empty rows must fit")
	}
	f.NewLine()
	if !errors.Is(f.Err(), application.ErrFrameOverflow) {
		t.Fatal("a third empty row must overflow")
	}
}

func TestFrameRejectsNonPrintable(t *testing.T) {
	for _, s := range []string{"\t", "\r", "\x1b", "\x7f", "\xc3\xa9"} {
		f := application.NewFrame(make([]byte, 64), 2)
		f.Text(s)
		if !errors.Is(f.Err(), application.ErrFrameText) {
			t.Errorf("%q: want ErrFrameText", s)
		}
	}
}

func TestFrameEnterGlyphIsOneColumn(t *testing.T) {
	f := application.NewFrame(make([]byte, 256), 1)
	f.Text(strings.Repeat("x", 79))
	f.Text("↵")
	if f.Err() != nil {
		t.Fatalf("Enter glyph should fit the 80th column: %v", f.Err())
	}
	if got := string(f.Out()); got != strings.Repeat("x", 79)+"↵" {
		t.Fatalf("got %q", got)
	}
	f.Text("x")
	if !errors.Is(f.Err(), application.ErrFrameOverflow) {
		t.Fatal("the glyph must occupy one column")
	}
}

func TestFrameHexNibbleFixtures(t *testing.T) {
	cases := []struct {
		name string
		in   byte
		want string
	}{
		{name: "nine", in: 0x09, want: "09"},
		{name: "ten", in: 0x0a, want: "0a"},
		{name: "fifteen", in: 0x0f, want: "0f"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := application.NewFrame(make([]byte, 16), 1)
			f.Hex([]byte{tc.in})
			if f.Err() != nil {
				t.Fatalf("Hex: %v", f.Err())
			}
			if got := string(f.Out()); got != tc.want {
				t.Errorf("Hex(%02x) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFrameWordUsesLookup(t *testing.T) {
	f := application.NewFrame(make([]byte, 256), 2)
	lookup := bip39.Lookup{}
	f.Word(0, lookup)
	f.Text(" ")
	f.Word(2047, lookup)
	if f.Err() != nil {
		t.Fatalf("Word: %v", f.Err())
	}
	if got, want := string(f.Out()), "abandon  zoo     "; got != want {
		t.Errorf("word slots: got %q, want %q", got, want)
	}
}

func TestFrameHexDigestFixture(t *testing.T) {
	in := []byte{
		0xfa, 0x09, 0x8e, 0xb8, 0x52, 0xb2, 0x66, 0x03,
		0x48, 0xb2, 0x1b, 0xb0, 0x0a, 0xd0, 0x3a, 0x49,
		0xcc, 0x17, 0x7e, 0xa0, 0x7e, 0xbe, 0x34, 0xf4,
		0x6b, 0x40, 0xba, 0xa8, 0x53, 0x13, 0x52, 0x5e,
	}
	f := application.NewFrame(make([]byte, 256), 2)
	f.Hex(in)
	if f.Err() != nil {
		t.Fatalf("Hex: %v", f.Err())
	}
	want := "fa098eb852b2660348b21bb00ad03a49cc177ea07ebe34f46b40baa85313525e"
	if got := string(f.Out()); got != want {
		t.Errorf("digest hex: got %q, want %q", got, want)
	}
}

func TestFrameBytes(t *testing.T) {
	f := application.NewFrame(make([]byte, 256), 2)
	f.Bytes([]byte("34165253123"))
	if f.Err() != nil {
		t.Fatalf("Bytes: %v", f.Err())
	}
	if got, want := string(f.Out()), "34165253123"; got != want {
		t.Errorf("bytes: got %q, want %q", got, want)
	}
}

func TestFrameBufferFull(t *testing.T) {
	f := application.NewFrame(make([]byte, 3), 2)
	f.Text("abcd")
	if !errors.Is(f.Err(), application.ErrFrameOverflow) {
		t.Fatal("want overflow")
	}
	if got := string(f.Out()); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestFrameErrorIsSticky(t *testing.T) {
	f := application.NewFrame(make([]byte, 64), 1)
	f.Text("a\x01")
	f.Text("bc")
	f.NewLine()
	if !errors.Is(f.Err(), application.ErrFrameText) {
		t.Fatal("first error must be kept")
	}
	if got := string(f.Out()); got != "a" {
		t.Fatalf("writes after an error must be ignored, got %q", got)
	}
}

func TestNewFrameWipesBuffer(t *testing.T) {
	buf := bytes.Repeat([]byte{'x'}, 16)
	f := application.NewFrame(buf, 2)
	f.Text("a")
	if buf[0] != 'a' || !bytes.Equal(buf[1:], make([]byte, 15)) {
		t.Fatal("previous frame contents must be wiped")
	}
}
