// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type RetryableTestData struct {
	id          common.Hash
	numTries    uint64
	from        common.Address
	to          common.Address
	callvalue   *big.Int
	beneficiary common.Address
	calldata    []byte
}

func TestOpenNonexistentRetryable(t *testing.T) {
	state, _ := arbosState.NewArbosMemoryBackedArbOSState()
	id := common.BigToHash(big.NewInt(978645611142))
	retryable, err := state.RetryableState().OpenRetryable(id, 0)
	Require(t, err)
	if retryable != nil {
		Fail(t)
	}
}

func TestRetryableLifecycle(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	state, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	retryableState := state.RetryableState()

	lifetime := uint64(retryables.RetryableLifetimeSeconds)
	timestampAtCreation := uint64(rand.Int63n(1 << 16))
	timeoutAtCreation := timestampAtCreation + lifetime
	currentTime := timeoutAtCreation

	setTime := func(timestamp uint64) uint64 {
		currentTime = timestamp
		// state.SetLastTimestampSeen(currentTime)
		colors.PrintGrey("Time is now ", timestamp)
		return currentTime
	}
	proveReapingDoesNothing := func() {
		t.Helper()
		stateCheck(t, statedb, false, "reaping had an effect", func() {
			evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, &params.ChainConfig{}, vm.Config{})
			_, _, err := retryableState.TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)
			Require(t, err)
		})
	}
	checkQueueSize := func(expected int, message string) {
		t.Helper()
		timeoutQueueSize, err := retryableState.TimeoutQueue.Size()
		Require(t, err)
		if timeoutQueueSize != uint64(expected) {
			Fail(t, currentTime, message, timeoutQueueSize)
		}
	}
	// TODO(magic)
	// stateBeforeEverything := statedb.IntermediateRoot(true)
	setTime(timestampAtCreation)

	ids := []common.Hash{}
	retriesData := []RetryableTestData{}
	for i := 0; i < 8; i++ {
		id := common.BigToHash(big.NewInt(rand.Int63n(1 << 32)))
		from := testhelpers.RandomAddress()
		to := testhelpers.RandomAddress()
		beneficiary := testhelpers.RandomAddress()
		callvalue := big.NewInt(rand.Int63n(1 << 32))
		calldata := testhelpers.RandomizeSlice(make([]byte, rand.Intn(1<<12)))
		retriesData = append(retriesData, RetryableTestData{id, 0, from, to, callvalue, beneficiary, calldata})
		timeout := timeoutAtCreation
		_, err := retryableState.CreateRetryable(id, timeout, from, &to, callvalue, beneficiary, calldata)
		Require(t, err)
		ids = append(ids, id)
	}
	proveReapingDoesNothing()

	// Advance half way to expiration and extend each retryable's lifetime by one period
	setTime((timestampAtCreation + timeoutAtCreation) / 2)
	for _, id := range ids {
		window := currentTime + lifetime
		newTimeout, err := retryableState.Keepalive(id, currentTime, window, lifetime)
		Require(t, err, "failed to extend the retryable's lifetime")
		proveReapingDoesNothing()
		if newTimeout != timeoutAtCreation+lifetime {
			Fail(t, "new timeout is wrong", newTimeout, timeoutAtCreation+lifetime)
		}

		// prove we need to wait before keepalive can succeed again
		_, err = retryableState.Keepalive(id, currentTime, window, lifetime)
		if err == nil {
			Fail(t, "keepalive should have failed")
		}
	}
	checkQueueSize(2*len(ids), "Queue should have twice as many entries as there are retryables")

	// Advance passed the original timeout and reap half the entries in the queue
	setTime(timeoutAtCreation + 1)
	burner, _ := state.Burner.(*burn.SystemBurner)
	for range ids {
		// check that our reap pricing is reflective of the true cost
		gasBefore := burner.Burned()
		evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, &params.ChainConfig{}, vm.Config{})
		_, _, err := retryableState.TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)
		Require(t, err)
		gasBurnedToReap := burner.Burned() - gasBefore
		if gasBurnedToReap != retryables.RetryableReapPrice {
			Fail(t, "reaping has been mispriced", gasBurnedToReap, retryables.RetryableReapPrice)
		}
	}
	checkQueueSize(len(ids), "Queue should have only one copy of each retryable")
	proveReapingDoesNothing()

	// Advanced passed the extended timeout and reap everything
	setTime(timeoutAtCreation + lifetime + 1)
	for _, id := range ids {
		// The retryable will be reaped, so opening it should fail
		shouldBeNil, err := retryableState.OpenRetryable(id, currentTime)
		Require(t, err)
		if shouldBeNil != nil {
			timeout, _ := shouldBeNil.CalculateTimeout()
			Fail(t, err, "read retryable after expiration", timeout, currentTime)
		}

		gasBefore := burner.Burned()
		evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, &params.ChainConfig{}, vm.Config{})
		_, _, err = retryableState.TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)
		Require(t, err)
		gasBurnedToReapAndDelete := burner.Burned() - gasBefore
		if gasBurnedToReapAndDelete <= retryables.RetryableReapPrice {
			Fail(t, "deletion was cheap", gasBurnedToReapAndDelete, retryables.RetryableReapPrice)
		}

		// The retryable has been deleted, so opening it should fail
		shouldBeNil, err = retryableState.OpenRetryable(id, currentTime)
		Require(t, err)
		if shouldBeNil != nil {
			timeout, _ := shouldBeNil.CalculateTimeout()
			Fail(t, err, "read retryable after deletion", timeout, currentTime)
		}
	}
	checkQueueSize(0, "Queue should be empty")
	proveReapingDoesNothing()

	cleared, err := retryableState.TimeoutQueue.Shift()
	Require(t, err)
	if !cleared /*|| stateBeforeEverything != statedb.IntermediateRoot(true)*/ {
		Fail(t, "reaping didn't reset the state", cleared)
	}

	// revive the retryables
}

func TestRetryableCleanup(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	state, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	retryableState := state.RetryableState()

	id := common.BigToHash(big.NewInt(rand.Int63n(1 << 32)))
	from := testhelpers.RandomAddress()
	to := testhelpers.RandomAddress()
	beneficiary := testhelpers.RandomAddress()

	// could be non-zero because we haven't actually minted funds like going through the submit process does
	callvalue := big.NewInt(0)
	calldata := testhelpers.RandomizeSlice(make([]byte, rand.Intn(1<<12)))

	timeout := uint64(rand.Int63n(1 << 16))
	timestamp := 2 * timeout

	// TODO(magic) check if the only state change is update of Expired accumulator
	stateCheck(t, statedb /*false*/, true, "state didn't change", func() {
		_, err := retryableState.CreateRetryable(id, timeout, from, &to, callvalue, beneficiary, calldata)
		Require(t, err)
		evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, &params.ChainConfig{}, vm.Config{})
		_, _, err = retryableState.TryToReapOneRetryable(timestamp, evm, util.TracingDuringEVM)
		Require(t, err)
		cleared, err := retryableState.TimeoutQueue.Shift()
		Require(t, err)
		if !cleared {
			Fail(t, "failed to reset the queue")
		}
	})
}

func TestRetryableCreate(t *testing.T) {
	state, _ := arbosState.NewArbosMemoryBackedArbOSState()
	id := common.BigToHash(big.NewInt(978645611142))
	lastTimestamp := uint64(0)

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

func stateCheck(t *testing.T, statedb *state.StateDB, change bool, message string, scope func()) {
	t.Helper()
	stateBefore := statedb.IntermediateRoot(true)
	dumpBefore := string(statedb.Dump(&state.DumpConfig{}))
	scope()
	if (stateBefore != statedb.IntermediateRoot(true)) != change {
		dumpAfter := string(statedb.Dump(&state.DumpConfig{}))
		colors.PrintRed(dumpBefore)
		colors.PrintRed(dumpAfter)
		Fail(t, message)
	}
}
