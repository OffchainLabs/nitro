// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_api

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/validator"
	"github.com/spf13/pflag"
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
}

type BatchInfoJson struct {
	Number  uint64
	DataB64 string
}

type RedisValidationServerConfig struct {
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
}

var DefaultRedisValidationServerConfig = RedisValidationServerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
}

var TestRedisValidationServerConfig = RedisValidationServerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
}

func RedisValidationServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
}
