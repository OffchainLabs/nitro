//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package l1pricing

import (
	"errors"
	"math/big"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

type L1PricingState struct {
	storage                     *storage.Storage
	defaultAggregator           storage.StorageBackedAddress
	l1BaseFeeEstimate           storage.StorageBackedBigInt
	l1BaseFeeEstimateInertia    storage.StorageBackedUint64
	lastL1BaseFeeUpdateTime     storage.StorageBackedUint64
	userSpecifiedAggregators    *storage.Storage
	refuseDefaultAggregator     *storage.Storage
	aggregatorFeeCollectors     *storage.Storage
	aggregatorCompressionRatios *storage.Storage
}

var (
	SequencerAddress = common.HexToAddress("0xA4B000000000000000000073657175656e636572")

	userSpecifiedAggregatorKey    = []byte{0}
	refuseDefaultAggregatorKey    = []byte{1}
	aggregatorFeeCollectorKey     = []byte{2}
	aggregatorCompressionRatioKey = []byte{3}
)

const (
	defaultAggregatorAddressOffset uint64 = 0
	l1BaseFeeEstimateOffset        uint64 = 1
	l1BaseFeeEstimateInertiaOffset uint64 = 2
	lastL1BaseFeeUpdateTimeOffset  uint64 = 3
)

const InitialL1BaseFeeEstimate = 50 * params.GWei
const InitialL1BaseFeeEstimateInertia = 24

func InitializeL1PricingState(sto *storage.Storage) error {
	// this deliberately doesn't initialize lastL1BaseFeeUpdateTime, because we want it to be zero
	err := sto.SetByUint64(defaultAggregatorAddressOffset, common.BytesToHash(SequencerAddress.Bytes()))
	if err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(l1BaseFeeEstimateInertiaOffset, InitialL1BaseFeeEstimateInertia); err != nil {
		return err
	}
	return sto.SetUint64ByUint64(l1BaseFeeEstimateOffset, InitialL1BaseFeeEstimate)
}

func OpenL1PricingState(sto *storage.Storage) *L1PricingState {
	return &L1PricingState{
		sto,
		sto.OpenStorageBackedAddress(defaultAggregatorAddressOffset),
		sto.OpenStorageBackedBigInt(l1BaseFeeEstimateOffset),
		sto.OpenStorageBackedUint64(l1BaseFeeEstimateInertiaOffset),
		sto.OpenStorageBackedUint64(lastL1BaseFeeUpdateTimeOffset),
		sto.OpenSubStorage(userSpecifiedAggregatorKey),
		sto.OpenSubStorage(refuseDefaultAggregatorKey),
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

func (ps *L1PricingState) L1BaseFeeEstimateWei() (*big.Int, error) {
	return ps.l1BaseFeeEstimate.Get()
}

func (ps *L1PricingState) SetL1BaseFeeEstimateWei(val *big.Int) error {
	return ps.l1BaseFeeEstimate.Set(val)
}

func (ps *L1PricingState) LastL1BaseFeeUpdateTime() (uint64, error) {
	return ps.lastL1BaseFeeUpdateTime.Get()
}

func (ps *L1PricingState) SetLastL1BaseFeeUpdateTime(t uint64) error {
	return ps.lastL1BaseFeeUpdateTime.Set(t)
}

// Update the pricing model with a finalized block's header
func (ps *L1PricingState) UpdatePricingModel(baseFeeSample *big.Int, currentTime uint64) {

	if baseFeeSample.Sign() == 0 {
		// The sequencer's normal messages do not include the l1 basefee, so ignore them
		return
	}

	// update the l1 basefee estimate, which is the weighted average of the past and present
	//     basefee' = weighted average of the historical rate and the current, discounting time passed
	//     basefee' = (memory * basefee + sqrt(passed) * sample) / (memory + sqrt(passed))
	//
	baseFee, _ := ps.L1BaseFeeEstimateWei()
	inertia, _ := ps.L1BaseFeeEstimateInertia()
	lastTime, _ := ps.LastL1BaseFeeUpdateTime()
	if currentTime <= lastTime {
		return
	}
	passedSqrt := arbmath.ApproxSquareRoot(currentTime - lastTime)
	newBaseFee := arbmath.BigDivByUint(
		arbmath.BigAdd(arbmath.BigMulByUint(baseFee, inertia), arbmath.BigMulByUint(baseFeeSample, passedSqrt)),
		inertia+passedSqrt,
	)

	_ = ps.SetL1BaseFeeEstimateWei(newBaseFee)
	_ = ps.SetLastL1BaseFeeUpdateTime(currentTime)
}

// Get how slowly ArbOS updates its estimate of the L1 basefee
func (ps *L1PricingState) L1BaseFeeEstimateInertia() (uint64, error) {
	return ps.l1BaseFeeEstimateInertia.Get()
}

// Set how slowly ArbOS updates its estimate of the L1 basefee
func (ps *L1PricingState) SetL1BaseFeeEstimateInertia(inertia uint64) error {
	return ps.l1BaseFeeEstimateInertia.Set(inertia)
}

func (ps *L1PricingState) userSpecifiedAggregatorsForAddress(sender common.Address) *addressSet.AddressSet {
	return addressSet.OpenAddressSet(ps.userSpecifiedAggregators.OpenSubStorage(sender.Bytes()))
}

// Get sender's user-specified aggregator, or nil if there is none. This does NOT fall back to the default aggregator
//     if there is no user-specified aggregator. If that is what you want, call ReimbursableAggregatorForSender instead.
func (ps *L1PricingState) UserSpecifiedAggregator(sender common.Address) (*common.Address, error) {
	return ps.userSpecifiedAggregatorsForAddress(sender).GetAnyMember()
}

func (ps *L1PricingState) SetUserSpecifiedAggregator(sender common.Address, maybeAggregator *common.Address) error {
	paSet := ps.userSpecifiedAggregatorsForAddress(sender)
	if err := paSet.Clear(); err != nil {
		return err
	}
	if maybeAggregator == nil {
		return nil
	}
	return paSet.Add(*maybeAggregator)
}

func (ps *L1PricingState) RefusesDefaultAggregator(addr common.Address) (bool, error) {
	val, err := ps.refuseDefaultAggregator.Get(common.BytesToHash(addr.Bytes()))
	if err != nil {
		return false, err
	}
	return val != (common.Hash{}), nil
}

func (ps *L1PricingState) SetRefusesDefaultAggregator(addr common.Address, refuses bool) error {
	val := uint64(0)
	if refuses {
		val = 1
	}
	return ps.refuseDefaultAggregator.Set(common.BytesToHash(addr.Bytes()), common.BigToHash(arbmath.UintToBig(val)))
}

// Get the aggregator who is eligible to be reimbursed for L1 costs of txs from sender, or nil if there is none.
func (ps *L1PricingState) ReimbursableAggregatorForSender(sender common.Address) (*common.Address, error) {
	fromTable, err := ps.UserSpecifiedAggregator(sender)
	if err != nil {
		return nil, err
	}
	if fromTable != nil {
		return fromTable, nil
	}

	refuses, err := ps.RefusesDefaultAggregator(sender)
	if err != nil || refuses {
		return nil, err
	}
	aggregator, err := ps.DefaultAggregator()
	if err != nil {
		return nil, err
	}
	if aggregator == (common.Address{}) {
		return nil, nil
	}
	return &aggregator, nil

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

func (ps *L1PricingState) AggregatorCompressionRatio(aggregator common.Address) (arbmath.Bips, error) {
	raw, err := ps.aggregatorCompressionRatios.Get(common.BytesToHash(aggregator.Bytes()))
	if raw == (common.Hash{}) {
		return arbmath.OneInBips, err
	} else {
		return arbmath.BigToBips(raw.Big()), err
	}
}

func (ps *L1PricingState) SetAggregatorCompressionRatio(aggregator common.Address, ratio arbmath.Bips) error {
	if ratio > arbmath.PercentToBips(200) {
		return errors.New("compression ratio out of bounds")
	}
	return ps.aggregatorCompressionRatios.Set(util.AddressToHash(aggregator), util.UintToHash(uint64(ratio)))
}

func (ps *L1PricingState) AddPosterInfo(tx *types.Transaction, sender, poster common.Address) {

	tx.PosterCost = big.NewInt(0)
	tx.PosterIsReimbursable = false

	aggregator, perr := ps.ReimbursableAggregatorForSender(sender)
	txBytes, merr := tx.MarshalBinary()
	txType := tx.Type()
	if !util.TxTypeHasPosterCosts(txType) || perr != nil || merr != nil || aggregator == nil || poster != *aggregator {
		return
	}

	l1Bytes, err := byteCountAfterBrotli0(txBytes)
	if err != nil {
		log.Error("failed to compress tx", "err", err)
		return
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	l1GasPrice, _ := ps.L1BaseFeeEstimateWei()
	l1BytePrice := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	l1Fee := arbmath.BigMulByUint(l1BytePrice, uint64(l1Bytes))

	// Adjust the price paid by the aggregator's reported improvements due to batching
	ratio, _ := ps.AggregatorCompressionRatio(poster)
	adjustedL1Fee := arbmath.BigMulByBips(l1Fee, ratio)

	tx.PosterIsReimbursable = true
	tx.PosterCost = adjustedL1Fee
}

const TxFixedCost = 140 // assumed maximum size in bytes of a typical RLP-encoded tx, not including its calldata

func (ps *L1PricingState) PosterDataCost(message core.Message, sender, poster common.Address) (*big.Int, bool) {

	if tx := message.UnderlyingTransaction(); tx != nil {
		if tx.PosterCost == nil {
			ps.AddPosterInfo(tx, sender, poster)
		}
		return tx.PosterCost, tx.PosterIsReimbursable
	}

	if message.RunMode() == types.MessageGasEstimationMode {
		// assume for the purposes of gas estimation that the poster will be the user's preferred aggregator
		aggregator, _ := ps.ReimbursableAggregatorForSender(sender)
		if aggregator != nil {
			poster = *aggregator
		} else {
			// assume the user will use the delayed inbox since there's no reimbursable party
			return big.NewInt(0), false
		}
	}

	byteCount, err := byteCountAfterBrotli0(message.Data())
	if err != nil {
		log.Error("failed to compress tx", "err", err)
		return big.NewInt(0), false
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	l1Bytes := byteCount + TxFixedCost
	l1GasPrice, _ := ps.L1BaseFeeEstimateWei()
	l1BytePrice := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	l1Fee := arbmath.BigMulByUint(l1BytePrice, uint64(l1Bytes))

	// Adjust the price paid by the aggregator's reported improvements due to batching
	ratio, _ := ps.AggregatorCompressionRatio(poster)
	return arbmath.BigMulByBips(l1Fee, ratio), true
}

func byteCountAfterBrotli0(input []byte) (uint64, error) {
	compressed, err := arbcompress.CompressFast(input)
	if err != nil {
		return 0, err
	}
	return uint64(len(compressed)), nil
}
