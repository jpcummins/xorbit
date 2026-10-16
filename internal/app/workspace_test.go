package application_test

import (
	"bytes"
	"errors"
	"testing"

	application "xorbit/internal/app"
	"xorbit/mock"
)

// heapAllocator hands out plain heap memory; tests don't need mlock.
func heapAllocator() *mock.Allocator {
	return &mock.Allocator{AllocFn: func(n int) ([]byte, error) {
		return make([]byte, n), nil
	}}
}

func newWorkspace(t *testing.T) *application.Workspace {
	t.Helper()
	w, err := application.NewWorkspace(heapAllocator())
	if err != nil {
		t.Fatal("workspace allocation failed")
	}
	return w
}

func buffers(w *application.Workspace) map[string][]byte {
	return map[string][]byte{
		"Pool": w.Pool, "Contrib": w.Contrib, "Mixed": w.Mixed, "Line": w.Line,
		"Frame": w.Frame, "Scratch": w.Scratch, "DiceScratch": w.DiceScratch,
		"Quiz": w.Quiz,
	}
}

func TestNewWorkspaceSizes(t *testing.T) {
	var requested []int
	a := &mock.Allocator{AllocFn: func(n int) ([]byte, error) {
		requested = append(requested, n)
		return make([]byte, n), nil
	}}
	w, err := application.NewWorkspace(a)
	if err != nil {
		t.Fatal("workspace allocation failed")
	}
	const quiz = 24 * 9
	wantTotal := 32 + 32 + 32 + 256 + 4096 + 32 + application.DiceScratchSize + quiz
	if len(requested) != 1 || requested[0] != wantTotal {
		t.Fatalf("want one allocation of %d bytes, got %v", wantTotal, requested)
	}
	want := map[string]int{
		"Pool": 32, "Contrib": 32, "Mixed": 32, "Line": 256,
		"Frame": 4096, "Scratch": 32, "DiceScratch": application.DiceScratchSize,
		"Quiz": quiz,
	}
	for name, b := range buffers(w) {
		if len(b) != want[name] || cap(b) != want[name] {
			t.Errorf("%s: len %d cap %d, want %d", name, len(b), cap(b), want[name])
		}
	}
}

func TestNewWorkspaceAllocError(t *testing.T) {
	errAlloc := errors.New("alloc failed")
	a := &mock.Allocator{AllocFn: func(int) ([]byte, error) { return nil, errAlloc }}
	if _, err := application.NewWorkspace(a); !errors.Is(err, errAlloc) {
		t.Fatal("expected the allocator's error")
	}
}

// Wiping one buffer up to its capacity must never touch its neighbours.
func TestWorkspaceBuffersAreIsolated(t *testing.T) {
	w := newWorkspace(t)
	bufs := buffers(w)
	fill := map[string]byte{
		"Pool": 1, "Contrib": 2, "Mixed": 3, "Line": 4,
		"Frame": 5, "Scratch": 6, "DiceScratch": 7, "Quiz": 8,
	}
	for name, b := range bufs {
		b = b[:cap(b)]
		for i := range b {
			b[i] = fill[name]
		}
	}
	for name, b := range bufs {
		if !bytes.Equal(b, bytes.Repeat([]byte{fill[name]}, len(b))) {
			t.Errorf("%s was overwritten by a neighbour", name)
		}
	}
}

// A source's scratch must not share memory with the App's buffers.
func TestDiceScratchIsSeparate(t *testing.T) {
	w := newWorkspace(t)
	for i := range w.DiceScratch {
		w.DiceScratch[i] = 0xAA
	}
	for name, b := range buffers(w) {
		if name == "DiceScratch" {
			continue
		}
		if !bytes.Equal(b, make([]byte, len(b))) {
			t.Errorf("dice scratch overlaps %s", name)
		}
	}
}

func TestWorkspaceWipe(t *testing.T) {
	w := newWorkspace(t)
	for _, b := range buffers(w) {
		for i := range b {
			b[i] = 0xAA
		}
	}
	w.Wipe()
	for name, b := range buffers(w) {
		if !bytes.Equal(b, make([]byte, len(b))) {
			t.Errorf("%s not wiped", name)
		}
	}
}

func TestWipeNilWorkspace(t *testing.T) {
	var w *application.Workspace
	w.Wipe()
}
