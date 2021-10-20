package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

type L1PricingState struct {
	segment                  *StorageSegment
	defaultAggregator        common.Address
	l1GasPriceEstimate       *big.Int
	preferredAggregators     EvmStorage
	aggregatorFixedCharges   EvmStorage
	aggregatorAddressesToPay EvmStorage
	aggregatorCoompressionRaios EvmStorage
}

const CompressionEstimateDenominator uint64 = 1000000

var (
	initialDefaultAggregator  = common.Address{} //TODO
	preferredAggregatorKey    = crypto.Keccak256Hash([]byte("Arbitrum ArbOS preferred aggregator key"))
	aggregatorFixedChargeKey  = crypto.Keccak256Hash([]byte("Arbitrum ArbOS aggregator fixed charge key"))
	aggregatorAddressToPayKey = crypto.Keccak256Hash([]byte("Arbitrum ArbOS aggregator address to pay key"))
	aggregatorCompressionRatioKey = crypto.Keccak256Hash([]byte("Arbitrum ArbOS aggregator compression ratio key"))
)

const (
	defaultAggregatorAddressOffset = 0
	l1GasPriceEstimateOffset       = 1
)
const L1PricingStateSize = 2

func AllocateL1PricingState(state *ArbosState) (*L1PricingState, common.Hash) {
	segment, err := state.AllocateSegment(L1PricingStateSize)
	if err != nil {
		panic("failed to allocate segment for L1 pricing state")
	}
	segment.Set(defaultAggregatorAddressOffset, common.BytesToHash(initialDefaultAggregator.Bytes()))
	l1PriceEstimate := big.NewInt(1 * params.GWei)
	segment.Set(l1GasPriceEstimateOffset, common.BigToHash(l1PriceEstimate))
	return &L1PricingState{
		segment,
		initialDefaultAggregator,
		l1PriceEstimate,
		NewVirtualStorage(state.backingStorage, preferredAggregatorKey),
		NewVirtualStorage(state.backingStorage, aggregatorFixedChargeKey),
		NewVirtualStorage(state.backingStorage, aggregatorAddressToPayKey),
		NewVirtualStorage(state.backingStorage, aggregatorCompressionRatioKey),
	}, segment.offset
}

func OpenL1PricingState(offset common.Hash, state *ArbosState) *L1PricingState {
	segment := state.OpenSegment(offset)
	defaultAggregator := common.BytesToAddress(segment.Get(defaultAggregatorAddressOffset).Bytes())
	l1GasPriceEstimate := segment.Get(l1GasPriceEstimateOffset).Big()
	return &L1PricingState{
		segment,
		defaultAggregator,
		l1GasPriceEstimate,
		NewVirtualStorage(state.backingStorage, preferredAggregatorKey),
		NewVirtualStorage(state.backingStorage, aggregatorFixedChargeKey),
		NewVirtualStorage(state.backingStorage, aggregatorAddressToPayKey),
		NewVirtualStorage(state.backingStorage, aggregatorCompressionRatioKey),
	}
}

func (ps *L1PricingState) SetDefaultAggregator(aggregator common.Address) {
	ps.defaultAggregator = aggregator
	ps.segment.Set(defaultAggregatorAddressOffset, common.BytesToHash(aggregator.Bytes()))
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
	ps.segment.Set(l1GasPriceEstimateOffset, common.BigToHash(ps.l1GasPriceEstimate))
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

func (ps *L1PricingState) AggregatorAddressToPay(aggregator common.Address, state *ArbosState) common.Address {
	raw := ps.aggregatorAddressesToPay.Get(common.BytesToHash(aggregator.Bytes()))
	if raw == (common.Hash{}) {
		return aggregator
	} else {
		return common.BytesToAddress(raw.Bytes())
	}
}

func (ps *L1PricingState) AggregatorCompressionRatio(aggregator common.Address) uint64 {
	raw := ps.aggregatorAddressesToPay.Get(common.BytesToHash(aggregator.Bytes()))
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
	ps.aggregatorCoompressionRaios.Set(common.BytesToHash(aggregator.Bytes()), common.BigToHash(big.NewInt(int64(val))))
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
