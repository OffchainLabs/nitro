package storage

import (
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
	for _, in := range []*big.Int{
		new(big.Int).Sub(twoToThe255, big.NewInt(1)), // MaxUint256
		new(big.Int).Neg(twoToThe255),                // MinUint256
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(33),
		big.NewInt(31591083),
		big.NewInt(-1),
		big.NewInt(-33),
		big.NewInt(-31591083),
	} {
		err := sbbi.Set(in)
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
		twoToThe255, // MaxUint256 + 1
		new(big.Int).Sub(big.NewInt(-1), twoToThe255), // MinUint256 - 1
		new(big.Int).Add(big.NewInt(1), twoToThe255),  // MaxUint256 + 2
		new(big.Int).Sub(big.NewInt(-2), twoToThe255), // MinUint256 - 2
		new(big.Int).Mul(big.NewInt(2), twoToThe255),  // MaxUint256 * 2
		new(big.Int).Mul(big.NewInt(-2), twoToThe255), // MinUint256 * 2
	} {
		requirePanic(t, in, func() {
			_ = sbbi.Set(in)
		})
	}
}
