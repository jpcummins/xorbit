package application

// drawMixScreen renders the three 256-bit values as hexadecimal directly
// from locked buffers, without converting them to strings.
func (a *Runner) drawMixScreen(msg string) error {
	f := NewFrame(a.WS.Frame, FrameRows)
	f.Text(screenHeader + "\n")
	f.Text(rule)
	f.NewLine()
	f.Text(mixTitle)
	f.NewLine()
	f.NewLine()
	f.Text(mixExplanation)
	f.NewLine()
	f.Text("Dice SHA-256:\n")
	f.Hex(a.WS.Pool)
	f.NewLine()
	f.NewLine()
	f.Text("Linux kernel RNG:\n")
	f.Hex(a.WS.Contrib)
	f.NewLine()
	f.NewLine()
	f.Text("Dice XOR Linux RNG:\n")
	f.Hex(a.WS.Mixed)
	f.NewLine()
	f.NewLine()
	// The note and transient status share a row so redraws stay within
	// the fixed frame budget while preserving the requested spacing.
	if msg != "" {
		f.Text(msg)
	} else {
		f.Text(verificationNote)
	}
	f.NewLine()
	f.NewLine()
	f.Text(mixCommands)
	f.endLine()
	return a.drawFrame(f)
}
