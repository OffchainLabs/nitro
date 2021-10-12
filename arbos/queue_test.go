package arbos

import "testing"

func TestQueue(t *testing.T) {
	state := OpenArbosStateForTest()
	q := AllocateQueueInStorage(state)

	if ! q.IsEmpty() {
		t.Fail()
	}

	val0 := int64(853139508)
	for i := 0; i < 150; i++ {
		val := IntToHash(val0 + int64(i))
		q.Put(val)
		if q.IsEmpty() {
			t.Fail()
		}
	}

	for i := 0; i < 150; i++ {
		val := IntToHash(val0 + int64(i))
		res := q.Get()
		if res.Big().Cmp(val.Big()) != 0 {
			t.Fail()
		}
	}

	if ! q.IsEmpty() {
		t.Fail()
	}
}


