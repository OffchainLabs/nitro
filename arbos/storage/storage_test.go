package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/burn"
	"math/big"
	"testing"
)

func TestStorageBackedBigInt(t *testing.T) {
	sto := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	sbbi := sto.OpenStorageBackedBigInt(0)

	for _, testCase := range []int64{0, 1, 33, 31591083, -1, -33, -31591083} {
		in := big.NewInt(testCase)
		err := sbbi.Set(in)
		if err != nil {
			t.Fatal(err)
		}
		out, err := sbbi.Get()
		if err != nil {
			t.Fatal(err)
		}
		if in.Cmp(out) != 0 {
			t.Fatal(in, out, common.BytesToHash(out.Bytes()))
		}

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

	twoToThe255 := new(big.Int).Lsh(big.NewInt(1), 255)
	for _, in := range []*big.Int{
		new(big.Int).Sub(twoToThe255, big.NewInt(1)), // MaxUint256
		new(big.Int).Neg(twoToThe255),                // MinUint256
	} {
		err := sbbi.Set(in)
		if err != nil {
			t.Fatal(err)
		}
		out, err := sbbi.Get()
		if err != nil {
			t.Fatal(err)
		}
		if in.Cmp(out) != 0 {
			t.Fatal(in, out, common.BytesToHash(out.Bytes()))
		}
	}
}
