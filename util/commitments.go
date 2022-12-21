package util

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type HistoryCommitment struct {
	Height uint64
	Merkle common.Hash
}

func (comm HistoryCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, comm.Height), comm.Merkle.Bytes())
}
