// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l1pricing

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func (ps *L1PricingState) _preversion10_UpdateForBatchPosterSpending(
	statedb vm.StateDB,
	evm *vm.EVM,
	arbosVersion uint64,
	updateTime, currentTime uint64,
	batchPoster common.Address,
	weiSpent *big.Int,
	l1Basefee *big.Int,
	scenario util.TracingScenario,
) error {
	if arbosVersion < params.ArbosVersion_2 {
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
	unitsAllocated := arbmath.SaturatingUMul(unitsSinceUpdate, allocationNumerator) / allocationDenominator
	unitsSinceUpdate -= unitsAllocated
	if err := ps.SetUnitsSinceUpdate(unitsSinceUpdate); err != nil {
		return err
	}

	// impose cap on amortized cost, if there is one
	if arbosVersion >= params.ArbosVersion_3 {
		amortizedCostCapBips, err := ps.AmortizedCostCapBips()
		if err != nil {
			return err
		}
		if amortizedCostCapBips != 0 {
			weiSpentCap := arbmath.BigMulByBips(
				arbmath.BigMulByUint(l1Basefee, unitsAllocated),
				arbmath.SaturatingCastToBips(amortizedCostCapBips),
			)
			if arbmath.BigLessThan(weiSpentCap, weiSpent) {
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
	err = posterState.SetFundsDue(arbmath.BigAdd(dueToPoster, weiSpent))
	if err != nil {
		return err
	}
	perUnitReward, err := ps.PerUnitReward()
	if err != nil {
		return err
	}
	fundsDueForRewards = arbmath.BigAdd(fundsDueForRewards, arbmath.BigMulByUint(arbmath.UintToBig(unitsAllocated), perUnitReward))
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}

	// pay rewards, as much as possible
	paymentForRewards := arbmath.BigMulByUint(arbmath.UintToBig(perUnitReward), unitsAllocated)
	availableFunds := statedb.GetBalance(L1PricerFundsPoolAddress)
	if arbmath.BigLessThan(availableFunds.ToBig(), paymentForRewards) {
		paymentForRewards = availableFunds.ToBig()
	}
	fundsDueForRewards = arbmath.BigSub(fundsDueForRewards, paymentForRewards)
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}
	payRewardsTo, err := ps.PayRewardsTo()
	if err != nil {
		return err
	}
	err = util.TransferBalance(
		&L1PricerFundsPoolAddress, &payRewardsTo, paymentForRewards, evm, scenario, tracing.BalanceChangeTransferBatchposterReward,
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
	if arbmath.BigLessThan(availableFunds.ToBig(), balanceToTransfer) {
		balanceToTransfer = availableFunds.ToBig()
	}
	if balanceToTransfer.Sign() > 0 {
		addrToPay, err := posterState.PayTo()
		if err != nil {
			return err
		}
		err = util.TransferBalance(
			&L1PricerFundsPoolAddress, &addrToPay, balanceToTransfer, evm, scenario, tracing.BalanceChangeTransferBatchposterRefund,
		)
		if err != nil {
			return err
		}
		balanceDueToPoster = arbmath.BigSub(balanceDueToPoster, balanceToTransfer)
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
		surplus := arbmath.BigSub(statedb.GetBalance(L1PricerFundsPoolAddress).ToBig(), arbmath.BigAdd(totalFundsDue, fundsDueForRewards))

		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilUnits, err := ps.EquilibrationUnits()
		if err != nil {
			return err
		}
		inertiaUnits := arbmath.BigDivByUint(equilUnits, inertia)
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}

		allocPlusInert := arbmath.BigAddByUint(inertiaUnits, unitsAllocated)
		oldSurplus, err := ps.LastSurplus()
		if err != nil {
			return err
		}

		desiredDerivative := arbmath.BigDiv(new(big.Int).Neg(surplus), equilUnits)
		actualDerivative := arbmath.BigDivByUint(arbmath.BigSub(surplus, oldSurplus), unitsAllocated)
		changeDerivativeBy := arbmath.BigSub(desiredDerivative, actualDerivative)
		priceChange := arbmath.BigDiv(arbmath.BigMulByUint(changeDerivativeBy, unitsAllocated), allocPlusInert)

		if err := ps.SetLastSurplus(surplus, arbosVersion); err != nil {
			return err
		}
		newPrice := arbmath.BigAdd(price, priceChange)
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
	oldSurplus := arbmath.BigSub(statedb.GetBalance(L1PricerFundsPoolAddress).ToBig(), arbmath.BigAdd(totalFundsDue, fundsDueForRewards))

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
	err = posterState.SetFundsDue(arbmath.BigAdd(dueToPoster, weiSpent))
	if err != nil {
		return err
	}
	perUnitReward, err := ps.PerUnitReward()
	if err != nil {
		return err
	}
	fundsDueForRewards = arbmath.BigAdd(fundsDueForRewards, arbmath.BigMulByUint(arbmath.UintToBig(unitsAllocated), perUnitReward))
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}

	// allocate funds to this update
	collectedSinceUpdate := statedb.GetBalance(L1PricerFundsPoolAddress)
	availableFunds := arbmath.BigDivByUint(arbmath.BigMulByUint(collectedSinceUpdate.ToBig(), allocationNumerator), allocationDenominator)

	// pay rewards, as much as possible
	paymentForRewards := arbmath.BigMulByUint(arbmath.UintToBig(perUnitReward), unitsAllocated)
	if arbmath.BigLessThan(availableFunds, paymentForRewards) {
		paymentForRewards = availableFunds
	}
	fundsDueForRewards = arbmath.BigSub(fundsDueForRewards, paymentForRewards)
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}
	payRewardsTo, err := ps.PayRewardsTo()
	if err != nil {
		return err
	}
	err = util.TransferBalance(
		&L1PricerFundsPoolAddress, &payRewardsTo, paymentForRewards, evm, scenario, tracing.BalanceChangeTransferBatchposterReward,
	)
	if err != nil {
		return err
	}
	availableFunds = arbmath.BigSub(availableFunds, paymentForRewards)

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
		if arbmath.BigLessThan(availableFunds, balanceToTransfer) {
			balanceToTransfer = availableFunds
		}
		if balanceToTransfer.Sign() > 0 {
			addrToPay, err := poster.PayTo()
			if err != nil {
				return err
			}
			err = util.TransferBalance(
				&L1PricerFundsPoolAddress, &addrToPay, balanceToTransfer, evm, scenario, tracing.BalanceChangeTransferBatchposterRefund,
			)
			if err != nil {
				return err
			}
			availableFunds = arbmath.BigSub(availableFunds, balanceToTransfer)
			balanceDueToPoster = arbmath.BigSub(balanceDueToPoster, balanceToTransfer)
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
		surplus := arbmath.BigSub(statedb.GetBalance(L1PricerFundsPoolAddress).ToBig(), arbmath.BigAdd(totalFundsDue, fundsDueForRewards))

		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilUnits, err := ps.EquilibrationUnits()
		if err != nil {
			return err
		}
		inertiaUnits := arbmath.BigDivByUint(equilUnits, inertia)
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}

		allocPlusInert := arbmath.BigAddByUint(inertiaUnits, unitsAllocated)
		priceChange := arbmath.BigDiv(
			arbmath.BigSub(
				arbmath.BigMul(surplus, arbmath.BigSub(equilUnits, common.Big1)),
				arbmath.BigMul(oldSurplus, equilUnits),
			),
			arbmath.BigMul(equilUnits, allocPlusInert),
		)

		newPrice := arbmath.BigAdd(price, priceChange)
		if newPrice.Sign() < 0 {
			newPrice = common.Big0
		}
		if err := ps.SetPricePerUnit(newPrice); err != nil {
			return err
		}
	}
	return nil
}
