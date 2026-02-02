// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"crypto/sha256"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func mustState(t *testing.T, s any) *HashedAddressCheckerState {
	t.Helper()
	state, ok := s.(*HashedAddressCheckerState)
	require.Truef(t, ok, "unexpected AddressCheckerState type %T", s)
	return state
}

func TestHashedAddressCheckerSimple(t *testing.T) {
	salt := []byte("test-salt")

	addrFiltered := common.HexToAddress("0x000000000000000000000000000000000000dead")
	addrAllowed := common.HexToAddress("0x000000000000000000000000000000000000beef")

	const cacheSize = 100
	store := NewHashStore(cacheSize)

	hash := sha256.Sum256(append(salt, addrFiltered.Bytes()...))
	store.Store(salt, []common.Hash{hash}, "test")

	checker := NewDefaultHashedAddressChecker(store)
	checker.Start(context.Background())

	// Tx 1: filtered address
	state1 := mustState(t, checker.NewTxState())
	state1.TouchAddress(addrFiltered)
	assert.True(t, state1.IsFiltered(), "expected transaction to be filtered")

	// Tx 2: allowed address
	state2 := mustState(t, checker.NewTxState())
	state2.TouchAddress(addrAllowed)
	assert.False(t, state2.IsFiltered(), "expected transaction NOT to be filtered")

	// Tx 3: mixed addresses
	state3 := mustState(t, checker.NewTxState())
	state3.TouchAddress(addrAllowed)
	state3.TouchAddress(addrFiltered)
	assert.True(t, state3.IsFiltered(), "expected transaction with mixed addresses to be filtered")

	// Tx 4: reuse HashStore cache across txs
	state4 := mustState(t, checker.NewTxState())
	state4.TouchAddress(addrFiltered)
	assert.True(t, state4.IsFiltered(), "expected cached filtered address to still be filtered")

	// Tx 5: queue overflow should not panic and must be conservative
	overflowChecker := NewHashedAddressChecker(
		store,
		/* workerCount */ 1,
		/* queueSize */ 0,
	)
	overflowChecker.Start(context.Background())

	// Tx 5: synchronous call
	overflowState := mustState(t, overflowChecker.NewTxState())
	overflowState.TouchAddress(addrFiltered)

	assert.True(
		t,
		overflowState.IsFiltered(),
		"expected cached filtered address to still be filtered",
	)
}

func TestHashedAddressCheckerHeavy(t *testing.T) {
	salt := []byte("heavy-salt")

	const filteredCount = 500
	const cacheSize = 100
	filteredAddrs := make([]common.Address, filteredCount)
	filteredHashes := make([]common.Hash, filteredCount)

	for i := range filteredAddrs {
		addr := common.BytesToAddress([]byte{byte(i + 1)})
		filteredAddrs[i] = addr
		filteredHashes[i] = sha256.Sum256(append(salt, addr.Bytes()...))
	}

	store := NewHashStore(cacheSize)
	store.Store(salt, filteredHashes, "heavy")

	checker := NewDefaultHashedAddressChecker(store)
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
					state.TouchAddress(filteredAddrs[i%filteredCount])
				} else {
					addr := common.BytesToAddress([]byte{byte(200 + i*tx)})
					state.TouchAddress(addr)
				}
			}

			results <- state.IsFiltered()
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
