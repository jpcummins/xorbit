package application

import (
	"encoding/hex"
	"strings"
	"testing"

	"xorbit"
)

func TestRenderDiceEntry(t *testing.T) {
	window := make([]byte, DiceWindowSize)
	copy(window, "361")
	view := DiceView{RollCount: 3, Window: window}
	f := NewFrame(make([]byte, 4096), diceBodyRows)
	if err := renderDiceBody(f, view); err != nil {
		t.Fatalf("render dice entry: %v", err)
	}
	want := "Enter each result as a digit from 1-6; spaces are fine.\n" +
		"Roll one or more fair six-sided dice at least 99 times.\n" +
		"Last 25 rolls:\n" +
		"  3 6 1 . .  . . . . .  . . . . .  . . . . .  . . . . .\n" +
		"\n" +
		"Rolls entered: 3 (at least 99 before you can finish)\n" +
		"\n" +
		"Next: Review the roll digest.\n"
	if got := string(f.Out()); got != want {
		t.Errorf("dice entry body:\n got  %q\n want %q", got, want)
	}
}

func TestRenderDiceWindowAndCount(t *testing.T) {
	cases := []struct {
		name      string
		count     int
		window    string
		wantRolls string
		wantCount string
	}{
		{
			name:      "sliding window",
			count:     30,
			window:    "6123456123456123456123456",
			wantRolls: "  6 1 2 3 4  5 6 1 2 3  4 5 6 1 2  3 4 5 6 1  2 3 4 5 6\n",
			wantCount: "Rolls entered: 30 (at least 99 before you can finish)",
		},
		{
			name:      "minimum reached",
			count:     xorbit.DiceMinimumRolls,
			window:    strings.Repeat("1", DiceWindowSize),
			wantRolls: "  1 1 1 1 1  1 1 1 1 1  1 1 1 1 1  1 1 1 1 1  1 1 1 1 1\n",
			wantCount: "Rolls entered: 99 (minimum reached; review or keep rolling)",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			view := DiceView{RollCount: tt.count, Window: []byte(tt.window)}
			f := NewFrame(make([]byte, 4096), diceBodyRows)
			if err := renderDiceBody(f, view); err != nil {
				t.Fatalf("render dice entry: %v", err)
			}
			got := string(f.Out())
			if !strings.Contains(got, tt.wantRolls) {
				t.Errorf("roll window missing:\n got  %q\n want row %q", got, tt.wantRolls)
			}
			if !strings.Contains(got, tt.wantCount) {
				t.Errorf("count line missing:\n got  %q\n want %q", got, tt.wantCount)
			}
		})
	}
}

func TestRenderDiceReview(t *testing.T) {
	digest, err := hex.DecodeString("fa098eb852b2660348b21bb00ad03a49cc177ea07ebe34f46b40baa85313525e")
	if err != nil {
		t.Fatal("invalid deterministic digest fixture")
	}
	view := DiceView{Review: true, Digest: digest}
	f := NewFrame(make([]byte, 4096), diceBodyRows)
	if err := renderDiceBody(f, view); err != nil {
		t.Fatalf("render dice review: %v", err)
	}
	want := "Below is a SHA-256 hash of the dice rolls you entered:\n" +
		"\n" +
		"  fa098eb8 52b26603 48b21bb0 0ad03a49\n" +
		"  cc177ea0 7ebe34f4 6b40baa8 5313525e\n" +
		"\n" +
		"You don't need to write these down; they're presented for verification.\n"
	if got := string(f.Out()); got != want {
		t.Errorf("dice review body:\n got  %q\n want %q", got, want)
	}
}

func TestDiceCommands(t *testing.T) {
	cases := []struct {
		name string
		view DiceView
		want string
	}{
		{"entry", DiceView{}, "[↵] Submit line\n[q] Quit"},
		{"minimum reached", DiceView{RollCount: xorbit.DiceMinimumRolls}, "[↵] Submit line\n[r] Review the roll digest\n[q] Quit"},
		{"review", DiceView{Review: true}, "[b] Back to rolling\n[n] Next: Generate Linux RNG entropy\n[q] Quit"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := diceCommands(tt.view); got != tt.want {
				t.Errorf("diceCommands() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderDiceBodyFitsBudget(t *testing.T) {
	cases := []DiceView{
		{Window: make([]byte, DiceWindowSize)},
		{RollCount: xorbit.DiceMinimumRolls, Window: []byte(strings.Repeat("1", DiceWindowSize))},
		{Review: true, Digest: make([]byte, 32)},
	}
	for i, view := range cases {
		f := NewFrame(make([]byte, 4096), diceBodyRows)
		if err := renderDiceBody(f, view); err != nil {
			t.Errorf("state %d exceeds the dice body budget: %v", i, err)
		}
	}
}
