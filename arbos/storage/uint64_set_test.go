package storage

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestUint64SetSplitKeyValue(t *testing.T) {
	set := Uint64Set{}
	k, v := set.splitKeyValue(uint64(0x123456789abcdef0))
	if k != uint64(0x123456789abcde00) {
		Fail(t, "invalid key, want:", uint64(0x123456789abcde00), "have:", k)
	}
	if v != int(0xf0) {
		Fail(t, "invalid value, want:", uint64(0xf0), "have:", v)
	}
	k, v = set.splitKeyValue(uint64(0))
	if k != uint64(0) {
		Fail(t, "invalid key, want:", uint64(0), "have:", k)
	}
	if v != int(0) {
		Fail(t, "invalid value, want:", uint64(0), "have:", v)
	}
}

func TestUint64Set(t *testing.T) {
	// TODO(magic) add more complex test
	testElement := uint64(0x1234)
	storage := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	InitializeUint64Set(storage) // called just in case we add some initialization in future
	set := OpenUint64Set(storage)
	isMember, err := set.IsMember(testElement)
	Require(t, err)
	if isMember {
		Fail(t, "invalid IsMember result, returned true for nonmember")
	}
	removed, err := set.Remove(testElement)
	Require(t, err)
	if removed {
		Fail(t, "invalid Remove result, returned true when removing nonmember")
	}
	added, err := set.Add(testElement)
	Require(t, err)
	if !added {
		Fail(t, "invalid Add result, returned false when adding new element")
	}
	isMember, err = set.IsMember(testElement)
	Require(t, err)
	if !isMember {
		Fail(t, "invalid IsMember result, returned false for member")
	}
	added, err = set.Add(testElement)
	Require(t, err)
	if added {
		Fail(t, "invalid Add result, returned true when adding already existing element")
	}
	removed, err = set.Remove(testElement)
	Require(t, err)
	if !removed {
		Fail(t, "invalid Remove result, returned false when removing member")
	}
	removed, err = set.Remove(testElement)
	Require(t, err)
	if removed {
		Fail(t, "invalid Remove result, returned true when removing nonmember")
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
