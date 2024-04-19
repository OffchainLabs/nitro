// Package validation is introduced to avoid cyclic depenency between validation
// client and validation api.
package validation

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/validator"
	"github.com/spf13/pflag"
)

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
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
}

var DefaultRedisValidationServerConfig = RedisValidationServerConfig{
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
}

var TestRedisValidationServerConfig = RedisValidationServerConfig{
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
}

func RedisValidationServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
}
