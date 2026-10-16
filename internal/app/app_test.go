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

// step is one scripted answer from the operator: a line, or an error.
type step struct {
	line string
	err  error
}

// scriptedApp returns an App whose terminal answers ReadLine with steps in
// order and records a copy of every frame drawn. The App is configured
// with mock dice and kernel adapters so Validate passes; tests that never
// reach the entropy flow ignore them.
func scriptedApp(t *testing.T, steps ...step) (*application.Runner, *[]string) {
	t.Helper()
	frames := []string{}
	term := &mock.Terminal{
		DrawFn: func(frame []byte) error {
			frames = append(frames, string(frame))
			return nil
		},
		ReadLineFn: func(dst []byte) (int, error) {
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
	app := &application.Runner{
		Term:          term,
		WS:            newWorkspace(t),
		Dice:          &mock.Dice{},
		Kernel:        &mock.Kernel{},
		Words:         bip39.Lookup{},
		CheckTerminal: func() error { return nil },
	}
	return app, &frames
}

var rule80 = strings.Repeat("=", 80) + "\n"

const testHeader = "xorbit - v0.0.0-dev\n"

func screenFrame(title, body, msg, commands string) string {
	frame := testHeader + rule80 + "\n" + title + "\n\n" + body
	if msg != "" {
		frame += msg + "\n"
	}
	return frame + "\n" + commands + "\n"
}

// aboutFrame returns the opening screen with the given message line.
func aboutFrame(msg string) string {
	return screenFrame("What is xorbit?",
		"xorbit creates a 24-word BIP-39 mnemonic code by mixing\n"+
			"entropy from physical dice rolls with entropy from the Linux kernel.\n", msg,
		"[n] Next: How it works\n[q] Quit")
}

// howFrame returns the instructions screen with the given message line.
func howFrame(msg string) string {
	return screenFrame("How it works",
		"1. Generate Dice entropy: roll one or more fair six-sided dice at least\n"+
			"   99 times; enter each result (1-6). Digits stream into SHA-256.\n"+
			"2. Generate RNG entropy: 256 bits come from the Linux kernel RNG.\n"+
			"3. Mix the entropy: combine two independently derived sources of\n"+
			"   entropy using bytewise XOR.\n"+
			"4. Reveal: view the 24-word BIP-39 mnemonic code.\n"+
			"5. Verify: type the 24 words from your paper, one at a time. The program\n"+
			"   checks each word against the mnemonic code before you finish.\n", msg,
		"[n] Next: Risks\n[q] Quit")
}

// warningFrame returns the Risks screen with the given message line.
func warningFrame(msg string) string {
	return screenFrame("Risks",
		"This program displays sensitive private-key data that can be used to\n"+
			"unlock funds: the 24-word BIP-39 mnemonic code and the entropy shown on\n"+
			"the XOR screen. Sensitive buffers are cleared on exit.\n"+
			"Losing or mistyping your only copy may make funds unrecoverable.\n"+
			"\n"+
			"Continue only if ALL of these are true:\n"+
			"  * Offline: Wi-Fi off, cable unplugged; fresh amnesic TailsOS session.\n"+
			"  * Alone; no camera/phone/recorder; fresh terminal, no tmux/screen/script.\n"+
			"  * Trust hardware, TailsOS, Linux kernel, and exact program build;\n"+
			"    absolutely trust the methods and personally inspect this version's source.\n"+
			"  * You can roll one or more fair six-sided dice unpredictably.\n"+
			"\n"+
			"Strongly recommended: use this for only one key in a multisig wallet.\n", msg,
		"[n] Next: Acknowledgement\n[q] Quit")
}

func riskAcceptanceFrame(msg string) string {
	return screenFrame("Acknowledgement",
		"Source code is available to inspect, and this build is reproducible.\n"+
			"Read the code yourself; an AI assistant can help explain it. Follow the\n"+
			"binary-verification steps on the project website to verify this binary.\n"+
			"The software is provided \"as is\"; the author is not liable for loss or\n"+
			"damage.\n"+
			"\n"+
			"Don't trust, verify.\n"+
			"\n"+
			"Do you acknowledge and accept the risks?\n", msg,
		"[a] Accept and continue\n[q] Quit")
}

// enter is the operator pressing Enter on a screen.
var enter = step{line: ""}

// quitFromWarning drives the flow through both opening screens and
// answers q on the Risks screen.
func TestOpeningThenWarningQuit(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("q on the warning must return ErrAbort, got %v", err)
	}
	want := []string{aboutFrame(""), howFrame(""), warningFrame("")}
	if len(*frames) != len(want) {
		t.Fatalf("want %d frames, got %d", len(want), len(*frames))
	}
	for i := range want {
		if (*frames)[i] != want[i] {
			t.Errorf("frame %d:\n got  %q\n want %q", i, (*frames)[i], want[i])
		}
	}
}

func TestOpeningQuitImmediately(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("q on the first screen must return ErrAbort, got %v", err)
	}
	if len(*frames) != 1 || (*frames)[0] != aboutFrame("") {
		t.Errorf("frames: %q", *frames)
	}
}

func TestOpeningQuitOnSecondScreen(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("q on the second screen must return ErrAbort, got %v", err)
	}
	if len(*frames) != 2 || (*frames)[1] != howFrame("") {
		t.Errorf("frames: %q", *frames)
	}
}

func TestOpeningInvalidChoice(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "x"}, step{line: "n"}, step{line: "n"}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if (*frames)[1] != aboutFrame("Not a valid choice. Use one of the options listed below.") {
		t.Errorf("frame 1:\n%q", (*frames)[1])
	}
}

func TestOpeningLineTooLong(t *testing.T) {
	app, frames := scriptedApp(t,
		step{err: application.ErrLineTooLong}, step{line: "n"}, step{line: "n"}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if (*frames)[1] != aboutFrame("Line too long. Nothing from it was kept.") {
		t.Errorf("frame 1:\n%q", (*frames)[1])
	}
}

func TestWarningAbort(t *testing.T) {
	for _, line := range []string{"Q", "q"} {
		app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{line: line})
		if err := app.Run(); !errors.Is(err, application.ErrAbort) {
			t.Fatalf("%q must return ErrAbort", line)
		}
		last := (*frames)[len(*frames)-1]
		if last != warningFrame("") {
			t.Errorf("%q last frame should remain the Risks screen:\n%q", line, last)
		}
	}
}

// N advances through the two first pages and Risks; q on the dice screen
// asks before quitting.
func TestWarningContinue(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{line: "n"}, step{line: "a"}, step{line: "q"}, step{line: "y"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if !strings.Contains((*frames)[3], "\nAcknowledgement\n") {
		t.Errorf("frame 3 should be the risk-acceptance screen:\n%q", (*frames)[3])
	}
	if !strings.Contains((*frames)[4], "\nRoll Dice\n") {
		t.Errorf("frame 4 should be the dice screen:\n%q", (*frames)[4])
	}
	if !strings.Contains((*frames)[5], "\nQuit?\n") {
		t.Errorf("frame 5 should be ConfirmAbort:\n%q", (*frames)[5])
	}
	if !strings.Contains((*frames)[5], "[y] Yes, quit\n[n] No, return to the current screen") {
		t.Error("ConfirmAbort must explain what happens after the answer")
	}
}

func TestRiskAcceptanceDeclined(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{line: "n"}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("declining the risks must return ErrAbort, got %v", err)
	}
	if len(*frames) != 4 || (*frames)[3] != riskAcceptanceFrame("") {
		t.Errorf("unexpected frames for declined risk acceptance: %q", *frames)
	}
}

func TestNavigationUsesNAndA(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "c"}, step{line: "n"},
		step{line: "c"}, step{line: "n"}, step{line: "a"}, step{line: "n"},
		step{line: "y"}, step{line: "a"}, step{line: "q"}, step{line: "y"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("N and A should progress; old C/Y commands should be rejected, got %v", err)
	}
	if (*frames)[1] != aboutFrame("Not a valid choice. Use one of the options listed below.") {
		t.Error("the About screen should reject C and direct the operator to N")
	}
	if (*frames)[3] != howFrame("Not a valid choice. Use one of the options listed below.") {
		t.Error("How it works should reject C and direct the operator to N")
	}
	if (*frames)[5] != warningFrame("Not a valid choice. Use one of the options listed below.") {
		t.Error("Risks should reject A and direct the operator to N")
	}
	if (*frames)[7] != riskAcceptanceFrame("Not a valid choice. Use one of the options listed below.") {
		t.Error("Acknowledgement should reject Y and direct the operator to A")
	}
}

func TestWarningInvalidChoice(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"},
		step{line: "x"}, step{line: "b"}, step{line: "cc"}, step{line: "1"},
		step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	want := warningFrame("Not a valid choice. Use one of the options listed below.")
	for i := 3; i < 7; i++ {
		if (*frames)[i] != want {
			t.Errorf("frame %d:\n%q", i, (*frames)[i])
		}
	}
}

func TestWarningLineTooLong(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"},
		step{err: application.ErrLineTooLong}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if (*frames)[3] != warningFrame("Line too long. Nothing from it was kept.") {
		t.Fatalf("unexpected frame:\n%q", (*frames)[3])
	}
}

func TestWarningEOF(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{err: application.ErrEOF})
	if err := app.Run(); !errors.Is(err, application.ErrEOF) {
		t.Fatal("end of input must return ErrEOF")
	}
	// EOF returns without drawing another screen because the terminal is gone.
	if len(*frames) != 3 {
		t.Fatalf("want 3 frames (opening screens and warning), got %d", len(*frames))
	}
}

func TestWarningReadError(t *testing.T) {
	errRead := errors.New("read failed")
	app, _ := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{err: errRead})
	if err := app.Run(); !errors.Is(err, errRead) {
		t.Fatal("read errors must be returned")
	}
}

func TestWarningDrawError(t *testing.T) {
	errDraw := errors.New("draw failed")
	term := &mock.Terminal{
		DrawFn:     func([]byte) error { return errDraw },
		ReadLineFn: func(dst []byte) (int, error) { return 0, nil },
	}
	app := &application.Runner{
		Term:          term,
		WS:            newWorkspace(t),
		Dice:          &mock.Dice{},
		Kernel:        &mock.Kernel{},
		Words:         bip39.Lookup{},
		CheckTerminal: func() error { return nil },
	}
	if err := app.Run(); !errors.Is(err, errDraw) {
		t.Fatal("draw errors must be returned")
	}
	if term.ReadLineInvoked {
		t.Fatal("must not read input after a failed draw")
	}
}

func TestRunWipesLineBuffer(t *testing.T) {
	app, _ := scriptedApp(t, step{line: "x"}, step{line: "q"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	if !bytes.Equal(app.WS.Line, make([]byte, len(app.WS.Line))) {
		t.Fatal("line buffer must be wiped after use")
	}
}

func TestFramesFitScreen(t *testing.T) {
	app, frames := scriptedApp(t, step{line: "n"}, step{line: "n"}, step{line: "n"}, step{line: "a"},
		step{line: "x"}, step{err: application.ErrLineTooLong},
		step{line: "a"}, step{line: "q"}, step{line: "y"})
	if err := app.Run(); !errors.Is(err, application.ErrAbort) {
		t.Fatalf("want ErrAbort, got %v", err)
	}
	for i, frame := range *frames {
		lines := strings.Split(frame, "\n")
		if len(lines) > application.FrameRows {
			t.Errorf("frame %d has %d rows", i, len(lines))
		}
		for _, l := range lines {
			if len(l) > application.FrameCols {
				t.Errorf("frame %d has a %d-column row", i, len(l))
			}
		}
	}
}
