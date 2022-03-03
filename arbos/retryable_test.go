//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/arbos/arbosState"

	"github.com/ethereum/go-ethereum/common"
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
	state, _ := arbosState.NewArbosMemoryBackedArbOSState()
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

	timestamp, err := state.LastTimestampSeen()
	Require(t, err)
	_, err = retryableState.CreateRetryable(timestamp, id, timeout, from, &to, callvalue, beneficiary, calldata)
	Require(t, err)

	timestamp, err = state.LastTimestampSeen()
	Require(t, err)
	reread, err := retryableState.OpenRetryable(id, timestamp)
	Require(t, err)
	if reread != nil {
		Fail(t)
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
	retryable, err := rstate.CreateRetryable(lastTimestamp, id, timeout, from, &to, callvalue, beneficiary, calldata)
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
