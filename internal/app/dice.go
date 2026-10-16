package application

// Dice flow and entropy mixing. The App owns screen transitions, input,
// quit confirmation, source order, and mixing.

import "errors"

// runDice drives the fixed interactive dice source. On success, its 32-byte
// contribution remains in Contrib for the caller to mix into Pool. Dice
// scratch is wiped on every way out.
func (a *Runner) runDice() error {
	scratch := a.WS.DiceScratch
	if err := a.Dice.Start(scratch); err != nil {
		wipe(a.WS.Contrib)
		wipe(scratch)
		return ErrSourceFailed
	}
	msg := ""
	for {
		if err := a.drawDiceScreen(msg); err != nil {
			wipe(scratch)
			return err
		}
		line, err := a.readLine()
		if errors.Is(err, ErrLineTooLong) {
			msg = msgTooLong
			continue
		}
		if err != nil {
			wipe(scratch)
			return err
		}
		// Handle q before passing other input to the dice source.
		if letter(line) == 'q' {
			wipe(a.WS.Line)
			if err := a.confirmAbort(); err != nil {
				wipe(scratch)
				return err
			}
			msg = ""
			continue
		}
		step := a.Dice.Input(line)
		wipe(a.WS.Line)
		msg = step.Message
		if !step.Done {
			continue
		}
		return a.finishDice(scratch)
	}
}

// finishDice writes the dice contribution and wipes its scratch buffer. A
// failed source cannot contribute partial bytes to the pool.
func (a *Runner) finishDice(scratch []byte) error {
	if err := a.Dice.Finish(a.WS.Contrib); err != nil {
		wipe(a.WS.Contrib)
		wipe(scratch)
		return ErrSourceFailed
	}
	wipe(scratch)
	return nil
}

// generateKernelEntropy reads the second contribution into Contrib. A
// failed read is wiped and reported as a fixed source failure.
func (a *Runner) generateKernelEntropy() error {
	if err := a.Kernel.Generate(a.WS.Contrib); err != nil {
		wipe(a.WS.Contrib)
		return ErrSourceFailed
	}
	return nil
}

// storeDiceContribution copies the dice contribution into the initially zero
// Pool and wipes Contrib. The kernel contribution is XORed later on the checked
// XOR screen.
func (a *Runner) storeDiceContribution() {
	copy(a.WS.Pool, a.WS.Contrib)
	wipe(a.WS.Contrib)
}

// mixScreen displays the dice digest, kernel RNG bytes, and their XOR result
// on the checked TTY. n commits the mixed value to Pool; q follows the usual
// quit-confirmation path.
func (a *Runner) mixScreen() error {
	copy(a.WS.Mixed, a.WS.Pool)
	xorInto(a.WS.Mixed, a.WS.Contrib)
	msg := ""
	for {
		if err := a.drawMixScreen(msg); err != nil {
			wipe(a.WS.Mixed)
			return err
		}
		line, err := a.readLine()
		if errors.Is(err, ErrLineTooLong) {
			msg = msgTooLong
			continue
		}
		if err != nil {
			wipe(a.WS.Mixed)
			return err
		}
		c := letter(line)
		wipe(a.WS.Line)
		switch c {
		case 'n':
			copy(a.WS.Pool, a.WS.Mixed)
			wipe(a.WS.Mixed)
			wipe(a.WS.Contrib)
			return nil
		case 'q':
			if err := a.confirmAbort(); err != nil {
				wipe(a.WS.Mixed)
				return err
			}
			msg = ""
		default:
			msg = msgNotValid
		}
	}
}
