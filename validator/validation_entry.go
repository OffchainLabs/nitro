package validator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/daprovider"
)

type BatchInfo struct {
	Number uint64
	Data   []byte
}

// lint:require-exhaustive-initialization
type ValidationInput struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	Preimages     daprovider.PreimagesMap
	UserWasms     map[rawdb.WasmTarget]map[common.Hash][]byte
	BatchInfo     []BatchInfo
	DelayedMsg    []byte
	StartState    GoGlobalState
	DebugChain    bool
}
