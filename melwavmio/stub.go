// Copyright 2026-2027, Offchain Labs, Inc.
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
	preimages               = make(map[common.Hash][]byte)
	startMelStateHash       = common.Hash{}
	melMsgHash              = common.Hash{}
	endMelStateHash         = common.Hash{} // This is set by the stubbed SetEndMELStateHash function
	endParentChainBlockHash = common.Hash{} // This is set by the stubbed GetEndParentChainBlockHash function
)

func StubInit() {
	endParentChainBlockHashFlag := flag.String("end-parent-chain-block-hash", "0000000000000000000000000000000000000000000000000000000000000000", "endParentChainBlockHash")
	startMelRootFlag := flag.String("start-mel-root", "0000000000000000000000000000000000000000000000000000000000000000", "startMelHash")
	preimagesPath := flag.String("preimages", "", "file to load preimages from")
	flag.Parse()
	endParentChainBlockHash = common.HexToHash(*endParentChainBlockHashFlag)
	startMelStateHash = common.HexToHash(*startMelRootFlag)
	fileBytes, err := os.ReadFile(*preimagesPath)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(fileBytes, &preimages); err != nil {
		panic(err)
	}
}

func StubFinal() {
	log.Info("endMelStateHash", endMelStateHash.Hex())
}

func GetMELMsgHash() (hash common.Hash) {
	hash = melMsgHash
	return
}

func SetMELMsgHash(hash common.Hash) {
	melMsgHash = hash
}

func GetStartMELRoot() (hash common.Hash) {
	hash = startMelStateHash
	return
}

func GetEndParentChainBlockHash() (hash common.Hash) {
	hash = endParentChainBlockHash
	return
}

func SetEndMELRoot(hash common.Hash) {
	endMelStateHash = hash
}

func GetPositionInMEL() uint64 {
	return 0
}

func IncreasePositionInMEL() {
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	val, ok := preimages[hash]
	if !ok {
		return []byte{}, errors.New("preimage not found")
	}
	return val, nil
}
