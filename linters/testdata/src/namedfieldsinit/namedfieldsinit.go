// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package namedfieldsinit

import "namedfieldsinit/otherpkg"

type SmallStruct struct {
	A int
	B string
	C bool
}

// EdgeCaseExactThreshold has exactly 5 fields (the threshold)
type EdgeCaseExactThreshold struct {
	Field1 int
	Field2 string
	Field3 bool
	Field4 float64
	Field5 int64
}

// EdgeCaseAboveThreshold has 6 fields (threshold + 1)
type EdgeCaseAboveThreshold struct {
	Field1 int
	Field2 string
	Field3 bool
	Field4 float64
	Field5 int64
	Field6 []byte
}

type LargeStruct struct {
	Field1 int
	Field2 string
	Field3 bool
	Field4 float64
	Field5 int64
	Field6 []byte
	Field7 map[string]int
	Field8 interface{}
}

func testStructs() {
	// These should be OK - small struct
	_ = SmallStruct{1, "hello", true}
	_ = SmallStruct{
		A: 1,
		B: "hello",
		C: true,
	}

	// These should trigger the linter - large struct with positional init
	_ = LargeStruct{1, "hello", true, 3.14, 42, []byte{}, nil, nil} // want `has 8 fields and must use named field initialization`

	// This should be OK - large struct with named fields
	_ = LargeStruct{
		Field1: 1,
		Field2: "hello",
		Field3: true,
		Field4: 3.14,
		Field5: 42,
		Field6: []byte{},
		Field7: nil,
		Field8: nil,
	}

	// Another positional initialization that should trigger the linter
	_ = LargeStruct{2, "world", false, 2.71, 99, []byte{1, 2}, make(map[string]int), "test"} // want `has 8 fields and must use named field initialization`

	// Edge case: struct with exactly threshold (5) fields - should be OK with positional
	_ = EdgeCaseExactThreshold{1, "test", true, 3.14, 42}

	// Edge case: struct with threshold + 1 (6) fields - should trigger linter
	_ = EdgeCaseAboveThreshold{1, "test", true, 3.14, 42, []byte{}} // want `has 6 fields and must use named field initialization`

	// Edge case with named fields - should be OK
	_ = EdgeCaseAboveThreshold{
		Field1: 1,
		Field2: "test",
		Field3: true,
		Field4: 3.14,
		Field5: 42,
		Field6: []byte{},
	}

	// Test pointer initialization - should also trigger the linter
	_ = &LargeStruct{1, "ptr", true, 2.0, 10, []byte{}, nil, nil} // want `has 8 fields and must use named field initialization`

	// Pointer with named fields - should be OK
	_ = &LargeStruct{
		Field1: 1,
		Field2: "ptr",
		Field3: true,
		Field4: 2.0,
		Field5: 10,
		Field6: []byte{},
		Field7: nil,
		Field8: nil,
	}

	// Pointer to edge case struct
	_ = &EdgeCaseAboveThreshold{2, "ptr", false, 1.0, 5, []byte{1}} // want `has 6 fields and must use named field initialization`
}

func testCrossPackage() {
	// Test cross-package: Small struct from other package - should be OK with positional
	_ = otherpkg.ExportedSmall{1, "test", true}

	// Large struct from other package - should trigger linter
	_ = otherpkg.ExportedLarge{1, "test", true, 3.14, 42, []byte{}} // want `has 6 fields and must use named field initialization`

	// Large struct from other package with named fields - should be OK
	_ = otherpkg.ExportedLarge{
		Field1: 1,
		Field2: "test",
		Field3: true,
		Field4: 3.14,
		Field5: 42,
		Field6: []byte{},
	}

	// Pointer to large struct from other package - should also trigger
	_ = &otherpkg.ExportedLarge{2, "ptr", false, 1.0, 5, []byte{1}} // want `has 6 fields and must use named field initialization`
}
