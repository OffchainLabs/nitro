//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)

type ArbGasInfo struct{}

func (con ArbGasInfo) GetGasAccountingParams(
	caller common.Address,
	st *state.StateDB,
) (*big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetGasAccountingParamsGasCost() *big.Int {
	return nil
}

func (con ArbGasInfo) GetPricesInArbGas(
	caller common.Address,
	st *state.StateDB,
) (*big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasGasCost() *big.Int {
	return nil
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregator(
	caller common.Address,
	st *state.StateDB,
	aggregator common.Address,
) (*big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregatorGasCost(aggregator common.Address) *big.Int {
	return nil
}

func (con ArbGasInfo) GetPricesInWei(
	caller common.Address,
	st *state.StateDB,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiGasCost() *big.Int {
	return nil
}

func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	caller common.Address,
	st *state.StateDB,
	aggregator common.Address,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiWithAggregatorGasCost(aggregator common.Address) *big.Int {
	return nil
}

func (con ArbGasInfo) GetL1GasPriceEstimate(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetL1GasPriceEstimateGasCost() *big.Int {
	return nil
}

func (con ArbGasInfo) SetL1GasPriceEstimate(caller common.Address, st *state.StateDB, priceInWei *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbGasInfo) SetL1GasPriceEstimateGasCost(priceInWei *big.Int) *big.Int {
	return nil
}
