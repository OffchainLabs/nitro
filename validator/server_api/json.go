// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_api

import (
	"encoding/base64"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"

	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/validator"
)

type BatchInfoJson struct {
	Number  uint64
	DataB64 string
}

type UserWasmJson struct {
	Module string
	Asm    string
}

type ValidationInputJson struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	PreimagesB64  jsonapi.PreimagesMapJson
	BatchInfo     []BatchInfoJson
	UserWasms     map[string]UserWasmJson
	DelayedMsgB64 string
	StartState    validator.GoGlobalState
	DebugChain    bool
}

func ValidationInputToJson(entry *validator.ValidationInput) *ValidationInputJson {
	res := &ValidationInputJson{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		DelayedMsgB64: base64.StdEncoding.EncodeToString(entry.DelayedMsg),
		StartState:    entry.StartState,
		PreimagesB64:  jsonapi.NewPreimagesMapJson(entry.Preimages),
		UserWasms:     make(map[string]UserWasmJson),
		DebugChain:    entry.DebugChain,
	}
	for _, binfo := range entry.BatchInfo {
		encData := base64.StdEncoding.EncodeToString(binfo.Data)
		res.BatchInfo = append(res.BatchInfo, BatchInfoJson{binfo.Number, encData})
	}
	for moduleHash, info := range entry.UserWasms {
		encModuleHash := base64.StdEncoding.EncodeToString(moduleHash[:])
		encWasm := UserWasmJson{
			Asm:    base64.StdEncoding.EncodeToString(info.Asm),
			Module: base64.StdEncoding.EncodeToString(info.Module),
		}
		res.UserWasms[encModuleHash] = encWasm
	}
	return res
}

func ValidationInputFromJson(entry *ValidationInputJson) (*validator.ValidationInput, error) {
	valInput := &validator.ValidationInput{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		StartState:    entry.StartState,
		Preimages:     entry.PreimagesB64.Map,
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
		decModuleHash, err := base64.StdEncoding.DecodeString(moduleHash)
		if err != nil {
			return nil, err
		}
		asm, err := base64.StdEncoding.DecodeString(info.Asm)
		if err != nil {
			return nil, err
		}
		module, err := base64.StdEncoding.DecodeString(info.Module)
		if err != nil {
			return nil, err
		}
		decInfo := state.ActivatedWasm{
			Asm:    asm,
			Module: module,
		}
		valInput.UserWasms[common.Hash(decModuleHash)] = decInfo
	}
	return valInput, nil
}

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
