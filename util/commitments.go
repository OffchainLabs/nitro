package util

import (
	"github.com/ethereum/go-ethereum/common"
)

type HistoryCommitment struct {
	Height uint64
	Merkle common.Hash
}
