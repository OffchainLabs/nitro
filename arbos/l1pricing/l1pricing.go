//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l1pricing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

type L1PricingState struct {
	storage                     *storage.Storage
	defaultAggregator           storage.StorageBackedAddress
	l1GasPriceEstimate          storage.StorageBackedBigInt
	preferredAggregators        *storage.Storage
	aggregatorFixedCharges      *storage.Storage
	aggregatorFeeCollectors     *storage.Storage
	aggregatorCompressionRatios *storage.Storage
}

var (
	SequencerAddress = common.HexToAddress("0xA4B000000000000000000073657175656e636572")

	preferredAggregatorKey        = []byte{0}
	aggregatorFixedChargeKey      = []byte{1}
	aggregatorFeeCollectorKey     = []byte{2}
	aggregatorCompressionRatioKey = []byte{3}
)

const (
	defaultAggregatorAddressOffset uint64 = 0
	l1GasPriceEstimateOffset       uint64 = 1
)

func InitializeL1PricingState(sto *storage.Storage) error {
	err := sto.SetByUint64(defaultAggregatorAddressOffset, common.BytesToHash(SequencerAddress.Bytes()))
	if err != nil {
		return err
	}
	return sto.SetByUint64(l1GasPriceEstimateOffset, common.BigToHash(big.NewInt(50*params.GWei)))
}

func OpenL1PricingState(sto *storage.Storage) *L1PricingState {
	return &L1PricingState{
		sto,
		sto.OpenStorageBackedAddress(defaultAggregatorAddressOffset),
		sto.OpenStorageBackedBigInt(l1GasPriceEstimateOffset),
		sto.OpenSubStorage(preferredAggregatorKey),
		sto.OpenSubStorage(aggregatorFixedChargeKey),
		sto.OpenSubStorage(aggregatorFeeCollectorKey),
		sto.OpenSubStorage(aggregatorCompressionRatioKey),
	}
}

func (ps *L1PricingState) DefaultAggregator() (common.Address, error) {
	return ps.defaultAggregator.Get()
}

func (ps *L1PricingState) SetDefaultAggregator(val common.Address) error {
	return ps.defaultAggregator.Set(val)
}

func (ps *L1PricingState) L1GasPriceEstimateWei() (*big.Int, error) {
	return ps.l1GasPriceEstimate.Get()
}

func (ps *L1PricingState) SetL1GasPriceEstimateWei(val *big.Int) error {
	return ps.l1GasPriceEstimate.Set(val)
}

const L1GasPriceEstimateMemoryWeight = 24

func (ps *L1PricingState) UpdateL1GasPriceEstimate(baseFeeWei *big.Int) error {
	curr, err := ps.L1GasPriceEstimateWei()
	if err != nil {
		return err
	}

	// new = (alpha * old + observed) / (alpha + 1)
	memory := new(big.Int).Mul(curr, big.NewInt(L1GasPriceEstimateMemoryWeight))
	impact := new(big.Int).Add(memory, baseFeeWei)
	update := new(big.Int).Div(impact, big.NewInt(L1GasPriceEstimateMemoryWeight+1))

	return ps.SetL1GasPriceEstimateWei(update)
}

func (ps *L1PricingState) SetPreferredAggregator(sender common.Address, aggregator common.Address) error {
	return ps.preferredAggregators.Set(common.BytesToHash(sender.Bytes()), common.BytesToHash(aggregator.Bytes()))
}

func (ps *L1PricingState) PreferredAggregator(sender common.Address) (common.Address, bool, error) {
	fromTable, err := ps.preferredAggregators.Get(common.BytesToHash(sender.Bytes()))
	if err != nil {
		return common.Address{}, false, err
	}
	if fromTable == (common.Hash{}) {
		aggregator, err := ps.DefaultAggregator()
		return aggregator, false, err
	} else {
		return common.BytesToAddress(fromTable.Bytes()), true, nil
	}
}

func (ps *L1PricingState) SetFixedChargeForAggregatorL1Gas(aggregator common.Address, chargeL1Gas *big.Int) error {
	return ps.aggregatorFixedCharges.Set(common.BytesToHash(aggregator.Bytes()), common.BigToHash(chargeL1Gas))
}

func (ps *L1PricingState) FixedChargeForAggregatorL1Gas(aggregator common.Address) (*big.Int, error) {
	value, err := ps.aggregatorFixedCharges.Get(common.BytesToHash(aggregator.Bytes()))
	return value.Big(), err
}
func (ps *L1PricingState) FixedChargeForAggregatorWei(aggregator common.Address) (*big.Int, error) {
	fixed, err := ps.FixedChargeForAggregatorL1Gas(aggregator)
	if err != nil {
		return nil, err
	}
	price, err := ps.L1GasPriceEstimateWei()
	if err != nil {
		return nil, err
	}
	return new(big.Int).Mul(fixed, price), nil
}

func (ps *L1PricingState) SetAggregatorFeeCollector(aggregator common.Address, addr common.Address) error {
	return ps.aggregatorFeeCollectors.Set(common.BytesToHash(aggregator.Bytes()), common.BytesToHash(addr.Bytes()))
}

func (ps *L1PricingState) AggregatorFeeCollector(aggregator common.Address) (common.Address, error) {
	raw, err := ps.aggregatorFeeCollectors.Get(common.BytesToHash(aggregator.Bytes()))
	if raw == (common.Hash{}) {
		return aggregator, err
	} else {
		return common.BytesToAddress(raw.Bytes()), err
	}
}

func (ps *L1PricingState) AggregatorCompressionRatio(aggregator common.Address) (uint64, error) {
	raw, err := ps.aggregatorCompressionRatios.Get(common.BytesToHash(aggregator.Bytes()))
	if raw == (common.Hash{}) {
		return DataWasNotCompressed, err
	} else {
		return raw.Big().Uint64(), err
	}
}

func (ps *L1PricingState) SetAggregatorCompressionRatio(aggregator common.Address, ratio *uint64) error {
	val := DataWasNotCompressed
	if (ratio != nil) && (*ratio < DataWasNotCompressed) {
		val = *ratio
	}
	return ps.aggregatorCompressionRatios.Set(util.AddressToHash(aggregator), util.UintToHash(val))
}

// Compression ratio is expressed in fixed-point representation.  A value of DataWasNotCompressed corresponds to
//    a compression ratio of 1, that is, no compression.
// A value of x (for x <= DataWasNotCompressed) corresponds to compression ratio of float(x) / float(DataWasNotCompressed).
// Values greater than DataWasNotCompressed are treated as equivalent to DataWasNotCompressed.

const DataWasNotCompressed uint64 = 1000000
const TxFixedCost = 64 // TODO: Pick a better fixed cost

func (ps *L1PricingState) PosterDataCost(
	sender common.Address,
	aggregator *common.Address,
	data []byte,
) (*big.Int, error) {
	if aggregator == nil {
		return big.NewInt(0), nil
	}
	preferredAggregator, _, err := ps.PreferredAggregator(sender)
	if err != nil {
		return nil, err
	}
	if preferredAggregator != *aggregator {
		return big.NewInt(0), nil
	}

	bytesToCharge := uint64(len(data) + TxFixedCost)

	ratio, err := ps.AggregatorCompressionRatio(preferredAggregator)
	if err != nil {
		return nil, err
	}

	dataGas := 16 * bytesToCharge * ratio / DataWasNotCompressed

	// add 5% to protect the aggregator bad price fluctuation luck
	dataGas = dataGas * 21 / 20

	price, err := ps.L1GasPriceEstimateWei()
	if err != nil {
		return nil, err
	}

	preferred, err := ps.FixedChargeForAggregatorWei(preferredAggregator)
	if err != nil {
		return nil, err
	}

	chargeForBytes := new(big.Int).Mul(big.NewInt(int64(dataGas)), price)
	return new(big.Int).Add(preferred, chargeForBytes), nil
}
