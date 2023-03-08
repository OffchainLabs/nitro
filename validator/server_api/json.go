package server_api

import (
	"encoding/base64"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

type BatchInfoJson struct {
	Number  uint64
	DataB64 string
}

type ValidationInputJson struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	PreimagesB64  map[string]string
	BatchInfo     []BatchInfoJson
	DelayedMsgB64 string
	StartState    validator.GoGlobalState
}

func ValidationInputToJson(entry *validator.ValidationInput) *ValidationInputJson {
	res := &ValidationInputJson{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		DelayedMsgB64: base64.StdEncoding.EncodeToString(entry.DelayedMsg),
		StartState:    entry.StartState,
		PreimagesB64:  make(map[string]string),
	}
	for hash, data := range entry.Preimages {
		encHash := base64.StdEncoding.EncodeToString(hash.Bytes())
		encData := base64.StdEncoding.EncodeToString(data)
		res.PreimagesB64[encHash] = encData
	}
	for _, binfo := range entry.BatchInfo {
		encData := base64.StdEncoding.EncodeToString(binfo.Data)
		res.BatchInfo = append(res.BatchInfo, BatchInfoJson{binfo.Number, encData})
	}
	return res
}

func ValidationInputFromJson(entry *ValidationInputJson) (*validator.ValidationInput, error) {
	valInput := &validator.ValidationInput{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		StartState:    entry.StartState,
		Preimages:     make(map[common.Hash][]byte),
	}
	delayed, err := base64.StdEncoding.DecodeString(entry.DelayedMsgB64)
	if err != nil {
		return nil, err
	}
	valInput.DelayedMsg = delayed
	for encHash, encData := range entry.PreimagesB64 {
		hash, err := base64.StdEncoding.DecodeString(encHash)
		if err != nil {
			return nil, err
		}
		data, err := base64.StdEncoding.DecodeString(encData)
		if err != nil {
			return nil, err
		}
		valInput.Preimages[common.BytesToHash(hash)] = data
	}
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
