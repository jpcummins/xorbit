package main

import (
	"errors"
	"fmt"
	"testing"

	application "xorbit/internal/app"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{nil, exitOK},
		{application.ErrAbort, exitAborted},
		{application.ErrEOF, exitAborted},
		{fmt.Errorf("wrapped: %w", application.ErrAbort), exitAborted},
		{errors.New("anything else"), exitFailure},
	}
	for _, tt := range tests {
		if got := exitCode(tt.err); got != tt.want {
			t.Errorf("exitCode(%v) = %d, want %d", tt.err, got, tt.want)
		}
	}
}
