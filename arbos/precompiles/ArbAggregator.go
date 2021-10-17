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

type ArbAggregator struct{}

func (con ArbAggregator) GetFeeCollector(st *state.StateDB, aggregator common.Address) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetDefaultAggregator(st *state.StateDB) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetPreferredAggregator(st *state.StateDB, addr common.Address) (common.Address, bool, error) {
	return common.Address{}, false, errors.New("unimplemented")
}

func (con ArbAggregator) GetTxBaseFee(st *state.StateDB, aggregator common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAggregator) SetFeeCollector(
	st *state.StateDB,
	aggregator common.Address,
	newFeeCollector common.Address,
) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetDefaultAggregator(st *state.StateDB, newDefault common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetPreferredAggregator(st *state.StateDB, prefAgg common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetTxBaseFee(st *state.StateDB, aggregator common.Address, feeInL1Gas *big.Int) error {
	return errors.New("unimplemented")
}
