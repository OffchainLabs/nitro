package chk

import (
	"testing"
	"unsafe"
)

func TestSize(t *testing.T) {
	// This test is here to ensure that the size of the Pos64 struct is 8 bytes.
	u := uint64(24601)
	p := Pos64{}

	want := unsafe.Sizeof(u)
	got := unsafe.Sizeof(p)
	if got != want {
		t.Errorf("Size of Pos64 want %d, got %d", want, got)
	}
}
