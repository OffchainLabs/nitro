// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package malicious

import "sync"

type Config struct {
	Enabled                   bool
	OverrideWasmModuleRoot    bool
	AllowGasEstimationFailure bool
}

// InboxMutationOffset selects a byte in the serialized sequencer batch data.
// 40 bytes of header + 24 bytes into payload keeps the header intact.
const InboxMutationOffset = 64

const inboxMutationMask = 0x01

var (
	mu  sync.RWMutex
	cfg Config
)

func SetConfig(c Config) {
	mu.Lock()
	cfg = c
	mu.Unlock()
}

func GetConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

func Enabled() bool {
	return GetConfig().Enabled
}

func OverrideWasmModuleRoot() bool {
	return GetConfig().OverrideWasmModuleRoot
}

func AllowGasEstimationFailure() bool {
	return GetConfig().AllowGasEstimationFailure
}

func MutateInboxMessage(data []byte) []byte {
	if !Enabled() {
		return data
	}
	if len(data) <= InboxMutationOffset {
		return data
	}
	mutated := make([]byte, len(data))
	copy(mutated, data)
	mutated[InboxMutationOffset] ^= inboxMutationMask
	return mutated
}

func MutateInboxMessageInPlace(data []byte) {
	if !Enabled() || len(data) <= InboxMutationOffset {
		return
	}
	data[InboxMutationOffset] ^= inboxMutationMask
}
