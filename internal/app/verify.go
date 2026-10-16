package application

import (
	"crypto/subtle"
	"errors"
	"fmt"
)

// Quiz storage layout:
//
// Each slot i (0..23) is 9 bytes inside Workspace.Quiz: a length byte
// (0..8; 0 means "nothing entered yet for this slot") followed by up to
// 8 ASCII bytes that hold the operator's typed word. The trailing bytes
// are zeroed on overwrite so wipe(Quiz) returns the buffer to a clean
// state with no leftover characters from a longer earlier entry.
//
// Comparison:
//
// The operator's typed word is padded to 8 bytes with spaces and
// compared to the BIP-39 word at the pool's word index for that slot.
// The typed word occupies bytes 8..15 of the locked Workspace.Scratch;
// the expected word occupies bytes 0..7. Both ranges are wiped afterward.
// WordLookup.WriteWord scans every one of the 2048 wordlist entries
// (constant-time over the secret index), and crypto/subtle's compare inspects
// every byte of both 8-byte ranges. The total work per slot check is fixed,
// so timing does not leak which slot or how many slots disagree.

// quizRow returns the quizSlotRow-byte row of slot i inside Workspace.Quiz.
func (a *Runner) quizRow(i int) []byte {
	start := i * quizSlotRow
	return a.WS.Quiz[start : start+quizSlotRow]
}

// quizHas reports whether the operator has entered anything for slot i.
func (a *Runner) quizHas(i int) bool {
	return a.quizRow(i)[0] != 0
}

// quizWord returns the operator's typed word for slot i. The slice
// aliases Workspace.Quiz and is up to 8 bytes long; its lifetime is the
// lifetime of the current readLine / draw.
func (a *Runner) quizWord(i int) []byte {
	row := a.quizRow(i)
	n := int(row[0])
	return row[1 : 1+n]
}

// quizSet stores `word` (up to 8 bytes) into slot i and zero-fills the
// row's unused bytes so a stale overflow cannot leak a previous entry.
func (a *Runner) quizSet(i int, word []byte) {
	n := len(word)
	if n > 8 {
		n = 8
	}
	row := a.quizRow(i)
	row[0] = byte(n)
	copy(row[1:1+n], word)
	for k := 1 + n; k < quizSlotRow; k++ {
		row[k] = 0
	}
}

// quizClearAll wipes every slot.
func (a *Runner) quizClearAll() {
	wipe(a.WS.Quiz)
}

// quizMatchSlot reports whether the operator's typed word in slot i
// matches the BIP-39 word that the pool resolves to for word position
// i+1. The check is constant-time over the pool:
//
//   - MnemonicWordIndex reads the appropriate entropy bits without
//     branching on their values.
//   - WriteWord inspects every one of the 2048 wordlist rows.
//   - subtle.ConstantTimeCompare walks all 8 bytes of both operand
//     ranges.
//
// Worked examples: typed "abandon" (length 7) with i=0 against an
// all-zeros pool compares padded "abandon " to the lookup for index 0
// and returns true. Typed "abandonn" (length 8) compares padded
// "abandonn" to "abandon " and returns false on the trailing bytes.
func (a *Runner) quizMatchSlot(i int) bool {
	row := a.quizRow(i)
	n := int(row[0])
	typed := row[1 : 1+n]
	wordIndex := a.mnemonicWordIndex(i)
	scratch := a.WS.Scratch[:16:16]
	word := scratch[:8:8]
	padded := scratch[8:16:16]
	copy(padded, typed)
	for k := n; k < 8; k++ {
		padded[k] = ' '
	}
	a.Words.WriteWord(word, wordIndex)
	matches := subtle.ConstantTimeCompare(padded, word) == 1
	wipe(padded)
	wipe(word)
	return matches
}

// quizScreen runs the verification step:
//
//  1. Wipe the 24-slot quiz state; the next slot to fill is 1.
//  2. Draw the verify screen with the slot prompt on the message line
//     and `q = quit` as the command line.
//  3. Read a line. `q` -> ConfirmAbort. Empty Enter during entry ->
//     redraw without advancing. Any non-empty line is stored into the
//     current slot. The slot is then compared against the pool's BIP-39
//     word in constant time; on a match the cursor advances and the
//     next loop draws the next slot prompt. On rejection the message
//     line says "Word N was not correct. Type word N again:" with the
//     slot number.
//  4. When all 24 slots match, the command line switches to
//     `[n] Next: Finish`, and n returns nil to the
//     caller (the App's finalFlow then runs Done).
func (a *Runner) quizScreen() error {
	a.quizClearAll()
	nextSlot := 0
	msg := ""
	for {
		allMatch := nextSlot >= quizSlots
		cmds := verifyCmdsDuring
		if allMatch {
			cmds = verifyCmdsAll
		}
		// Message line: slot prompt by default; the all-match message
		// once every slot has matched.
		lineMsg := msg
		if lineMsg == "" {
			if allMatch {
				lineMsg = verifyMsgAllMatch
			} else {
				lineMsg = fmt.Sprintf(verifyMsgPrompt, nextSlot+1)
			}
		}
		if err := a.drawVerify(lineMsg, cmds); err != nil {
			return err
		}
		line, err := a.readLine()
		if errors.Is(err, ErrLineTooLong) {
			msg = msgTooLong
			continue
		}
		if err != nil {
			return err
		}
		c := letter(line)
		switch {
		case c == 'q':
			if err := a.confirmAbort(); err != nil {
				return err
			}
			msg = ""
		case allMatch:
			if c == 'n' {
				wipe(a.WS.Line)
				return nil
			}
			wipe(a.WS.Line)
			msg = msgNotValid
		case len(line) == 0:
			// Empty Enter during entry: redraw with the same prompt.
			wipe(a.WS.Line)
			msg = ""
		default:
			a.quizSet(nextSlot, line)
			wipe(a.WS.Line)
			if a.quizMatchSlot(nextSlot) {
				nextSlot++
				msg = "" // next loop builds the right prompt
				continue
			}
			msg = fmt.Sprintf(verifyMsgRejected, nextSlot+1, nextSlot+1)
		}
	}
}

// doneScreen waits for an empty Enter on the Done screen. Other input and
// overlong lines redraw it; a read error short-circuits to the caller.
// The Done screen has no `b` target and no `q` confirmation: once every
// slot has matched, exiting is the only step that remains.
func (a *Runner) doneScreen() error {
	for {
		if err := a.drawDone(); err != nil {
			return err
		}
		line, err := a.readLine()
		if errors.Is(err, ErrLineTooLong) {
			continue
		}
		if err != nil {
			return err
		}
		empty := len(line) == 0
		wipe(a.WS.Line)
		if empty {
			return nil
		}
	}
}
