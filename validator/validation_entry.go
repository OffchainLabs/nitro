package validator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/espresso"
	"github.com/offchainlabs/nitro/arbutil"
)

type BatchInfo struct {
	Number            uint64
	HotShotCommitment *espresso.Commitment
	Data              []byte
}

type ValidationInput struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	Preimages     map[arbutil.PreimageType]map[common.Hash][]byte
	BatchInfo     []BatchInfo
	DelayedMsg    []byte
	StartState    GoGlobalState
}
