package dice_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"xorbit"
	"xorbit/dice"
	app "xorbit/internal/app"
)

// This assertion verifies the dice adapter implements the App's contract.
var _ app.DiceSource = (*dice.Source)(nil)

const scratchSize = app.DiceScratchSize

func newSource(t *testing.T) *dice.Source {
	t.Helper()
	s := &dice.Source{}
	if err := s.Start(make([]byte, scratchSize)); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return s
}

// finish drives the source through the review state to Finish and
// returns the 32-byte contribution.
func finish(t *testing.T, s *dice.Source) []byte {
	t.Helper()
	s.Input([]byte("r")) // enter the review state (>= 99 rolls)
	step := s.Input([]byte("n"))
	if !step.Done {
		t.Fatalf("n on the review screen must be Done, got %+v", step)
	}
	dst := make([]byte, 32)
	if err := s.Finish(dst); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	return dst
}

func TestViewStateByRollState(t *testing.T) {
	s := newSource(t)
	view := s.View()
	if view.Review || view.RollCount != 0 {
		t.Errorf("initial view = %+v, want entry state with zero rolls", view)
	}
	if len(view.Window) != app.DiceWindowSize || len(view.Digest) != 32 {
		t.Fatal("view must return slices for the display window and digest")
	}
	s.Input([]byte(strings.Repeat("1", 99)))
	if view := s.View(); view.Review || view.RollCount != 99 {
		t.Errorf("ready view = %+v, want entry state with 99 rolls", view)
	}
	if step := s.Input(nil); step.Message != "Type [r] to review the roll digest." {
		t.Errorf("empty line after the minimum = %+v", step)
	}
	s.Input([]byte("r"))
	if view := s.View(); !view.Review || view.RollCount != 99 {
		t.Errorf("review view = %+v, want review state with 99 rolls", view)
	}
}

func TestViewSlicesAliasScratch(t *testing.T) {
	s := newSource(t)
	entryView := s.View()
	s.Input([]byte("3"))
	if entryView.Window[0] != '3' {
		t.Error("View window must alias the locked scratch buffer")
	}

	reviewSource := newSource(t)
	reviewSource.Input([]byte(strings.Repeat("1", xorbit.DiceMinimumRolls)))
	reviewView := reviewSource.View()
	reviewSource.Input([]byte("r"))
	want, err := hex.DecodeString("fa098eb852b2660348b21bb00ad03a49cc177ea07ebe34f46b40baa85313525e")
	if err != nil {
		t.Fatal("invalid deterministic digest fixture")
	}
	if !bytes.Equal(reviewView.Digest, want) {
		t.Error("View digest must alias the review digest in locked scratch")
	}
}

func TestStartRejectsWrongScratch(t *testing.T) {
	s := &dice.Source{}
	if err := s.Start(make([]byte, scratchSize-1)); err == nil {
		t.Error("Start with a short scratch must fail")
	}
}

// The fixtures are SHA-256 digests of the roll digits in order, with
// spaces ignored; they can be independently checked with sha256sum.
func TestFinishFixtures(t *testing.T) {
	cases := []struct {
		name  string
		rolls string
		want  string
	}{
		{"99 rolls of 1", strings.Repeat("1", 99),
			"fa098eb852b2660348b21bb00ad03a49cc177ea07ebe34f46b40baa85313525e"},
		{"99 rolls of 6", strings.Repeat("6", 99),
			"7efb8e5d1353a90137755f711e1763fd7301a033fbb854889e127ff79c389131"},
		{"123456 x17 (102 rolls)", strings.Repeat("123456", 17),
			"8fd128918b2e29d6dcbfa5b9a118e5c16d60498c7ba107922a8eb6eb1d36c112"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newSource(t)
			s.Input([]byte(c.rolls))
			got := finish(t, s)
			want, err := hex.DecodeString(c.want)
			if err != nil {
				t.Fatal("bad fixture hex")
			}
			if !bytes.Equal(got, want) {
				t.Errorf("contribution = %x, want %s", got, c.want)
			}
		})
	}
}

// Rolls over several lines are exactly the rolls concatenated; spaces
// and tabs are ignored.
func TestMultiLineAndSpaces(t *testing.T) {
	a := newSource(t)
	a.Input([]byte("11 1"))
	a.Input([]byte(strings.Repeat("1", 96)))
	gotA := finish(t, a)

	b := newSource(t)
	b.Input([]byte("1 1\t1 " + strings.Repeat("1", 96)))
	gotB := finish(t, b)

	if !bytes.Equal(gotA, gotB) {
		t.Errorf("same rolls in different lines must hash the same:\n%x\n%x", gotA, gotB)
	}
}

// More than 99 rolls is fine and changes the digest.
func TestUnlimitedRolls(t *testing.T) {
	s := newSource(t)
	s.Input([]byte(strings.Repeat("1", 99)))
	// Enter the review state, return to rolls, then keep rolling.
	if step := s.Input([]byte("r")); step.Done || step.Message != "" {
		t.Fatalf("entering review: got %+v", step)
	}
	if step := s.Input([]byte("b")); step.Done || step.Message != "" {
		t.Fatalf("b should return to roll entry: got %+v", step)
	}
	s.Input([]byte("6")) // resume entry
	got := finish(t, s)
	ninetyNine, _ := hex.DecodeString("fa098eb852b2660348b21bb00ad03a49cc177ea07ebe34f46b40baa85313525e")
	if bytes.Equal(got, ninetyNine) {
		t.Error("the 100th roll must change the digest")
	}
}

func TestMinimumEnforced(t *testing.T) {
	for _, n := range []int{0, 50, 98} {
		s := newSource(t)
		if n > 0 {
			s.Input([]byte(strings.Repeat("1", n)))
		}
		step := s.Input(nil)
		if step.Done {
			t.Errorf("%d rolls: empty line must not be Done", n)
		}
		want := "At least 99 rolls are needed before you can finish. Keep rolling."
		if step.Message != want {
			t.Errorf("%d rolls: message = %q, want %q", n, step.Message, want)
		}
	}
}

func TestReviewState(t *testing.T) {
	s := newSource(t)
	s.Input([]byte(strings.Repeat("1", 99)))
	step := s.Input([]byte("r"))
	if step.Done {
		t.Fatal("entering review must not be Done")
	}
	// b returns to entry; roll input is accepted there.
	if step := s.Input([]byte("b")); step.Done || step.Message != "" {
		t.Errorf("b should return from review: got %+v", step)
	}
	if step := s.Input([]byte("2")); step.Done || step.Message != "" {
		t.Errorf("roll input after b: got %+v", step)
	}
	// Re-enter review; unrelated input stays in review.
	s.Input([]byte("r"))
	step = s.Input([]byte("x"))
	if step.Done || step.Message == "" {
		t.Errorf("bad line in review: got %+v", step)
	}
	if step := s.Input([]byte("n")); !step.Done {
		t.Error("n in review must be Done")
	}
}

func TestBadCharRejectsWholeLine(t *testing.T) {
	s := newSource(t)
	s.Input([]byte("123"))
	step := s.Input([]byte("45x6"))
	want := "Rejected: use only the digits 1-6 and spaces. That line was not kept."
	if step.Message != want {
		t.Errorf("message = %q, want %q", step.Message, want)
	}
	// The rejected line added nothing: 3 more rolls must reach 6, not 7.
	s.Input([]byte("456"))
	if got := s.View().RollCount; got != 6 {
		t.Errorf("rejected line must not count rolls: got %d, want 6", got)
	}
}

func TestNotReady(t *testing.T) {
	s := &dice.Source{}
	if step := s.Input([]byte("1")); step.Message == "" {
		t.Error("Input before Start must report the source as unavailable")
	}
}

func TestFinishPreconditions(t *testing.T) {
	// Fresh source.
	if err := (&dice.Source{}).Finish(make([]byte, 32)); err == nil {
		t.Error("Finish before Start must fail")
	}
	// Under 99 rolls.
	s := newSource(t)
	s.Input([]byte(strings.Repeat("1", 50)))
	if err := s.Finish(make([]byte, 32)); err == nil {
		t.Error("Finish below 99 rolls must fail")
	}
	// Wrong dst length.
	s2 := newSource(t)
	s2.Input([]byte(strings.Repeat("1", 99)))
	s2.Input([]byte("r"))
	if err := s2.Finish(make([]byte, 16)); err == nil {
		t.Error("Finish with a 16-byte dst must fail")
	}
}

// Start fully resets: the same rolls give the same digest across runs.
func TestStartReset(t *testing.T) {
	s := newSource(t)
	s.Input([]byte(strings.Repeat("6", 99)))
	first := finish(t, s)
	if err := s.Start(make([]byte, scratchSize)); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	s.Input([]byte(strings.Repeat("6", 99)))
	second := finish(t, s)
	if !bytes.Equal(first, second) {
		t.Error("same rolls after a reset must give the same digest")
	}
}
