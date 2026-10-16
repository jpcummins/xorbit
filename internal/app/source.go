package application

import "errors"

// ErrSourceFailed is returned when dice processing or kernel entropy
// generation fails. The App shows the fixed source-failure screen and exits
// through its central wipe path.
var ErrSourceFailed = errors.New("source failed")

// DiceWindowSize is the number of recent rolls shown on the dice screen.
// DiceScratchSize stores that display window followed by a SHA-256 digest.
const (
	DiceWindowSize  = 25
	DiceScratchSize = DiceWindowSize + 32
)

// DiceStep is the result of one line entered on the dice screen. Message is
// fixed text and never contains the operator's input.
type DiceStep struct {
	Done    bool
	Message string
}

// DiceView is the short-lived presentation state for the dice screen. Window
// and Digest alias the locked scratch buffer supplied to DiceSource.Start
// (Workspace.DiceScratch in the App); renderers must not retain or copy their
// bytes, and the value is used only while drawing the dice screen.
type DiceView struct {
	RollCount int
	Review    bool
	Window    []byte // DiceWindowSize bytes
	Digest    []byte // 32 bytes
}

// DiceSource provides the fixed interactive dice screen. The App owns the
// terminal, input loop, quit confirmation, and mixing; the dice adapter only
// owns roll state and processes one bounded line at a time.
type DiceSource interface {
	// Start resets the dice screen using the locked 57-byte scratch buffer.
	Start(scratch []byte) error
	// View returns the public count and slices into locked scratch for rendering.
	View() DiceView
	// Input processes one trimmed line and returns fixed status text.
	Input(line []byte) DiceStep
	// Finish writes the 32-byte SHA-256 dice contribution.
	Finish(dst []byte) error
}

// KernelRNG fills dst with 32 bytes from the Linux kernel RNG. The App calls
// it only after the dice contribution has been mixed into the pool.
type KernelRNG interface {
	// Generate fills dst with exactly 32 bytes from the kernel RNG.
	Generate(dst []byte) error
}
