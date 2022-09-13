// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/util/arbmath"
	am "github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
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
	unitsSinceUpdate     storage.StorageBackedUint64 // calldata units collected for since last update
	pricePerUnit         storage.StorageBackedBigInt // current price per calldata unit
	lastSurplus          storage.StorageBackedBigInt // introduced in ArbOS version 2
	perBatchGasCost      storage.StorageBackedInt64  // introduced in ArbOS version 3
	amortizedCostCapBips storage.StorageBackedUint64 // in basis points; introduced in ArbOS version 3
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
	lastSurplusOffset
	perBatchGasCostOffset
	amortizedCostCapBipsOffset
)

const (
	InitialInertia           = 10
	InitialPerUnitReward     = 10
	InitialPricePerUnitWei   = 50 * params.GWei
	InitialPerBatchGasCostV6 = 100000
)

// one minute at 100000 bytes / sec
var InitialEquilibrationUnitsV0 = arbmath.UintToBig(60 * params.TxDataNonZeroGasEIP2028 * 100000)
var InitialEquilibrationUnitsV6 = arbmath.UintToBig(params.TxDataNonZeroGasEIP2028 * 10000000)

func InitializeL1PricingState(sto *storage.Storage, initialRewardsRecipient common.Address) error {
	bptStorage := sto.OpenSubStorage(BatchPosterTableKey)
	if err := InitializeBatchPostersTable(bptStorage); err != nil {
		return err
	}
	bpTable := OpenBatchPostersTable(bptStorage)
	if _, err := bpTable.AddPoster(BatchPosterAddress, BatchPosterPayToAddress); err != nil {
		return err
	}
	if err := sto.SetByUint64(payRewardsToOffset, util.AddressToHash(initialRewardsRecipient)); err != nil {
		return err
	}
	equilibrationUnits := sto.OpenStorageBackedBigInt(equilibrationUnitsOffset)
	if err := equilibrationUnits.Set(InitialEquilibrationUnitsV0); err != nil {
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
	if err := pricePerUnit.SetByUint(InitialPricePerUnitWei); err != nil {
		return err
	}
	return nil
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
		sto.OpenStorageBackedBigInt(lastSurplusOffset),
		sto.OpenStorageBackedInt64(perBatchGasCostOffset),
		sto.OpenStorageBackedUint64(amortizedCostCapBipsOffset),
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

func (ps *L1PricingState) LastSurplus() (*big.Int, error) {
	return ps.lastSurplus.Get()
}

func (ps *L1PricingState) SetLastSurplus(val *big.Int, arbosVersion uint64) error {
	if arbosVersion < 7 {
		return ps.lastSurplus.Set_preVersion7(val)
	}
	return ps.lastSurplus.Set(val)
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

func (ps *L1PricingState) PerBatchGasCost() (int64, error) {
	return ps.perBatchGasCost.Get()
}

func (ps *L1PricingState) SetPerBatchGasCost(cost int64) error {
	return ps.perBatchGasCost.Set(cost)
}

func (ps *L1PricingState) AmortizedCostCapBips() (uint64, error) {
	return ps.amortizedCostCapBips.Get()
}

func (ps *L1PricingState) SetAmortizedCostCapBips(cap uint64) error {
	return ps.amortizedCostCapBips.Set(cap)
}

// Update the pricing model based on a payment by a batch poster
func (ps *L1PricingState) UpdateForBatchPosterSpending(
	statedb vm.StateDB,
	evm *vm.EVM,
	arbosVersion uint64,
	updateTime, currentTime uint64,
	batchPoster common.Address,
	weiSpent *big.Int,
	l1Basefee *big.Int,
	scenario util.TracingScenario,
) error {
	if arbosVersion < 2 {
		return ps._preVersion2_UpdateForBatchPosterSpending(statedb, evm, updateTime, currentTime, batchPoster, weiSpent, scenario)
	}

	batchPosterTable := ps.BatchPosterTable()
	posterState, err := batchPosterTable.OpenPoster(batchPoster, true)
	if err != nil {
		return err
	}

	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return err
	}

	// compute allocation fraction -- will allocate updateTimeDelta/timeDelta fraction of units and funds to this update
	lastUpdateTime, err := ps.LastUpdateTime()
	if err != nil {
		return err
	}
	if lastUpdateTime == 0 && updateTime > 0 { // it's the first update, so there isn't a last update time
		lastUpdateTime = updateTime - 1
	}
	if updateTime > currentTime || updateTime < lastUpdateTime {
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
	unitsAllocated := am.SaturatingUMul(unitsSinceUpdate, allocationNumerator) / allocationDenominator
	unitsSinceUpdate -= unitsAllocated
	if err := ps.SetUnitsSinceUpdate(unitsSinceUpdate); err != nil {
		return err
	}

	// impose cap on amortized cost, if there is one
	if arbosVersion >= 3 {
		amortizedCostCapBips, err := ps.AmortizedCostCapBips()
		if err != nil {
			return err
		}
		if amortizedCostCapBips != 0 {
			weiSpentCap := am.BigMulByBips(
				am.BigMulByUint(l1Basefee, unitsAllocated),
				am.SaturatingCastToBips(amortizedCostCapBips),
			)
			if am.BigLessThan(weiSpentCap, weiSpent) {
				// apply the cap on assignment of amortized cost;
				// the difference will be a loss for the batch poster
				weiSpent = weiSpentCap
			}
		}
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

	// pay rewards, as much as possible
	paymentForRewards := am.BigMulByUint(am.UintToBig(perUnitReward), unitsAllocated)
	availableFunds := statedb.GetBalance(L1PricerFundsPoolAddress)
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
	err = util.TransferBalance(
		&L1PricerFundsPoolAddress, &payRewardsTo, paymentForRewards, evm, scenario, "batchPosterReward",
	)
	if err != nil {
		return err
	}
	availableFunds = statedb.GetBalance(L1PricerFundsPoolAddress)

	// settle up payments owed to the batch poster, as much as possible
	balanceDueToPoster, err := posterState.FundsDue()
	if err != nil {
		return err
	}
	balanceToTransfer := balanceDueToPoster
	if am.BigLessThan(availableFunds, balanceToTransfer) {
		balanceToTransfer = availableFunds
	}
	if balanceToTransfer.Sign() > 0 {
		addrToPay, err := posterState.PayTo()
		if err != nil {
			return err
		}
		err = util.TransferBalance(
			&L1PricerFundsPoolAddress, &addrToPay, balanceToTransfer, evm, scenario, "batchPosterRefund",
		)
		if err != nil {
			return err
		}
		balanceDueToPoster = am.BigSub(balanceDueToPoster, balanceToTransfer)
		err = posterState.SetFundsDue(balanceDueToPoster)
		if err != nil {
			return err
		}
	}

	// update time
	if err := ps.SetLastUpdateTime(updateTime); err != nil {
		return err
	}

	// adjust the price
	if unitsAllocated > 0 {
		totalFundsDue, err := batchPosterTable.TotalFundsDue()
		if err != nil {
			return err
		}
		fundsDueForRewards, err = ps.FundsDueForRewards()
		if err != nil {
			return err
		}
		surplus := am.BigSub(statedb.GetBalance(L1PricerFundsPoolAddress), am.BigAdd(totalFundsDue, fundsDueForRewards))

		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilUnits, err := ps.EquilibrationUnits()
		if err != nil {
			return err
		}
		inertiaUnits := am.BigDivByUint(equilUnits, inertia)
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}

		allocPlusInert := am.BigAddByUint(inertiaUnits, unitsAllocated)
		oldSurplus, err := ps.LastSurplus()
		if err != nil {
			return err
		}

		desiredDerivative := am.BigDiv(new(big.Int).Neg(surplus), equilUnits)
		actualDerivative := am.BigDivByUint(am.BigSub(surplus, oldSurplus), unitsAllocated)
		changeDerivativeBy := am.BigSub(desiredDerivative, actualDerivative)
		priceChange := am.BigDiv(am.BigMulByUint(changeDerivativeBy, unitsAllocated), allocPlusInert)

		if err := ps.SetLastSurplus(surplus, arbosVersion); err != nil {
			return err
		}
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

func (ps *L1PricingState) _preVersion2_UpdateForBatchPosterSpending(
	statedb vm.StateDB,
	evm *vm.EVM,
	updateTime, currentTime uint64,
	batchPoster common.Address,
	weiSpent *big.Int,
	scenario util.TracingScenario,
) error {
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
	if updateTime >= currentTime || updateTime < lastUpdateTime {
		return nil // historically this returned an error
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
	err = util.TransferBalance(
		&L1PricerFundsPoolAddress, &payRewardsTo, paymentForRewards, evm, scenario, "batchPosterReward",
	)
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
			err = util.TransferBalance(
				&L1PricerFundsPoolAddress, &addrToPay, balanceToTransfer, evm, scenario, "batchPosterRefund",
			)
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

		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilUnits, err := ps.EquilibrationUnits()
		if err != nil {
			return err
		}
		inertiaUnits := am.BigDivByUint(equilUnits, inertia)
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}

		allocPlusInert := am.BigAddByUint(inertiaUnits, unitsAllocated)
		priceChange := am.BigDiv(
			am.BigSub(
				am.BigMul(surplus, am.BigSub(equilUnits, common.Big1)),
				am.BigMul(oldSurplus, equilUnits),
			),
			am.BigMul(equilUnits, allocPlusInert),
		)

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

func (ps *L1PricingState) getPosterUnitsWithoutCache(tx *types.Transaction, posterAddr common.Address) uint64 {

	if posterAddr != BatchPosterAddress {
		return 0
	}
	txBytes, merr := tx.MarshalBinary()
	txType := tx.Type()
	if !util.TxTypeHasPosterCosts(txType) || merr != nil {
		return 0
	}

	l1Bytes, err := byteCountAfterBrotli0(txBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to compress tx: %v", err))
	}
	return l1Bytes * params.TxDataNonZeroGasEIP2028
}

// Returns the poster cost and the calldata units for a transaction
func (ps *L1PricingState) GetPosterInfo(tx *types.Transaction, poster common.Address) (*big.Int, uint64) {
	if poster != BatchPosterAddress {
		return common.Big0, 0
	}
	units := atomic.LoadUint64(&tx.CalldataUnits)
	if units == 0 {
		units = ps.getPosterUnitsWithoutCache(tx, poster)
		atomic.StoreUint64(&tx.CalldataUnits, units)
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	pricePerUnit, _ := ps.PricePerUnit()
	return am.BigMulByUint(pricePerUnit, units), units
}

// We don't have the full tx in gas estimation, so we assume it might be a bit bigger in practice.
const estimationPaddingUnits = 16 * params.TxDataNonZeroGasEIP2028
const estimationPaddingBasisPoints = 100

var randomNonce = binary.BigEndian.Uint64(crypto.Keccak256([]byte("Nonce"))[:8])
var randomGasTipCap = new(big.Int).SetBytes(crypto.Keccak256([]byte("GasTipCap"))[:4])
var randomGasFeeCap = new(big.Int).SetBytes(crypto.Keccak256([]byte("GasFeeCap"))[:4])
var RandomGas = uint64(binary.BigEndian.Uint32(crypto.Keccak256([]byte("Gas"))[:4]))
var randV = arbmath.BigMulByUint(params.ArbitrumOneChainConfig().ChainID, 3)
var randR = crypto.Keccak256Hash([]byte("R")).Big()
var randS = crypto.Keccak256Hash([]byte("S")).Big()

// The returned tx will be invalid, likely for a number of reasons such as an invalid signature.
// It's only used to check how large it is after brotli level 0 compression.
func makeFakeTxForMessage(message core.Message) *types.Transaction {
	nonce := message.Nonce()
	if nonce == 0 {
		nonce = randomNonce
	}
	gasTipCap := message.GasTipCap()
	if gasTipCap.Sign() == 0 {
		gasTipCap = randomGasTipCap
	}
	gasFeeCap := message.GasFeeCap()
	if gasFeeCap.Sign() == 0 {
		gasFeeCap = randomGasFeeCap
	}
	gas := message.Gas()
	if gas == 0 {
		gas = RandomGas
	}
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:      nonce,
		GasTipCap:  gasTipCap,
		GasFeeCap:  gasFeeCap,
		Gas:        gas,
		To:         message.To(),
		Value:      message.Value(),
		Data:       message.Data(),
		AccessList: message.AccessList(),
		V:          randV,
		R:          randR,
		S:          randS,
	})
}

func (ps *L1PricingState) PosterDataCost(message core.Message, poster common.Address) (*big.Int, uint64) {
	tx := message.UnderlyingTransaction()
	if tx != nil {
		return ps.GetPosterInfo(tx, poster)
	}

	// Otherwise, we don't have an underlying transaction, so we're likely in gas estimation.
	// We'll instead make a fake tx from the message info we do have, and then pad our cost a bit to be safe.
	tx = makeFakeTxForMessage(message)
	units := ps.getPosterUnitsWithoutCache(tx, poster)
	units = arbmath.UintMulByBips(units+estimationPaddingUnits, arbmath.OneInBips+estimationPaddingBasisPoints)
	pricePerUnit, _ := ps.PricePerUnit()
	return am.BigMulByUint(pricePerUnit, units), units
}

func byteCountAfterBrotli0(input []byte) (uint64, error) {
	compressed, err := arbcompress.CompressFast(input)
	if err != nil {
		return 0, err
	}
	return uint64(len(compressed)), nil
}
