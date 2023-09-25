// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"encoding/hex"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/merkletree"
	"github.com/offchainlabs/nitro/util/testhelpers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

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
	_ = testhelpers.InitTestLog(t, log.LvlWarn)
	rand.Seed(time.Now().UTC().UnixNano())
	state, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	retryableState := state.RetryableState()

	lifetime := uint64(retryables.RetryableLifetimeSeconds)
	timestampAtCreation := uint64(rand.Int63n(1 << 16))
	timeoutAtCreation := timestampAtCreation + lifetime
	timestampAtRevival := timeoutAtCreation + 2 + uint64(rand.Int63n(1<<16))
	// timeoutAtRevival := timestampAtRevival + lifetime
	currentTime := timeoutAtCreation

	setTime := func(timestamp uint64) uint64 {
		currentTime = timestamp
		// state.SetLastTimestampSeen(currentTime)
		colors.PrintGrey("Time is now ", timestamp)
		return currentTime
	}
	proveReapingDoesNothing := func() {
		t.Helper()
		stateCheck(t, statedb, statedb, false, "reaping had an effect", func() {
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
	stateBeforeEverything := statedb.Copy()
	setTime(timestampAtCreation)

	// TODO(magic) remove ids (already in retries data)
	ids := []common.Hash{}
	retriesData := []retryables.TestRetryableData{}
	for i := 0; i < 8; i++ {
		id := common.BigToHash(big.NewInt(rand.Int63n(1 << 32)))
		from := testhelpers.RandomAddress()
		to := testhelpers.RandomAddress()
		beneficiary := testhelpers.RandomAddress()
		callValue := big.NewInt(rand.Int63n(1 << 32))
		callData := testhelpers.RandomizeSlice(make([]byte, rand.Intn(1<<12)))
		retriesData = append(retriesData, retryables.TestRetryableData{
			Id:          id,
			NumTries:    0,
			From:        from,
			To:          to,
			CallValue:   callValue,
			Beneficiary: beneficiary,
			CallData:    callData,
		})
		timeout := timeoutAtCreation
		_, err := retryableState.CreateRetryable(id, timeout, from, &to, callValue, beneficiary, callData)
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
	var merkleUpdateEvents []merkleAccumulator.MerkleTreeNodeEvent
	var expiredRetryablesLeaves []retryables.ExpiredRetryableLeaf
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
		events, leaf, err := retryableState.TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)
		Require(t, err)
		t.Log("events:", events)
		merkleUpdateEvents = append(merkleUpdateEvents, events...)
		if leaf == nil {
			Fail(t, "reaping retryable returned no expired retryable leaf")
		}
		t.Log("New expired leaf:", leaf)
		expiredRetryablesLeaves = append(expiredRetryablesLeaves, *leaf)
		merkleUpdateEvents = append(merkleUpdateEvents, merkleAccumulator.MerkleTreeNodeEvent{
			Level:     0,
			NumLeaves: leaf.Index,
			Hash:      leaf.Hash,
		})
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
	if !cleared {
		Fail(t, "reaping didn't clear TimeoutQueue")
	}
	expectedState := stateBeforeEverything.Copy()
	expectedArbosState, err := arbosState.OpenSystemArbosState(expectedState, nil, false)
	Require(t, err)
	expectedRetryableState := expectedArbosState.RetryableState()
	for _, retryData := range retriesData {
		_, _, err := expectedRetryableState.Expired.Append(retryData.Hash())
		Require(t, err)
	}
	if expectedState.IntermediateRoot(true) != statedb.IntermediateRoot(true) {
		Fail(t, "unexpected state after reaping")
	}

	setTime(timestampAtRevival)
	// revive the retryables
	for _, retryData := range retriesData {
		size, err := retryableState.Expired.Size()
		Require(t, err)
		rootHash, err := retryableState.Expired.Root()
		Require(t, err)
		t.Log("accumulator size:", size, "rootHash:", rootHash)
		newTimeout, err := retryableState.Revive(expiredRetryableReviveData(t, retryData, merkleUpdateEvents, expiredRetryablesLeaves, currentTime, lifetime))
		Require(t, err, "failed to revive the retryable")
		if newTimeout != currentTime+lifetime {
			Fail(t, "new timeout after revival is wrong", newTimeout, currentTime+lifetime)
		}
		shouldntBeNil, err := retryableState.OpenRetryable(retryData.Id, currentTime)
		Require(t, err)
		if shouldntBeNil == nil {
			Fail(t, err, "failed to open retryable after revival", retryData)
		}
	}
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
	callValue := big.NewInt(0)
	callData := testhelpers.RandomizeSlice(make([]byte, rand.Intn(1<<12)))

	timeout := uint64(rand.Int63n(1 << 16))
	timestamp := 2 * timeout

	expectedState := statedb.Copy()
	expectedArbosState, err := arbosState.OpenSystemArbosState(expectedState, nil, false)
	Require(t, err)
	expectedRetryableState := expectedArbosState.RetryableState()
	_, _, err = expectedRetryableState.Expired.Append(retryables.RetryableHash(id, 0, from, to, callValue, beneficiary, callData))
	Require(t, err)
	stateCheck(t, statedb, expectedState, false, "state has changed", func() {
		_, err := retryableState.CreateRetryable(id, timeout, from, &to, callValue, beneficiary, callData)
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
	callValue := big.NewInt(0)
	beneficiary := common.BytesToAddress([]byte{3, 1, 4, 1, 5, 9, 2, 6})
	callData := make([]byte, 42)
	for i := range callData {
		callData[i] = byte(i + 3)
	}
	rstate := state.RetryableState()
	retryable, err := rstate.CreateRetryable(id, timeout, from, &to, callValue, beneficiary, callData)
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

func stateCheck(t *testing.T, statedb *state.StateDB, expectedStateDb *state.StateDB, change bool, message string, scope func()) {
	t.Helper()
	expectedRoot := expectedStateDb.IntermediateRoot(true)
	dumpBefore := string(expectedStateDb.Dump(&state.DumpConfig{}))
	scope()
	if (expectedRoot != statedb.IntermediateRoot(true)) != change {
		dumpAfter := string(statedb.Dump(&state.DumpConfig{}))
		colors.PrintRed(dumpBefore)
		colors.PrintRed(dumpAfter)
		Fail(t, message)
	}
}

func expiredRetryableReviveData(
	t *testing.T,
	retryData retryables.TestRetryableData,
	events []merkleAccumulator.MerkleTreeNodeEvent,
	leaves []retryables.ExpiredRetryableLeaf,
	now,
	lifetime uint64,
) (
	ticketId common.Hash,
	numTries uint64,
	from common.Address,
	to common.Address,
	callValue *big.Int,
	beneficiary common.Address,
	callData []byte,
	rootHash common.Hash,
	leafIndex uint64,
	proof []common.Hash,
	currentTimestamp uint64,
	timeToAdd uint64,
) {
	leafHash := crypto.Keccak256Hash(retryData.Hash().Bytes())
	var treeSize uint64
	for _, leaf := range leaves {
		if leaf.TicketId == retryData.Id {
			leafIndex = leaf.Index
			if leaf.Hash != leafHash {
				Fail(t, "invalid leaf hash in ExpiredRetryableLeaf, want:", leafHash, "have:", leaf.Hash)
			}
		}
		if leaf.Index+1 > treeSize {
			treeSize = leaf.Index + 1
		}
	}

	balanced := treeSize == arbmath.NextPowerOf2(treeSize)/2
	treeLevels := int(arbmath.Log2ceil(treeSize)) // the # of levels in the tree
	proofLevels := treeLevels - 1                 // the # of levels where a hash is needed (all but root)
	walkLevels := treeLevels                      // the # of levels we need to consider when building walks
	if balanced {
		walkLevels -= 1 // skip the root
	}
	t.Log("Tree has", treeSize, "leaves and", treeLevels, "levels")
	t.Log("Balanced:", balanced)
	// find which nodes we'll want in our proof up to a partial
	query := make(map[merkletree.LevelAndLeaf]struct{}) // the nodes we'll query for
	nodes := make([]merkletree.LevelAndLeaf, 0)         // the nodes needed (might not be found from query)
	which := uint64(1)                                  // which bit to flip & set
	place := uint64(leafIndex)                          // where we are in the tree
	t.Log("start place:", place)
	for level := 0; level < walkLevels; level++ {
		sibling := place ^ which
		position := merkletree.LevelAndLeaf{
			Level: uint64(level),
			Leaf:  sibling,
		}
		query[position] = struct{}{}
		nodes = append(nodes, position)
		place |= which // set the bit so that we approach from the right
		which <<= 1    // advance to the next bit
	}
	// find all the partials
	partials := make(map[merkletree.LevelAndLeaf]common.Hash)
	if !balanced {
		power := uint64(1) << proofLevels
		total := uint64(0)
		for level := proofLevels; level >= 0; level-- {
			if (power & treeSize) > 0 { // the partials map to the binary representation of the tree size
				total += power    // The actual leaf for a given partial is the sum of the powers of 2
				leaf := total - 1 // preceding it. We subtract 1 since we count from 0
				partial := merkletree.LevelAndLeaf{
					Level: uint64(level),
					Leaf:  leaf,
				}
				query[partial] = struct{}{}
				partials[partial] = common.Hash{}
			}
			power >>= 1
		}
	}
	t.Log("Query:", query)
	t.Log("Found", len(partials), "partials:", partials)
	known := make(map[merkletree.LevelAndLeaf]common.Hash) // all values in the tree we know
	partialsByLevel := make(map[uint64]common.Hash)        // maps for each level the partial it may have
	var minPartialPlace *merkletree.LevelAndLeaf           // the lowest-level partial
	// search all events
	for _, event := range events {
		level := event.Level
		leaf := event.NumLeaves
		hash := event.Hash
		t.Log("event:\n\tposition: level", level, "leaf", leaf, "\n\thash:    ", hash)
		place := merkletree.LevelAndLeaf{
			Level: level,
			Leaf:  leaf,
		}
		if _, ok := query[place]; ok {
			t.Log("Found queried place:", place)
			known[place] = hash
		}
		if zero, ok := partials[place]; ok {
			if zero != (common.Hash{}) {
				if zero != hash {
					Fail(t, "Somehow got 2 partials for the same level\n\t1st:", zero, "\n\t2nd:", hash, "place:", place)
				}
				continue
			}
			partials[place] = hash
			partialsByLevel[level] = hash
			if minPartialPlace == nil || level < minPartialPlace.Level {
				minPartialPlace = &place
			}
		}
	}
	for place, hash := range known {
		t.Log("known  ", place.Level, hash, "@", place)
	}
	t.Log(len(known), "values are known\n")
	for place, hash := range partials {
		t.Log("partial", place.Level, hash, "@", place)
	}
	t.Log("resolving frontiers\n", "minPartialPlace:", minPartialPlace)
	if !balanced {
		// This tree isn't balanced, so we'll need to use the partials to recover the missing info.
		// To do this, we'll walk the boundary of what's known, computing hashes along the way
		zero := common.Hash{}
		step := *minPartialPlace
		step.Leaf += 1 << step.Level // we start on the min partial's zero-hash sibling
		known[step] = zero
		t.Log("zeroing:", step)
		for step.Level < uint64(treeLevels) {
			curr, ok := known[step]
			if !ok {
				Fail(t, "We should know the current node's value")
			}
			left := curr
			right := curr
			if _, ok := partialsByLevel[step.Level]; ok {
				// a partial on the frontier can only appear on the left
				// moving leftward for a level l skips 2^l leaves
				step.Leaf -= 1 << step.Level
				partial, ok := known[step]
				if !ok {
					Fail(t, "There should be a partial here")
				}
				left = partial
			} else {
				// getting to the next partial means covering its mirror subtree, so we look right
				// moving rightward for a level l skips 2^l leaves
				step.Leaf += 1 << step.Level
				known[step] = zero
				right = zero
			}
			// move to the parent
			step.Level += 1
			step.Leaf |= 1 << (step.Level - 1)
			known[step] = crypto.Keccak256Hash(left.Bytes(), right.Bytes())
		}
		for place, hash := range known {
			t.Log("known", place, hash)
		}
	}
	t.Log("Complete proof of leaf", leafIndex)
	proof = make([]common.Hash, len(nodes))
	for i, place := range nodes {
		hash, ok := known[place]
		if !ok {
			Fail(t, "We're missing data for the node at position", place)
		}
		proof[i] = hash
		t.Log("node", place, hash)
	}
	rootHash = leafHash
	index := leafIndex
	for _, hashFromProof := range proof {
		if index&1 == 0 {
			rootHash = crypto.Keccak256Hash(rootHash.Bytes(), hashFromProof.Bytes())
		} else {
			rootHash = crypto.Keccak256Hash(hashFromProof.Bytes(), rootHash.Bytes())
		}
		index = index / 2
	}
	if index != 0 {
		Fail(t, "internal test error - failed to compute root hash")
	}
	t.Log("Root hash", hex.EncodeToString(rootHash[:]))
	merkleProof := &merkletree.MerkleProof{
		RootHash:  rootHash,
		LeafHash:  leafHash,
		LeafIndex: leafIndex,
		Proof:     proof,
	}
	if !merkleProof.IsCorrect() {
		Fail(t, "internal test error - incorrect proof")
	}
	ticketId, numTries, from, to, callValue, beneficiary, callData = retryData.Id, retryData.NumTries, retryData.From, retryData.To, retryData.CallValue, retryData.Beneficiary, retryData.CallData
	currentTimestamp = now
	timeToAdd = lifetime
	return
}
