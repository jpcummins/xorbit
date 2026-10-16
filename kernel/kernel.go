// Package kernel provides the Linux kernel RNG source. It reads 32 bytes
// from crypto/rand for the App to mix with the dice contribution and display
// on the checked XOR screen.
//
// This package is the only one in the project that imports crypto/rand
// (intended to make audits easier: reviewers can confirm it is the only
// place Linux kernel RNG bytes enter the program).
package kernel

import (
	"crypto/rand"
	"io"

	app "xorbit/internal/app"
)

// Source is the Linux kernel RNG entropy source.
//
// Reader supplies the 32 bytes; nil means crypto/rand.Reader. Tests inject a
// fixed reader so runs are reproducible.
type Source struct {
	Reader io.Reader
}

// Generate reads exactly 32 bytes from the configured reader into dst. A
// short read or an error fails the source, and the App fails closed. The App
// retains the contribution for the checked XOR display, then commits the
// result and wipes the contribution.
func (s *Source) Generate(dst []byte) error {
	if len(dst) != 32 {
		return app.ErrConfig
	}
	r := s.Reader
	if r == nil {
		r = rand.Reader
	}
	if _, err := io.ReadFull(r, dst); err != nil {
		return app.ErrSourceFailed
	}
	return nil
}
