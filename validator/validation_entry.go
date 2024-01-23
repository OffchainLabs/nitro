package validator

import (
	"github.com/ethereum/go-ethereum/common"
)

type BatchInfo struct {
	Number    uint64
	Data      []byte
	BlockHash common.Hash
}

type ValidationInput struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	Preimages     map[common.Hash][]byte
	BatchInfo     []BatchInfo
	DelayedMsg    []byte
	StartState    GoGlobalState
}
