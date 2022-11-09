package storage

import (
	"bytes"
	"github.com/offchainlabs/nitro/arbos/util"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func requirePanic(t *testing.T, testCase interface{}, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("panic expected but function exited successfully for test case", testCase)
		}
	}()
	f()
}

func TestStorageBackedBigInt(t *testing.T) {
	sto := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	sbbi := sto.OpenStorageBackedBigInt(0)
	rawSlot := sto.NewSlot(0)

	twoToThe255 := new(big.Int).Lsh(big.NewInt(1), 255)
	maxUint256 := new(big.Int).Sub(twoToThe255, big.NewInt(1))
	minUint256 := new(big.Int).Neg(twoToThe255)
	for _, in := range []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(33),
		big.NewInt(31591083),
		big.NewInt(-1),
		big.NewInt(-33),
		big.NewInt(-31591083),
		maxUint256,
		minUint256,
	} {
		err := sbbi.SetChecked(in)
		if err != nil {
			t.Fatal(err)
		}
		rawVal, err := rawSlot.Get()
		if err != nil {
			t.Fatal(err)
		}
		// Verify that our encoding matches geth's signed complement impl
		expectedRawVal := common.BigToHash(math.U256(new(big.Int).Set(in)))
		if rawVal != expectedRawVal {
			t.Fatal("for input", in, "expected raw value", expectedRawVal, "but got", rawVal)
		}
		gotInverse := math.S256(rawVal.Big())
		if !arbmath.BigEquals(gotInverse, in) {
			t.Fatal("for input", in, "expected raw value", rawVal, "to convert back into input but got", gotInverse)
		}
		out, err := sbbi.Get()
		if err != nil {
			t.Fatal(err)
		}
		if in.Cmp(out) != 0 {
			t.Fatal(in, out, common.BytesToHash(out.Bytes()))
		}

		if in.BitLen() < 200 {
			err = sbbi.Set_preVersion7(in)
			if err != nil {
				t.Fatal(err)
			}
			out, err = sbbi.Get()
			if err != nil {
				t.Fatal(err)
			}
			if new(big.Int).Abs(in).Cmp(out) != 0 {
				t.Fatal(in, out, common.BytesToHash(out.Bytes()))
			}
		}
	}
	for _, in := range []*big.Int{
		new(big.Int).Add(maxUint256, big.NewInt(1)),
		new(big.Int).Sub(minUint256, big.NewInt(1)),
		new(big.Int).Mul(maxUint256, big.NewInt(2)),
		new(big.Int).Mul(minUint256, big.NewInt(2)),
		new(big.Int).Exp(maxUint256, big.NewInt(1025), nil),
		new(big.Int).Exp(minUint256, big.NewInt(1025), nil),
	} {
		requirePanic(t, in, func() {
			_ = sbbi.SetChecked(in)
		})
	}
}

func TestStorageBackedQueue(t *testing.T) {
	rootStorage := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	key1 := []byte("a key")
	key2 := []byte("another key")
	subSto := rootStorage.OpenSubStorage(key1).OpenSubStorage(key2)
	Require(t, InitializeQueue(subSto))

	q1 := OpenQueue(subSto)

	queueMap := MakeMapForQueue(RootStorageKey.SubspaceKey(key1).SubspaceKey(key2))
	q2 := queueMap.Open(rootStorage)

	if !bytes.Equal(q1.storage.StorageKey, q2.storage.StorageKey) {
		t.Fatal()
	}

	empty, err := q1.IsEmpty()
	Require(t, err)
	if !empty {
		t.Fatal()
	}
	empty, err = q2.IsEmpty()
	Require(t, err)
	if !empty {
		t.Fatal()
	}

	Require(t, q1.Put(util.UintToHash(13)))
	Require(t, q2.Put(util.UintToHash(34)))
	res, err := q2.Get()
	Require(t, err)
	if *res != util.UintToHash(13) {
		t.Fatal()
	}
	res, err = q1.Get()
	Require(t, err)
	if *res != util.UintToHash(34) {
		t.Fatal()
	}

	empty, err = q2.IsEmpty()
	Require(t, err)
	if !empty {
		t.Fatal()
	}
}

func Require(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
