//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbAggregator struct {
	Address addr
}

func (con ArbAggregator) GetFeeCollector(caller addr, evm mech, aggregator addr) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetFeeCollectorGasCost(aggregator addr) uint64 {
	return 0
}

func (con ArbAggregator) GetDefaultAggregator(caller addr, evm mech) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbAggregator) GetDefaultAggregatorGasCost() uint64 {
	return 0
}

func (con ArbAggregator) GetPreferredAggregator(caller addr, evm mech, address addr) (addr, bool, error) {
	return addr{}, false, errors.New("unimplemented")
}

func (con ArbAggregator) GetPreferredAggregatorGasCost(addr addr) uint64 {
	return 0
}

func (con ArbAggregator) GetTxBaseFee(caller addr, evm mech, aggregator addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAggregator) GetTxBaseFeeGasCost(aggregator addr) uint64 {
	return 0
}

func (con ArbAggregator) SetFeeCollector(caller addr, evm mech, aggregator addr, newFeeCollector addr) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetFeeCollectorGasCost(aggregator addr, newFeeCollector addr) uint64 {
	return 0
}

func (con ArbAggregator) SetDefaultAggregator(caller addr, evm mech, newDefault addr) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetDefaultAggregatorGasCost(newDefault addr) uint64 {
	return 0
}

func (con ArbAggregator) SetPreferredAggregator(caller addr, evm mech, prefAgg addr) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetPreferredAggregatorGasCost(prefAgg addr) uint64 {
	return 0
}

func (con ArbAggregator) SetTxBaseFee(caller addr, evm mech, aggregator addr, feeInL1Gas huge) error {
	return errors.New("unimplemented")
}

func (con ArbAggregator) SetTxBaseFeeGasCost(aggregator addr, feeInL1Gas huge) uint64 {
	return 0
}
