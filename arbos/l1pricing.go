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
	compressionEstimate uint64 // estimate compression ratio is this/CompressionEstimateDenominator
}

const L1PricingStateSize = 3
const CompressionEstimateDenominator uint64 = 1000000

var (
	initialDefaultAggregator = common.Address{} //TODO
	preferredAggregatorKey   = crypto.Keccak256Hash([]byte("Arbitrum ArbOS preferred aggregator key")).Bytes()
	aggregatorFixedChargeKey = crypto.Keccak256Hash([]byte("Arbitrum ArbOS aggregator fixed charge key")).Bytes()
)

func AllocateL1PricingState(state *ArbosState) (*L1PricingState, common.Hash) {
	segment, err := state.AllocateSegment(L1PricingStateSize)
	if err != nil {
		panic("failed to allocate segment for L1 pricing state")
	}
	segment.Set(0, common.BytesToHash(initialDefaultAggregator.Bytes()))
	l1PriceEstimate := big.NewInt(1 * params.GWei)
	segment.Set(1, common.BigToHash(l1PriceEstimate))
	compressionEstimate := CompressionEstimateDenominator
	segment.Set(2, IntToHash(int64(CompressionEstimateDenominator)))
	return &L1PricingState{
		segment,
		initialDefaultAggregator,
		l1PriceEstimate,
		compressionEstimate,
	}, segment.offset
}

func OpenL1PricingState(offset common.Hash, state *ArbosState) *L1PricingState {
	segment := state.OpenSegment(offset)
	defaultAggregator := common.BytesToAddress(segment.Get(0).Bytes())
	l1GasPriceEstimate := segment.Get(1).Big()
	compressionEstimate := segment.Get(2).Big().Uint64()
	return &L1PricingState{
		segment,
		defaultAggregator,
		l1GasPriceEstimate,
		compressionEstimate,
	}
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
	ps.segment.Set(1, common.BigToHash(ps.l1GasPriceEstimate))
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
	ps.compressionEstimate = (compressedSize*CompressionEstimateDenominator +
		(BytesInCompressionEstimate-decompressedSize)*ps.compressionEstimate) / BytesInCompressionEstimate
	ps.segment.Set(2, IntToHash(int64(ps.compressionEstimate)))
}

func (ps *L1PricingState) PreferredAggregator(sender common.Address) common.Address {
	fromTable := common.BytesToAddress(
		ps.segment.storage.Get(crypto.Keccak256Hash(preferredAggregatorKey, sender.Bytes())).Bytes()[:20],
	)
	if fromTable == (common.Address{}) {
		return ps.defaultAggregator
	} else {
		return fromTable
	}
}

func (ps *L1PricingState) FixedChargeForAggregatorWei(aggregator common.Address) *big.Int {
	return ps.segment.storage.Get(crypto.Keccak256Hash(aggregatorFixedChargeKey, aggregator.Bytes())).Big()
}
