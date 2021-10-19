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

func (con ArbAggregator) GetFeeCollector(
	caller common.Address,
	st *state.StateDB,
	aggregator common.Address,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetFeeCollectorGasCost(aggregator common.Address) uint64 {
	return 0
}

func (con ArbAggregator) GetDefaultAggregator(caller common.Address, st *state.StateDB) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetDefaultAggregatorGasCost() uint64 {
	return 0
}

func (con ArbAggregator) GetPreferredAggregator(
	caller common.Address,
	st *state.StateDB,
	addr common.Address,
) (common.Address, bool, error) {
	return common.Address{}, false, errors.New("unimplemented")
}

func (con ArbAggregator) GetPreferredAggregatorGasCost(addr common.Address) uint64 {
	return 0
}

func (con ArbAggregator) GetTxBaseFee(
	caller common.Address,
	st *state.StateDB,
	aggregator common.Address,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAggregator) GetTxBaseFeeGasCost(aggregator common.Address) uint64 {
	return 0
}

func (con ArbAggregator) SetFeeCollector(
	caller common.Address,
	st *state.StateDB,
	aggregator common.Address,
	newFeeCollector common.Address,
) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetFeeCollectorGasCost(aggregator common.Address, newFeeCollector common.Address) uint64 {
	return 0
}

func (con ArbAggregator) SetDefaultAggregator(
	caller common.Address,
	st *state.StateDB,
	newDefault common.Address,
) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetDefaultAggregatorGasCost(newDefault common.Address) uint64 {
	return 0
}

func (con ArbAggregator) SetPreferredAggregator(
	caller common.Address,
	st *state.StateDB,
	prefAgg common.Address,
) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetPreferredAggregatorGasCost(prefAgg common.Address) uint64 {
	return 0
}

func (con ArbAggregator) SetTxBaseFee(
	caller common.Address,
	st *state.StateDB,
	aggregator common.Address,
	feeInL1Gas *big.Int,
) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetTxBaseFeeGasCost(aggregator common.Address, feeInL1Gas *big.Int) uint64 {
	return 0
}
