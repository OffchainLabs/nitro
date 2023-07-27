// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package prefixproofs

import (
	"github.com/ethereum/go-ethereum/common"
)

type MerkleExpansion []common.Hash

func NewEmptyMerkleExpansion() MerkleExpansion {
	return []common.Hash{}
}

func (me MerkleExpansion) Clone() MerkleExpansion {
	return append([]common.Hash{}, me...)
}

func (me MerkleExpansion) Compact() ([]common.Hash, uint64) {
	var comp []common.Hash
	size := uint64(0)
	for level, h := range me {
		if h != (common.Hash{}) {
			comp = append(comp, h)
			size += 1 << level
		}
	}
	return comp, size
}

func MerkleExpansionFromCompact(comp []common.Hash, size uint64) (MerkleExpansion, uint64) {
	var me []common.Hash
	numRead := uint64(0)
	i := uint64(1)
	for i <= size {
		if i&size != 0 {
			numRead++
			me = append(me, comp[0])
			comp = comp[1:]
		} else {
			me = append(me, common.Hash{})
		}
		i <<= 1
	}
	return me, numRead
}

func ExpansionFromLeaves(leaves []common.Hash) (MerkleExpansion, error) {
	ret := NewEmptyMerkleExpansion()
	for _, leaf := range leaves {
		appended, err := AppendLeaf(ret, leaf)
		if err != nil {
			return nil, err
		}
		ret = appended
	}
	return ret, nil
}

type MerkleExpansionRootFetcherFunc = func(leaves []common.Hash, upTo uint64) (common.Hash, error)

func RootFetcherFromExpansion(leaves []common.Hash, upTo uint64) (common.Hash, error) {
	exp, err := ExpansionFromLeaves(leaves[:upTo])
	if err != nil {
		return common.Hash{}, err
	}
	return Root(exp)
}
