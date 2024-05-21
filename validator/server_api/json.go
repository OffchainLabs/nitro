// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_api

import (
	"fmt"

	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"

	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/validator"
)

const Namespace string = "validation"

type MachineStepResultJson struct {
	Hash        common.Hash
	Position    uint64
	Status      uint8
	GlobalState validator.GoGlobalState
}

func MachineStepResultToJson(result *validator.MachineStepResult) *MachineStepResultJson {
	return &MachineStepResultJson{
		Hash:        result.Hash,
		Position:    result.Position,
		Status:      uint8(result.Status),
		GlobalState: result.GlobalState,
	}
}

func MachineStepResultFromJson(resultJson *MachineStepResultJson) (*validator.MachineStepResult, error) {

	return &validator.MachineStepResult{
		Hash:        resultJson.Hash,
		Position:    resultJson.Position,
		Status:      validator.MachineStatus(resultJson.Status),
		GlobalState: resultJson.GlobalState,
	}, nil
}

func RedisStreamForRoot(moduleRoot common.Hash) string {
	return fmt.Sprintf("stream:%s", moduleRoot.Hex())
}

type Request struct {
	Input      *InputJSON
	ModuleRoot common.Hash
}

type InputJSON struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	PreimagesB64  map[arbutil.PreimageType]*jsonapi.PreimagesMapJson
	BatchInfo     []BatchInfoJson
	DelayedMsgB64 string
	StartState    validator.GoGlobalState
	UserWasms     map[common.Hash]UserWasmJson
	DebugChain    bool

	L1BlockHeight       uint64
	HotShotAvailability bool
	HotShotCommitment   espressoTypes.Commitment
}

type UserWasmJson struct {
	Module string
	Asm    string
}

type BatchInfoJson struct {
	Number  uint64
	DataB64 string
}
