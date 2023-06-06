// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

type ClassicOutboxRetriever struct {
	db ethdb.Database
}

func NewClassicOutboxRetriever(db ethdb.Database) *ClassicOutboxRetriever {
	return &ClassicOutboxRetriever{
		db: db,
	}
}

type ClassicOutboxMsg struct {
	ProofNodes [][32]byte
	PathInt    *big.Int
	Data       []byte
}

func msgBatchKey(batchNum *big.Int) []byte {
	return crypto.Keccak256(append([]byte("msgBatch"), batchNum.Bytes()...))
}

func (m *ClassicOutboxRetriever) Msg(batchNum *big.Int, index uint64) (*ClassicOutboxMsg, error) {
	batchHeader, err := m.db.Get(msgBatchKey(batchNum))
	if err != nil {
		return nil, fmt.Errorf("%w: batch %d not found", err, batchNum)
	}
	if len(batchHeader) != 40 {
		return nil, fmt.Errorf("unexpected batch header: %v", batchHeader)
	}
	merkleSize := binary.BigEndian.Uint64(batchHeader[0:8])
	lowest := uint64(0)
	var root common.Hash
	copy(root[:], batchHeader[8:40])
	if merkleSize < index {
		return nil, fmt.Errorf("batch %d only has %d indexes", batchNum, merkleSize)
	}
	proofNodes := [][32]byte{}
	pathInt := big.NewInt(0)
	for merkleSize > 1 {
		merkleNode, err := m.db.Get(root[:])
		if err != nil {
			return nil, err
		}
		if len(merkleNode) != 64 {
			return nil, errors.New("unexpected merkle node")
		}
		// left side is always full
		var merkleLeftSize uint64
		if bits.OnesCount64(merkleSize) == 1 {
			merkleLeftSize = merkleSize / 2
		} else {
			merkleLeftSize = uint64(1) << (bits.Len64(merkleSize) - 1)
		}
		var leftHash, rightHash [32]byte
		copy(leftHash[:], merkleNode[0:32])
		copy(rightHash[:], merkleNode[32:64])
		pathInt.Mul(pathInt, common.Big2)
		if index < lowest+merkleLeftSize {
			// take a left turn
			copy(root[:], leftHash[:])
			proofNodes = append([][32]byte{rightHash}, proofNodes...)
			// lowest doesn't change
			merkleSize = merkleLeftSize
			pathInt.Add(pathInt, common.Big1)
		} else {
			// take a right turn
			copy(root[:], rightHash[:])
			proofNodes = append([][32]byte{leftHash}, proofNodes...)
			lowest = lowest + merkleLeftSize
			merkleSize = merkleSize - merkleLeftSize
			// equivalent bit in pathInt is zero
		}
	}
	if index != lowest {
		return nil, errors.New("unexpected error moving through merkle tree")
	}
	data, err := m.db.Get(root[:])
	if err != nil {
		return nil, err
	}
	return &ClassicOutboxMsg{
		ProofNodes: proofNodes,
		PathInt:    pathInt,
		Data:       data,
	}, nil
}
