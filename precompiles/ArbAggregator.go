//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbAggregator struct{}

func (con ArbAggregator) GetFeeCollector(caller addr, st *stateDB, aggregator addr) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetFeeCollectorGasCost(aggregator addr) uint64 {
	return 0
}

func (con ArbAggregator) GetDefaultAggregator(caller addr, st *stateDB) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetDefaultAggregatorGasCost() uint64 {
	return 0
}

func (con ArbAggregator) GetPreferredAggregator(caller addr, st *stateDB, address addr) (addr, bool, error) {
	return addr{}, false, errors.New("unimplemented")
}

func (con ArbAggregator) GetPreferredAggregatorGasCost(addr addr) uint64 {
	return 0
}

func (con ArbAggregator) GetTxBaseFee(caller addr, st *stateDB, aggregator addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAggregator) GetTxBaseFeeGasCost(aggregator addr) uint64 {
	return 0
}

func (con ArbAggregator) SetFeeCollector(caller addr, st *stateDB, aggregator addr, newFeeCollector addr) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetFeeCollectorGasCost(aggregator addr, newFeeCollector addr) uint64 {
	return 0
}

func (con ArbAggregator) SetDefaultAggregator(caller addr, st *stateDB, newDefault addr) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetDefaultAggregatorGasCost(newDefault addr) uint64 {
	return 0
}

func (con ArbAggregator) SetPreferredAggregator(caller addr, st *stateDB, prefAgg addr) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetPreferredAggregatorGasCost(prefAgg addr) uint64 {
	return 0
}

func (con ArbAggregator) SetTxBaseFee(caller addr, st *stateDB, aggregator addr, feeInL1Gas huge) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetTxBaseFeeGasCost(aggregator addr, feeInL1Gas huge) uint64 {
	return 0
}
