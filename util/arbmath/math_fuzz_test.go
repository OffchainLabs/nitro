// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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

// Fuzzing tests for Log2Floor and Log2Ceil functions
func FuzzLog2Floor(f *testing.F) {
	// Add some seed values to help the fuzzer
	seedValues := []uint64{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		15, 16, 17, 31, 32, 33, 63, 64, 65,
		127, 128, 129, 255, 256, 257,
		1023, 1024, 1025, 2047, 2048, 2049,
		65535, 65536, 65537,
		4294967295, 4294967296, 4294967297,
		18446744073709551615, // max uint64
	}
	for _, val := range seedValues {
		f.Add(val)
	}

	f.Fuzz(func(t *testing.T, u uint64) {
		result := Log2Floor(u)

		// Verify the result is non-negative
		if result < 0 {
			t.Errorf("Log2Floor(%d) = %d, expected non-negative result", u, result)
		}

		// For u == 0, result should be 0
		if u == 0 {
			if result != 0 {
				t.Errorf("Log2Floor(0) = %d, expected 0", result)
			}
			return
		}

		// For u > 0, verify that 2^result <= u < 2^(result+1)
		lowerBound := uint64(1) << result
		upperBound := uint64(1) << (result + 1)

		if u < lowerBound || u >= upperBound {
			t.Errorf("Log2Floor(%d) = %d, but 2^%d = %d and 2^%d = %d",
				u, result, result, lowerBound, result+1, upperBound)
		}
	})
}

func FuzzLog2Ceil(f *testing.F) {
	// Add some seed values to help the fuzzer
	seedValues := []uint64{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		15, 16, 17, 31, 32, 33, 63, 64, 65,
		127, 128, 129, 255, 256, 257,
		1023, 1024, 1025, 2047, 2048, 2049,
		65535, 65536, 65537,
		4294967295, 4294967296, 4294967297,
		18446744073709551615, // max uint64
	}
	for _, val := range seedValues {
		f.Add(val)
	}

	f.Fuzz(func(t *testing.T, u uint64) {
		result := Log2Ceil(u)

		// Verify the result is non-negative
		if result < 0 {
			t.Errorf("Log2Ceil(%d) = %d, expected non-negative result", u, result)
		}

		// For u == 0, result should be 0
		if u == 0 {
			if result != 0 {
				t.Errorf("Log2Ceil(0) = %d, expected 0", result)
			}
			return
		}

		// For u > 0, verify that 2^(result-1) < u <= 2^result
		if result > 0 {
			lowerBound := uint64(1) << (result - 1)
			upperBound := uint64(1) << result

			if u <= lowerBound || u > upperBound {
				t.Errorf("Log2Ceil(%d) = %d, but 2^%d = %d and 2^%d = %d",
					u, result, result-1, lowerBound, result, upperBound)
			}
		}

		// Verify that Log2Ceil(u) >= Log2Floor(u)
		floorResult := Log2Floor(u)
		if result < floorResult {
			t.Errorf("Log2Ceil(%d) = %d < Log2Floor(%d) = %d", u, result, u, floorResult)
		}

		// Verify that Log2Ceil(u) <= Log2Floor(u) + 1
		if result > floorResult+1 {
			t.Errorf("Log2Ceil(%d) = %d > Log2Floor(%d) + 1 = %d", u, result, u, floorResult+1)
		}
	})
}
