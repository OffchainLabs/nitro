// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import (
	"math/big"
	"testing"
)

func toBig[T Signed](a T) *big.Int {
	return big.NewInt(int64(a))
}

func saturatingBigToInt[T Signed](a *big.Int) T {
	// MinSignedValue and MaxSignedValue are already separately tested
	if a.Cmp(toBig(MaxSignedValue[T]())) > 0 {
		return MaxSignedValue[T]()
	}
	if a.Cmp(toBig(MinSignedValue[T]())) < 0 {
		return MinSignedValue[T]()
	}
	return T(a.Int64())
}

func fuzzSaturatingAdd[T Signed](f *testing.F) {
	f.Fuzz(func(t *testing.T, a, b T) {
		got := SaturatingAdd(a, b)
		expected := saturatingBigToInt[T](new(big.Int).Add(toBig(a), toBig(b)))
		if got != expected {
			t.Errorf("SaturatingAdd(%v, %v) = %v, expected %v", a, b, got, expected)
		}
	})
}

func fuzzSaturatingMul[T Signed](f *testing.F) {
	f.Fuzz(func(t *testing.T, a, b T) {
		got := SaturatingMul(a, b)
		expected := saturatingBigToInt[T](new(big.Int).Mul(toBig(a), toBig(b)))
		if got != expected {
			t.Errorf("SaturatingMul(%v, %v) = %v, expected %v", a, b, got, expected)
		}
	})
}

func fuzzSaturatingNeg[T Signed](f *testing.F) {
	f.Fuzz(func(t *testing.T, a T) {
		got := SaturatingNeg(a)
		expected := saturatingBigToInt[T](new(big.Int).Neg(toBig(a)))
		if got != expected {
			t.Errorf("SaturatingNeg(%v) = %v, expected %v", a, got, expected)
		}
	})
}

func FuzzSaturatingAddInt8(f *testing.F) {
	fuzzSaturatingAdd[int8](f)
}

func FuzzSaturatingAddInt16(f *testing.F) {
	fuzzSaturatingAdd[int16](f)
}

func FuzzSaturatingAddInt32(f *testing.F) {
	fuzzSaturatingAdd[int32](f)
}

func FuzzSaturatingAddInt64(f *testing.F) {
	fuzzSaturatingAdd[int64](f)
}

func FuzzSaturatingSub(f *testing.F) {
	f.Fuzz(func(t *testing.T, a, b int64) {
		got := SaturatingSub(a, b)
		expected := saturatingBigToInt[int64](new(big.Int).Sub(toBig(a), toBig(b)))
		if got != expected {
			t.Errorf("SaturatingSub(%v, %v) = %v, expected %v", a, b, got, expected)
		}
	})
}

func FuzzSaturatingMulInt8(f *testing.F) {
	fuzzSaturatingMul[int8](f)
}

func FuzzSaturatingMulInt16(f *testing.F) {
	fuzzSaturatingMul[int16](f)
}

func FuzzSaturatingMulInt32(f *testing.F) {
	fuzzSaturatingMul[int32](f)
}

func FuzzSaturatingMulInt64(f *testing.F) {
	fuzzSaturatingMul[int64](f)
}

func FuzzSaturatingNegInt8(f *testing.F) {
	fuzzSaturatingNeg[int8](f)
}

func FuzzSaturatingNegInt16(f *testing.F) {
	fuzzSaturatingNeg[int16](f)
}

func FuzzSaturatingNegInt32(f *testing.F) {
	fuzzSaturatingNeg[int32](f)
}

func FuzzSaturatingNegInt64(f *testing.F) {
	fuzzSaturatingNeg[int64](f)
}
