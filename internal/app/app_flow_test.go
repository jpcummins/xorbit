package application_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"xorbit/bip39"
	application "xorbit/internal/app"
	"xorbit/mock"
)

// flowRig wires an App for flow tests: a scripted terminal, a recording
// display, a mock dice source that completes on n, and a mock kernel RNG.
type flowRig struct {
	app     *application.Runner
	frames  *[]string
	shown   [][]byte
	dice    *mock.Dice
	kernel  *mock.Kernel
	display *mock.Display
	reads   int
}

func newFlowRig(t *testing.T, steps ...step) *flowRig {
	t.Helper()
	frames := []string{}
	var ws *application.Workspace
	var r *flowRig
	term := &mock.Terminal{
		DrawFn: func(frame []byte) error {
			frames = append(frames, string(frame))
			return nil
		},
		ReadLineFn: func(dst []byte) (int, error) {
			r.reads++
			if ws != nil {
				for _, b := range ws.Frame {
					if b != 0 {
						t.Error("screen frame must be wiped before the next input")
						break
					}
				}
			}
			if len(steps) == 0 {
				t.Fatal("script exhausted")
			}
			s := steps[0]
			steps = steps[1:]
			if s.err != nil {
				return 0, s.err
			}
			return copy(dst, s.line), nil
		},
	}
	r = &flowRig{frames: &frames}
	display := &mock.Display{ShowFn: func(result []byte) error {
		r.shown = append(r.shown, append([]byte(nil), result...))
		return nil
	}}
	r.display = display
	doneOnNext := func(line []byte) application.DiceStep {
		if len(line) == 1 && (line[0] == 'n' || line[0] == 'N') {
			return application.DiceStep{Done: true}
		}
		return application.DiceStep{}
	}
	r.dice = &mock.Dice{InputFn: doneOnNext}
	r.kernel = &mock.Kernel{}
	r.app = &application.Runner{
		Term:          term,
		Display:       display,
		WS:            newWorkspace(t),
		Dice:          r.dice,
		Kernel:        r.kernel,
		Words:         bip39.Lookup{},
		CheckTerminal: func() error { return nil },
	}
	ws = r.app.WS
	return r
}

// stepsToVerify returns the script that takes the flow from the start
// to the verify screen with an all-zero pool: opening screens, Risks,
// acceptance, the dice screen, XOR display, ConfirmReveal, and Reveal.
func stepsToVerify() []step {
	return []step{{line: "n"}, {line: "n"}, {line: "n"}, {line: "a"},
		{line: "n"}, {line: "n"}, {line: "n"}, {line: "n"}}
}

// zeroPoolWords are the mnemonic words for 256 zero bits: "abandon" 23
// times, followed by "art".
func zeroPoolWords() []step {
	steps := make([]step, 0, 24)
	for i := 0; i < 23; i++ {
		steps = append(steps, step{line: "abandon"})
	}
	return append(steps, step{line: "art"})
}

// TestHappyPath drives the complete flow to Done with an all-zero pool.
func TestHappyPath(t *testing.T) {
	steps := append(stepsToVerify(), zeroPoolWords()...)
	steps = append(steps, step{line: "n"}, enter) // all-match Next, then Done ack
	r := newFlowRig(t, steps...)
	if err := r.app.Run(); err != nil {
		t.Fatalf("happy path must return nil, got %v", err)
	}
	if !r.display.ShowInvoked || len(r.shown) != 1 {
		t.Fatalf("Reveal must be shown exactly once, shown %d times", len(r.shown))
	}
	reveal := string(r.shown[0])
	for _, w := range []string{"abandon", "art"} {
		if !strings.Contains(reveal, w) {
			t.Errorf("reveal frame must contain %q", w)
		}
	}
	if !strings.Contains(reveal, "[n] Next: Verify your copy") {
		t.Error("Reveal must explain that verification is next")
	}
	if !strings.Contains(reveal, "[q] Quit") {
		t.Error("Reveal must offer the quit command")
	}
	joined := strings.Join(*r.frames, "\x00")
	for _, want := range []string{
		"xorbit - v0.0.0-dev",
		"What is xorbit?",
		"How it works",
		"Risks",
		"Acknowledgement",
		"XOR mix",
		"BIP-39 Mnemonic Code",
		"Confirm your 24-word BIP-39 mnemonic code",
		"You're done",
		"[n] Next: How it works",
		"[n] Next: Risks",
		"[n] Next: Acknowledgement",
		"[a] Accept and continue",
		"Dice SHA-256:",
		"Linux kernel RNG:",
		"Dice XOR Linux RNG:",
		"[n] Next: Review mnemonic code instructions",
		"[q] Quit",
		"[n] Next: Reveal mnemonic code",
		"[n] Next: Finish",
		"Don't trust, verify.",
		"Read the code yourself; an AI assistant can help explain it.",
		"This program displays sensitive private-key data",
		"When you exit, the program clears its sensitive buffers.",
		"those 256 mixed bits. The last 8 bits are a checksum.",
		"[↵] Exit",
		"Word 1 of 24: enter it below.",
		"All 24 slots match.",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("screens must include %q", want)
		}
	}
	if r.dice.FinishCalls != 1 || r.kernel.GenerateCalls != 1 {
		t.Errorf("dice must finish and kernel RNG must generate once: %d, %d",
			r.dice.FinishCalls, r.kernel.GenerateCalls)
	}
}

// The pool is the bytewise XOR of the dice and kernel contributions.
func TestPoolIsXOROfSources(t *testing.T) {
	steps := append(stepsToVerify()[:6], step{line: "q"}, step{line: "y"})
	r := newFlowRig(t, steps...)
	r.dice.FinishFn = func(dst []byte) error {
		for i := range dst {
			dst[i] = 0x01
		}
		return nil
	}
	r.kernel.GenerateFn = func(dst []byte) error {
		for i := range dst {
			dst[i] = 0x02
		}
		return nil
	}
	if err := r.app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if !bytes.Equal(r.app.WS.Pool, bytes.Repeat([]byte{0x03}, 32)) {
		t.Errorf("pool = %x, want 0x03 x 32", r.app.WS.Pool)
	}
	joined := strings.Join(*r.frames, "\x00")
	for _, want := range []string{
		strings.Repeat("01", 32),
		strings.Repeat("02", 32),
		strings.Repeat("03", 32),
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("XOR screen must show the deterministic fixture %q", want)
		}
	}
}

// A Start error shows the fixed failure screen and returns ErrSourceFailed.
func TestDiceStartFailure(t *testing.T) {
	r := newFlowRig(t, step{line: "n"}, step{line: "n"}, step{line: "n"}, step{line: "a"}, enter)
	r.dice.StartErr = errors.New("no such device")
	if err := r.app.Run(); !errors.Is(err, application.ErrSourceFailed) {
		t.Fatalf("want ErrSourceFailed, got %v", err)
	}
	joined := strings.Join(*r.frames, "\x00")
	if !strings.Contains(joined, "Something went wrong") ||
		!strings.Contains(joined, "That source failed. No 24-word BIP-39 mnemonic code was created.") {
		t.Errorf("the source-failed screen must be shown")
	}
	if r.dice.FinishCalls != 0 {
		t.Error("Finish must not run after a failed Start")
	}
}

func TestDiceFinishFailure(t *testing.T) {
	r := newFlowRig(t, step{line: "n"}, step{line: "n"}, step{line: "n"}, step{line: "a"},
		step{line: "n"}, enter)
	r.dice.FinishErr = application.ErrSourceFailed
	if err := r.app.Run(); !errors.Is(err, application.ErrSourceFailed) {
		t.Fatalf("want ErrSourceFailed, got %v", err)
	}
	joined := strings.Join(*r.frames, "\x00")
	if !strings.Contains(joined, "That source failed. No 24-word BIP-39 mnemonic code was created.") {
		t.Errorf("the source-failed screen must be shown")
	}
	if !bytes.Equal(r.app.WS.Pool, make([]byte, 32)) {
		t.Error("nothing must be mixed in after a failed Finish")
	}
}

func TestKernelGenerationFailure(t *testing.T) {
	r := newFlowRig(t, step{line: "n"}, step{line: "n"}, step{line: "n"}, step{line: "a"}, step{line: "n"}, enter)
	r.kernel.GenerateErr = errors.New("kernel RNG unavailable")
	if err := r.app.Run(); !errors.Is(err, application.ErrSourceFailed) {
		t.Fatalf("want ErrSourceFailed, got %v", err)
	}
	joined := strings.Join(*r.frames, "\x00")
	if !strings.Contains(joined, "That source failed. No 24-word BIP-39 mnemonic code was created.") ||
		strings.Contains(joined, "\nXOR mix\n") {
		t.Error("kernel generation failure must fail closed before the XOR screen")
	}
	if !bytes.Equal(r.app.WS.Contrib, make([]byte, len(r.app.WS.Contrib))) {
		t.Error("failed kernel contribution must be wiped")
	}
}

// q on the dice screen asks first; declining redraws the dice screen.
func TestAbortDeclinedOnDiceScreen(t *testing.T) {
	steps := []step{{line: "n"}, {line: "n"}, {line: "n"}, {line: "a"},
		{line: "q"}, {line: "n"}, // declined
		{line: "q"}, {line: "y"}, // confirmed
	}
	r := newFlowRig(t, steps...)
	checks := 0
	r.app.CheckTerminal = func() error {
		checks++
		return nil
	}
	if err := r.app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if checks != 8 {
		t.Errorf("terminal scan calls = %d, want one before each of 8 input reads", checks)
	}
	quitCount := 0
	diceScreenCount := 0
	for _, f := range *r.frames {
		if strings.Contains(f, "Quit?\n") {
			quitCount++
		}
		if strings.Contains(f, "\nRoll Dice\n") {
			diceScreenCount++
		}
	}
	if quitCount != 2 {
		t.Errorf("ConfirmAbort must be shown twice, got %d", quitCount)
	}
	if diceScreenCount != 2 {
		t.Errorf("the dice screen must be drawn once and redrawn after a decline, got %d draws", diceScreenCount)
	}
}

func TestAbortFromConfirmReveal(t *testing.T) {
	steps := append(stepsToVerify()[:6], step{line: "q"}, step{line: "y"})
	r := newFlowRig(t, steps...)
	if err := r.app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if r.display.ShowInvoked {
		t.Error("Reveal must not be shown after an abort")
	}
}

func TestAbortFromReveal(t *testing.T) {
	steps := append(stepsToVerify()[:7], step{line: "q"}, step{line: "y"})
	r := newFlowRig(t, steps...)
	if err := r.app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if len(r.shown) != 1 {
		t.Fatalf("Reveal must be shown exactly once, got %d displays", len(r.shown))
	}
	joined := strings.Join(*r.frames, "\x00")
	for _, want := range []string{
		"The 24-word BIP-39 mnemonic code has already been shown",
		"[n] No, continue to Verify",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("reveal quit confirmation must include %q", want)
		}
	}
}

func TestDeclineQuitFromRevealContinuesToVerifyOnce(t *testing.T) {
	steps := append(stepsToVerify()[:7], step{line: "q"}, step{line: "n"})
	steps = append(steps, zeroPoolWords()...)
	steps = append(steps, step{line: "n"}, enter)
	r := newFlowRig(t, steps...)
	if err := r.app.Run(); err != nil {
		t.Fatalf("declining quit at Reveal should continue to Verify: %v", err)
	}
	if len(r.shown) != 1 {
		t.Fatalf("Reveal must be shown exactly once, got %d displays", len(r.shown))
	}
	joined := strings.Join(*r.frames, "\x00")
	if !strings.Contains(joined, "Confirm your 24-word BIP-39 mnemonic code") {
		t.Error("declining quit should continue to Verify")
	}
}

func TestAbortFromVerify(t *testing.T) {
	steps := append(stepsToVerify(), step{line: "q"}, step{line: "y"})
	r := newFlowRig(t, steps...)
	if err := r.app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
}

// A failed terminal re-scan before the reveal fails closed: the fixed
// failure screen, the error, and no reveal (AGENTS.md step 11).
func TestTerminalRecheckFailure(t *testing.T) {
	steps := append(stepsToVerify()[:6], step{line: "n"}, enter)
	r := newFlowRig(t, steps...)
	errCheck := errors.New("terminal held by another process")
	checks := 0
	r.app.CheckTerminal = func() error {
		checks++
		if checks == 9 {
			return errCheck
		}
		return nil
	}
	if err := r.app.Run(); !errors.Is(err, errCheck) {
		t.Fatalf("want the check's error, got %v", err)
	}
	joined := strings.Join(*r.frames, "\x00")
	if !strings.Contains(joined, "The terminal security check failed. No 24-word BIP-39 mnemonic code was") {
		t.Error("the check-failed screen must be shown")
	}
	if !strings.Contains(joined, "\nBIP-39 Mnemonic Code\n") || checks != 10 {
		t.Error("the terminal must be rechecked after confirmation and immediately before reveal")
	}
	if r.display.ShowInvoked {
		t.Error("Reveal must not be shown after a failed terminal check")
	}
}

func TestTerminalCheckBeforeXORFailure(t *testing.T) {
	steps := append(stepsToVerify()[:5], enter)
	r := newFlowRig(t, steps...)
	errCheck := errors.New("terminal held by another process")
	checks := 0
	r.app.CheckTerminal = func() error {
		checks++
		if checks == 6 {
			return errCheck
		}
		return nil
	}
	if err := r.app.Run(); !errors.Is(err, errCheck) {
		t.Fatalf("want the terminal-check error, got %v", err)
	}
	joined := strings.Join(*r.frames, "\x00")
	if !strings.Contains(joined, "The terminal security check failed. No 24-word BIP-39 mnemonic code was") ||
		strings.Contains(joined, "\nXOR mix\n") || checks != 7 {
		t.Error("a failed pre-mix terminal check must prevent the XOR screen")
	}
}

func TestTerminalCheckFailurePreventsInputRead(t *testing.T) {
	r := newFlowRig(t, step{line: "n"})
	errCheck := errors.New("terminal held by another process")
	checks := 0
	r.app.CheckTerminal = func() error {
		checks++
		return errCheck
	}
	if err := r.app.Run(); !errors.Is(err, errCheck) {
		t.Fatalf("want the terminal-check error, got %v", err)
	}
	if checks != 1 || r.reads != 0 {
		t.Errorf("checks = %d and input reads = %d, want 1 and 0", checks, r.reads)
	}
}

// These worked examples cover quizMatchSlot's constant-time comparison for
// word 1; the rest of the zero-pool mnemonic also exercises word 2 and word 24.
func TestVerifySlotComparisonFixtures(t *testing.T) {
	cases := []struct {
		name       string
		firstWord  string
		wantReject bool
	}{
		{"matching first word", "abandon", false},
		{"extra letter rejected", "abandonn", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := append(stepsToVerify(), step{line: tc.firstWord})
			remainingWords := zeroPoolWords()[1:]
			if tc.wantReject {
				remainingWords = zeroPoolWords()
			}
			steps = append(steps, remainingWords...)
			steps = append(steps, step{line: "n"}, enter)
			r := newFlowRig(t, steps...)
			if err := r.app.Run(); err != nil {
				t.Fatalf("want nil, got %v", err)
			}
			if !bytes.Equal(r.app.WS.Scratch, make([]byte, len(r.app.WS.Scratch))) {
				t.Error("quiz comparison scratch must be wiped after every slot check")
			}
			joined := strings.Join(*r.frames, "\x00")
			gotReject := strings.Contains(joined, "Word 1 was not correct. Type word 1 again:")
			if gotReject != tc.wantReject {
				t.Errorf("rejection feedback = %t, want %t", gotReject, tc.wantReject)
			}
		})
	}
}

// Buffer hygiene, mirroring the central exit path: after a full run and
// a Wipe, every sensitive buffer is zero.
func TestWipeAfterRun(t *testing.T) {
	steps := append(stepsToVerify(), zeroPoolWords()...)
	steps = append(steps, step{line: "n"}, enter)
	r := newFlowRig(t, steps...)
	if err := r.app.Run(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	r.app.WS.Wipe()
	for name, b := range map[string][]byte{
		"Pool": r.app.WS.Pool, "Contrib": r.app.WS.Contrib, "Mixed": r.app.WS.Mixed,
		"Line": r.app.WS.Line, "Frame": r.app.WS.Frame, "DiceScratch": r.app.WS.DiceScratch,
		"Quiz": r.app.WS.Quiz,
	} {
		if !bytes.Equal(b, make([]byte, len(b))) {
			t.Errorf("%s must be wiped", name)
		}
	}
}

func TestValidate(t *testing.T) {
	goodWS := newWorkspace(t)
	check := func() error { return nil }
	badWS := newWorkspace(t)
	badWS.DiceScratch = badWS.DiceScratch[:len(badWS.DiceScratch)-1]
	cases := []struct {
		name   string
		dice   application.DiceSource
		kernel application.KernelRNG
		words  application.WordLookup
		ws     *application.Workspace
		check  func() error
	}{
		{"missing dice", nil, &mock.Kernel{}, bip39.Lookup{}, goodWS, check},
		{"missing kernel", &mock.Dice{}, nil, bip39.Lookup{}, goodWS, check},
		{"missing word lookup", &mock.Dice{}, &mock.Kernel{}, nil, goodWS, check},
		{"missing workspace", &mock.Dice{}, &mock.Kernel{}, bip39.Lookup{}, nil, check},
		{"missing terminal check", &mock.Dice{}, &mock.Kernel{}, bip39.Lookup{}, goodWS, nil},
		{"wrong dice scratch size", &mock.Dice{}, &mock.Kernel{}, bip39.Lookup{}, badWS, check},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := &application.Runner{
				WS: c.ws, Dice: c.dice, Kernel: c.kernel,
				Words: c.words, CheckTerminal: c.check,
			}
			if err := app.Validate(); !errors.Is(err, application.ErrConfig) {
				t.Errorf("want ErrConfig, got %v", err)
			}
		})
	}
}

func TestValidateGood(t *testing.T) {
	app := &application.Runner{
		WS:            newWorkspace(t),
		Dice:          &mock.Dice{},
		Kernel:        &mock.Kernel{},
		Words:         bip39.Lookup{},
		CheckTerminal: func() error { return nil },
	}
	if err := app.Validate(); err != nil {
		t.Errorf("a valid configuration must pass, got %v", err)
	}
}
