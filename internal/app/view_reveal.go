package application

// drawReveal renders and displays the mnemonic frame exactly once. The
// locked frame is wiped immediately after Display.Show returns.
func (a *Runner) drawReveal() error {
	if a.Display == nil {
		return ErrConfig
	}
	f := NewFrame(a.WS.Frame, FrameRows)
	f.Text(screenHeader + "\n")
	f.Text(rule)
	f.NewLine()
	f.Text(revealTitle)
	f.NewLine()
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
			index := a.mnemonicWordIndex(i)
			f.Word(index, a.Words)
		}
		f.NewLine()
	}
	f.NewLine()
	f.Text(rule)
	f.Text(revealFooter)
	f.NewLine()
	f.Text(revealCommands)
	f.endLine()
	if err := f.Err(); err != nil {
		wipe(a.WS.Frame)
		return err
	}
	err := a.Display.Show(f.Out())
	wipe(a.WS.Frame)
	return err
}
