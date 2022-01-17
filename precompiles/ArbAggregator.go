//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

// Provides aggregators and their users methods for configuring how they participate in L1 aggregation.
// Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless SetPreferredAggregator()
// is invoked to change it.
type ArbAggregator struct {
	Address addr // 0x6d
}

// Gets an account's preferred aggregator
func (con ArbAggregator) GetPreferredAggregator(c ctx, evm mech, account addr) (addr, bool, error) {
	return c.state.L1PricingState().PreferredAggregator(account)
}

// Sets the caller's preferred aggregator to that provided
func (con ArbAggregator) SetPreferredAggregator(c ctx, evm mech, prefAgg addr) error {
	return c.state.L1PricingState().SetPreferredAggregator(c.caller, prefAgg)
}

// Gets the rollup's default aggregator
func (con ArbAggregator) GetDefaultAggregator(c ctx, evm mech) (addr, error) {
	return c.state.L1PricingState().DefaultAggregator()
}

// Sets the rollup's default aggregator (caller must be the current default aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetDefaultAggregator(c ctx, evm mech, newDefault addr) error {
	l1State := c.state.L1PricingState()
	defaultAgg, err := l1State.DefaultAggregator()
	if err != nil {
		return err
	}
	collector, err := l1State.AggregatorFeeCollector(defaultAgg)
	if err != nil {
		return err
	}
	if (c.caller != defaultAgg) || (c.caller != collector) {
		member, err := c.state.ChainOwners().IsMember(c.caller)
		if err != nil {
			return err
		}
		if !member {
			return errors.New("Only chain owners and the current default aggregator may use this method")
		}
	}
	return l1State.SetDefaultAggregator(newDefault)
}

// Gets an aggregator's fee collector
func (con ArbAggregator) GetFeeCollector(c ctx, evm mech, aggregator addr) (addr, error) {
	return c.state.L1PricingState().AggregatorFeeCollector(aggregator)
}

// Sets an aggregator's fee collector (caller must be the aggregator)
func (con ArbAggregator) SetFeeCollector(c ctx, evm mech, aggregator addr, newFeeCollector addr) error {
	l1State := c.state.L1PricingState()
	collector, err := l1State.AggregatorFeeCollector(aggregator)
	if err != nil {
		return err
	}
	if (c.caller != aggregator) && (c.caller != collector) {
		// only the aggregator and its current fee collector can change the aggregator's fee collector
		return errors.New("non-authorized c.caller in ArbAggregator.SetFeeCollector")
	}
	return l1State.SetAggregatorFeeCollector(aggregator, newFeeCollector)
}

// Gets an aggregator's current fixed cost charge to submit a tx
func (con ArbAggregator) GetTxBaseFee(c ctx, evm mech, aggregator addr) (huge, error) {
	return c.state.L1PricingState().FixedChargeForAggregatorL1Gas(aggregator)
}

// Sets an aggregator's fixed cost (caller must be the aggregator or a chain owner)
func (con ArbAggregator) SetTxBaseFee(c ctx, evm mech, aggregator addr, feeInL1Gas huge) error {
	if c.caller != aggregator {
		member, err := c.state.ChainOwners().IsMember(c.caller)
		if err != nil {
			return err
		}
		if !member {
			return errors.New("Only an aggregator (or a chain owner) may change its fixed fee")
		}
	}
	return c.state.L1PricingState().SetFixedChargeForAggregatorL1Gas(aggregator, feeInL1Gas)
}
