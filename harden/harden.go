// Package harden applies Linux process hardening. It must run first in main.
package harden

import (
	"errors"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
)

const (
	prSetDumpable   = 4
	prSetNoNewPrivs = 38
)

// ErrHardening is the only error Run returns.
var ErrHardening = errors.New("hardening failed")

// runtimeVars are environment variables the Go runtime reads before main.
var runtimeVars = []string{
	"GOTRACEBACK", "GODEBUG", "GOGC", "GOMEMLIMIT", "GOMAXPROCS",
	"GORACE", "GOCOVERDIR", "GOFIPS140", "GOROOT",
}

// Run applies every hardening step in order and fails closed. onSignal is
// called from a dedicated goroutine when a termination signal arrives; it
// must not return (it should be the central exit path).
func Run(onSignal func()) error {
	// 1. Refuse runtime environment variables.
	for _, kv := range os.Environ() {
		for _, name := range runtimeVars {
			if strings.HasPrefix(kv, name+"=") {
				return ErrHardening
			}
		}
	}

	// 2. Silence stdout and stderr.
	devnull, err := syscall.Open("/dev/null", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return ErrHardening
	}
	if syscall.Dup3(devnull, 1, 0) != nil || syscall.Dup3(devnull, 2, 0) != nil {
		syscall.Close(devnull)
		return ErrHardening
	}
	if syscall.Close(devnull) != nil {
		return ErrHardening
	}

	// 3. Not dumpable.
	if _, _, e := syscall.RawSyscall(syscall.SYS_PRCTL, prSetDumpable, 0, 0); e != 0 {
		return ErrHardening
	}

	// 4. No new privileges.
	if _, _, e := syscall.RawSyscall(syscall.SYS_PRCTL, prSetNoNewPrivs, 1, 0); e != 0 {
		return ErrHardening
	}

	// 5. No core dumps.
	if syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0}) != nil {
		return ErrHardening
	}

	// 6. No swap: handled per buffer by the memlock allocator.

	// 7. Minimal tracebacks.
	debug.SetTraceback("none")

	// 8. Signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		defer func() {
			if recover() != nil {
				onSignal()
			}
		}()
		<-sigs
		onSignal()
	}()

	// 9. Umask.
	syscall.Umask(0o077)

	// 10. No debugger attached.
	if CheckTraced() != nil {
		return ErrHardening
	}

	// 11. Terminal scan.
	if CheckTerminal() != nil {
		return ErrHardening
	}

	return nil
}
