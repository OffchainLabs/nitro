// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm
// +build !wasm

package melwavmio

import (
	"encoding/json"
	"errors"
	"flag"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

var (
	preimages     = make(map[common.Hash][]byte)
	lastBlockHash = common.Hash{}
	startMelRoot  = common.Hash{}
	endMelRoot    = common.Hash{} // This is set by the stubbed SetEndMELRoot function
)

func StubInit() {
	lastBlockFlag := flag.String("last-block-hash", "0000000000000000000000000000000000000000000000000000000000000000", "lastBlockHash")
	startMelRootFlag := flag.String("start-mel-root", "0000000000000000000000000000000000000000000000000000000000000000", "startMelRoot")
	preimagesPath := flag.String("preimages", "", "file to load preimages from")
	flag.Parse()
	lastBlockHash = common.HexToHash(*lastBlockFlag)
	startMelRoot = common.HexToHash(*startMelRootFlag)
	fileBytes, err := os.ReadFile(*preimagesPath)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(fileBytes, &preimages); err != nil {
		panic(err)
	}
}

func StubFinal() {
	log.Info("endMELRoot", endMelRoot.Hex())
}

func GetStartMELRoot() (hash common.Hash) {
	hash = startMelRoot
	return
}

func GetEndParentChainBlockHash() (hash common.Hash) {
	hash = lastBlockHash
	return
}

func SetEndMELRoot(hash common.Hash) {
	endMelRoot = hash
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	val, ok := preimages[hash]
	if !ok {
		return []byte{}, errors.New("preimage not found")
	}
	return val, nil
}
