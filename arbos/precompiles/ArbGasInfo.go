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

func (con ArbGasInfo) GetGasAccountingParams(st *state.StateDB) (*big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGas(st *state.StateDB) (*big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregator(
	st *state.StateDB,
	aggregator common.Address,
) (*big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWei(
	st *state.StateDB,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	st *state.StateDB,
	aggregator common.Address,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetL1GasPriceEstimate(st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbGasInfo) SetL1GasPriceEstimate(st *state.StateDB, priceInWei *big.Int) error {
	return errors.New("unimplemented")
}
