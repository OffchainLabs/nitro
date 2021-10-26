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

func (con ArbGasInfo) GetGasAccountingParams(b burn, caller addr, evm mech) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGas(b burn, caller addr, evm mech) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInArbGasWithAggregator(
	b burn,
	caller addr,
	evm mech,
	aggregator addr,
) (huge, huge, huge, error) {
	return nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWei(b burn, caller addr, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	b burn,
	caller addr,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbGasInfo) GetL1GasPriceEstimate(b burn, caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbGasInfo) SetL1GasPriceEstimate(b burn, caller addr, evm mech, priceInWei huge) error {
	return errors.New("unimplemented")
}
