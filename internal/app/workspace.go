package application

import "runtime"

// Buffer sizes in bytes. NewWorkspace gets one locked allocation from
// Allocator and divides it into these fixed regions.
const (
	poolSize    = 32                      // 256-bit entropy pool
	contribSize = 32                      // one source's 256 bits before XOR
	mixSize     = 32                      // XOR display buffer
	lineSize    = 256                     // one line of operator input
	frameSize   = 4096                    // one rendered screen
	scratchSize = 32                      // BIP-39 checksum and word-comparison scratch
	quizSlots   = 24                      // words in the BIP-39 mnemonic code
	quizSlotRow = 9                       // 1 length byte + 8 bytes of typed word per slot
	quizSize    = quizSlots * quizSlotRow // 216 bytes of quiz state
)

// Workspace holds every sensitive buffer the App uses. All buffers are
// carved from one locked allocation made once at startup, reused for the
// whole run, and wiped on every exit path.
type Workspace struct {
	Pool        []byte
	Contrib     []byte
	Mixed       []byte
	Line        []byte
	Frame       []byte
	Scratch     []byte
	DiceScratch []byte
	Quiz        []byte // typed words entered on the verification screen

	all []byte // the single locked allocation the buffers above share
}

// NewWorkspace allocates and locks every App buffer in one call to a.
// Dice scratch is fixed-size and allocated once for the program's lifetime;
// allocating it per run would leak locked pages.
func NewWorkspace(a Allocator) (*Workspace, error) {
	total := poolSize + contribSize + mixSize + lineSize + frameSize + scratchSize +
		DiceScratchSize + quizSize
	all, err := a.Alloc(total)
	if err != nil {
		return nil, err
	}
	w := &Workspace{all: all}
	rest := all
	w.Pool, rest = carve(rest, poolSize)
	w.Contrib, rest = carve(rest, contribSize)
	w.Mixed, rest = carve(rest, mixSize)
	w.Line, rest = carve(rest, lineSize)
	w.Frame, rest = carve(rest, frameSize)
	w.Scratch, rest = carve(rest, scratchSize)
	w.DiceScratch, rest = carve(rest, DiceScratchSize)
	w.Quiz, rest = carve(rest, quizSize)
	// Keep the rest of the allocation quiet: clear it so we don't leave
	// old data in the working set.
	wipe(rest)
	return w, nil
}

// carve splits n bytes off the front of b. The returned buffer's capacity
// is exactly n, so wiping it with clear(s[:cap(s)]) can never reach the
// neighbouring buffer.
func carve(b []byte, n int) (s, rest []byte) {
	if n < 0 || n > len(b) {
		panic("carve: out of range")
	}
	return b[:n:n], b[n:]
}

// Wipe clears every buffer. It is safe to call on a nil Workspace, so
// the central exit path can call it before the Workspace exists.
func (w *Workspace) Wipe() {
	if w == nil {
		return
	}
	wipe(w.all)
}

// wipe clears b up to its capacity. KeepAlive stops the compiler from
// treating the clear as a dead store and removing it.
func wipe(b []byte) {
	clear(b[:cap(b)])
	runtime.KeepAlive(b)
}
