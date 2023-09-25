package storage

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func requirePanic(t *testing.T, testCase interface{}, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			Fatal(t, "panic expected but function exited successfully for test case", testCase)
		}
	}()
	f()
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fatal(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
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
		Require(t, err)
		rawVal, err := rawSlot.Get()
		Require(t, err)
		// Verify that our encoding matches geth's signed complement impl
		expectedRawVal := common.BigToHash(math.U256(new(big.Int).Set(in)))
		if rawVal != expectedRawVal {
			Fatal(t, "for input", in, "expected raw value", expectedRawVal, "but got", rawVal)
		}
		gotInverse := math.S256(rawVal.Big())
		if !arbmath.BigEquals(gotInverse, in) {
			Fatal(t, "for input", in, "expected raw value", rawVal, "to convert back into input but got", gotInverse)
		}
		out, err := sbbi.Get()
		Require(t, err)
		if in.Cmp(out) != 0 {
			Fatal(t, in, out, common.BytesToHash(out.Bytes()))
		}

		if in.BitLen() < 200 {
			err = sbbi.Set_preVersion7(in)
			Require(t, err)
			out, err = sbbi.Get()
			Require(t, err)
			if new(big.Int).Abs(in).Cmp(out) != 0 {
				Fatal(t, in, out, common.BytesToHash(out.Bytes()))
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
