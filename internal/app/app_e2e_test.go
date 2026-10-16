package application_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"xorbit/bip39"
	"xorbit/dice"
	application "xorbit/internal/app"
	"xorbit/kernel"
	"xorbit/mock"
)

// e2eRig wires the real dice and kernel sources into an App with a
// scripted terminal and a recording display. The kernel source's reader
// is fixed so the run is reproducible.
type e2eRig struct {
	app    *application.Runner
	frames *[]string
	shown  [][]byte
}

func newE2eRig(t *testing.T, kernelBytes []byte, steps ...step) *e2eRig {
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
	r := &e2eRig{frames: &frames}
	display := &mock.Display{ShowFn: func(result []byte) error {
		r.shown = append(r.shown, append([]byte(nil), result...))
		return nil
	}}
	diceSource := &dice.Source{}
	kernelRNG := &kernel.Source{Reader: bytes.NewReader(kernelBytes)}
	ws, err := application.NewWorkspace(heapAllocator())
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	r.app = &application.Runner{
		Term:          term,
		Display:       display,
		WS:            ws,
		Dice:          diceSource,
		Kernel:        kernelRNG,
		Words:         bip39.Lookup{},
		CheckTerminal: func() error { return nil },
	}
	return r
}

// wordSteps turns a mnemonic into one script step per word.
func wordSteps(mnemonic string) []step {
	var steps []step
	for _, w := range strings.Fields(mnemonic) {
		steps = append(steps, step{line: w})
	}
	return steps
}

func TestEndToEnd(t *testing.T) {
	cases := []struct {
		name           string
		kernelBytes    []byte
		wantMnemonic   string
		wantReviewPart string
	}{
		{
			// XOR with 32 zero bytes leaves the dice digest unchanged.
			name:        "kernel all zeros",
			kernelBytes: make([]byte, 32),
			wantMnemonic: "wheel erase puppy pistol chapter accuse carpet drop " +
				"quote final attend near scrap satisfy limit style crunch person " +
				"south inspire lunch meadow enact tattoo",
			wantReviewPart: "fa098eb8 52b26603 48b21bb0 0ad03a49",
		},
		{
			// XOR with 32 bytes of 0xff complements every digest byte.
			name:        "kernel all ones",
			kernelBytes: bytes.Repeat([]byte{0xff}, 32),
			wantMnemonic: "album receive faint fold sunset wrestle tag route exit " +
				"practice vapor hockey direct doll lawn choice slam frequent " +
				"cool melt kiss iron repair capital",
			wantReviewPart: "fa098eb8 52b26603 48b21bb0 0ad03a49",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			steps := []step{
				{line: "n"}, {line: "n"}, {line: "n"}, {line: "a"},
				{line: strings.Repeat("1", 99)},   // the dice rolls
				{line: "r"},                       // open the digest review
				{line: "n"},                       // finish the dice source
				{line: "x"},                       // show the invalid-choice message
				{err: application.ErrLineTooLong}, // exercise the long-line redraw
				{line: "n"},                       // accept the combined RNG and XOR screen
				{line: "n"},                       // ConfirmReveal
				{line: "n"},                       // Reveal acknowledgment
			}
			steps = append(steps, wordSteps(c.wantMnemonic)...)
			steps = append(steps, step{line: "n"}, enter) // all-match Next, Done ack
			r := newE2eRig(t, c.kernelBytes, steps...)
			if err := r.app.Run(); err != nil {
				t.Fatalf("full run must return nil, got %v", err)
			}
			if len(r.shown) != 1 {
				t.Fatalf("Reveal must be shown exactly once, got %d", len(r.shown))
			}
			reveal := string(r.shown[0])
			for _, w := range strings.Fields(c.wantMnemonic) {
				if !strings.Contains(reveal, w) {
					t.Errorf("reveal frame must contain %q:\n%s", w, reveal)
				}
			}
			// The dice review screen shows a digest the operator can
			// independently verify with sha256sum.
			joined := strings.Join(*r.frames, "\x00")
			if !strings.Contains(joined, c.wantReviewPart) {
				t.Errorf("the review screen must show the dice digest %q", c.wantReviewPart)
			}
			for _, next := range []string{
				"Roll Dice",
				"Digest Review",
				"[↵] Submit line",
				"[r] Review the roll digest",
				"[b] Back to rolling",
				"[n] Next: Generate Linux RNG entropy",
				"[n] Next: Review mnemonic code instructions",
				"[q] Quit",
				"Below is a SHA-256 hash of the dice rolls you entered:",
				"You don't need to write these down; they're presented for verification.",
				"Mixing independent sources can help if one is unreliable.",
				"Not a valid choice. Use one of the options listed below.",
				"Line too long. Nothing from it was kept.",
			} {
				if !strings.Contains(joined, next) {
					t.Errorf("screen must explain what comes next: %q", next)
				}
			}
			if strings.Contains(joined, "\nLinux kernel RNG\n") {
				t.Error("kernel RNG must not have a separate input screen")
			}
			foundDigestReview := false
			for _, frame := range *r.frames {
				if !strings.Contains(frame, "\nDigest Review\n") {
					continue
				}
				foundDigestReview = true
				lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
				foundReviewNote := false
				for row, line := range lines {
					if line != "You don't need to write these down; they're presented for verification." {
						continue
					}
					foundReviewNote = true
					if row == 0 || row+1 == len(lines) || lines[row-1] != "" || lines[row+1] != "" {
						t.Error("Digest Review verification note must have a blank line above and below")
					}
					break
				}
				if !foundReviewNote {
					t.Error("Digest Review must include its verification note")
				}
				for _, absent := range []string{
					"Enter each result as a digit from 1-6; spaces are fine.",
					"Rolls entered:",
					"Roll order matters; spaces are ignored.",
				} {
					if strings.Contains(frame, absent) {
						t.Errorf("Digest Review should omit %q", absent)
					}
				}
			}
			if !foundDigestReview {
				t.Error("the dice digest review screen must be shown")
			}
			diceBytes, err := hex.DecodeString(strings.ReplaceAll(c.wantReviewPart, " ", ""))
			if err != nil {
				t.Fatal("invalid deterministic digest fixture")
			}
			finalBits := make([]byte, len(diceBytes))
			for i := range diceBytes {
				finalBits[i] = diceBytes[i] ^ c.kernelBytes[i]
			}
			for _, bits := range []string{
				strings.ReplaceAll(c.wantReviewPart, " ", ""),
				hex.EncodeToString(c.kernelBytes),
				hex.EncodeToString(finalBits),
			} {
				if !strings.Contains(joined, bits) {
					t.Errorf("XOR screen must show deterministic fixture bits %q", bits)
				}
			}
			for i, frame := range *r.frames {
				lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
				if len(lines) > application.FrameRows {
					t.Errorf("frame %d has %d rows; maximum is %d", i, len(lines), application.FrameRows)
				}
				for _, line := range lines {
					if len(line) > application.FrameCols {
						t.Errorf("frame %d has a %d-column line", i, len(line))
					}
				}
			}
		})
	}
}
