// Package dice implements the dice entropy source: at least 99 rolls are
// required, and SHA-256 is updated as rolls are typed.
//
// The operator rolls one or more fair six-sided dice and types the rolls.
// Each accepted roll is fed into SHA-256 as it is typed (streaming); the
// rolls themselves are never stored, so there is no upper bound on rolls.
// There is no undo. A mistyped roll is still fresh entropy; starting over
// is cheap. The 256-bit digest is the dice contribution, mixed (XOR)
// with the kernel's bits by the App.
//
// The SHA-256 input is exactly the ASCII characters '1'..'6' in entry
// order; spaces and tabs are ignored, and no newline is included. The
// operator can verify the review digest with: printf '111...1' | sha256sum.
package dice

import (
	"crypto/sha256"
	"hash"
	"runtime"

	"xorbit"
	app "xorbit/internal/app"
)

// Scratch-buffer layout.
const (
	minRolls    = xorbit.DiceMinimumRolls
	windowSize  = app.DiceWindowSize
	digestStart = windowSize
	scratchSize = app.DiceScratchSize
)

// Source is the dice entropy source.
//
// Mutable state on the struct:
//
//	h        the SHA-256 digest the rolls are streamed into. Created on
//	         the first Start and reused across runs (Reset in Start and
//	         Finish). The digest struct is runtime-managed heap memory
//	         the program cannot mlock. TailsOS has no swap; per-buffer
//	         locking is defense in depth.
//	scratch  the locked scratch buffer passed in Start (length 57);
//	         not owned by the source, wiped on mix-in and on every exit
//	         path by Workspace.Wipe().
//	count    rolls accepted so far. Public: it is shown on screen.
//	review   true once the operator pressed Enter at >= 99 rolls and
//	         the review screen (with the digest) is shown.
type Source struct {
	h       hash.Hash
	scratch []byte
	count   int
	review  bool
}

// windowOf and digestOf name the two regions of scratch.
func windowOf(scratch []byte) []byte { return scratch[:windowSize] }
func digestOf(scratch []byte) []byte { return scratch[digestStart:scratchSize] }

// View returns the current dice screen state. Its byte slices alias locked
// scratch and are used only by the renderer for this screen draw.
func (s *Source) View() app.DiceView {
	view := app.DiceView{
		RollCount: s.count,
		Review:    s.review,
	}
	if s.scratch != nil {
		view.Window = windowOf(s.scratch)
		view.Digest = digestOf(s.scratch)
	}
	return view
}

// Start prepares the dice screen. It zeroes the scratch, creates
// or resets the digest, and sets count and review to zero. It does not
// read entropy: the rolls come from the operator's keyboard.
func (s *Source) Start(scratch []byte) error {
	if len(scratch) != scratchSize {
		// Programmer error: Workspace should have allocated exactly
		// DiceScratchSize bytes.
		return app.ErrConfig
	}
	clear(scratch)
	if s.h == nil {
		s.h = sha256.New()
	} else {
		s.h.Reset()
	}
	s.scratch = scratch
	s.count = 0
	s.review = false
	runtime.KeepAlive(scratch)
	return nil
}

// Input handles one line from the dice screen.
//
// line is the trimmed operator input. Roll lines contain digits 1-6 and
// optional whitespace. Commands r, b, and n manage digest review.
//
// It updates the running SHA-256 digest with accepted roll digits and
// maintains the public roll count.
//
// Algorithm:
//  1. In entry state, r at 99 or more rolls enters digest review. An empty
//     line at the minimum asks the operator to use r. The digest is copied
//     into scratch for display; hash.Hash.Sum preserves the running state.
//  2. In review state, b returns to entry and n returns Done so the App can
//     finish the dice source. Other input stays in review.
//  3. Roll lines are validated character by character, then each digit is
//     fed into SHA-256 and pushed into the 25-roll display window.
//
// Worked example: after 99 rolls of '1', the review screen and Finish show
//
//	fa098eb852b2660348b21bb00ad03a49cc177ea07ebe34f46b40baa85313525e
//
// which any sha256sum of the 99 characters '1' reproduces.
//
// Constant time: per accepted byte the work is one fixed-size Write, and
// SHA-256's block function is data-oblivious. Branches are on input class,
// public count, or explicit commands, never on a roll's value.
func (s *Source) Input(line []byte) app.DiceStep {
	if s.h == nil || s.scratch == nil {
		return app.DiceStep{Message: msgNotReady}
	}
	if s.review {
		switch {
		case isCommand(line, 'b'):
			s.review = false
			return app.DiceStep{}
		case isCommand(line, 'n'):
			return app.DiceStep{Done: true}
		default:
			return app.DiceStep{Message: msgReviewChoice}
		}
	}
	if isCommand(line, 'r') {
		if s.count < minRolls {
			return app.DiceStep{Message: msgNeedMore}
		}
		s.enterReview()
		return app.DiceStep{}
	}
	if len(line) == 0 {
		if s.count >= minRolls {
			return app.DiceStep{Message: msgReviewCommand}
		}
		return app.DiceStep{Message: msgNeedMore}
	}
	badChar := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == ' ' || c == '\t' {
			continue
		}
		if c < '1' || c > '6' {
			badChar = true
			break
		}
	}
	if badChar {
		return app.DiceStep{Message: msgBadChar}
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == ' ' || c == '\t' {
			continue
		}
		s.h.Write(line[i : i+1])
		s.pushRoll(c)
	}
	return app.DiceStep{}
}

func (s *Source) enterReview() {
	s.h.Sum(digestOf(s.scratch)[:0])
	s.review = true
}

func isCommand(line []byte, want byte) bool {
	if len(line) != 1 {
		return false
	}
	c := line[0]
	if 'A' <= c && c <= 'Z' {
		c += 'a' - 'A'
	}
	return c == want
}

// pushRoll appends one roll to the display window and to the count.
// While fewer than 25 rolls are in, the roll takes the next free slot;
// from 25 rolls on, the window slides left by one (a fixed 24 byte
// moves, independent of the roll's value) so it always shows the last
// 25 rolls, oldest first.
func (s *Source) pushRoll(c byte) {
	w := windowOf(s.scratch)
	if s.count >= windowSize {
		copy(w[:windowSize-1], w[1:])
		w[windowSize-1] = c
	} else {
		w[s.count] = c
	}
	s.count++
	runtime.KeepAlive(s.scratch)
}

// Finish writes the dice contribution. It requires the review state
// and at least 99 rolls (both enforced by Input), writes the digest into
// dst, and resets the digest so the long-lived hash.Hash does not
// retain the run's state. The value is recomputed from the hash state
// (not copied from the review display), so Finish is independent of
// whatever the screen showed; they are the same value.
func (s *Source) Finish(dst []byte) error {
	if len(dst) != 32 || s.h == nil || !s.review || s.count < minRolls {
		return app.ErrConfig
	}
	s.h.Sum(dst[:0])
	s.h.Reset()
	runtime.KeepAlive(dst)
	return nil
}

// Fixed user-facing messages returned in DiceStep.Message.
const (
	msgBadChar       = "Rejected: use only the digits 1-6 and spaces. That line was not kept."
	msgNeedMore      = "At least 99 rolls are needed before you can finish. Keep rolling."
	msgReviewChoice  = "Choose [b] to return to rolls or [n] to continue."
	msgReviewCommand = "Type [r] to review the roll digest."
	msgNotReady      = "Source unavailable."
)
