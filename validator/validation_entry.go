package validator

import (
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
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
	UserWasms     state.UserWasms
	BatchInfo     []BatchInfo
	DelayedMsg    []byte
	StartState    GoGlobalState
	DebugChain    bool

	L1BlockHeight uint64
	// The validating hotshot commitment
	HotShotCommitment   espressoTypes.Commitment
	HotShotAvailability bool
}
