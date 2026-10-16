// Command xorbit guides the operator through generating a BIP-39
// mnemonic on the controlling terminal.
package main

import (
	"errors"
	"os"
	"sync"

	"xorbit/bip39"
	"xorbit/dice"
	"xorbit/harden"
	application "xorbit/internal/app"
	"xorbit/kernel"
	"xorbit/memlock"
	"xorbit/tty"
)

// Exit codes identify a failure category, never data.
const (
	exitOK        = 0
	exitFailure   = 1
	exitHardening = 2
	exitNoTTY     = 3
	exitSignal    = 4
	exitAborted   = 5
	exitUsage     = 6
)

// State the central exit path needs. It is written and read only while
// holding exitMu, because exit can run on the signal goroutine.
var (
	exitMu    sync.Mutex
	workspace *application.Workspace
	terminal  *tty.Terminal
)

// exit is the central exit path: wipe all sensitive memory, restore the
// screen, and exit with a fixed code. os.Exit skips deferred calls, so
// nothing here relies on defer. exitMu is never released: the first
// caller exits the process and any other caller blocks until it does.
func exit(code int) {
	exitMu.Lock()
	workspace.Wipe()
	if terminal != nil {
		terminal.Close()
	}
	os.Exit(code)
}

func setWorkspace(w *application.Workspace) {
	exitMu.Lock()
	workspace = w
	exitMu.Unlock()
}

func setTerminal(t *tty.Terminal) {
	exitMu.Lock()
	terminal = t
	exitMu.Unlock()
}

// exitCode maps the result of Runner.Run to an exit code.
func exitCode(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, application.ErrAbort), errors.Is(err, application.ErrEOF):
		return exitAborted
	default:
		return exitFailure
	}
}

func main() {
	defer func() {
		if recover() != nil {
			exit(exitFailure)
		}
	}()

	if harden.Run(func() { exit(exitSignal) }) != nil {
		exit(exitHardening)
	}

	if len(os.Args) > 1 {
		exit(exitUsage)
	}

	// The mix is fixed: dice rolls first, then Linux kernel RNG.
	diceSource := &dice.Source{}
	kernelRNG := &kernel.Source{}
	w, err := application.NewWorkspace(memlock.Allocator{})
	if err != nil {
		exit(exitFailure)
	}
	setWorkspace(w)

	t, err := tty.Open()
	if err != nil {
		exit(exitNoTTY)
	}
	setTerminal(t)

	runner := &application.Runner{
		Term:    t,
		Display: t,
		WS:      w,
		Dice:    diceSource,
		Kernel:  kernelRNG,
		Words:   bip39.Lookup{},
		// The terminal scan runs at startup, before every input read, and
		// before the XOR display and one-time reveal (AGENTS.md step 11).
		CheckTerminal: harden.CheckTerminal,
	}
	exit(exitCode(runner.Run()))
}
