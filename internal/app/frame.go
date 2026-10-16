package application

import "crypto/subtle"

// Screen limits. Frames use at most 22 rows, leaving space below them for
// the operator's echoed input.
const (
	FrameCols = 80
	FrameRows = 22
)

const enterGlyph = "↵"

// Frame builds one screen of printable ASCII text in a locked buffer.
// Errors are sticky: after the first one, further writes are ignored and
// Err reports it.
//
// Each Frame is one screen. The App shares a single Workspace.Frame buffer
// with sections that overlap on row budgets: the chrome, source body,
// message, and bottom command list. To enforce the 10-row body budget, the
// App calls
// Reset before invoking a source's Render so the per-screen row counter
// starts at the body's first row and the body-budget check is local.
//
// Why this matters: every formatter (Word, Hex, Bytes, Padded) operates
// on secret bytes. The order and column alignment of what they write
// must not depend on a secret, so each writes a fixed-width record and
// splits rows with a single NewLine.
type Frame struct {
	buf     []byte
	n       int // bytes written
	col     int // characters on the current row
	rows    int // completed rows (resettable)
	maxRows int
	err     error
}

// NewFrame wipes buf and returns a Frame that writes into it, allowing at
// most maxRows rows.
func NewFrame(buf []byte, maxRows int) *Frame {
	wipe(buf)
	return &Frame{buf: buf, maxRows: maxRows}
}

// Reset starts a new section inside the same buffer. The byte-write position
// and column are unchanged; the row counter is reset to 0 and maxRows is set
// for the new section.
//
// Reset is not the same as NewFrame: the existing buffer is preserved
// (chrome stays on screen) and the row counter restarts from 0 so the
// section-local maxRows check fires at the right boundary.
func (f *Frame) Reset(maxRows int) {
	f.rows = 0
	f.maxRows = maxRows
}

// Text writes fixed program text. A '\n' ends the current row; all other
// text must be printable ASCII except for the single allowed Enter glyph.
func (f *Frame) Text(s string) {
	for i := 0; i < len(s); {
		if s[i] == '\n' {
			f.NewLine()
			i++
		} else if len(s)-i >= len(enterGlyph) && s[i:i+len(enterGlyph)] == enterGlyph {
			f.enterGlyph()
			i += len(enterGlyph)
		} else {
			f.put(s[i])
			i++
		}
	}
}

// enterGlyph writes the one permitted three-byte UTF-8 symbol as one column.
func (f *Frame) enterGlyph() {
	if f.err != nil {
		return
	}
	if f.col == 0 && f.rows == f.maxRows || f.col == FrameCols || len(f.buf)-f.n < len(enterGlyph) {
		f.err = ErrFrameOverflow
		return
	}
	f.buf[f.n] = enterGlyph[0]
	f.buf[f.n+1] = enterGlyph[1]
	f.buf[f.n+2] = enterGlyph[2]
	f.n += len(enterGlyph)
	f.col++
}

// NewLine ends the current row. An empty row still counts against maxRows.
func (f *Frame) NewLine() {
	if f.err != nil {
		return
	}
	if f.col == 0 && f.rows == f.maxRows {
		f.err = ErrFrameOverflow
		return
	}
	if f.n == len(f.buf) {
		f.err = ErrFrameOverflow
		return
	}
	f.buf[f.n] = '\n'
	f.n++
	f.rows++
	f.col = 0
}

// put writes one printable character on the current row.
func (f *Frame) put(c byte) {
	if f.err != nil {
		return
	}
	if c < ' ' || c > '~' {
		f.err = ErrFrameText
		return
	}
	if f.col == 0 && f.rows == f.maxRows {
		f.err = ErrFrameOverflow
		return
	}
	if f.col == FrameCols || f.n == len(f.buf) {
		f.err = ErrFrameOverflow
		return
	}
	f.buf[f.n] = c
	f.n++
	f.col++
}

// endLine ends the current row if anything has been written on it, so the
// next write starts on a fresh row. It never adds an empty row.
func (f *Frame) endLine() {
	if f.col > 0 {
		f.NewLine()
	}
}

// Byte writes c as one printable character. Screen renderers use it to show a
// single symbol or digit derived from operator input without building a
// string or a temporary slice.
func (f *Frame) Byte(c byte) {
	f.put(c)
}

// Bytes writes each byte of b as one printable character. The frame
// enforces printable ASCII; bytes must be derived from operator-supplied
// input, never pool contents directly.
func (f *Frame) Bytes(b []byte) {
	for _, c := range b {
		f.put(c)
	}
}

// Padded writes b followed by enough space bytes to fill exactly `width`
// columns. Bytes beyond `width` in b are dropped with ErrFrameOverflow,
// the same way an overlong line would be. Used by the verify screen to keep
// variable-length BIP-39 words aligned in 8-column cells.
func (f *Frame) Padded(b []byte, width int) {
	if width < 0 {
		f.err = ErrFrameOverflow
		return
	}
	n := len(b)
	if n > width {
		n = width
		f.err = ErrFrameOverflow
	}
	for i := 0; i < n; i++ {
		f.put(b[i])
	}
	for i := n; i < width; i++ {
		f.put(' ')
	}
}

// Word writes the 8-byte BIP-39 word for a valid index 0..2047 into the locked
// frame buffer. The lookup scans the wordlist in constant time and writes
// directly into that buffer, without a temporary string or array.
func (f *Frame) Word(index uint16, words WordLookup) {
	if f.err != nil {
		return
	}
	if words == nil {
		f.err = ErrConfig
		return
	}
	if (f.col == 0 && f.rows == f.maxRows) || f.col+8 > FrameCols || len(f.buf)-f.n < 8 {
		f.err = ErrFrameOverflow
		return
	}
	words.WriteWord(f.buf[f.n:f.n+8], index)
	f.n += 8
	f.col += 8
}

// Hex formats each input byte as two lowercase ASCII hexadecimal characters,
// with no separators. For example, []byte{0x09, 0x0a, 0x0f} becomes
// "090a0f" (the fixture covers nibbles 9, a, and f). Each nibble is converted
// by hexDigitCT, whose mask selection has no secret-dependent branch or
// indexed lookup.
func (f *Frame) Hex(buf []byte) {
	for _, b := range buf {
		f.put(hexDigitCT(b >> 4))
		f.put(hexDigitCT(b & 0x0f))
	}
}

// hexDigitCT converts one nibble (a 4-bit value 0..15) to one lowercase
// ASCII hexadecimal byte. It computes both candidate characters, uses
// ConstantTimeLessOrEq to make a 0/0xff mask for values 0..9 versus 10..15,
// then selects the character with bitwise operations. The fixtures are
// 9 -> '9', 10 -> 'a', and 15 -> 'f'. Its work and memory accesses do not
// depend on the nibble value.
func hexDigitCT(nibble byte) byte {
	isDigit := byte(subtle.ConstantTimeLessOrEq(int(nibble), 9))
	letterMask := byte(0) - byte(1-isDigit)
	digit := byte('0') + nibble
	letter := byte('a') + (nibble - 10)
	return (digit &^ letterMask) | (letter & letterMask)
}

// rightAligned4 spells a public value 0..9999 in 4 characters with
// leading zeros replaced by spaces (the last digit is always shown).
// Examples: 1024 -> "1024", 16 -> "  16", 0 -> "   0". It branches on the
// value, so it MUST only be used for public numbers.
func rightAligned4(n uint16) [4]byte {
	out := [4]byte{
		byte('0' + (n/1000)%10),
		byte('0' + (n/100)%10),
		byte('0' + (n/10)%10),
		byte('0' + n%10),
	}
	for i := 0; i < 3 && out[i] == '0'; i++ {
		out[i] = ' '
	}
	return out
}

// Err returns the first error, if any.
func (f *Frame) Err() error {
	return f.err
}

// Out returns the bytes written so far. The slice aliases the locked buffer.
func (f *Frame) Out() []byte {
	return f.buf[:f.n]
}
