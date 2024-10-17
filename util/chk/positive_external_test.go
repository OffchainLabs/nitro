package chk_test

import (
	"math"
	"testing"

	"github.com/offchainlabs/nitro/util/chk"
)

func TestNewPos64(t *testing.T) {
	v, err := chk.NewPos64(1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if v.Val() != 1 {
		t.Errorf("v.Val() want 1, got %d", v)
	}
}

func TestMustPos64(t *testing.T) {
	v := chk.MustPos64(1)
	if v.Val() != 1 {
		t.Errorf("v.Val() want 1, got %d", v)
	}
}

func TestNewPos64_error(t *testing.T) {
	_, err := chk.NewPos64(0)
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if err.Error() != "v must be positive. got: 0" {
		t.Errorf("Expected error message 'value must be positive', got '%s'", err.Error())
	}
}

func TestMustPos64_panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic, got nil")
		}
	}()
	chk.MustPos64(0)
}

func BenchmarkAdding(b *testing.B) {
	x := chk.MustPos64(1)
	y := chk.MustPos64(2)
	for i := 0; i < b.N; i++ {
		_ = x.Val() + y.Val()
	}
}

func BenchmarkAddingUint64(b *testing.B) {
	x := uint64(1)
	y := uint64(2)
	for i := 0; i < b.N; i++ {
		_ = x + y
	}
}

// Test zero value.
func TestZeroValue(t *testing.T) {
	var p chk.Pos64
	if p.Val() != 1 {
		t.Errorf("want 1, got %d", p.Val())
	}
}

// Test MaxUint64 value.
func TestMaxUint64(t *testing.T) {
	p := chk.MustPos64(math.MaxUint64)
	if p.Val() != math.MaxUint64 {
		t.Errorf("want math.MaxUint64, got %d", p.Val())
	}
}

// Cations are always positive.
func handleCation(c chk.Pos64) uint64 {
	return c.Val()
}

func TestPassingToFunction(t *testing.T) {
	want := uint64(1)
	got := handleCation(chk.MustPos64(1))
	if got != want {
		t.Errorf("want %d, got %d", want, got)
	}
}

// Uncomment to see that these lines don't compile.
// func doesNotCompile() {
// 	_ = chk.Pos64{100}
// 	_ = chk.MustPos64(50).value
// 	handleCation(0)
// 	handleCation(uint64(0))
// }
