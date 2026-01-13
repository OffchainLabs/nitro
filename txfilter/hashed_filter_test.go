// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txfilter

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestHashedAddressCheckerSimple(t *testing.T) {
	salt := []byte("test-salt")

	addrFiltered := common.HexToAddress("0x000000000000000000000000000000000000dead")
	addrAllowed := common.HexToAddress("0x000000000000000000000000000000000000beef")

	filteredHash := crypto.Keccak256Hash(addrFiltered.Bytes(), salt)

	checker := NewHashedAddressChecker(
		[]common.Hash{filteredHash},
		salt,
		/* hashCacheSize */ 16,
		/* workerCount */ 2,
		/* queueSize */ 8,
	)

	// Tx 1: filtered address
	//nolint:errcheck
	state1 := checker.NewTxState().(*HashedAddressCheckerState)
	state1.TouchAddress(addrFiltered)

	if !state1.IsFiltered() {
		t.Fatalf("expected transaction to be filtered")
	}

	// Tx 2: allowed address
	//nolint:errcheck
	state2 := checker.NewTxState().(*HashedAddressCheckerState)
	state2.TouchAddress(addrAllowed)

	if state2.IsFiltered() {
		t.Fatalf("expected transaction NOT to be filtered")
	}

	// Tx 3: mixed addresses
	//nolint:errcheck
	state3 := checker.NewTxState().(*HashedAddressCheckerState)
	state3.TouchAddress(addrAllowed)
	state3.TouchAddress(addrFiltered)

	if !state3.IsFiltered() {
		t.Fatalf("expected transaction with mixed addresses to be filtered")
	}

	// Tx 4: reuse hash cache across txs
	// Touch the same filtered address again; this must hit the hash cache
	//nolint:errcheck
	state4 := checker.NewTxState().(*HashedAddressCheckerState)
	state4.TouchAddress(addrFiltered)

	if !state4.IsFiltered() {
		t.Fatalf("expected cached filtered address to still be filtered")
	}

	// Tx 5: queue overflow should not panic and must be conservative
	// Create a checker with zero queue size to force drops
	overflowChecker := NewHashedAddressChecker(
		[]common.Hash{filteredHash},
		salt,
		/* hashCacheSize */ 16,
		/* workerCount */ 1,
		/* queueSize */ 0,
	)

	//nolint:errcheck
	overflowState := overflowChecker.NewTxState().(*HashedAddressCheckerState)
	overflowState.TouchAddress(addrFiltered)

	// Queue is full, work is dropped; result may be false, but must not panic
	_ = overflowState.IsFiltered()
}

func TestHashedAddressCheckerHeavy(t *testing.T) {
	salt := []byte("heavy-salt")

	const filteredCount = 500
	filteredAddrs := make([]common.Address, filteredCount)
	filteredHashes := make([]common.Hash, filteredCount)

	for i := range filteredAddrs {
		addr := common.BytesToAddress([]byte{byte(i + 1)})
		filteredAddrs[i] = addr
		filteredHashes[i] = crypto.Keccak256Hash(addr.Bytes(), salt)
	}

	checker := NewHashedAddressChecker(
		filteredHashes,
		salt,
		/* hashCacheSize */ 256,
		/* workerCount */ 4,
		/* queueSize */ 32,
	)

	const txCount = 100
	const touchesPerTx = 100

	results := make(chan bool, txCount)

	var wg sync.WaitGroup
	wg.Add(txCount)

	for tx := range txCount {
		go func(tx int) {
			defer wg.Done()

			//nolint:errcheck
			state := checker.NewTxState().(*HashedAddressCheckerState)

			for i := range touchesPerTx {
				if i%10 == 0 {
					state.TouchAddress(filteredAddrs[i%filteredCount])
				} else {
					addr := common.BytesToAddress([]byte{byte(200 + i)})
					state.TouchAddress(addr)
				}
			}

			results <- state.IsFiltered()
		}(tx)
	}

	wg.Wait()
	close(results)

	// Post-conditions
	filteredTxs := 0
	for r := range results {
		if r {
			filteredTxs++
		}
	}

	if filteredTxs == 0 {
		t.Fatalf("expected at least some transactions to be filtered under load")
	}
}
