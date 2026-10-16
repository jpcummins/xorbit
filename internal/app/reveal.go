package application

import "errors"

// confirmRevealScreen asks before the one-time reveal. It shows fixed text
// only: no part of the entropy or the mnemonic code. n
// advances to Reveal; q goes through ConfirmAbort. There is no `b`: the
// pool cannot be un-mixed.
func (a *Runner) confirmRevealScreen() error {
	msg := ""
	for {
		if err := a.drawScreen(confirmRevealTitle, confirmRevealBody, msg, confirmRevealCmds); err != nil {
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
		wipe(a.WS.Line)
		switch {
		case c == 'n':
			return nil
		case c == 'q':
			if err := a.confirmAbort(); err != nil {
				return err
			}
			msg = ""
		default:
			msg = msgNotValid
		}
	}
}

// revealScreen shows the 24 words exactly once through Display.Show and waits
// for n. The frame is
// built in the locked Workspace.Frame and wiped right after it is shown;
// it is never drawn again.
func (a *Runner) revealScreen() error {
	if err := a.drawReveal(); err != nil {
		return err
	}
	for {
		line, readErr := a.readLine()
		if errors.Is(readErr, ErrLineTooLong) {
			continue
		}
		if readErr != nil {
			return readErr
		}
		command := letter(line)
		wipe(a.WS.Line)
		switch command {
		case 'n':
			return nil
		case 'q':
			// ConfirmAbort clears the screen. If quit is declined, proceed
			// to Verify instead of redrawing the one-time mnemonic frame.
			return a.confirmAbortAfterReveal()
		}
	}
}
