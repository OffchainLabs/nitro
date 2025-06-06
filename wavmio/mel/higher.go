// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm
// +build wasm

package melwavmio

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

func StubInit() {
}

func StubFinal() {
}

func GetStartMELRoot() (hash common.Hash) {
	return
}

func GetEndParentChainBlockHash() (hash common.Hash) {
	return
}

func SetMELStateHash(hash common.Hash) {
	// This function is a stub and does not do anything in this context.
	// In a real implementation, it would set the MEL state hash in the global state.
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return []byte{}, errors.New("preimage not found")
}
