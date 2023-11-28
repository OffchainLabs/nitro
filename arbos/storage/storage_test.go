package storage

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
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
		expectedRawVal := common.BigToHash(arbmath.U256(in))
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

func TestOpenCachedSubStorage(t *testing.T) {
	s := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	var subSpaceIDs [][]byte
	for i := 0; i < 20; i++ {
		subSpaceIDs = append(subSpaceIDs, []byte{byte(rand.Intn(0xff))})
	}
	var expectedKeys [][]byte
	for _, subSpaceID := range subSpaceIDs {
		expectedKeys = append(expectedKeys, crypto.Keccak256(s.storageKey, subSpaceID))
	}
	n := len(subSpaceIDs) * 50
	start := make(chan struct{})
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		j := i % len(subSpaceIDs)
		subSpaceID, expectedKey := subSpaceIDs[j], expectedKeys[j]
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ss := s.OpenCachedSubStorage(subSpaceID)
			if !bytes.Equal(ss.storageKey, expectedKey) {
				errs <- fmt.Errorf("unexpected storage key, want: %v, have: %v", expectedKey, ss.storageKey)
			}
		}()
	}
	close(start)
	wg.Wait()
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
}

func TestMapAddressCache(t *testing.T) {
	s := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	var keys []common.Hash
	for i := 0; i < 20; i++ {
		keys = append(keys, common.BytesToHash([]byte{byte(rand.Intn(0xff))}))
	}
	var expectedMapped []common.Hash
	for _, key := range keys {
		expectedMapped = append(expectedMapped, s.mapAddress(key))
	}
	n := len(keys) * 50
	start := make(chan struct{})
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		j := i % len(keys)
		key, expected := keys[j], expectedMapped[j]
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			mapped := s.mapAddress(key)
			if !bytes.Equal(mapped.Bytes(), expected.Bytes()) {
				errs <- fmt.Errorf("unexpected storage key, want: %v, have: %v", expected, mapped)
			}
		}()
	}
	close(start)
	wg.Wait()
	if len(errs) > 0 {
		t.Fatal(<-errs)
	}
}
