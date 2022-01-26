//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/arbstate/arbos/arbosState"
)

type ArbAggregator struct {
	Address addr
}

func (con ArbAggregator) GetPreferredAggregator(c ctx, evm mech, address addr) (addr, bool, error) {
	l1p := c.state.L1PricingState()
	prefAgg, err := l1p.GetAnyPreferredAggregator(address)
	if err != nil {
		return common.Address{}, false, err
	}
	if prefAgg != nil {
		return *prefAgg, false, nil
	}
	prefAgg, err = l1p.GetAnyDefaultAggregator()
	if err != nil {
		return common.Address{}, false, err
	}
	if prefAgg == nil {
		return addr{}, true, err
	}
	return *prefAgg, true, nil
}

func (con ArbAggregator) SetPreferredAggregator(c ctx, evm mech, prefAgg addr) error {
	l1p := c.state.L1PricingState()
	err := l1p.RemoveAllPreferredAggregators(c.caller)
	if err != nil {
		return err
	}
	if prefAgg == (common.Address{}) {
		return nil
	}
	return l1p.AddPreferredAggregator(c.caller, prefAgg)
}

func (con ArbAggregator) AddPreferredAggregator(c ctx, evm mech, aggToAdd addr) error {
	return c.state.L1PricingState().AddPreferredAggregator(c.caller, aggToAdd)
}

func (con ArbAggregator) RemovePreferredAggregator(c ctx, evm mech, aggToRemove addr) error {
	return c.state.L1PricingState().RemovePreferredAggregator(c.caller, aggToRemove)
}

const MaxGetAggregatorSize = 256

func (con ArbAggregator) GetPreferredAggregators(c ctx, evm mech, address addr) ([]common.Address, error) {
	prefAggs, err := c.state.L1PricingState().GetPreferredAggregators(address, false)
	if err != nil {
		return nil, err
	}
	if len(prefAggs) > MaxGetAggregatorSize {
		prefAggs = prefAggs[:MaxGetAggregatorSize]
	}
	return prefAggs, nil
}

func (con ArbAggregator) IsPreferredAggregator(c ctx, evm mech, address addr, agg addr) (bool, error) {
	return c.state.L1PricingState().IsPreferredAggregator(address, agg, false)
}

func (con ArbAggregator) GetDefaultAggregator(c ctx, evm mech) (addr, error) {
	addr, err := c.state.L1PricingState().GetAnyDefaultAggregator()
	if err != nil || addr == nil {
		return common.Address{}, err
	}
	return *addr, nil
}

func (con ArbAggregator) SetDefaultAggregator(c ctx, evm mech, newDefault addr) error {
	l1p := c.state.L1PricingState()
	err := l1p.RemoveAllDefaultAggregators()
	if err != nil {
		return err
	}
	if newDefault == (common.Address{}) {
		return nil
	}
	return l1p.AddDefaultAggregator(newDefault)
}

func (con ArbAggregator) GetCompressionRatio(c ctx, evm mech, aggregator addr) (uint64, error) {
	return c.state.L1PricingState().AggregatorCompressionRatio(aggregator)
}

func (con ArbAggregator) SetCompressionRatio(c ctx, evm mech, aggregator addr, newRatio uint64) error {
	allowed, err := accountIsAggregatorOrCollectorOrOwner(c.caller, aggregator, c.state)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only an aggregator (or its fee collector / chain owner) may change its compression ratio")
	}
	return c.state.L1PricingState().SetAggregatorCompressionRatio(aggregator, newRatio)
}

func (con ArbAggregator) GetFeeCollector(c ctx, evm mech, aggregator addr) (addr, error) {
	return c.state.L1PricingState().AggregatorFeeCollector(aggregator)
}

func (con ArbAggregator) SetFeeCollector(c ctx, evm mech, aggregator addr, newFeeCollector addr) error {
	allowed, err := accountIsAggregatorOrCollectorOrOwner(c.caller, aggregator, c.state)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only an aggregator (or its fee collector / chain owner) may change its fee collector")
	}
	return c.state.L1PricingState().SetAggregatorFeeCollector(aggregator, newFeeCollector)
}

func (con ArbAggregator) GetTxBaseFee(c ctx, evm mech, aggregator addr) (huge, error) {
	return c.state.L1PricingState().FixedChargeForAggregatorL1Gas(aggregator)
}

func (con ArbAggregator) SetTxBaseFee(c ctx, evm mech, aggregator addr, feeInL1Gas huge) error {
	allowed, err := accountIsAggregatorOrCollectorOrOwner(c.caller, aggregator, c.state)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only an aggregator (or its fee collector / chain owner) may change its fee collector")
	}
	return c.state.L1PricingState().SetFixedChargeForAggregatorL1Gas(aggregator, feeInL1Gas)
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
