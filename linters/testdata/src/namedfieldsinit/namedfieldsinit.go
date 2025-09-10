package namedfieldsinit

type SmallStruct struct {
	A int
	B string
	C bool
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
}
