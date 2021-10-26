//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
)

type ArbAggregator struct {
	Address addr
}

func (con ArbAggregator) GetFeeCollector(b burn, caller addr, evm mech, aggregator addr) (addr, error) {
	if err := b(params.SloadGas); err != nil {
		return addr{}, err
	}
	return arbos.OpenArbosState(evm.StateDB).L1PricingState().AggregatorFeeCollector(aggregator), nil
}

func (con ArbAggregator) GetDefaultAggregator(b burn, caller addr, evm mech) (addr, error) {
	if err := b(params.SloadGas); err != nil {
		return addr{}, err
	}
	return arbos.OpenArbosState(evm.StateDB).L1PricingState().DefaultAggregator(), nil
}

func (con ArbAggregator) GetPreferredAggregator(b burn, caller addr, evm mech, address addr) (addr, bool, error) {
	if err := b(params.SloadGas); err != nil {
		return addr{}, false, err
	}
	res, exists := arbos.OpenArbosState(evm.StateDB).L1PricingState().PreferredAggregator(address)
	return res, exists, nil
}

func (con ArbAggregator) GetTxBaseFee(b burn, caller addr, evm mech, aggregator addr) (huge, error) {
	if err := b(params.SloadGas); err != nil {
		return nil, err
	}
	return arbos.OpenArbosState(evm.StateDB).L1PricingState().FixedChargeForAggregatorL1Gas(aggregator), nil
}

func (con ArbAggregator) SetFeeCollector(b burn, caller addr, evm mech, aggregator addr, newFeeCollector addr) error {
	if err := b(params.SloadGas + params.SstoreSetGas); err != nil {
		return err
	}
	l1State := arbos.OpenArbosState(evm.StateDB).L1PricingState()
	if (caller != aggregator) && (caller != l1State.AggregatorFeeCollector(aggregator)) {
		// only the aggregator and its current fee collector can change the aggregator's fee collector
		return errors.New("non-authorized caller in ArbAggregator.SetFeeCollector")
	}
	l1State.SetAggregatorFeeCollector(aggregator, newFeeCollector)
	return nil
}

func (con ArbAggregator) SetDefaultAggregator(b burn, caller addr, evm mech, newDefault addr) error {
	if err := b(params.SstoreSetGas); err != nil {
		return err
	}
	arbos.OpenArbosState(evm.StateDB).L1PricingState().SetDefaultAggregator(newDefault)
	return nil
}

func (con ArbAggregator) SetPreferredAggregator(b burn, caller addr, evm mech, prefAgg addr) error {
	if err := b(params.SstoreSetGas); err != nil {
		return err
	}
	arbos.OpenArbosState(evm.StateDB).L1PricingState().SetPreferredAggregator(caller, prefAgg)
	return nil
}

func (con ArbAggregator) SetTxBaseFee(b burn, caller addr, evm mech, aggregator addr, feeInL1Gas huge) error {
	if err := b(params.SstoreSetGas); err != nil {
		return err
	}
	arbos.OpenArbosState(evm.StateDB).L1PricingState().SetFixedChargeForAggregatorL1Gas(aggregator, feeInL1Gas)
	return nil
}
