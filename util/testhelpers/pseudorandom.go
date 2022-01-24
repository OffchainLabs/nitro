//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package testhelpers

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type PseudoRandomDataSource struct {
	rand *rand.Rand
}

// pseudorandom source that repeats on different executions
// T param is to make sure it's only used in testing
func NewPseudoRandomDataSource(_ *testing.T, seed int64) *PseudoRandomDataSource {
	return &PseudoRandomDataSource{
		rand: rand.New(rand.NewSource(seed)),
	}
}

func (r *PseudoRandomDataSource) GetHash() common.Hash {
	var outHash common.Hash
	r.rand.Read(outHash[:])
	return outHash
}

func (r *PseudoRandomDataSource) GetAddress() common.Address {
	return common.BytesToAddress(r.GetHash().Bytes()[:20])
}

func (r *PseudoRandomDataSource) GetUint64() uint64 {
	return r.rand.Uint64()
}

func (r *PseudoRandomDataSource) GetData(size int) []byte {
	outArray := make([]byte, size)
	r.rand.Read(outArray)
	return outArray
}
