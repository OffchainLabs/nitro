// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
)

func mustState(t *testing.T, s any) *HashedAddressCheckerState {
	t.Helper()
	state, ok := s.(*HashedAddressCheckerState)
	require.Truef(t, ok, "unexpected AddressCheckerState type %T", s)
	return state
}

func TestHashedAddressCheckerSimple(t *testing.T) {
	salt, err := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")
	require.NoError(t, err, "failed to parse salt UUID")

	addrFiltered := common.HexToAddress("0xddfAbCdc4D8FfC6d5beaf154f18B778f892A0740")
	addrFiltered2 := common.HexToAddress("0xdead000000000000000000000000000000000001")
	addrAllowed := common.HexToAddress("0x000000000000000000000000000000000000beef")

	const cacheSize = 100
	store := NewHashStore(cacheSize)

	// These values are test values from the provider, to cross-check the salting/hashing algorithm.
	hash := common.HexToHash("0x8fb74f22f0aed996e7548101ae1cea812ccdf86e7ad8a781eebea00f797ce4a6")
	hash2 := common.HexToHash("0xe4c758332a0fe49872f79ae15d2e1c0d76daeb5a9b33578e7f11d3e2571dad1a")
	store.Store(uuid.New(), salt, []common.Hash{hash, hash2}, "test")

	checker := NewHashedAddressChecker(store, 4, 8192)
	checker.Start(context.Background())

	// Tx 1: filtered address
	state1 := mustState(t, checker.NewTxState())
	state1.TouchAddress(&filter.FilteredAddressRecord{Address: addrFiltered, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
	filtered1, records1 := state1.IsFiltered()
	assert.True(t, filtered1, "expected transaction to be filtered")
	require.Len(t, records1, 1)
	assert.Equal(t, addrFiltered, records1[0].Address)
	assert.Equal(t, filter.ReasonFrom, records1[0].Reason)

	// Tx 2: allowed address
	state2 := mustState(t, checker.NewTxState())
	state2.TouchAddress(&filter.FilteredAddressRecord{Address: addrAllowed, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
	filtered2, records2 := state2.IsFiltered()
	assert.False(t, filtered2, "expected transaction NOT to be filtered")
	assert.Empty(t, records2)

	// Tx 3: mixed addresses
	state3 := mustState(t, checker.NewTxState())
	state3.TouchAddress(&filter.FilteredAddressRecord{Address: addrAllowed, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
	state3.TouchAddress(&filter.FilteredAddressRecord{Address: addrFiltered, FilterReason: filter.FilterReason{Reason: filter.ReasonTo, EventRuleMatch: nil}})
	filtered3, records3 := state3.IsFiltered()
	assert.True(t, filtered3, "expected transaction with mixed addresses to be filtered")
	require.Len(t, records3, 1)
	assert.Equal(t, addrFiltered, records3[0].Address)
	assert.Equal(t, filter.ReasonTo, records3[0].Reason)

	// Tx 4: multiple filtered addresses
	state4 := mustState(t, checker.NewTxState())
	state4.TouchAddress(&filter.FilteredAddressRecord{Address: addrFiltered, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
	state4.TouchAddress(&filter.FilteredAddressRecord{Address: addrAllowed, FilterReason: filter.FilterReason{Reason: filter.ReasonTo, EventRuleMatch: nil}})
	state4.TouchAddress(&filter.FilteredAddressRecord{Address: addrFiltered2, FilterReason: filter.FilterReason{Reason: filter.ReasonContractAddress, EventRuleMatch: nil}})
	filtered4, records4 := state4.IsFiltered()
	assert.True(t, filtered4, "expected transaction with multiple filtered addresses to be filtered")
	require.Len(t, records4, 2)
	recordsByAddr := make(map[common.Address]filter.FilteredAddressRecord)
	for _, r := range records4 {
		recordsByAddr[r.Address] = r
	}
	assert.Equal(t, filter.ReasonFrom, recordsByAddr[addrFiltered].Reason)
	assert.Equal(t, filter.ReasonContractAddress, recordsByAddr[addrFiltered2].Reason)

	// Tx 5: reuse HashStore cache across txs
	state5 := mustState(t, checker.NewTxState())
	state5.TouchAddress(&filter.FilteredAddressRecord{Address: addrFiltered, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
	filtered5, _ := state5.IsFiltered()
	assert.True(t, filtered5, "expected cached filtered address to still be filtered")

	// Tx 6: unbuffered channel (synchronous send) should not panic
	overflowChecker := NewHashedAddressChecker(
		store,
		/* workerCount */ 1,
		/* queueSize */ 0,
	)
	overflowChecker.Start(context.Background())

	// Tx 6: synchronous call
	overflowState := mustState(t, overflowChecker.NewTxState())
	overflowState.TouchAddress(&filter.FilteredAddressRecord{Address: addrFiltered, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})

	filtered6, _ := overflowState.IsFiltered()
	assert.True(
		t,
		filtered6,
		"expected cached filtered address to still be filtered",
	)
}

func TestHashedAddressCheckerHeavy(t *testing.T) {
	salt, err := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")
	require.NoError(t, err, "failed to parse salt UUID")

	const filteredCount = 500
	const cacheSize = 100
	filteredAddrs := make([]common.Address, filteredCount)
	filteredHashes := make([]common.Hash, filteredCount)

	hashPrefix := GetHashInputPrefix(salt)
	for i := range filteredAddrs {
		addr := common.BytesToAddress([]byte{byte(i + 1)})
		filteredAddrs[i] = addr
		filteredHashes[i] = HashWithPrefix(hashPrefix, addr)
	}

	store := NewHashStore(cacheSize)
	store.Store(uuid.New(), salt, filteredHashes, "heavy")

	checker := NewHashedAddressChecker(store, 4, 8192)
	checker.Start(context.Background())

	const txCount = 100
	const touchesPerTx = 100

	results := make(chan bool, txCount)

	var wg sync.WaitGroup
	wg.Add(txCount)

	for tx := range txCount {
		go func(tx int) {
			defer wg.Done()

			state := mustState(t, checker.NewTxState())

			for i := range touchesPerTx {
				if i%10 == 0 {
					state.TouchAddress(&filter.FilteredAddressRecord{Address: filteredAddrs[i%filteredCount], FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
				} else {
					addr := common.BytesToAddress([]byte{byte(200 + i*tx)})
					state.TouchAddress(&filter.FilteredAddressRecord{Address: addr, FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil}})
				}
			}

			filtered, _ := state.IsFiltered()
			results <- filtered
		}(tx)
	}

	wg.Wait()
	close(results)

	filteredTxs := 0
	for r := range results {
		if r {
			filteredTxs++
		}
	}

	assert.Greater(
		t,
		filteredTxs,
		0,
		"expected at least some transactions to be filtered under load",
	)
}
