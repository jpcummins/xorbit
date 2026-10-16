package application

// The opening screens, Risks screen, and Acknowledgement are fixed-text
// gates before entropy is collected.

import "errors"

// openingScreen shows one of the fixed educational screens. n advances;
// q exits with ErrAbort; other input redraws with an invalid-choice message.
func (a *Runner) openingScreen(title, body, commands string) error {
	msg := ""
	for {
		if err := a.drawScreen(title, body, msg, commands); err != nil {
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
		switch c {
		case 'n':
			return nil
		case 'q':
			return ErrAbort
		default:
			msg = msgNotValid
		}
	}
}

// riskScreen is the detailed operator gate. It returns:
//
//	nil        operator chose "N" to continue to risk acknowledgement
//	ErrAbort   operator chose "Q" to quit
func (a *Runner) riskScreen() error {
	msg := ""
	for {
		if err := a.drawScreen(riskTitle, risksBody, msg, risksCommands); err != nil {
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
		switch c {
		case 'n':
			return nil
		case 'q':
			return ErrAbort
		default:
			msg = msgNotValid
		}
	}
}

// riskAcceptanceScreen requires A to acknowledge the risks before any
// entropy source starts. q exits without collecting entropy.
func (a *Runner) riskAcceptanceScreen() error {
	msg := ""
	for {
		if err := a.drawScreen(riskAcceptanceTitle, riskAcceptanceBody, msg, riskAcceptanceCommands); err != nil {
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
		case c == 'a':
			return nil
		case c == 'q':
			return ErrAbort
		default:
			msg = msgNotValid
		}
	}
}
