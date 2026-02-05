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
	preimages               = make(map[arbutil.PreimageType]map[common.Hash][]byte)
	startMelStateHash       = common.Hash{}
	melMsgHash              = common.Hash{}
	endMelStateHash         = common.Hash{} // This is set by the stubbed SetEndMELStateHash function
	endParentChainBlockHash = common.Hash{} // This is set by the stubbed GetEndParentChainBlockHash function
	lastBlockHash           = common.Hash{} // This is set by the stubbed GetLastBlockHash function
	positionInMEL           = uint64(0)
)

func StubInit() {
	endParentChainBlockHashFlag := flag.String("end-parent-chain-block-hash", "0000000000000000000000000000000000000000000000000000000000000000", "endParentChainBlockHash")
	startMelRootFlag := flag.String("start-mel-state-hash", "0000000000000000000000000000000000000000000000000000000000000000", "startMelHash")
	preimagesPath := flag.String("preimages", "", "file to load preimages from")
	positionInMELFlag := flag.Uint64("position-in-mel", 0, "positionInMEL")
	lastBlockHashFlag := flag.String("last-block-hash", "0000000000000000000000000000000000000000000000000000000000000000", "lastBlockHash")
	flag.Parse()
	endParentChainBlockHash = common.HexToHash(*endParentChainBlockHashFlag)
	startMelStateHash = common.HexToHash(*startMelRootFlag)
	positionInMEL = *positionInMELFlag
	lastBlockHash = common.HexToHash(*lastBlockHashFlag)
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

func GetLastBlockHash() (hash common.Hash) {
	return lastBlockHash
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

func SetLastBlockHash(hash [32]byte) {
	lastBlockHash = hash
}

func GetPositionInMEL() uint64 {
	return positionInMEL
}

func IncreasePositionInMEL() {
	positionInMEL++
}

func SetSendRoot(hash [32]byte) {
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	val, ok := preimages[ty][hash]
	if !ok {
		return []byte{}, errors.New("preimage not found")
	}
	return val, nil
}
