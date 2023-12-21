package validator

import (
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

type BatchInfo struct {
	Number uint64
	Data   []byte
}

type ValidationInput struct {
	Id                uint64
	HasDelayedMsg     bool
	DelayedMsgNr      uint64
	Preimages         map[arbutil.PreimageType]map[common.Hash][]byte
	BatchInfo         []BatchInfo
	DelayedMsg        []byte
	StartState        GoGlobalState
	HotShotCommitment espressoTypes.Commitment
}
