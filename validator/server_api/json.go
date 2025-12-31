// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package server_api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
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
	Id              uint64
	HasDelayedMsg   bool
	DelayedMsgNr    uint64
	PreimagesB64    map[arbutil.PreimageType]*jsonapi.PreimagesMapJson
	BatchInfo       []BatchInfoJson
	DelayedMsgB64   string
	StartState      validator.GoGlobalState
	UserWasms       map[rawdb.WasmTarget]map[common.Hash]string
	DebugChain      bool
	MaxUserWasmSize uint64 `json:"max-user-wasmSize,omitempty"`
}

// Marshal returns the JSON encoding of the InputJSON.
func (i *InputJSON) Marshal() ([]byte, error) {
	return json.MarshalIndent(i, "", "    ")
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
		UserWasms:     make(map[rawdb.WasmTarget]map[common.Hash]string),
		DebugChain:    entry.DebugChain,
	}
	for _, binfo := range entry.BatchInfo {
		encData := base64.StdEncoding.EncodeToString(binfo.Data)
		res.BatchInfo = append(res.BatchInfo, BatchInfoJson{Number: binfo.Number, DataB64: encData})
	}
	maxWasmSize := 0
	for target, wasms := range entry.UserWasms {
		archWasms := make(map[common.Hash]string)
		for moduleHash, data := range wasms {
			if len(data) > maxWasmSize {
				maxWasmSize = len(data)
			}
			compressed, err := arbcompress.CompressLevel(data, 1)
			if err != nil {
				continue
			}
			archWasms[moduleHash] = base64.StdEncoding.EncodeToString(compressed)
		}
		res.UserWasms[target] = archWasms
	}
	res.MaxUserWasmSize = uint64(maxWasmSize) //nolint:gosec
	return res
}

func ValidationInputFromJson(entry *InputJSON) (*validator.ValidationInput, error) {
	preimages := make(daprovider.PreimagesMap)
	for ty, jsonPreimages := range entry.PreimagesB64 {
		preimages[ty] = jsonPreimages.Map
	}
	valInput := &validator.ValidationInput{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		StartState:    entry.StartState,
		Preimages:     preimages,
		UserWasms:     make(map[rawdb.WasmTarget]map[common.Hash][]byte),
		BatchInfo:     nil,
		DelayedMsg:    nil,
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
	maxWasmSize := entry.MaxUserWasmSize + 10_000
	if maxWasmSize < 2_000_000 {
		maxWasmSize = 2_000_000
	}
	if maxWasmSize > 256_000_000 {
		return nil, fmt.Errorf("refusing maxWasmSize: %d", maxWasmSize)
	}
	for target, wasms := range entry.UserWasms {
		archWasms := make(map[common.Hash][]byte)
		for moduleHash, encoded := range wasms {
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, err
			}
			uncompressed, err := arbcompress.Decompress(decoded, int(maxWasmSize))
			if err != nil {
				return nil, err
			}
			archWasms[moduleHash] = uncompressed
		}
		valInput.UserWasms[target] = archWasms
	}
	return valInput, nil
}
