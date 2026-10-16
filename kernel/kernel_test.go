package kernel_test

import (
	"bytes"
	"errors"
	"testing"

	app "xorbit/internal/app"
	"xorbit/kernel"
)

// This assertion verifies the kernel adapter implements the App's contract.
var _ app.KernelRNG = (*kernel.Source)(nil)

// Generate copies exactly 32 bytes from the injected reader.
func TestGenerateInjectedReader(t *testing.T) {
	fixed := make([]byte, 32)
	for i := range fixed {
		fixed[i] = byte(i)
	}
	s := &kernel.Source{Reader: bytes.NewReader(fixed)}
	dst := make([]byte, 32)
	if err := s.Generate(dst); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Equal(dst, fixed) {
		t.Errorf("dst = %x, want %x", dst, fixed)
	}
}

// A short read is a failure, and the App fails closed on it.
func TestGenerateShortRead(t *testing.T) {
	s := &kernel.Source{Reader: bytes.NewReader(make([]byte, 10))}
	err := s.Generate(make([]byte, 32))
	if !errors.Is(err, app.ErrSourceFailed) {
		t.Errorf("short read must return ErrSourceFailed, got %v", err)
	}
}

func TestGenerateBadLength(t *testing.T) {
	s := &kernel.Source{Reader: bytes.NewReader(make([]byte, 32))}
	if err := s.Generate(make([]byte, 16)); err == nil {
		t.Error("Generate on a 16-byte dst should error")
	}
}
