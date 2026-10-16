// Package application owns the terminal, display, and allocator ports used
// by the application flow.
package application

import "errors"

// Terminal is the operator's screen and keyboard. Only the App uses it.
type Terminal interface {
	// Draw clears the screen and writes frame: ASCII text with '\n' line
	// ends, at most FrameCols x FrameRows.
	Draw(frame []byte) error
	// ReadLine reads one line of input into dst, without the newline, and
	// returns its length. It returns ErrLineTooLong if the line did not fit
	// (the rest of the line is discarded) and ErrEOF at end of input.
	ReadLine(dst []byte) (int, error)
}

// Display shows the final result to the operator exactly once.
type Display interface {
	Show(result []byte) error
}

// Allocator provides zeroed, swap-protected heap buffers of exactly n bytes.
type Allocator interface {
	Alloc(n int) ([]byte, error)
}

// Sentinel errors. None of them carries data.
var (
	ErrAbort         = errors.New("aborted by operator")
	ErrEOF           = errors.New("end of input")
	ErrLineTooLong   = errors.New("input line too long")
	ErrFrameOverflow = errors.New("frame exceeds screen size")
	ErrFrameText     = errors.New("frame text is not printable ASCII")
	ErrConfig        = errors.New("invalid program configuration")
)
