// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"

	arbosutil "github.com/offchainlabs/nitro/arbos/util"
)

// recordingCheckerState is a fake state.AddressCheckerState that records every
// touched address with its reason.
type recordingCheckerState struct {
	touched []filter.FilteredAddressWithReason
}

func (r *recordingCheckerState) TouchAddress(t *filter.FilteredAddressWithReason) {
	r.touched = append(r.touched, *t)
}

func (r *recordingCheckerState) IsFiltered() (bool, []filter.FilteredAddressRecord) {
	return false, nil
}

func (r *recordingCheckerState) hasTouch(addr common.Address, reason filter.FilterReasonType) bool {
	for _, t := range r.touched {
		if t.Address == addr && t.Reason == reason {
			return true
		}
	}
	return false
}

// TestTouchAddressesDealiasedFrom verifies that touchAddresses touches the
// de-aliased sender for every tx type whose sender is aliased by the L1
// bridge, including submit retryables and deposits, which are aliased
// unconditionally by the Inbox but are not covered by DoesTxTypeAlias.
func TestTouchAddressesDealiasedFrom(t *testing.T) {
	l1Sender := common.HexToAddress("0x1111000000000000000000000000000000000000")
	aliasedSender := arbosutil.RemapL1Address(l1Sender)
	dest := common.HexToAddress("0x2222000000000000000000000000000000000000")
	chainID := big.NewInt(412346)

	testCases := []struct {
		name              string
		tx                *types.Transaction
		expectDealiasedTo bool
	}{
		{
			name: "SubmitRetryableTx",
			tx: types.NewTx(&types.ArbitrumSubmitRetryableTx{
				ChainId:          chainID,
				RequestId:        common.Hash{1},
				From:             aliasedSender,
				L1BaseFee:        common.Big0,
				DepositValue:     big.NewInt(1e18),
				GasFeeCap:        big.NewInt(1e9),
				Gas:              100000,
				RetryTo:          &dest,
				RetryValue:       common.Big0,
				Beneficiary:      dest,
				MaxSubmissionFee: big.NewInt(1e16),
				FeeRefundAddr:    dest,
				RetryData:        nil,
			}),
			expectDealiasedTo: true,
		},
		{
			name: "DepositTx",
			tx: types.NewTx(&types.ArbitrumDepositTx{
				ChainId:     chainID,
				L1RequestId: common.Hash{2},
				From:        aliasedSender,
				To:          dest,
				Value:       big.NewInt(1e18),
			}),
			expectDealiasedTo: true,
		},
		{
			name: "UnsignedTx",
			tx: types.NewTx(&types.ArbitrumUnsignedTx{
				ChainId:   chainID,
				From:      aliasedSender,
				Nonce:     0,
				GasFeeCap: big.NewInt(1e9),
				Gas:       100000,
				To:        &dest,
				Value:     common.Big0,
				Data:      nil,
			}),
			expectDealiasedTo: true,
		},
		{
			name: "DynamicFeeTx",
			tx: types.NewTx(&types.DynamicFeeTx{
				ChainID:   chainID,
				Nonce:     0,
				GasFeeCap: big.NewInt(1e9),
				Gas:       100000,
				To:        &dest,
				Value:     common.Big0,
			}),
			expectDealiasedTo: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
			require.NoError(t, err)
			checker := &recordingCheckerState{}
			db.SetAddressCheckerState(checker)

			touchAddresses(db, tc.tx, aliasedSender)

			require.True(t, checker.hasTouch(aliasedSender, filter.ReasonFrom), "sender should be touched")
			require.Equal(t, tc.expectDealiasedTo, checker.hasTouch(l1Sender, filter.ReasonDealiasedFrom),
				"unexpected de-aliased sender touch state")
		})
	}
}
