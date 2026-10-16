package application

// This file holds the Runner type and top-level flow. Screen transitions and
// input handling live in dice.go, warning.go, confirm.go, reveal.go, verify.go,
// and exit.go. Frame rendering lives in view*.go; fixed screen text is in
// screens.go. Input parsing and configuration validation live in input.go
// and validate.go.

import "errors"

// Runner executes the operator-facing flow.
type Runner struct {
	Term    Terminal
	Display Display
	WS      *Workspace

	// Dice and Kernel are the fixed sources of entropy. The App runs the
	// dice screen first, then generates kernel entropy and shows the XOR mix.
	Dice   DiceSource
	Kernel KernelRNG
	Words  WordLookup // injected BIP-39 lookup; root does not import its package

	// CheckTerminal re-runs the hardening terminal scan (AGENTS.md
	// hardening step 11) before every input read and before the XOR display
	// and reveal. Wired to harden.CheckTerminal by main; required by Validate.
	CheckTerminal func() error
}

// Run drives the operator-facing flow:
//
//	What is xorbit? -> How it works -> Risks -> Acknowledgement
//	-> Dice -> generate Linux kernel RNG entropy -> XOR display
//	-> ConfirmReveal -> Reveal -> Verify -> Done
//
// It returns nil after Done is acknowledged, ErrAbort if the operator
// aborted, ErrEOF at end of input, ErrSourceFailed after showing the
// source-failed screen, and any other error on failure. The caller's
// central exit path wipes memory.
func (a *Runner) Run() error {
	if err := a.Validate(); err != nil {
		return err
	}
	if err := a.openingScreen(aboutTitle, aboutBody, aboutCommands); err != nil {
		return err
	}
	if err := a.openingScreen(howTitle, howBody, howCommands); err != nil {
		return err
	}
	if err := a.riskScreen(); err != nil {
		return err
	}
	if err := a.riskAcceptanceScreen(); err != nil {
		return err
	}
	if err := a.runDice(); err != nil {
		return a.showSourceFailure(err)
	}
	a.storeDiceContribution()

	if err := a.generateKernelEntropy(); err != nil {
		return a.showSourceFailure(err)
	}
	// The XOR screen displays both sensitive contributions and their result,
	// so re-scan the terminal immediately before drawing it.
	if err := a.scanTerminalOrFail(); err != nil {
		return err
	}
	if err := a.mixScreen(); err != nil {
		return err
	}
	if err := a.confirmRevealScreen(); err != nil {
		return err
	}
	// Re-scan after the reveal confirmation and immediately before the
	// one-time mnemonic-code display (AGENTS.md hardening step 11).
	if err := a.scanTerminalOrFail(); err != nil {
		return err
	}
	if err := a.revealScreen(); err != nil {
		return err
	}
	if err := a.quizScreen(); err != nil {
		return err
	}
	if err := a.doneScreen(); err != nil {
		return err
	}
	return nil
}

// showSourceFailure draws the fixed failure screen for an entropy-source
// failure. Other errors pass through unchanged.
func (a *Runner) showSourceFailure(err error) error {
	if !errors.Is(err, ErrSourceFailed) {
		return err
	}
	if drawErr := a.exitScreen(failedTitle, failedBody); drawErr != nil {
		return drawErr
	}
	return err
}

// scanTerminalOrFail rechecks the controlling terminal before sensitive
// display. On failure, the fixed error screen is shown and the caller
// returns the check error through the central exit path.
func (a *Runner) scanTerminalOrFail() error {
	if err := a.CheckTerminal(); err != nil {
		if drawErr := a.exitScreen(failedTitle, checkFailedBody); drawErr != nil {
			return drawErr
		}
		return err
	}
	return nil
}
