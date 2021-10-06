package arbos

import "testing"

func TestQueue(t *testing.T) {
	state := OpenArbosState(NewMemoryBackingEvmStorage(), IntToHash(100))
	q, err := NewQueue(state)
	if err != nil {
		t.Error(err)
	}

	if ! q.IsEmpty() {
		t.Fail()
	}

	val0 := int64(853139508)
	for i := 0; i < 150; i++ {
		val := IntToHash(val0 + int64(i))
		err = q.Put(val)
		if err != nil {
			t.Error(err)
		}
		if q.IsEmpty() {
			t.Fail()
		}
	}

	for i := 0; i < 150; i++ {
		val := IntToHash(val0 + int64(i))
		res, err := q.Get()
		if err != nil {
			t.Fatal(i, err)
		}
		if res.Big().Cmp(val.Big()) != 0 {
			t.Fail()
		}
	}

	if ! q.IsEmpty() {
		t.Fail()
	}
}


