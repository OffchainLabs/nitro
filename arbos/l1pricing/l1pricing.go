//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l1pricing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"math/big"
)

type L1PricingState struct {
	storage                  *storage.Storage
	defaultAggregator        common.Address
	l1GasPriceEstimate       *big.Int
	preferredAggregators     *storage.Storage
	aggregatorFixedCharges   *storage.Storage
	aggregatorAddressesToPay *storage.Storage
	aggregatorCompressionRatios *storage.Storage
}

var (
	initialDefaultAggregator = common.Address{} //TODO

	preferredAggregatorKey    = []byte{0}
	aggregatorFixedChargeKey  = []byte{1}
	aggregatorAddressToPayKey = []byte{2}
	aggregatorCompressionRatioKey = []byte{3}
)

const (
	defaultAggregatorAddressOffset int64 = 0
	l1GasPriceEstimateOffset       int64 = 1
)

func InitializeL1PricingState(sto *storage.Storage) {
	sto.SetByInt64(defaultAggregatorAddressOffset, common.BytesToHash(initialDefaultAggregator.Bytes()))
	sto.SetByInt64(l1GasPriceEstimateOffset, common.BigToHash(big.NewInt(50*params.GWei)))
}

func OpenL1PricingState(sto *storage.Storage) *L1PricingState {
	defaultAggregator := common.BytesToAddress(sto.GetByInt64(defaultAggregatorAddressOffset).Bytes())
	l1GasPriceEstimate := sto.GetByInt64(l1GasPriceEstimateOffset).Big()
	return &L1PricingState{
		sto,
		defaultAggregator,
		l1GasPriceEstimate,
		sto.OpenSubStorage(preferredAggregatorKey),
		sto.OpenSubStorage(aggregatorFixedChargeKey),
		sto.OpenSubStorage(aggregatorAddressToPayKey),
		sto.OpenSubStorage(aggregatorCompressionRatioKey),
	}
}

func (ps *L1PricingState) SetDefaultAggregator(aggregator common.Address) {
	ps.defaultAggregator = aggregator
	ps.storage.SetByInt64(defaultAggregatorAddressOffset, common.BytesToHash(aggregator.Bytes()))
}

func (ps *L1PricingState) L1GasPriceEstimateWei() *big.Int {
	return ps.l1GasPriceEstimate
}

const L1GasPriceEstimateSamplesInAverage = 25

func (ps *L1PricingState) UpdateL1GasPriceEstimate(baseFeeWei *big.Int) {
	ps.l1GasPriceEstimate = new(big.Int).Div(
		new(big.Int).Add(
			baseFeeWei,
			new(big.Int).Mul(ps.l1GasPriceEstimate, big.NewInt(L1GasPriceEstimateSamplesInAverage-1)),
		),
		big.NewInt(L1GasPriceEstimateSamplesInAverage),
	)
	ps.storage.SetByInt64(l1GasPriceEstimateOffset, common.BigToHash(ps.l1GasPriceEstimate))
}

func (ps *L1PricingState) SetPreferredAggregator(sender common.Address, aggregator common.Address) {
	ps.preferredAggregators.Set(common.BytesToHash(sender.Bytes()), common.BytesToHash(aggregator.Bytes()))
}

func (ps *L1PricingState) PreferredAggregator(sender common.Address) common.Address {
	fromTable := ps.preferredAggregators.Get(common.BytesToHash(sender.Bytes()))
	if fromTable == (common.Hash{}) {
		return ps.defaultAggregator
	} else {
		return common.BytesToAddress(fromTable.Bytes())
	}
}

func (ps *L1PricingState) SetFixedChargeForAggregatorWei(aggregator common.Address, chargeL1Gas *big.Int) {
	ps.aggregatorFixedCharges.Set(common.BytesToHash(aggregator.Bytes()), common.BigToHash(chargeL1Gas))
}

func (ps *L1PricingState) FixedChargeForAggregatorWei(aggregator common.Address) *big.Int {
	fixedChargeL1Gas := ps.aggregatorFixedCharges.Get(common.BytesToHash(aggregator.Bytes())).Big()
	return new(big.Int).Mul(fixedChargeL1Gas, ps.L1GasPriceEstimateWei())
}

func (ps *L1PricingState) SetAggregatorAddressToPay(aggregator common.Address, addr common.Address) {
	ps.aggregatorAddressesToPay.Set(common.BytesToHash(aggregator.Bytes()), common.BytesToHash(addr.Bytes()))
}

func (ps *L1PricingState) AggregatorAddressToPay(aggregator common.Address) common.Address {
	raw := ps.aggregatorAddressesToPay.Get(common.BytesToHash(aggregator.Bytes()))
	if raw == (common.Hash{}) {
		return aggregator
	} else {
		return common.BytesToAddress(raw.Bytes())
	}
}

func (ps *L1PricingState) AggregatorCompressionRatio(aggregator common.Address) uint64 {
	raw := ps.aggregatorCompressionRatios.Get(common.BytesToHash(aggregator.Bytes()))
	if raw == (common.Hash{}) {
		return DataWasNotCompressed
	} else {
		return raw.Big().Uint64()
	}
}

func (ps *L1PricingState) SetAggregatorCompressionRatio(aggregator common.Address, ratio *uint64) {
	val := DataWasNotCompressed
	if (ratio != nil) && (*ratio < DataWasNotCompressed) {
		val = *ratio
	}
	ps.aggregatorCompressionRatios.Set(common.BytesToHash(aggregator.Bytes()), common.BigToHash(big.NewInt(int64(val))))
}

// Compression ratio is expressed in fixed-point representation.  A value of DataWasNotCompressed corresponds to
//    a compression ratio of 1, that is, no compression.
// A value of x (for x <= DataWasNotCompressed) corresponds to compression ratio of float(x) / float(DataWasNotCompressed).
// Values greater than DataWasNotCompressed are treated as equivalent to DataWasNotCompressed.

const DataWasNotCompressed uint64 = 1000000

func (ps *L1PricingState) GetL1Charges(
	sender common.Address,
	aggregator *common.Address,
	data []byte,
	wasCompressed bool,
) *big.Int {
	if aggregator == nil {
		return big.NewInt(0)
	}
	preferredAggregator := ps.PreferredAggregator(sender)
	if preferredAggregator != *aggregator {
		return big.NewInt(0)
	}

	var dataGas uint64
	if wasCompressed {
		dataGas = 16 * uint64(len(data)) * ps.AggregatorCompressionRatio(preferredAggregator) / DataWasNotCompressed
	} else {
		var err error
		dataGas, err = core.IntrinsicGas(data, nil, false, true, true)
		if err == nil {
			dataGas -= params.TxGas
		} else {
			dataGas = 16 * uint64(len(data))
		}
	}

	// add 5% to protect the aggregator bad price fluctuation luck
	dataGas = dataGas * 21 / 20

	chargeForBytes := new(big.Int).Mul(big.NewInt(int64(dataGas)), ps.L1GasPriceEstimateWei())
	return new(big.Int).Add(ps.FixedChargeForAggregatorWei(preferredAggregator), chargeForBytes)
}
