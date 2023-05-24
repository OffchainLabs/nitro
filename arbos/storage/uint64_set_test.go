package storage

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestUint64Set(t *testing.T) {
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
