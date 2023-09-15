package storage

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/burn"
)

func TestUint64SetSplitKeyValue(t *testing.T) {
	set := Uint64Set{}
	k, v := set.splitKeyValue(uint64(0x123456789abcdef0))
	if k != uint64(0x00123456789abcde) {
		Fatal(t, "invalid key, want:", uint64(0x00123456789abcde), "have:", k)
	}
	if v != int(0xf0) {
		Fatal(t, "invalid value, want:", uint64(0xf0), "have:", v)
	}
	k, v = set.splitKeyValue(uint64(0))
	if k != uint64(0) {
		Fatal(t, "invalid key, want:", uint64(0), "have:", k)
	}
	if v != int(0) {
		Fatal(t, "invalid value, want:", uint64(0), "have:", v)
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
		Fatal(t, "invalid IsMember result, returned true for nonmember")
	}
	removed, err := set.Remove(testElement)
	Require(t, err)
	if removed {
		Fatal(t, "invalid Remove result, returned true when removing nonmember")
	}
	added, err := set.Add(testElement)
	Require(t, err)
	if !added {
		Fatal(t, "invalid Add result, returned false when adding new element")
	}
	isMember, err = set.IsMember(testElement)
	Require(t, err)
	if !isMember {
		Fatal(t, "invalid IsMember result, returned false for member")
	}
	added, err = set.Add(testElement)
	Require(t, err)
	if added {
		Fatal(t, "invalid Add result, returned true when adding already existing element")
	}
	removed, err = set.Remove(testElement)
	Require(t, err)
	if !removed {
		Fatal(t, "invalid Remove result, returned false when removing member")
	}
	removed, err = set.Remove(testElement)
	Require(t, err)
	if removed {
		Fatal(t, "invalid Remove result, returned true when removing nonmember")
	}
}
