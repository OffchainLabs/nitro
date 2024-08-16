package validator

import (
	"encoding/json"
	"fmt"

	"github.com/cespare/xxhash/v2"
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

	SelfHash string // Is a unique identifier which can be used to compare any two instances of validationInput
}

// SetSelfHash should be only called once. In the context of redis streams- by the producer, before submitting a request
func (v *ValidationInput) SetSelfHash() {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return
	}
	v.SelfHash = fmt.Sprintf("%d", xxhash.Sum64(jsonData))
}
