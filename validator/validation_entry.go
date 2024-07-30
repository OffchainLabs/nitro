package validator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

type BatchInfo struct {
	Number    uint64
	BlockHash common.Hash
	Data      []byte
}

type ValidationInput struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	Preimages     map[arbutil.PreimageType]map[common.Hash][]byte
	UserWasms     map[string]map[common.Hash][]byte
	BatchInfo     []BatchInfo
	DelayedMsg    []byte
	StartState    GoGlobalState
	DebugChain    bool
}
