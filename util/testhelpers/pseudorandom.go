// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package testhelpers

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type PseudoRandomDataSource struct {
	rand *rand.Rand
}

// NewPseudoRandomDataSource is the pseudorandom source that repeats on different executions
// T param is to make sure it's only used in testing
func NewPseudoRandomDataSource(_ *testing.T, seed int64) *PseudoRandomDataSource {
	return &PseudoRandomDataSource{
		rand: rand.New(rand.NewSource(seed)),
	}
}

func (r *PseudoRandomDataSource) Hash() common.Hash {
	var outHash common.Hash
	r.rand.Read(outHash[:])
	return outHash
}

func (r *PseudoRandomDataSource) Address() common.Address {
	return common.BytesToAddress(r.Hash().Bytes()[:20])
}

func (r *PseudoRandomDataSource) Uint64() uint64 {
	return r.rand.Uint64()
}

func (r *PseudoRandomDataSource) Data(size int) []byte {
	outArray := make([]byte, size)
	r.rand.Read(outArray)
	return outArray
}
