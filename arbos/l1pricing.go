package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

type L1PricingState struct {
	segment             *StorageSegment
	defaultAggregator   common.Address
	l1GasPriceEstimate  *big.Int
	compressionEstimate uint64 // estimated compression ratio is this/CompressionEstimateDenominator
}

const CompressionEstimateDenominator uint64 = 1000000

var (
	initialDefaultAggregator = common.Address{} //TODO
	preferredAggregatorKey   = crypto.Keccak256([]byte("Arbitrum ArbOS preferred aggregator key"))
	aggregatorFixedChargeKey = crypto.Keccak256([]byte("Arbitrum ArbOS aggregator fixed charge key"))
	aggregatorAddressToPayKey = crypto.Keccak256([]byte("Arbitrum ArbOS aggregator address to pay key"))
)

const (
	defaultAggregatorAddressOffset = 0
	l1GasPriceEstimateOffset       = 1
	compressionEstimateOffset      = 2
)
const L1PricingStateSize = 3

func AllocateL1PricingState(state *ArbosState) (*L1PricingState, common.Hash) {
	segment, err := state.AllocateSegment(L1PricingStateSize)
	if err != nil {
		panic("failed to allocate segment for L1 pricing state")
	}
	segment.Set(defaultAggregatorAddressOffset, common.BytesToHash(initialDefaultAggregator.Bytes()))
	l1PriceEstimate := big.NewInt(1 * params.GWei)
	segment.Set(l1GasPriceEstimateOffset, common.BigToHash(l1PriceEstimate))
	compressionEstimate := CompressionEstimateDenominator
	segment.Set(compressionEstimateOffset, IntToHash(int64(CompressionEstimateDenominator)))
	return &L1PricingState{
		segment,
		initialDefaultAggregator,
		l1PriceEstimate,
		compressionEstimate,
	}, segment.offset
}

func OpenL1PricingState(offset common.Hash, state *ArbosState) *L1PricingState {
	segment := state.OpenSegment(offset)
	defaultAggregator := common.BytesToAddress(segment.Get(defaultAggregatorAddressOffset).Bytes())
	l1GasPriceEstimate := segment.Get(l1GasPriceEstimateOffset).Big()
	compressionEstimate := segment.Get(compressionEstimateOffset).Big().Uint64()
	return &L1PricingState{
		segment,
		defaultAggregator,
		l1GasPriceEstimate,
		compressionEstimate,
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

func (ps *L1PricingState) CompressedSizeEstimate(decompressedSize uint64) uint64 {
	return 1 + (decompressedSize * ps.compressionEstimate / CompressionEstimateDenominator)
}

const BytesInCompressionEstimate = 100000
const MinSamplesInCompressionEstimate = 10
const MaxSampleForCompressionEstimate = BytesInCompressionEstimate / MinSamplesInCompressionEstimate

func (ps *L1PricingState) UpdateCompressedSizeEstimate(compressedSize uint64, decompressedSize uint64) {
	if decompressedSize == 0 {
		return
	}
	if decompressedSize > MaxSampleForCompressionEstimate {
		// sample is large, so limit its influence to 1 part in MinSamplesInCompressionEstimate
		compressedSize = MaxSampleForCompressionEstimate * compressedSize / decompressedSize
		decompressedSize = MaxSampleForCompressionEstimate
	}
	ps.compressionEstimate = compressedSize*CompressionEstimateDenominator + (BytesInCompressionEstimate-decompressedSize)*ps.compressionEstimate
	ps.segment.Set(compressionEstimateOffset, IntToHash(int64(ps.compressionEstimate)))
}

func offsetForPreferredAggregator(sender common.Address) common.Hash {
	return crypto.Keccak256Hash(preferredAggregatorKey, sender.Bytes())
}

func (ps *L1PricingState) SetPreferredAggregator(sender common.Address, aggregator common.Address) {
	ps.segment.storage.Set(offsetForPreferredAggregator(sender), common.BytesToHash(aggregator.Bytes()))
}

func (ps *L1PricingState) PreferredAggregator(sender common.Address) common.Address {
	fromTable := common.BytesToAddress(ps.segment.storage.Get(offsetForPreferredAggregator(sender)).Bytes()[:20])
	if fromTable == (common.Address{}) {
		return ps.defaultAggregator
	} else {
		return fromTable
	}
}

func offsetForAggregatorCharge(aggregator common.Address) common.Hash {
	return crypto.Keccak256Hash(aggregatorFixedChargeKey, aggregator.Bytes())
}

func (ps *L1PricingState) SetFixedChargeForAggregatorWei(aggregator common.Address, chargeL1Gas *big.Int) {
	ps.segment.storage.Set(offsetForAggregatorCharge(aggregator), common.BigToHash(chargeL1Gas))
}

func (ps *L1PricingState) FixedChargeForAggregatorWei(aggregator common.Address) *big.Int {
	fixedChargeL1Gas := ps.segment.storage.Get(offsetForAggregatorCharge(aggregator)).Big()
	return new(big.Int).Mul(fixedChargeL1Gas, ps.L1GasPriceEstimateWei())
}

func offsetForAggregatorAddressToPay(aggregator common.Address) common.Hash {
	return crypto.Keccak256Hash(aggregatorAddressToPayKey, aggregator.Bytes())
}

func SetAggregatorAddressToPay(aggregator common.Address, addr common.Address, state *ArbosState) {
	state.backingStorage.Set(offsetForAggregatorCharge(aggregator), common.BytesToHash(addr.Bytes()))
}

func AggregatorAddressToPay(aggregator common.Address, state *ArbosState) common.Address {
	raw := state.backingStorage.Get(offsetForAggregatorAddressToPay(aggregator))
	if raw == (common.Hash{}) {
		return aggregator
	} else {
		return common.BytesToAddress(raw.Bytes())
	}
}

func (ps *L1PricingState) GetL1Charges(
	sender common.Address,
	aggregator *common.Address,
	sizeBytes uint64,
	wasCompressed bool,
) *big.Int {
	if aggregator == nil {
		return big.NewInt(0)
	}
	preferredAggregator := ps.PreferredAggregator(sender)
	if preferredAggregator != *aggregator {
		return big.NewInt(0)
	}

	if wasCompressed {
		sizeBytes = ps.CompressedSizeEstimate(sizeBytes)
	}

	chargeForBytes := new(big.Int).Mul(big.NewInt(int64(sizeBytes*16)), ps.L1GasPriceEstimateWei())
	return new(big.Int).Add(ps.FixedChargeForAggregatorWei(preferredAggregator), chargeForBytes)
}
