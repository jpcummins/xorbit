package memlock_test

import (
	"testing"

	"xorbit/memlock"
)

func TestAlloc(t *testing.T) {
	b, err := memlock.Allocator{}.Alloc(64)
	if err != nil {
		t.Fatal("lock failed")
	}
	if len(b) != 64 || cap(b) != 64 {
		t.Fatal("unexpected size")
	}
}

func TestAllocRejectsNonPositive(t *testing.T) {
	if _, err := (memlock.Allocator{}).Alloc(0); err == nil {
		t.Fatal("expected error")
	}
}
