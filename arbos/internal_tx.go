//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
)

// Types of ArbitrumInternalTx, distinguished by the first data byte
const (
	// Contains 8 bytes indicating the big endian L1 block number to set
	arbInternalTxUpdateL1BlockNumber uint8 = 0
)

func InternalTxUpdateL1BlockNumber(l1BlockNumber uint64) []byte {
	data := make([]byte, 9)
	data[0] = arbInternalTxUpdateL1BlockNumber
	binary.BigEndian.PutUint64(data[1:], l1BlockNumber)
	return data
}

func ApplyInternalTxUpdate(data []byte, state *arbosState.ArbosState, blockContext vm.BlockContext) error {
	if len(data) == 0 {
		return errors.New("no internal tx data")
	}
	tipe := data[0]
	data = data[1:]
	if tipe == arbInternalTxUpdateL1BlockNumber {
		if len(data) != 8 {
			return fmt.Errorf("bad data length of UpdateL1BlockNumber: %v", len(data))
		}
		var prevHash common.Hash
		if blockContext.BlockNumber.Sign() > 0 {
			prevHash = blockContext.GetHash(blockContext.BlockNumber.Uint64() - 1)
		}
		return state.Blockhashes().RecordNewL1Block(binary.BigEndian.Uint64(data), prevHash)
	} else {
		return fmt.Errorf("unknown ArbitrumInternalTx type %v", tipe)
	}
}
