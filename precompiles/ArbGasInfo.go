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

func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGas(c ctx, evm mech) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	c ctx,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbGasInfo) SetL1GasPriceEstimate(c ctx, evm mech, priceInWei huge) error {
	return errors.New("unimplemented")
}
