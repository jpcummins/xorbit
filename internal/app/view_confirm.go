package application

// drawConfirm renders the fixed yes/no confirmation screen.
func (a *Runner) drawConfirm(title, question, instruction string) error {
	f := NewFrame(a.WS.Frame, FrameRows)
	f.Text(screenHeader + "\n")
	f.Text(rule)
	f.NewLine()
	f.Text(title)
	f.NewLine()
	f.NewLine()
	f.Text(question)
	f.endLine()
	f.NewLine()
	f.Text(instruction)
	f.endLine()
	return a.drawFrame(f)
}
