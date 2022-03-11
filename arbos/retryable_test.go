//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestOpenNonexistentRetryable(t *testing.T) {
	state, _ := arbosState.NewArbosMemoryBackedArbOSState()
	id := common.BigToHash(big.NewInt(978645611142))
	lastTimestamp, err := state.LastTimestampSeen()
	Require(t, err)
	retryable, err := state.RetryableState().OpenRetryable(id, lastTimestamp)
	Require(t, err)
	if retryable != nil {
		Fail(t)
	}
}

func TestOpenExpiredRetryable(t *testing.T) {
	state, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	originalTimestamp, err := state.LastTimestampSeen()
	Require(t, err)
	newTimestamp := originalTimestamp + 42
	state.SetLastTimestampSeen(newTimestamp)

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := originalTimestamp // in the past
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	beneficiary := common.BytesToAddress([]byte{3, 1, 4, 1, 5, 9, 2, 6})
	calldata := []byte{42}
	retryableState := state.RetryableState()

	_, err = retryableState.CreateRetryable(id, timeout, from, &to, callvalue, beneficiary, calldata)
	Require(t, err)

	timestamp, err := state.LastTimestampSeen()
	Require(t, err)
	reread, err := retryableState.OpenRetryable(id, timestamp)
	Require(t, err)
	if reread != nil {
		Fail(t)
	}

	// check that our reap pricing is reflective of the true cost
	burner, _ := state.Burner.(*burn.SystemBurner)
	gasBefore := burner.Burned()
	evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, &params.ChainConfig{}, vm.Config{})
	Require(t, retryableState.TryToReapOneRetryable(timestamp, evm, util.TracingDuringEVM))
	gasBurnedToReap := burner.Burned() - gasBefore
	if gasBurnedToReap != retryables.RetryableReapPrice {
		Fail(t, "reaping has been mispriced", gasBurnedToReap, retryables.RetryableReapPrice)
	}

	// ensure the retryable is gone
	queueSize, err := retryableState.TimeoutQueueSize()
	Require(t, err)
	if queueSize != 0 {
		Fail(t, "failed to reap", queueSize)
	}
}

func TestRetryableCreate(t *testing.T) {
	state, _ := arbosState.NewArbosMemoryBackedArbOSState()
	id := common.BigToHash(big.NewInt(978645611142))
	lastTimestamp, err := state.LastTimestampSeen()
	Require(t, err)

	timeout := lastTimestamp + 10000000
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	beneficiary := common.BytesToAddress([]byte{3, 1, 4, 1, 5, 9, 2, 6})
	calldata := make([]byte, 42)
	for i := range calldata {
		calldata[i] = byte(i + 3)
	}
	rstate := state.RetryableState()
	retryable, err := rstate.CreateRetryable(id, timeout, from, &to, callvalue, beneficiary, calldata)
	Require(t, err)

	reread, err := rstate.OpenRetryable(id, lastTimestamp)
	Require(t, err)
	if reread == nil {
		Fail(t)
	}
	equal, err := reread.Equals(retryable)
	Require(t, err)

	if !equal {
		Fail(t)
	}
}
