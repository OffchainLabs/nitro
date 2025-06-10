// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm
// +build !wasm

package melwavmio

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

var (
	preimages = make(map[common.Hash][]byte)
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

func SetEndMELRoot(hash common.Hash) {
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	val, ok := preimages[hash]
	if !ok {
		return []byte{}, errors.New("preimage not found")
	}
	return val, nil
}
