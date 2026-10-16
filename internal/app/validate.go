package application

// Configuration validation runs once at the start of Runner.Run and rejects
// missing fixed dependencies or an invalid dice scratch buffer before any
// screen is drawn.

// Validate checks that the fixed dice and kernel adapters, workspace, and
// terminal re-scan callback are present and ready.
func (a *Runner) Validate() error {
	if a.Dice == nil || a.Kernel == nil || a.Words == nil || a.WS == nil || a.CheckTerminal == nil {
		return ErrConfig
	}
	if len(a.WS.DiceScratch) != DiceScratchSize || cap(a.WS.DiceScratch) != DiceScratchSize {
		return ErrConfig
	}
	return nil
}
