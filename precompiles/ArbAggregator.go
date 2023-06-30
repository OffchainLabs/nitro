// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
	"math/big"

	"github.com/offchainlabs/nitro/arbos/l1pricing"
)

// ArbAggregator provides aggregators and their users methods for configuring how they participate in L1 aggregation.
// Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless SetPreferredAggregator()
// is invoked to change it.
type ArbAggregator struct {
	Address addr // 0x6d
}

var ErrNotOwner = errors.New("must be called by chain owner")

// GetPreferredAggregator returns the preferred aggregator address.
// Deprecated: Do not use this method.
func (con ArbAggregator) GetPreferredAggregator(c ctx, evm mech, address addr) (prefAgg addr, isDefault bool, err error) {
	return l1pricing.BatchPosterAddress, true, err
}

// GetDefaultAggregator returns the default aggregator address.
// Deprecated: Do not use this method.
func (con ArbAggregator) GetDefaultAggregator(c ctx, evm mech) (addr, error) {
	return l1pricing.BatchPosterAddress, nil
}

// GetBatchPosters gets the addresses of all current batch posters
func (con ArbAggregator) GetBatchPosters(c ctx, evm mech) ([]addr, error) {
	return c.State.L1PricingState().BatchPosterTable().AllPosters(65536)
}

func (con ArbAggregator) AddBatchPoster(c ctx, evm mech, newBatchPoster addr) error {
	isOwner, err := c.State.ChainOwners().IsMember(c.caller)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotOwner
	}
	batchPosterTable := c.State.L1PricingState().BatchPosterTable()
	isBatchPoster, err := batchPosterTable.ContainsPoster(newBatchPoster)
	if err != nil {
		return err
	}
	if !isBatchPoster {
		_, err = batchPosterTable.AddPoster(newBatchPoster, newBatchPoster)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFeeCollector gets a batch poster's fee collector
func (con ArbAggregator) GetFeeCollector(c ctx, evm mech, batchPoster addr) (addr, error) {
	posterInfo, err := c.State.L1PricingState().BatchPosterTable().OpenPoster(batchPoster, false)
	if err != nil {
		return addr{}, err
	}
	return posterInfo.PayTo()
}

// SetFeeCollector sets a batch poster's fee collector (caller must be the batch poster, its fee collector, or an owner)
func (con ArbAggregator) SetFeeCollector(c ctx, evm mech, batchPoster addr, newFeeCollector addr) error {
	posterInfo, err := c.State.L1PricingState().BatchPosterTable().OpenPoster(batchPoster, false)
	if err != nil {
		return err
	}
	oldFeeCollector, err := posterInfo.PayTo()
	if err != nil {
		return err
	}
	if c.caller != batchPoster && c.caller != oldFeeCollector {
		isOwner, err := c.State.ChainOwners().IsMember(c.caller)
		if err != nil {
			return err
		}
		if !isOwner {
			return errors.New("only a batch poster (or its fee collector / chain owner) may change its fee collector")
		}
	}
	return posterInfo.SetPayTo(newFeeCollector)
}

// GetTxBaseFee gets an aggregator's current fixed fee to submit a tx
func (con ArbAggregator) GetTxBaseFee(c ctx, evm mech, aggregator addr) (huge, error) {
	// This is deprecated and now always returns zero.
	return big.NewInt(0), nil
}

// SetTxBaseFee sets an aggregator's fixed fee (caller must be the aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetTxBaseFee(c ctx, evm mech, aggregator addr, feeInL1Gas huge) error {
	// This is deprecated and is now a no-op.
	return nil
}
