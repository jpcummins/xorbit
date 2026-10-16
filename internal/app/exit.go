package application

// Source and terminal-check failure screens wait for Enter before the
// program leaves through the central exit path.

import "errors"

// exitScreen draws a fixed exit screen and waits for an empty Enter. Other
// input redraws the screen; a read error short-circuits back to the caller.
func (a *Runner) exitScreen(title, body string) error {
	for {
		if err := a.drawScreen(title, body, "", exitCommands); err != nil {
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
