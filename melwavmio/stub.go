// Copyright 2026-2027, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm
// +build !wasm

package melwavmio

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
)

var (
	preimages                = make(map[arbutil.PreimageType]map[common.Hash][]byte)
	relevantTxIndicesByBlock = make(map[common.Hash][]uint)
	startMelStateHash        = common.Hash{}
	melMsgHash               = common.Hash{}
	endMelStateHash          = common.Hash{} // This is set by the stubbed SetEndMELStateHash function
	endParentChainBlockHash  = common.Hash{} // This is set by the stubbed GetEndParentChainBlockHash function
	positionInMEL            = uint64(0)
)

func StubInit() {
	endParentChainBlockHashFlag := flag.String("end-parent-chain-block-hash", "0000000000000000000000000000000000000000000000000000000000000000", "endParentChainBlockHash")
	startMelRootFlag := flag.String("start-mel-state-hash", "0000000000000000000000000000000000000000000000000000000000000000", "startMelHash")
	preimagesPath := flag.String("preimages", "", "file to load preimages from")
	positionInMELFlag := flag.Uint64("position-in-mel", 1, "positionInMEL")
	relevantIndicesPath := flag.String("relevant-tx-indices", "", "file to load relevant tx indices from")
	flag.Parse()
	endParentChainBlockHash = common.HexToHash(*endParentChainBlockHashFlag)
	startMelStateHash = common.HexToHash(*startMelRootFlag)
	positionInMEL = *positionInMELFlag
	fileBytes, err := os.ReadFile(*preimagesPath)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(fileBytes, &preimages); err != nil {
		panic(err)
	}
	fileBytes, err = os.ReadFile(*relevantIndicesPath)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(fileBytes, &relevantTxIndicesByBlock); err != nil {
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

func GetRelevantTxIndices(parentChainBlockHash common.Hash) ([]byte, error) {
	txIndices, ok := relevantTxIndicesByBlock[parentChainBlockHash]
	if !ok {
		return nil, errors.New("no relevant tx indices for block hash")
	}
	return rlp.EncodeToBytes(txIndices)
}

func SetEndMELRoot(hash common.Hash) {
	endMelStateHash = hash
}

func GetPositionInMEL() uint64 {
	return positionInMEL
}

func IncreasePositionInMEL() {
	positionInMEL++
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	val, ok := preimages[ty][hash]
	if !ok {
		return []byte{}, errors.New("preimage not found")
	}
	if ty == arbutil.Keccak256PreimageType {
		if hash != crypto.Keccak256Hash(val) {
			return []byte{}, fmt.Errorf("preimage did not rehash to expected hash: %v", hash)
		}
	}
	return val, nil
}
