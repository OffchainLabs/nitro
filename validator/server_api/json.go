// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
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

func RedisStreamForRoot(prefix string, moduleRoot common.Hash) string {
	return fmt.Sprintf("%sstream:%s", prefix, moduleRoot.Hex())
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
}

func (i *InputJSON) WriteToFile() error {
	contents, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(fmt.Sprintf("block_inputs_%d.json", i.Id), contents, 0600); err != nil {
		return err
	}
	return nil
}

type UserWasmJson struct {
	Module string
	Asm    map[string]string
}

type BatchInfoJson struct {
	Number  uint64
	DataB64 string
}

func ValidationInputToJson(entry *validator.ValidationInput) *InputJSON {
	jsonPreimagesMap := make(map[arbutil.PreimageType]*jsonapi.PreimagesMapJson)
	for ty, preimages := range entry.Preimages {
		jsonPreimagesMap[ty] = jsonapi.NewPreimagesMapJson(preimages)
	}
	res := &InputJSON{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		DelayedMsgB64: base64.StdEncoding.EncodeToString(entry.DelayedMsg),
		StartState:    entry.StartState,
		PreimagesB64:  jsonPreimagesMap,
		UserWasms:     make(map[common.Hash]UserWasmJson),
		DebugChain:    entry.DebugChain,
	}
	for _, binfo := range entry.BatchInfo {
		encData := base64.StdEncoding.EncodeToString(binfo.Data)
		res.BatchInfo = append(res.BatchInfo, BatchInfoJson{Number: binfo.Number, DataB64: encData})
	}
	for moduleHash, asmMap := range entry.UserWasms {
		asmMapEncoded := make(map[string]string, len(asmMap))
		for target, asm := range asmMap {
			asmMapEncoded[target] = base64.StdEncoding.EncodeToString(asm)
		}
		// TODO
		module := asmMapEncoded[rawdb.TargetWavm]
		encWasm := UserWasmJson{
			Asm:    asmMapEncoded,
			Module: module,
		}
		res.UserWasms[moduleHash] = encWasm
	}
	return res
}

func ValidationInputFromJson(entry *InputJSON) (*validator.ValidationInput, error) {
	preimages := make(map[arbutil.PreimageType]map[common.Hash][]byte)
	for ty, jsonPreimages := range entry.PreimagesB64 {
		preimages[ty] = jsonPreimages.Map
	}
	valInput := &validator.ValidationInput{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		StartState:    entry.StartState,
		Preimages:     preimages,
		UserWasms:     make(state.UserWasms),
		DebugChain:    entry.DebugChain,
	}
	delayed, err := base64.StdEncoding.DecodeString(entry.DelayedMsgB64)
	if err != nil {
		return nil, err
	}
	valInput.DelayedMsg = delayed
	for _, binfo := range entry.BatchInfo {
		data, err := base64.StdEncoding.DecodeString(binfo.DataB64)
		if err != nil {
			return nil, err
		}
		decInfo := validator.BatchInfo{
			Number: binfo.Number,
			Data:   data,
		}
		valInput.BatchInfo = append(valInput.BatchInfo, decInfo)
	}
	for moduleHash, info := range entry.UserWasms {
		asmMapDecoded := make(map[string][]byte, len(info.Asm))
		for target, asmString := range info.Asm {
			asm, err := base64.StdEncoding.DecodeString(asmString)
			if err != nil {
				return nil, err
			}
			asmMapDecoded[target] = asm
		}
		// TODO
		module, err := base64.StdEncoding.DecodeString(info.Module)
		if err != nil {
			return nil, err
		}
		asmMapDecoded[rawdb.TargetWavm] = module
		valInput.UserWasms[moduleHash] = asmMapDecoded
	}
	return valInput, nil
}
