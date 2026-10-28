// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

var testPendingSender = common.HexToAddress("0x1234")

func makePendingTx(nonce uint64, tipCap, feeCap int64) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: big.NewInt(tipCap),
		GasFeeCap: big.NewInt(feeCap),
		Gas:       21000,
	})
}

func mustAdmit(t *testing.T, set *pendingTxs, sender common.Address, tx *types.Transaction, strategy orderingStrategy) func() {
	t.Helper()
	release, err := set.admit(sender, tx, strategy)
	if err != nil {
		t.Fatalf("admit(%s) = %v, want nil", tx.Hash(), err)
	}
	return release
}

func TestPendingTxsRejectsSameHash(t *testing.T) {
	set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
	tx := makePendingTx(0, 1, 100)
	mustAdmit(t, set, testPendingSender, tx, orderingStrategyFIFO)
	for _, strategy := range []orderingStrategy{orderingStrategyUninitialized, orderingStrategyFIFO, orderingStrategyPGA} {
		if _, err := set.admit(testPendingSender, tx, strategy); !errors.Is(err, txpool.ErrAlreadyKnown) {
			t.Errorf("admit(same hash, %s) = %v, want ErrAlreadyKnown", strategy, err)
		}
	}
	if set.len() != 1 {
		t.Errorf("len() = %d, want 1", set.len())
	}
}

func TestPendingTxsRejectsSameNonceOutsidePGA(t *testing.T) {
	for _, strategy := range []orderingStrategy{orderingStrategyUninitialized, orderingStrategyFIFO} {
		set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
		mustAdmit(t, set, testPendingSender, makePendingTx(0, 1, 100), strategy)
		if _, err := set.admit(testPendingSender, makePendingTx(0, 1000, 100000), strategy); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
			t.Errorf("admit(second, %s) = %v, want ErrReplaceUnderpriced", strategy, err)
		}
	}
}

func TestPendingTxsRejectsUnderpricedReplacement(t *testing.T) {
	set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
	mustAdmit(t, set, testPendingSender, makePendingTx(0, 10, 100), orderingStrategyPGA)
	if _, err := set.admit(testPendingSender, makePendingTx(0, 10, 109), orderingStrategyPGA); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Errorf("admit(second) = %v, want ErrReplaceUnderpriced", err)
	}
}

func TestPendingTxsReplacesWithFeeBump(t *testing.T) {
	set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
	releaseFirst := mustAdmit(t, set, testPendingSender, makePendingTx(0, 10, 100), orderingStrategyPGA)
	second := makePendingTx(0, 11, 110)
	releaseSecond := mustAdmit(t, set, testPendingSender, second, orderingStrategyPGA)
	if set.len() != 1 {
		t.Fatalf("len() after replacement = %d, want 1", set.len())
	}

	releaseFirst()
	if set.len() != 1 {
		t.Fatalf("len() after releasing the replaced tx = %d, want 1", set.len())
	}
	if _, err := set.admit(testPendingSender, second, orderingStrategyPGA); !errors.Is(err, txpool.ErrAlreadyKnown) {
		t.Errorf("admit(second again) = %v, want ErrAlreadyKnown", err)
	}
	releaseSecond()
	if set.len() != 0 {
		t.Errorf("len() after releasing the replacement = %d, want 0", set.len())
	}
}

func TestPendingTxsReplacementReleasedBeforeReplaced(t *testing.T) {
	set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
	first := makePendingTx(0, 10, 100)
	releaseFirst := mustAdmit(t, set, testPendingSender, first, orderingStrategyPGA)
	second := makePendingTx(0, 11, 110)
	releaseSecond := mustAdmit(t, set, testPendingSender, second, orderingStrategyPGA)

	releaseSecond()
	if set.len() != 1 {
		t.Fatalf("len() after releasing the replacement = %d, want 1", set.len())
	}
	if _, err := set.admit(testPendingSender, first, orderingStrategyPGA); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Errorf("admit(first again) = %v, want ErrReplaceUnderpriced", err)
	}
	if _, err := set.admit(testPendingSender, makePendingTx(0, 10, 100), orderingStrategyFIFO); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Errorf("admit(third, FIFO) = %v, want ErrReplaceUnderpriced", err)
	}
	if _, err := set.admit(testPendingSender, second, orderingStrategyPGA); !errors.Is(err, txpool.ErrAlreadyKnown) {
		t.Errorf("admit(second again) = %v, want ErrAlreadyKnown", err)
	}
	releaseThird := mustAdmit(t, set, testPendingSender, makePendingTx(0, 13, 121), orderingStrategyPGA)

	releaseFirst()
	if set.len() != 1 {
		t.Fatalf("len() after releasing the replaced tx = %d, want 1", set.len())
	}
	releaseThird()
	if set.len() != 0 {
		t.Errorf("len() after releasing every holder = %d, want 0", set.len())
	}
}

func TestPendingTxsReleaseFreesNonce(t *testing.T) {
	set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
	release := mustAdmit(t, set, testPendingSender, makePendingTx(0, 1, 100), orderingStrategyFIFO)
	release()
	mustAdmit(t, set, testPendingSender, makePendingTx(0, 1, 100), orderingStrategyFIFO)
}

func TestPendingTxsKeysBySenderAndNonce(t *testing.T) {
	set := newPendingTxs(DefaultPGAConfig.MinPriorityFeePercentBumpToReplace)
	tx := makePendingTx(0, 1, 100)
	mustAdmit(t, set, testPendingSender, tx, orderingStrategyFIFO)
	mustAdmit(t, set, testPendingSender, makePendingTx(1, 1, 100), orderingStrategyFIFO)
	mustAdmit(t, set, common.HexToAddress("0x5678"), tx, orderingStrategyFIFO)
	if set.len() != 3 {
		t.Errorf("len() = %d, want 3", set.len())
	}
}

func TestIsFeeBumped(t *testing.T) {
	type fees struct{ tip, feeCap int64 }
	tests := []struct {
		name        string
		old         fees
		replacement fees
		want        bool
	}{
		{"identical", fees{10, 100}, fees{10, 100}, false},
		{"only tip bumped", fees{10, 100}, fees{11, 100}, false},
		{"only fee cap bumped", fees{10, 100}, fees{10, 110}, false},
		{"both bumped below threshold", fees{10, 100}, fees{11, 109}, false},
		{"tip below threshold", fees{100, 100}, fees{105, 200}, false},
		{"fee cap below threshold", fees{100, 100}, fees{200, 105}, false},
		{"both bumped at threshold", fees{10, 100}, fees{11, 110}, true},
		{"both bumped above threshold", fees{10, 100}, fees{20, 200}, true},
		{"lower fees", fees{10, 100}, fees{5, 50}, false},
		{"zero tip bumped to one wei", fees{0, 100}, fees{1, 110}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := makePendingTx(0, tt.old.tip, tt.old.feeCap)
			replacement := makePendingTx(0, tt.replacement.tip, tt.replacement.feeCap)
			if got := isFeeBumped(old, replacement, 10); got != tt.want {
				t.Errorf("isFeeBumped() = %v, want %v", got, tt.want)
			}
		})
	}
}
