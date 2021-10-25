//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"math/big"
)

type ArbAggregator struct {
	Address addr
}

func (con ArbAggregator) GetFeeCollector(
	caller common.Address,
	evm mech,
	aggregator common.Address,
) (common.Address, error) {
	return arbos.OpenArbosState(evm.StateDB).L1PricingState().AggregatorFeeCollector(aggregator), nil
}

func (con ArbAggregator) GetFeeCollectorGasCost(aggregator common.Address) uint64 {
	return params.SloadGas
}

func (con ArbAggregator) GetDefaultAggregator(caller common.Address, st *state.StateDB) (common.Address, error) {
	return arbos.OpenArbosState(st).L1PricingState().DefaultAggregator(), nil
}

func (con ArbAggregator) GetDefaultAggregatorGasCost() uint64 {
	return params.SloadGas
}

func (con ArbAggregator) GetPreferredAggregator(
	caller common.Address,
	evm mech,
	addr common.Address,
) (common.Address, bool, error) {
	res, exists := arbos.OpenArbosState(evm.StateDB).L1PricingState().PreferredAggregator(addr)
	return res, exists, nil
}

func (con ArbAggregator) GetPreferredAggregatorGasCost(addr common.Address) uint64 {
	return params.SloadGas
}

func (con ArbAggregator) GetTxBaseFee(
	caller common.Address,
	evm mech,
	aggregator common.Address,
) (*big.Int, error) {
	return arbos.OpenArbosState(evm.StateDB).L1PricingState().FixedChargeForAggregatorL1Gas(aggregator), nil
}

func (con ArbAggregator) GetTxBaseFeeGasCost(aggregator common.Address) uint64 {
	return params.SloadGas
}

func (con ArbAggregator) SetFeeCollector(
	caller common.Address,
	evm mech,
	aggregator common.Address,
	newFeeCollector common.Address,
) error {
	l1State := arbos.OpenArbosState(evm.StateDB).L1PricingState()
	if (caller != aggregator) && (caller != l1State.AggregatorFeeCollector(aggregator)) {
		// only the aggregator and its current fee collector can change the aggregator's fee collector
		return errors.New("non-authorized caller in ArbAggregator.SetFeeCollector")
	}
	l1State.SetAggregatorFeeCollector(aggregator, newFeeCollector)
	return nil
}

func (con ArbAggregator) SetFeeCollectorGasCost(aggregator common.Address, newFeeCollector common.Address) uint64 {
	return params.SloadGas + params.SstoreSetGas
}

func (con ArbAggregator) SetDefaultAggregator(
	caller common.Address,
	evm mech,
	newDefault common.Address,
) error {
	arbos.OpenArbosState(evm.StateDB).L1PricingState().SetDefaultAggregator(newDefault)
	return nil
}

func (con ArbAggregator) SetDefaultAggregatorGasCost(newDefault common.Address) uint64 {
	return params.SstoreSetGas
}

func (con ArbAggregator) SetPreferredAggregator(
	caller common.Address,
	evm mech,
	prefAgg common.Address,
) error {
	arbos.OpenArbosState(evm.StateDB).L1PricingState().SetPreferredAggregator(caller, prefAgg)
	return nil
}

func (con ArbAggregator) SetPreferredAggregatorGasCost(prefAgg common.Address) uint64 {
	return params.SstoreSetGas
}

func (con ArbAggregator) SetTxBaseFee(
	caller common.Address,
	evm mech,
	aggregator common.Address,
	feeInL1Gas *big.Int,
) error {
	arbos.OpenArbosState(evm.StateDB).L1PricingState().SetFixedChargeForAggregatorL1Gas(aggregator, feeInL1Gas)
	return nil
}

func (con ArbAggregator) SetTxBaseFeeGasCost(aggregator common.Address, feeInL1Gas *big.Int) uint64 {
	return params.SstoreSetGas
}
