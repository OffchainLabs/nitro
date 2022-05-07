// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"

	"github.com/offchainlabs/nitro/arbos/arbosState"
)

// Provides aggregators and their users methods for configuring how they participate in L1 aggregation.
// Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless SetPreferredAggregator()
// is invoked to change it.
type ArbAggregator struct {
	Address addr // 0x6d
}

// Gets an account's preferred aggregator
func (con ArbAggregator) GetPreferredAggregator(c ctx, evm mech, address addr) (prefAgg addr, isDefault bool, err error) {
	sequencer, err := c.State.L1PricingState().Sequencer()
	return sequencer, true, err
}

// Gets the chain's default aggregator
func (con ArbAggregator) GetDefaultAggregator(c ctx, evm mech) (addr, error) {
	return c.State.L1PricingState().Sequencer()
}

// Sets the chain's default aggregator (caller must be the current default aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetDefaultAggregator(c ctx, evm mech, newDefault addr) error {
	l1State := c.State.L1PricingState()
	allowed, err := accountIsSequencerOrCollectorOrOwner(c.caller, c.State)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only the current default (or its fee collector / chain owner) may change the default")
	}
	return l1State.SetSequencer(newDefault)
}

// Gets an aggregator's fee collector
func (con ArbAggregator) GetFeeCollector(c ctx, evm mech, aggregator addr) (addr, error) {
	l1p := c.State.L1PricingState()
	sequencer, err := l1p.Sequencer()
	if err != nil {
		return common.Address{}, err
	}
	if aggregator != sequencer {
		return common.Address{}, nil
	}
	return c.State.L1PricingState().PaySequencerFeesTo()
}

// Sets an aggregator's fee collector (caller must be the aggregator, its fee collector, or an owner)
func (con ArbAggregator) SetFeeCollector(c ctx, evm mech, aggregator addr, newFeeCollector addr) error {
	sequencer, err := c.State.L1PricingState().Sequencer()
	if err != nil {
		return err
	}
	if sequencer != aggregator {
		return errors.New("Cannot set fee collector for non-sequencer address")
	}
	allowed, err := accountIsSequencerOrCollectorOrOwner(c.caller, c.State)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("Only an aggregator (or its fee collector / chain owner) may change its fee collector")
	}
	return c.State.L1PricingState().SetPaySequencerFeesTo(newFeeCollector)
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

func accountIsSequencerOrCollectorOrOwner(account addr, state *arbosState.ArbosState) (bool, error) {
	l1State := state.L1PricingState()
	sequencer, err := l1State.Sequencer()
	if account == sequencer || err != nil {
		return true, err
	}
	collector, err := l1State.PaySequencerFeesTo()
	if account == collector || err != nil {
		return true, err
	}
	return state.ChainOwners().IsMember(account)
}
