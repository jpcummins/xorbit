package application

// Quit confirmation is fixed text. Only y confirms; anything else declines,
// so callers can choose the appropriate next screen without losing work.

import "errors"

// confirmAbort returns ErrAbort if the operator confirms, nil if they
// decline, and any read or draw error unchanged.
func (a *Runner) confirmAbort() error {
	return a.confirmAbortWith(abortBody, abortInstruction)
}

// confirmAbortAfterReveal handles quit from the one-time reveal. Declining
// quit continues to Verify because the reveal frame must never be redrawn.
func (a *Runner) confirmAbortAfterReveal() error {
	return a.confirmAbortWith(revealAbortBody, revealAbortInstruction)
}

// confirmAbortWith renders a fixed quit question with the supplied text.
func (a *Runner) confirmAbortWith(body, instruction string) error {
	yes, err := a.confirmYN(abortTitle, body, instruction)
	if err != nil {
		return err
	}
	if yes {
		return ErrAbort
	}
	return nil
}

// confirmYN reads one line for a yes/no confirmation. Only y (either case)
// is yes; anything else, including an empty or too-long line, declines.
func (a *Runner) confirmYN(title, question, instruction string) (bool, error) {
	if err := a.drawConfirm(title, question, instruction); err != nil {
		return false, err
	}
	line, err := a.readLine()
	if errors.Is(err, ErrLineTooLong) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	yes := letter(line) == 'y'
	wipe(a.WS.Line)
	return yes, nil
}
