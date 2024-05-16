// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMath(t *testing.T) {
	cases := []uint64{0, 1, 2, 3, 4, 7, 13, 28, 64}
	correctPower := []uint64{1, 2, 4, 4, 8, 8, 16, 32, 128}
	correctLog := []uint64{0, 1, 2, 2, 3, 3, 4, 5, 7}

	for i := 0; i < len(cases); i++ {
		calculated := NextPowerOf2(cases[i])
		if calculated != correctPower[i] {
			Fail(t, "expected power", correctPower[i], "but got", calculated)
		}
		calculated = Log2ceil(cases[i])
		if calculated != correctLog[i] {
			Fail(t, "expected log", correctLog[i], "but got", calculated)
		}
	}

	// try large random sqrts
	for i := 0; i < 100000; i++ {
		input := rand.Uint64() / 256
		approx := ApproxSquareRoot(input)
		correct := math.Sqrt(float64(input))
		diff := int(approx) - int(correct)
		if diff < -1 || diff > 1 {
			Fail(t, "sqrt approximation off by too much", diff, input, approx, correct)
		}
	}

	// try the first million sqrts
	for i := 0; i < 1000000; i++ {
		input := uint64(i)
		approx := ApproxSquareRoot(input)
		correct := math.Sqrt(float64(input))
		diff := int(approx) - int(correct)
		if diff < 0 || diff > 1 {
			Fail(t, "sqrt approximation off by too much", diff, input, approx, correct)
		}
	}

	// try powers of 2
	for i := 0; i < 63; i++ {
		input := uint64(1 << i)
		approx := ApproxSquareRoot(input)
		correct := math.Sqrt(float64(input))
		diff := int(approx) - int(correct)
		if diff != 0 {
			Fail(t, "incorrect", "2^", i, diff, approx, correct)
		}
	}

	assert := func(cond bool) {
		t.Helper()
		if !cond {
			Fail(t)
		}
	}
	assert(uint64(math.MaxInt64) == SaturatingUCast[uint64](int64(math.MaxInt64)))
	assert(uint64(math.MaxInt64-1) == SaturatingUCast[uint64](int64(math.MaxInt64-1)))
	assert(uint64(math.MaxInt64-1) == SaturatingUCast[uint64](math.MaxInt64-1))
	assert(uint64(math.MaxInt64) == SaturatingUCast[uint64](math.MaxInt64))
	assert(uint32(math.MaxUint32) == SaturatingUCast[uint32](math.MaxInt64-1))
	assert(uint16(math.MaxUint16) == SaturatingUCast[uint16](math.MaxInt32))
	assert(uint16(math.MaxUint16) == SaturatingUCast[uint16](math.MaxInt32-1))
	assert(uint16(math.MaxUint16) == SaturatingUCast[uint16](math.MaxInt-1))
	assert(uint8(math.MaxUint8) == SaturatingUCast[uint8](math.MaxInt-1))
	assert(uint(math.MaxInt-1) == SaturatingUCast[uint](math.MaxInt-1))
	assert(uint(math.MaxInt-1) == SaturatingUCast[uint](int64(math.MaxInt-1)))

	assert(int64(math.MaxInt64) == SaturatingCast[int64, uint64](math.MaxUint64))
	assert(int64(math.MaxInt64) == SaturatingCast[int64, uint64](math.MaxUint64-1))
	assert(int32(math.MaxInt32) == SaturatingCast[int32, uint64](math.MaxUint64))
	assert(int32(math.MaxInt32) == SaturatingCast[int32, uint64](math.MaxUint64-1))
	assert(int8(math.MaxInt8) == SaturatingCast[int8, uint16](math.MaxUint16))
	assert(int8(32) == SaturatingCast[int8, uint16](32))
	assert(int16(0) == SaturatingCast[int16, uint32](0))
	assert(int16(math.MaxInt16) == SaturatingCast[int16, uint32](math.MaxInt16))
	assert(int16(math.MaxInt16) == SaturatingCast[int16, uint16](math.MaxInt16))
	assert(int16(math.MaxInt8) == SaturatingCast[int16, uint8](math.MaxInt8))

	assert(uint32(math.MaxUint32) == SaturatingUUCast[uint32, uint64](math.MaxUint64))
	assert(uint32(math.MaxUint16) == SaturatingUUCast[uint32, uint64](math.MaxUint16))
	assert(uint32(math.MaxUint16) == SaturatingUUCast[uint32, uint16](math.MaxUint16))
	assert(uint16(math.MaxUint16) == SaturatingUUCast[uint16, uint16](math.MaxUint16))
}

func TestSlices(t *testing.T) {
	assert_eq := func(left, right []uint8) {
		t.Helper()
		if !bytes.Equal(left, right) {
			Fail(t, common.Bytes2Hex(left), " ", common.Bytes2Hex(right))
		}
	}

	data := []uint8{0, 1, 2, 3}
	assert_eq(SliceWithRunoff(data, 4, 4), data[0:0])
	assert_eq(SliceWithRunoff(data, 1, 0), data[0:0])
	assert_eq(SliceWithRunoff(data, 0, 0), data[0:0])
	assert_eq(SliceWithRunoff(data, 0, 1), data[0:1])
	assert_eq(SliceWithRunoff(data, 1, 3), data[1:3])
	assert_eq(SliceWithRunoff(data, 0, 4), data[0:4])
	assert_eq(SliceWithRunoff(data, 0, 5), data[0:4])
	assert_eq(SliceWithRunoff(data, 2, math.MaxUint8), data[2:4])

	assert_eq(SliceWithRunoff(data, -1, -2), []uint8{})
	assert_eq(SliceWithRunoff(data, 5, 3), []uint8{})
	assert_eq(SliceWithRunoff(data, 7, 8), []uint8{})
}

func testMinMaxSignedValues[T Signed](t *testing.T, min T, max T) {
	gotMin := MinSignedValue[T]()
	if gotMin != min {
		Fail(t, "expected min", min, "but got", gotMin)
	}
	gotMax := MaxSignedValue[T]()
	if gotMax != max {
		Fail(t, "expected max", max, "but got", gotMax)
	}
}

func TestMinMaxSignedValues(t *testing.T) {
	testMinMaxSignedValues[int8](t, math.MinInt8, math.MaxInt8)
	testMinMaxSignedValues[int16](t, math.MinInt16, math.MaxInt16)
	testMinMaxSignedValues[int32](t, math.MinInt32, math.MaxInt32)
	testMinMaxSignedValues[int64](t, math.MinInt64, math.MaxInt64)
}

func TestSaturatingAdd(t *testing.T) {
	tests := []struct {
		a, b, expected int64
	}{
		{2, 3, 5},
		{-1, -2, -3},
		{math.MaxInt64, 1, math.MaxInt64},
		{math.MaxInt64, math.MaxInt64, math.MaxInt64},
		{math.MinInt64, -1, math.MinInt64},
		{math.MinInt64, math.MinInt64, math.MinInt64},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v + %v = %v", tc.a, tc.b, tc.expected), func(t *testing.T) {
			sum := SaturatingAdd(int64(tc.a), int64(tc.b))
			if sum != tc.expected {
				t.Errorf("SaturatingAdd(%v, %v) = %v; want %v", tc.a, tc.b, sum, tc.expected)
			}
		})
	}
}

func TestSaturatingSub(t *testing.T) {
	tests := []struct {
		a, b, expected int64
	}{
		{5, 3, 2},
		{-3, -2, -1},
		{math.MinInt64, 1, math.MinInt64},
		{math.MinInt64, -1, math.MinInt64 + 1},
		{math.MinInt64, math.MinInt64, 0},
		{0, math.MinInt64, math.MaxInt64},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			sum := SaturatingSub(int64(tc.a), int64(tc.b))
			if sum != tc.expected {
				t.Errorf("SaturatingSub(%v, %v) = %v; want %v", tc.a, tc.b, sum, tc.expected)
			}
		})
	}
}

func TestSaturatingMul(t *testing.T) {
	tests := []struct {
		a, b, expected int64
	}{
		{5, 3, 15},
		{-3, -2, 6},
		{math.MaxInt64, 2, math.MaxInt64},
		{math.MinInt64, 2, math.MinInt64},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v - %v = %v", tc.a, tc.b, tc.expected), func(t *testing.T) {
			sum := SaturatingMul(int64(tc.a), int64(tc.b))
			if sum != tc.expected {
				t.Errorf("SaturatingMul(%v, %v) = %v; want %v", tc.a, tc.b, sum, tc.expected)
			}
		})
	}
}

func TestSaturatingNeg(t *testing.T) {
	tests := []struct {
		value    int64
		expected int64
	}{
		{0, 0},
		{5, -5},
		{-5, 5},
		{math.MinInt64, math.MaxInt64},
		{math.MaxInt64, math.MinInt64 + 1},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("-%v = %v", tc.value, tc.expected), func(t *testing.T) {
			result := SaturatingNeg(tc.value)
			if result != tc.expected {
				t.Errorf("SaturatingNeg(%v) = %v: expected %v", tc.value, result, tc.expected)
			}
		})
	}
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
