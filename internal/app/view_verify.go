package application

// verifyBody renders the verify screen. Unfilled cells are eight spaces;
// filled cells show only the operator's typed word, never the pool's word.
func (a *Runner) verifyBody(f *Frame) {
	f.Text(verifyIntro)
	f.NewLine()
	for row := 0; row < 6; row++ {
		for col := 0; col < 4; col++ {
			i := col*6 + row
			if col > 0 {
				f.Text(" ")
			}
			for _, c := range rightAligned4(uint16(i + 1)) {
				f.Byte(c)
			}
			f.Text(" ")
			if a.quizHas(i) {
				f.Padded(a.quizWord(i), 8)
			} else {
				for k := 0; k < 8; k++ {
					f.Byte(' ')
				}
			}
		}
		f.NewLine()
	}
}

// drawVerify renders the verify screen's title, body, message, and commands.
func (a *Runner) drawVerify(msg, cmds string) error {
	f := NewFrame(a.WS.Frame, FrameRows)
	f.Text(screenHeader + "\n")
	f.Text(rule)
	f.NewLine()
	f.Text(verifyTitle)
	f.NewLine()
	f.NewLine()
	a.verifyBody(f)
	f.endLine()
	if msg != "" {
		f.Text(msg)
		f.NewLine()
	}
	f.NewLine()
	f.Text(cmds)
	f.endLine()
	return a.drawFrame(f)
}

// drawDone renders the final fixed-text Done screen.
func (a *Runner) drawDone() error {
	return a.drawScreen(doneTitle, doneBody, "", exitCommands)
}
