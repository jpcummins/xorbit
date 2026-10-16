package application

// Shared view helpers. Renderers build directly in the locked frame buffer
// and draw it without making a string copy.

// drawScreen renders the shared chrome for fixed App screens: header, rule,
// title, body, optional message, and bottom command list.
func (a *Runner) drawScreen(title, body, msg, cmd string) error {
	f := NewFrame(a.WS.Frame, FrameRows)
	f.Text(screenHeader + "\n")
	f.Text(rule)
	f.NewLine()
	f.Text(title)
	f.NewLine()
	f.NewLine()
	if body != "" {
		f.Text(body)
	}
	f.endLine()
	if msg != "" {
		f.Text(msg)
		f.NewLine()
	}
	f.NewLine() // blank line before the command list
	f.Text(cmd)
	f.endLine()
	return a.drawFrame(f)
}

// drawFrame checks and writes a rendered frame, then wipes its locked buffer.
func (a *Runner) drawFrame(f *Frame) error {
	if err := f.Err(); err != nil {
		wipe(a.WS.Frame)
		return err
	}
	err := a.Term.Draw(f.Out())
	wipe(a.WS.Frame)
	return err
}
