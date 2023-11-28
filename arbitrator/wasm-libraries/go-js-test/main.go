// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build js && wasm

package main

import "testing"

func main() {
	tests := []testing.InternalTest{
		{"TestBool", TestBool},
		{"TestString", TestString},
		{"TestInt", TestInt},
		{"TestIntConversion", TestIntConversion},
		{"TestFloat", TestFloat},
		{"TestObject", TestObject},
		{"TestEqual", TestEqual},
		{"TestNaN", TestNaN},
		{"TestUndefined", TestUndefined},
		{"TestNull", TestNull},
		{"TestLength", TestLength},
		{"TestGet", TestGet},
		{"TestSet", TestSet},
		{"TestIndex", TestIndex},
		{"TestSetIndex", TestSetIndex},
		{"TestCall", TestCall},
		{"TestInvoke", TestInvoke},
		{"TestNew", TestNew},
		{"TestType", TestType},
		{"TestValueOf", TestValueOf},
		{"TestZeroValue", TestZeroValue},
		{"TestFuncOf", TestFuncOf},
		{"TestTruthy", TestTruthy},
		{"TestCopyBytesToGo", TestCopyBytesToGo},
		{"TestCopyBytesToJS", TestCopyBytesToJS},
		{"TestGlobal", TestGlobal},
		{"TestPoolHash", TestPoolHash},
	}

	// include all tests
	match := func(pat, str string) (bool, error) {
		return true, nil
	}
	testing.Main(match, tests, nil, nil)
}
