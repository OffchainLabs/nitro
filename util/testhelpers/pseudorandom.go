//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package testhelpers

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type PseudoRandomDataSource struct {
	salt  common.Hash
	index int64
}

// pseudorandom source that repeats on different executions
// T param is to make sure it's only used in testing
func NewPseudoRandomDataSource(_ *testing.T, saltParam int) *PseudoRandomDataSource {
	salt := crypto.Keccak256Hash([]byte{'s'}, common.BigToHash(big.NewInt(int64(saltParam))).Bytes())
	return &PseudoRandomDataSource{
		salt:  salt,
		index: 0,
	}
}

func (r *PseudoRandomDataSource) GetHash() common.Hash {
	r.index++
	return crypto.Keccak256Hash(r.salt[:], common.BigToHash(big.NewInt(r.index)).Bytes())
}

func (r *PseudoRandomDataSource) GetAddress() common.Address {
	return common.BytesToAddress(r.GetHash().Bytes()[:20])
}

func (r *PseudoRandomDataSource) GetUint64() uint64 {
	return binary.BigEndian.Uint64(r.GetHash().Bytes()[:8])
}

func (r *PseudoRandomDataSource) GetData(size int) []byte {
	ret := []byte{}
	for len(ret) < size {
		ret = append(ret, r.GetHash().Bytes()...)
	}
	return ret[:size]
}
