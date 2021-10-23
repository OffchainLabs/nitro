//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbGasInfo struct {
	Address addr
}

func (con ArbGasInfo) GetGasAccountingParams(caller addr, evm mech) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetGasAccountingParamsGasCost() uint64 {
	return 0
}

func (con ArbGasInfo) GetPricesInArbGas(caller addr, evm mech) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasGasCost() uint64 {
	return 0
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregator(
	caller addr,
	evm mech,
	aggregator addr,
) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregatorGasCost(aggregator addr) uint64 {
	return 0
}

func (con ArbGasInfo) GetPricesInWei(caller addr, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiGasCost() uint64 {
	return 0
}

func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	caller addr,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiWithAggregatorGasCost(aggregator addr) uint64 {
	return 0
}

func (con ArbGasInfo) GetL1GasPriceEstimate(caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetL1GasPriceEstimateGasCost() uint64 {
	return 0
}

func (con ArbGasInfo) SetL1GasPriceEstimate(caller addr, evm mech, priceInWei huge) error {
	return errors.New("unimplemented")
}

func (con ArbGasInfo) SetL1GasPriceEstimateGasCost(priceInWei huge) uint64 {
	return 0
}
