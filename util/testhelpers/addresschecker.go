// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package testhelpers

import (
	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
)

// RecordingCheckerState is a fake state.AddressCheckerState that records every
// touched address with its reason.
type RecordingCheckerState struct {
	touched []filter.FilteredAddressWithReason
}

func (r *RecordingCheckerState) TouchAddress(t *filter.FilteredAddressWithReason) {
	r.touched = append(r.touched, *t)
}

func (r *RecordingCheckerState) IsFiltered() (bool, []filter.FilteredAddressRecord) {
	return false, nil
}

func (r *RecordingCheckerState) CountTouches(addr common.Address, reason filter.FilterReasonType) int {
	count := 0
	for _, t := range r.touched {
		if t.Address == addr && t.Reason == reason {
			count++
		}
	}
	return count
}
