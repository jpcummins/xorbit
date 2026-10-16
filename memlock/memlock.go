// Package memlock implements application.Allocator with mlock(2).
package memlock

import (
	"errors"
	"syscall"

	app "xorbit/internal/app"
)

var _ app.Allocator = Allocator{}

// ErrLock is returned when a buffer cannot be locked into RAM.
var ErrLock = errors.New("memory lock failed")

// Allocator returns heap buffers locked into RAM. Locks are never released;
// they are page-granular and do not nest, so the kernel drops them at exit.
type Allocator struct{}

// Alloc returns a zeroed, locked buffer of exactly n bytes.
func (Allocator) Alloc(n int) ([]byte, error) {
	if n <= 0 {
		return nil, ErrLock
	}
	b := make([]byte, n)
	if syscall.Mlock(b) != nil {
		return nil, ErrLock
	}
	return b, nil
}
