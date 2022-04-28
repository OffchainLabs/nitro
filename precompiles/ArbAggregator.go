// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Provides aggregators and their users methods for configuring how they participate in L1 aggregation.
// Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless SetPreferredAggregator()
// is invoked to change it.
type ArbAggregator struct {
	Address addr // 0x6d
}

// Gets an account's preferred aggregator
func (con ArbAggregator) GetPreferredAggregator(c ctx, evm mech, address addr) (prefAgg addr, isDefault bool, err error) {
	l1p := c.State.L1PricingState()
	maybePrefAgg, err := l1p.UserSpecifiedAggregator(address)
	if err != nil {
		return common.Address{}, false, err
	}
	if maybePrefAgg != nil {
		return *maybePrefAgg, false, nil
	}
	maybeReimbursableAgg, err := l1p.ReimbursableAggregatorForSender(address)
	if err != nil || maybeReimbursableAgg == nil {
		return common.Address{}, false, err
	}
	return *maybeReimbursableAgg, true, nil
}

// Sets the caller's preferred aggregator to that provided
func (con ArbAggregator) SetPreferredAggregator(c ctx, evm mech, prefAgg addr) error {
	var maybePrefAgg *common.Address
	if prefAgg != (common.Address{}) {
		maybePrefAgg = &prefAgg
	}
	return c.State.L1PricingState().SetUserSpecifiedAggregator(c.caller, maybePrefAgg)
}

// Gets the chain's default aggregator
func (con ArbAggregator) GetDefaultAggregator(c ctx, evm mech) (addr, error) {
	return c.State.L1PricingState().DefaultAggregator()
}

// Sets the chain's default aggregator (caller must be the current default aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetDefaultAggregator(c ctx, evm mech, newDefault addr) error {
	l1State := c.State.L1PricingState()
	defaultAgg, err := l1State.DefaultAggregator()
	if err != nil {
		return err
	}
	allowed, err := accountIsAggregatorOrCollectorOrOwner(c.caller, defaultAgg, c.State)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only the current default (or its fee collector / chain owner) may change the default")
	}
	return l1State.SetDefaultAggregator(newDefault)
}

// Get the aggregator's compression ratio, measured in basis points
func (con ArbAggregator) GetCompressionRatio(c ctx, evm mech, aggregator addr) (uint64, error) {
	ratio, err := c.State.L1PricingState().AggregatorCompressionRatio(aggregator)
	return uint64(ratio), err
}

// Set the aggregator's compression ratio, measured in basis points
func (con ArbAggregator) SetCompressionRatio(c ctx, evm mech, aggregator addr, newRatio uint64) error {
	allowed, err := accountIsAggregatorOrCollectorOrOwner(c.caller, aggregator, c.State)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only an aggregator (or its fee collector / chain owner) may change its compression ratio")
	}
	return c.State.L1PricingState().SetAggregatorCompressionRatio(aggregator, arbmath.Bips(newRatio))
}

// Gets an aggregator's fee collector
func (con ArbAggregator) GetFeeCollector(c ctx, evm mech, aggregator addr) (addr, error) {
	return c.State.L1PricingState().AggregatorFeeCollector(aggregator)
}

// Sets an aggregator's fee collector (caller must be the aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetFeeCollector(c ctx, evm mech, aggregator addr, newFeeCollector addr) error {
	allowed, err := accountIsAggregatorOrCollectorOrOwner(c.caller, aggregator, c.State)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only an aggregator (or its fee collector / chain owner) may change its fee collector")
	}
	return c.State.L1PricingState().SetAggregatorFeeCollector(aggregator, newFeeCollector)
}

// Gets an aggregator's current fixed fee to submit a tx
func (con ArbAggregator) GetTxBaseFee(c ctx, evm mech, aggregator addr) (huge, error) {
	// This is deprecated and now always returns zero.
	return big.NewInt(0), nil
}

// Sets an aggregator's fixed fee (caller must be the aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetTxBaseFee(c ctx, evm mech, aggregator addr, feeInL1Gas huge) error {
	// This is deprecated and is now a no-op.
	return nil
}

func accountIsAggregatorOrCollectorOrOwner(account, aggregator addr, state *arbosState.ArbosState) (bool, error) {
	if account == aggregator {
		return true, nil
	}
	l1State := state.L1PricingState()
	collector, err := l1State.AggregatorFeeCollector(aggregator)
	if account == collector || err != nil {
		return true, err
	}
	return state.ChainOwners().IsMember(account)
}
