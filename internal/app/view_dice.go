package application

import (
	"strconv"

	"xorbit"
)

const (
	diceBodyRows   = 10
	diceFooterRows = 8
)

// drawDiceScreen renders the dice view state supplied by the dice adapter.
// Its byte slices alias locked scratch and are written directly into the
// locked frame; no string copy of the rolls or digest is made.
func (a *Runner) drawDiceScreen(msg string) error {
	view := a.Dice.View()
	if view.RollCount < 0 || len(view.Window) != DiceWindowSize || len(view.Digest) != 32 {
		return ErrConfig
	}
	f := NewFrame(a.WS.Frame, 6)
	f.Text(screenHeader + "\n")
	f.Text(rule)
	f.NewLine()
	if view.Review {
		f.Text("Digest Review")
	} else {
		f.Text("Roll Dice")
	}
	f.NewLine()
	f.NewLine()

	f.Reset(diceBodyRows)
	if err := renderDiceBody(f, view); err != nil {
		wipe(a.WS.Frame)
		return err
	}
	f.endLine()

	f.Reset(diceFooterRows)
	if msg != "" {
		f.NewLine()
		f.Text(msg)
		f.NewLine()
	}
	f.NewLine()
	f.Text(diceCommands(view))
	f.endLine()
	if !view.Review {
		f.NewLine()
		f.Text("> ")
	}
	return a.drawFrame(f)
}

// renderDiceBody writes the entry body or digest review from a DiceView.
func renderDiceBody(f *Frame, view DiceView) error {
	if view.Review {
		renderDiceReview(f, view.Digest)
	} else {
		renderDiceEntry(f, view)
	}
	return f.Err()
}

// renderDiceEntry writes the instructions, recent rolls, and public roll
// count. The window is shown oldest first and grouped in fives.
func renderDiceEntry(f *Frame, view DiceView) {
	f.Text("Enter each result as a digit from 1-6; spaces are fine.\n")
	f.Text("Roll one or more fair six-sided dice at least 99 times.\n")
	f.Text("Last 25 rolls:\n")
	f.Text("  ")
	for i := 0; i < DiceWindowSize; i++ {
		if i > 0 {
			f.Text(" ")
			if i%5 == 0 {
				f.Text(" ")
			}
		}
		if view.RollCount >= DiceWindowSize || i < view.RollCount {
			f.Byte(view.Window[i])
		} else {
			f.Text(".")
		}
	}
	f.NewLine()
	f.NewLine()
	f.Text("Rolls entered: " + strconv.Itoa(view.RollCount))
	if view.RollCount < xorbit.DiceMinimumRolls {
		f.Text(" (at least 99 before you can finish)")
	} else {
		f.Text(" (minimum reached; review or keep rolling)")
	}
	f.NewLine()
	if view.RollCount < xorbit.DiceMinimumRolls {
		f.NewLine()
		f.Text("Next: Review the roll digest.\n")
	}
}

// renderDiceReview writes the reviewed SHA-256 digest in two rows of grouped
// hexadecimal. The digest bytes stay in locked scratch until written.
func renderDiceReview(f *Frame, digest []byte) {
	f.Text("Below is a SHA-256 hash of the dice rolls you entered:\n")
	f.NewLine()
	for row := 0; row < 2; row++ {
		f.Text("  ")
		for group := 0; group < 4; group++ {
			if group > 0 {
				f.Text(" ")
			}
			start := row*16 + group*4
			f.Hex(digest[start : start+4])
		}
		f.NewLine()
	}
	f.NewLine()
	f.Text(verificationNote)
	f.NewLine()
}

// diceCommands returns fixed commands for the current public dice state.
func diceCommands(view DiceView) string {
	if view.Review {
		return "[b] Back to rolling\n[n] Next: Generate Linux RNG entropy\n[q] Quit"
	}
	if view.RollCount >= xorbit.DiceMinimumRolls {
		return "[↵] Submit line\n[r] Review the roll digest\n[q] Quit"
	}
	return "[↵] Submit line\n[q] Quit"
}
