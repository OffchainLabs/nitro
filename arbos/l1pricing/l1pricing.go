// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/math"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/util/arbmath"
	am "github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"

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
	batchPosterTable   *BatchPostersTable
	payRewardsTo       storage.StorageBackedAddress
	equilibrationUnits storage.StorageBackedBigInt
	inertia            storage.StorageBackedUint64
	perUnitReward      storage.StorageBackedUint64
	// variables
	lastUpdateTime     storage.StorageBackedUint64 // timestamp of the last update from L1 that we processed
	fundsDueForRewards storage.StorageBackedBigInt
	// funds collected since update are recorded as the balance in account L1PricerFundsPoolAddress
	unitsSinceUpdate storage.StorageBackedUint64 // calldata units collected for since last update
	pricePerUnit     storage.StorageBackedBigInt // current price per calldata unit
}

var (
	BatchPosterTableKey      = []byte{0}
	BatchPosterAddress       = common.HexToAddress("0xA4B000000000000000000073657175656e636572")
	BatchPosterPayToAddress  = BatchPosterAddress
	L1PricerFundsPoolAddress = common.HexToAddress("0xA4B00000000000000000000000000000000000f6")

	ErrInvalidTime = errors.New("invalid timestamp")
)

const (
	payRewardsToOffset uint64 = iota
	equilibrationUnitsOffset
	inertiaOffset
	perUnitRewardOffset
	lastUpdateTimeOffset
	fundsDueForRewardsOffset
	unitsSinceOffset
	pricePerUnitOffset
)

const (
	InitialEquilibrationUnits uint64 = 60 * params.TxDataNonZeroGasEIP2028 * 100000 // one minute at 100000 bytes / sec
	InitialInertia                   = 10
	InitialPerUnitReward             = 10
	InitialPricePerUnitWei           = 50 * params.GWei
)

func InitializeL1PricingState(sto *storage.Storage) error {
	bptStorage := sto.OpenSubStorage(BatchPosterTableKey)
	if err := InitializeBatchPostersTable(bptStorage); err != nil {
		return err
	}
	bpTable := OpenBatchPostersTable(bptStorage)
	if _, err := bpTable.AddPoster(BatchPosterAddress, BatchPosterPayToAddress); err != nil {
		return err
	}
	if err := sto.SetByUint64(payRewardsToOffset, util.AddressToHash(BatchPosterAddress)); err != nil {
		return err
	}
	equilibrationUnits := sto.OpenStorageBackedBigInt(equilibrationUnitsOffset)
	if err := equilibrationUnits.Set(am.UintToBig(InitialEquilibrationUnits)); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(inertiaOffset, InitialInertia); err != nil {
		return err
	}
	fundsDueForRewards := sto.OpenStorageBackedBigInt(fundsDueForRewardsOffset)
	if err := fundsDueForRewards.Set(common.Big0); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(perUnitRewardOffset, InitialPerUnitReward); err != nil {
		return err
	}
	pricePerUnit := sto.OpenStorageBackedBigInt(pricePerUnitOffset)
	return pricePerUnit.SetByUint(InitialPricePerUnitWei)
}

func OpenL1PricingState(sto *storage.Storage) *L1PricingState {
	return &L1PricingState{
		sto,
		OpenBatchPostersTable(sto.OpenSubStorage(BatchPosterTableKey)),
		sto.OpenStorageBackedAddress(payRewardsToOffset),
		sto.OpenStorageBackedBigInt(equilibrationUnitsOffset),
		sto.OpenStorageBackedUint64(inertiaOffset),
		sto.OpenStorageBackedUint64(perUnitRewardOffset),
		sto.OpenStorageBackedUint64(lastUpdateTimeOffset),
		sto.OpenStorageBackedBigInt(fundsDueForRewardsOffset),
		sto.OpenStorageBackedUint64(unitsSinceOffset),
		sto.OpenStorageBackedBigInt(pricePerUnitOffset),
	}
}

func (ps *L1PricingState) BatchPosterTable() *BatchPostersTable {
	return ps.batchPosterTable
}

func (ps *L1PricingState) PayRewardsTo() (common.Address, error) {
	return ps.payRewardsTo.Get()
}

func (ps *L1PricingState) SetPayRewardsTo(addr common.Address) error {
	return ps.payRewardsTo.Set(addr)
}

func (ps *L1PricingState) EquilibrationUnits() (*big.Int, error) {
	return ps.equilibrationUnits.Get()
}

func (ps *L1PricingState) SetEquilibrationUnits(equilUnits *big.Int) error {
	return ps.equilibrationUnits.Set(equilUnits)
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

func (ps *L1PricingState) SetPerUnitReward(weiPerUnit uint64) error {
	return ps.perUnitReward.Set(weiPerUnit)
}

func (ps *L1PricingState) LastUpdateTime() (uint64, error) {
	return ps.lastUpdateTime.Get()
}

func (ps *L1PricingState) SetLastUpdateTime(t uint64) error {
	return ps.lastUpdateTime.Set(t)
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

func (ps *L1PricingState) AddToUnitsSinceUpdate(units uint64) error {
	oldUnits, err := ps.unitsSinceUpdate.Get()
	if err != nil {
		return err
	}
	return ps.unitsSinceUpdate.Set(oldUnits + units)
}

func (ps *L1PricingState) PricePerUnit() (*big.Int, error) {
	return ps.pricePerUnit.Get()
}

func (ps *L1PricingState) SetPricePerUnit(price *big.Int) error {
	return ps.pricePerUnit.Set(price)
}

func (ps *L1PricingState) L1BaseFeeEstimate() (*big.Int, error) {
	perUnit, err := ps.pricePerUnit.Get()
	if err != nil {
		return nil, err
	}
	return arbmath.BigMulByUint(perUnit, 16), nil
}

// Update the pricing model based on a payment by a batch poster
func (ps *L1PricingState) UpdateForBatchPosterSpending(statedb vm.StateDB, evm *vm.EVM, arbosVersion uint64, updateTime uint64, currentTime uint64, batchPoster common.Address, weiSpent *big.Int) error {
	batchPosterTable := ps.BatchPosterTable()
	posterState, err := batchPosterTable.OpenPoster(batchPoster, true)
	if err != nil {
		return err
	}

	// compute previous shortfall
	totalFundsDue, err := batchPosterTable.TotalFundsDue()
	if err != nil {
		return err
	}
	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return err
	}
	oldSurplus := am.BigSub(statedb.GetBalance(L1PricerFundsPoolAddress), am.BigAdd(totalFundsDue, fundsDueForRewards))

	// compute allocation fraction -- will allocate updateTimeDelta/timeDelta fraction of units and funds to this update
	lastUpdateTime, err := ps.LastUpdateTime()
	if err != nil {
		return err
	}
	if lastUpdateTime == 0 && currentTime > 0 { // it's the first update, so there isn't a last update time
		lastUpdateTime = updateTime - 1
	}
	if updateTime > currentTime || (arbosVersion < 2 && updateTime == currentTime) || updateTime < lastUpdateTime {
		return ErrInvalidTime
	}
	allocationNumerator := updateTime - lastUpdateTime
	allocationDenominator := currentTime - lastUpdateTime
	if allocationDenominator == 0 {
		allocationNumerator = 1
		allocationDenominator = 1
	}

	// allocate units to this update
	unitsSinceUpdate, err := ps.UnitsSinceUpdate()
	if err != nil {
		return err
	}
	unitsAllocated := unitsSinceUpdate * allocationNumerator / allocationDenominator
	unitsSinceUpdate -= unitsAllocated
	if err := ps.SetUnitsSinceUpdate(unitsSinceUpdate); err != nil {
		return err
	}

	dueToPoster, err := posterState.FundsDue()
	if err != nil {
		return err
	}
	err = posterState.SetFundsDue(am.BigAdd(dueToPoster, weiSpent))
	if err != nil {
		return err
	}
	perUnitReward, err := ps.PerUnitReward()
	if err != nil {
		return err
	}
	fundsDueForRewards = am.BigAdd(fundsDueForRewards, am.BigMulByUint(am.UintToBig(unitsAllocated), perUnitReward))
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}

	// allocate funds to this update
	collectedSinceUpdate := statedb.GetBalance(L1PricerFundsPoolAddress)
	availableFunds := am.BigDivByUint(am.BigMulByUint(collectedSinceUpdate, allocationNumerator), allocationDenominator)

	// pay rewards, as much as possible
	paymentForRewards := am.BigMulByUint(am.UintToBig(perUnitReward), unitsAllocated)
	if am.BigLessThan(availableFunds, paymentForRewards) {
		paymentForRewards = availableFunds
	}
	fundsDueForRewards = am.BigSub(fundsDueForRewards, paymentForRewards)
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}
	payRewardsTo, err := ps.PayRewardsTo()
	if err != nil {
		return err
	}
	err = util.TransferBalance(&L1PricerFundsPoolAddress, &payRewardsTo, paymentForRewards, evm, util.TracingBeforeEVM, "batchPosterReward")
	if err != nil {
		return err
	}
	availableFunds = am.BigSub(availableFunds, paymentForRewards)

	// settle up our batch poster payments owed, as much as possible
	allPosterAddrs, err := batchPosterTable.AllPosters(math.MaxUint64)
	if err != nil {
		return err
	}
	for _, posterAddr := range allPosterAddrs {
		poster, err := batchPosterTable.OpenPoster(posterAddr, false)
		if err != nil {
			return err
		}
		balanceDueToPoster, err := poster.FundsDue()
		if err != nil {
			return err
		}
		balanceToTransfer := balanceDueToPoster
		if am.BigLessThan(availableFunds, balanceToTransfer) {
			balanceToTransfer = availableFunds
		}
		if balanceToTransfer.Sign() > 0 {
			addrToPay, err := poster.PayTo()
			if err != nil {
				return err
			}
			err = util.TransferBalance(&L1PricerFundsPoolAddress, &addrToPay, balanceToTransfer, evm, util.TracingBeforeEVM, "batchPosterRefund")
			if err != nil {
				return err
			}
			availableFunds = am.BigSub(availableFunds, balanceToTransfer)
			balanceDueToPoster = am.BigSub(balanceDueToPoster, balanceToTransfer)
			err = poster.SetFundsDue(balanceDueToPoster)
			if err != nil {
				return err
			}
		}
	}

	// update time
	if err := ps.SetLastUpdateTime(updateTime); err != nil {
		return err
	}

	// adjust the price
	if unitsAllocated > 0 {
		totalFundsDue, err = batchPosterTable.TotalFundsDue()
		if err != nil {
			return err
		}
		fundsDueForRewards, err = ps.FundsDueForRewards()
		if err != nil {
			return err
		}
		surplus := am.BigSub(statedb.GetBalance(L1PricerFundsPoolAddress), am.BigAdd(totalFundsDue, fundsDueForRewards))
		colors.PrintBlue("Units allocated: ", unitsAllocated)
		colors.PrintMint("Surplus: ", surplus)
		colors.PrintMint("Old surplus: ", oldSurplus)

		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilUnits, err := ps.EquilibrationUnits()
		if err != nil {
			return err
		}
		colors.PrintGrey("Equilibration units: ", equilUnits)
		inertiaUnits := am.BigDivByUint(equilUnits, inertia)
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}
		colors.PrintGrey("Inertia units: ", inertiaUnits)

		allocPlusInert := am.BigAddByUint(inertiaUnits, unitsAllocated)
		priceChange := am.BigDiv(
			am.BigSub(
				am.BigMul(surplus, am.BigSub(equilUnits, common.Big1)),
				am.BigMul(oldSurplus, equilUnits),
			),
			am.BigMul(equilUnits, allocPlusInert),
		)
		colors.PrintBlue("Price change: ", priceChange)

		newPrice := am.BigAdd(price, priceChange)
		if newPrice.Sign() < 0 {
			newPrice = common.Big0
		}
		if err := ps.SetPricePerUnit(newPrice); err != nil {
			return err
		}
	}
	return nil
}

func (ps *L1PricingState) getPosterInfoWithoutCache(tx *types.Transaction, posterAddr common.Address) (*big.Int, uint64) {

	if posterAddr != BatchPosterAddress {
		return common.Big0, 0
	}
	txBytes, merr := tx.MarshalBinary()
	txType := tx.Type()
	if !util.TxTypeHasPosterCosts(txType) || merr != nil {
		return common.Big0, 0
	}

	l1Bytes, err := byteCountAfterBrotli0(txBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to compress tx: %v", err))
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	pricePerUnit, _ := ps.PricePerUnit()
	numUnits := l1Bytes * params.TxDataNonZeroGasEIP2028
	return am.BigMulByUint(pricePerUnit, numUnits), numUnits
}

// Returns the poster cost and the calldata units for a transaction
func (ps *L1PricingState) GetPosterInfo(tx *types.Transaction, poster common.Address) (*big.Int, uint64) {
	cost, _ := tx.PosterCost.Load().(*big.Int)
	if cost != nil {
		return cost, atomic.LoadUint64(&tx.CalldataUnits)
	}
	cost, units := ps.getPosterInfoWithoutCache(tx, poster)
	atomic.StoreUint64(&tx.CalldataUnits, units)
	tx.PosterCost.Store(cost)
	return cost, units
}

const TxFixedCost = 140 // assumed maximum size in bytes of a typical RLP-encoded tx, not including its calldata

func (ps *L1PricingState) PosterDataCost(message core.Message, poster common.Address) (*big.Int, uint64) {
	if tx := message.UnderlyingTransaction(); tx != nil {
		return ps.GetPosterInfo(tx, poster)
	}

	if poster != BatchPosterAddress {
		return common.Big0, 0
	}

	byteCount, err := byteCountAfterBrotli0(message.Data())
	if err != nil {
		log.Error("failed to compress tx", "err", err)
		return common.Big0, 0
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	l1Bytes := byteCount + TxFixedCost
	pricePerUnit, _ := ps.PricePerUnit()

	units := l1Bytes * params.TxDataNonZeroGasEIP2028
	return am.BigMulByUint(pricePerUnit, units), units
}

func byteCountAfterBrotli0(input []byte) (uint64, error) {
	compressed, err := arbcompress.CompressFast(input)
	if err != nil {
		return 0, err
	}
	return uint64(len(compressed)), nil
}
