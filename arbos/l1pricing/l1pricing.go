// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	am "github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

type L1PricingState struct {
	storage *storage.Storage
	// parameters
	sequencer          storage.StorageBackedAddress
	paySequencerFeesTo storage.StorageBackedAddress
	payRewardsTo       storage.StorageBackedAddress
	equilibrationTime  storage.StorageBackedUint64
	inertia            storage.StorageBackedUint64
	perUnitReward      storage.StorageBackedUint64
	// variables
	currentTime         storage.StorageBackedUint64
	lastUpdateTime      storage.StorageBackedUint64
	availableFunds      storage.StorageBackedBigInt
	fundsDueToSequencer storage.StorageBackedBigInt
	fundsDueForRewards  storage.StorageBackedBigInt
	// funds collected since update are recorded as the balance in account L1PricerFundsPoolAddress
	unitsSinceUpdate storage.StorageBackedUint64
	pricePerUnit     storage.StorageBackedBigInt
}

var (
	SequencerAddress         = common.HexToAddress("0xA4B000000000000000000073657175656e636572")
	L1PricerFundsPoolAddress = common.HexToAddress("0xA4B0000000000000000000000000000000000f6")

	ErrInvalidTime = errors.New("invalid timestamp")
)

const (
	sequencerOffset uint64 = iota
	paySequencerFeesToOffset
	payRewardsToOffset
	equilibrationTimeOffset
	inertiaOffset
	perUnitRewardOffset
	currentTimeOffset
	lastUpdateTimeOffset
	availableFundsOffset
	fundsDueToSequencerOffset
	fundsDueForRewards
	unitsSinceOffset
	pricePerUnitOffset
)

const (
	InitialEquilibrationTime = 10000000
	InitialInertia           = 10
	InitialPerUnitReward     = 10
	InitialPricePerUnitGwei  = 50
)

func InitializeL1PricingState(sto *storage.Storage) error {
	if err := sto.SetByUint64(sequencerOffset, util.AddressToHash(SequencerAddress)); err != nil {
		return err
	}
	if err := sto.SetByUint64(paySequencerFeesToOffset, util.AddressToHash(SequencerAddress)); err != nil {
		return err
	}
	if err := sto.SetByUint64(payRewardsToOffset, util.AddressToHash(SequencerAddress)); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(equilibrationTimeOffset, InitialEquilibrationTime); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(inertiaOffset, InitialInertia); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(perUnitRewardOffset, InitialPerUnitReward); err != nil {
		return err
	}
	pricePerUnit := sto.OpenStorageBackedBigInt(pricePerUnitOffset)
	return pricePerUnit.SetByUint(InitialPricePerUnitGwei * 1000000000)
}

func OpenL1PricingState(sto *storage.Storage) *L1PricingState {
	return &L1PricingState{
		sto,
		sto.OpenStorageBackedAddress(sequencerOffset),
		sto.OpenStorageBackedAddress(paySequencerFeesToOffset),
		sto.OpenStorageBackedAddress(payRewardsToOffset),
		sto.OpenStorageBackedUint64(equilibrationTimeOffset),
		sto.OpenStorageBackedUint64(inertiaOffset),
		sto.OpenStorageBackedUint64(perUnitRewardOffset),
		sto.OpenStorageBackedUint64(currentTimeOffset),
		sto.OpenStorageBackedUint64(lastUpdateTimeOffset),
		sto.OpenStorageBackedBigInt(availableFundsOffset),
		sto.OpenStorageBackedBigInt(fundsDueToSequencerOffset),
		sto.OpenStorageBackedBigInt(fundsDueForRewards),
		sto.OpenStorageBackedUint64(unitsSinceOffset),
		sto.OpenStorageBackedBigInt(pricePerUnitOffset),
	}
}

func (ps *L1PricingState) Sequencer() (common.Address, error) {
	return ps.sequencer.Get()
}

func (ps *L1PricingState) SetSequencer(seq common.Address) error {
	return ps.sequencer.Set(seq)
}

func (ps *L1PricingState) PaySequencerFeesTo() (common.Address, error) {
	return ps.paySequencerFeesTo.Get()
}

func (ps *L1PricingState) SetPaySequencerFeesTo(addr common.Address) error {
	return ps.paySequencerFeesTo.Set(addr)
}

func (ps *L1PricingState) PayRewardsTo() (common.Address, error) {
	return ps.payRewardsTo.Get()
}

func (ps *L1PricingState) EquilibrationTime() (uint64, error) {
	return ps.equilibrationTime.Get()
}

func (ps *L1PricingState) Inertia() (uint64, error) {
	return ps.inertia.Get()
}

func (ps *L1PricingState) SetInertia(inertia uint64) error {
	return ps.inertia.Set(inertia)
}

func (ps *L1PricingState) PerUnitReward() (uint64, error) {
	return ps.perUnitReward.Get()
}

func (ps *L1PricingState) CurrentTime() (uint64, error) {
	return ps.currentTime.Get()
}

func (ps *L1PricingState) SetCurrentTime(t uint64) error {
	return ps.currentTime.Set(t)
}

func (ps *L1PricingState) LastUpdateTime() (uint64, error) {
	return ps.lastUpdateTime.Get()
}

func (ps *L1PricingState) SetLastUpdateTime(t uint64) error {
	return ps.lastUpdateTime.Set(t)
}

func (ps *L1PricingState) AvailableFunds() (*big.Int, error) {
	return ps.availableFunds.Get()
}

func (ps *L1PricingState) SetAvailableFunds(amt *big.Int) error {
	return ps.availableFunds.Set(amt)
}

func (ps *L1PricingState) FundsDueToSequencer() (*big.Int, error) {
	return ps.fundsDueToSequencer.Get()
}

func (ps *L1PricingState) SetFundsDueToSequencer(amt *big.Int) error {
	return ps.fundsDueToSequencer.Set(amt)
}

func (ps *L1PricingState) FundsDueForRewards() (*big.Int, error) {
	return ps.fundsDueForRewards.Get()
}

func (ps *L1PricingState) SetFundsDueForRewards(amt *big.Int) error {
	return ps.fundsDueForRewards.Set(amt)
}

func (ps *L1PricingState) UnitsSinceUpdate() (uint64, error) {
	return ps.unitsSinceUpdate.Get()
}

func (ps *L1PricingState) SetUnitsSinceUpdate(units uint64) error {
	return ps.unitsSinceUpdate.Set(units)
}

func (ps *L1PricingState) PricePerUnit() (*big.Int, error) {
	return ps.pricePerUnit.Get()
}

func (ps *L1PricingState) SetPricePerUnit(price *big.Int) error {
	return ps.pricePerUnit.Set(price)
}

// Update the pricing model with info from the start of a block
func (ps *L1PricingState) UpdateTime(currentTime uint64) {
	_ = ps.SetCurrentTime(currentTime)
}

// Update the pricing model based on a payment by the sequencer
func (ps *L1PricingState) UpdateForSequencerSpending(statedb vm.StateDB, updateTime uint64, currentTime uint64, weiSpent *big.Int) error {
	// compute previous shortfall
	fundsDueToSequencer, err := ps.FundsDueToSequencer()
	if err != nil {
		return err
	}
	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return err
	}
	availableFunds, err := ps.AvailableFunds()
	if err != nil {
		return err
	}
	oldShortfall := am.BigSub(am.BigAdd(fundsDueToSequencer, fundsDueForRewards), availableFunds)

	// compute allocation fraction
	lastUpdateTime, err := ps.LastUpdateTime()
	if err != nil {
		return err
	}
	if updateTime > currentTime || updateTime < lastUpdateTime || currentTime == lastUpdateTime {
		return ErrInvalidTime
	}
	updateTimeDelta := updateTime - lastUpdateTime
	timeDelta := currentTime - lastUpdateTime

	// allocate units to this update
	unitsSinceUpdate, err := ps.UnitsSinceUpdate()
	if err != nil {
		return err
	}
	unitsAllocated := unitsSinceUpdate * (updateTime - lastUpdateTime) / (currentTime - lastUpdateTime)
	unitsSinceUpdate -= unitsAllocated
	if err := ps.SetUnitsSinceUpdate(unitsSinceUpdate); err != nil {
		return err
	}

	// allocate funds to this update
	collectedSinceUpdate := statedb.GetBalance(L1PricerFundsPoolAddress)
	fundsToMove := am.BigDivByUint(am.BigMulByUint(collectedSinceUpdate, updateTimeDelta), timeDelta)
	statedb.SubBalance(L1PricerFundsPoolAddress, fundsToMove)
	availableFunds = am.BigAdd(availableFunds, fundsToMove)

	// update amounts due
	perUnitReward, err := ps.PerUnitReward()
	if err != nil {
		return err
	}
	fundsDueToSequencer = am.BigAdd(fundsDueToSequencer, weiSpent)
	if err := ps.SetFundsDueToSequencer(fundsDueToSequencer); err != nil {
		return err
	}
	newRewards := am.SaturatingUMul(updateTimeDelta, perUnitReward) / timeDelta
	fundsDueForRewards = am.BigAddByUint(fundsDueForRewards, newRewards)
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}

	// settle up, by paying out available funds
	payRewardsTo, err := ps.PayRewardsTo()
	if err != nil {
		return err
	}
	paymentForRewards := availableFunds
	if am.BigLessThan(fundsDueForRewards, paymentForRewards) {
		paymentForRewards = fundsDueForRewards
	}
	if paymentForRewards.Sign() > 0 {
		availableFunds = am.BigSub(availableFunds, paymentForRewards)
		if err := ps.SetAvailableFunds(availableFunds); err != nil {
			return err
		}
		fundsDueForRewards = am.BigSub(fundsDueForRewards, paymentForRewards)
		if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
			return err
		}
		core.Transfer(statedb, L1PricerFundsPoolAddress, payRewardsTo, paymentForRewards)
	}
	sequencerPaymentAddr, err := ps.Sequencer()
	if err != nil {
		return err
	}
	paymentForSequencer := availableFunds
	if am.BigLessThan(fundsDueToSequencer, paymentForSequencer) {
		paymentForSequencer = fundsDueToSequencer
	}
	if paymentForSequencer.Sign() > 0 {
		availableFunds = new(big.Int).Sub(availableFunds, paymentForSequencer)
		if err := ps.SetAvailableFunds(availableFunds); err != nil {
			return err
		}
		fundsDueToSequencer = new(big.Int).Sub(fundsDueToSequencer, paymentForSequencer)
		if err := ps.SetFundsDueToSequencer(fundsDueToSequencer); err != nil {
			return err
		}
		core.Transfer(statedb, L1PricerFundsPoolAddress, sequencerPaymentAddr, paymentForSequencer)
	}

	// update time
	if err := ps.SetLastUpdateTime(updateTime); err != nil {
		return err
	}

	// adjust the price
	if unitsAllocated > 0 {
		shortfall := am.BigSub(am.BigAdd(fundsDueToSequencer, fundsDueForRewards), availableFunds)
		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilTime, err := ps.EquilibrationTime()
		if err != nil {
			return err
		}
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}

		priceChange := am.BigDivByUint(
			am.BigAdd(am.BigSub(shortfall, oldShortfall), am.BigDivByUint(shortfall, equilTime)),
			unitsAllocated+equilTime/inertia,
		)

		newPrice := am.BigSub(price, priceChange)
		if newPrice.Sign() >= 0 {
			price = newPrice
		} else {
			price = big.NewInt(0)
		}
		if err := ps.SetPricePerUnit(price); err != nil {
			return err
		}
	}
	return nil
}

func (ps *L1PricingState) AddPosterInfo(tx *types.Transaction, poster common.Address) {
	tx.PosterCost = big.NewInt(0)
	tx.PosterIsReimbursable = false

	sequencer, perr := ps.Sequencer()
	txBytes, merr := tx.MarshalBinary()
	txType := tx.Type()
	if !util.TxTypeHasPosterCosts(txType) || perr != nil || merr != nil || poster != sequencer {
		return
	}

	l1Bytes, err := byteCountAfterBrotli0(txBytes)
	if err != nil {
		log.Error("failed to compress tx", "err", err)
		return
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	pricePerUnit, _ := ps.PricePerUnit()
	tx.PosterCost = am.BigMulByUint(pricePerUnit, l1Bytes*params.TxDataNonZeroGasEIP2028)
	tx.PosterIsReimbursable = true
}

const TxFixedCost = 140 // assumed maximum size in bytes of a typical RLP-encoded tx, not including its calldata

func (ps *L1PricingState) PosterDataCost(message core.Message, poster common.Address) (*big.Int, bool) {
	if tx := message.UnderlyingTransaction(); tx != nil {
		if tx.PosterCost == nil {
			ps.AddPosterInfo(tx, poster)
		}
		return tx.PosterCost, tx.PosterIsReimbursable
	}

	byteCount, err := byteCountAfterBrotli0(message.Data())
	if err != nil {
		log.Error("failed to compress tx", "err", err)
		return big.NewInt(0), false
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	l1Bytes := byteCount + TxFixedCost
	pricePerUnit, _ := ps.PricePerUnit()

	return am.BigMulByUint(pricePerUnit, l1Bytes*params.TxDataNonZeroGasEIP2028), true
}

func byteCountAfterBrotli0(input []byte) (uint64, error) {
	compressed, err := arbcompress.CompressFast(input)
	if err != nil {
		return 0, err
	}
	return uint64(len(compressed)), nil
}
