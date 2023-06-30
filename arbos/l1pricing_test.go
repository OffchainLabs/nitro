// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbos

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/burn"
)

type l1PricingTest struct {
	unitReward              uint64
	unitsPerSecond          uint64
	fundsCollectedPerSecond uint64
	fundsSpent              uint64
	amortizationCapBips     uint64
	l1BasefeeGwei           uint64
}

type l1TestExpectedResults struct {
	rewardRecipientBalance *big.Int
	unitsRemaining         uint64
	fundsReceived          *big.Int
	fundsStillHeld         *big.Int
}

func TestL1Pricing(t *testing.T) {
	inputs := []*l1PricingTest{
		{
			unitReward:              10,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 7800,
			fundsSpent:              3000,
			amortizationCapBips:     math.MaxUint64,
			l1BasefeeGwei:           10,
		},
		{
			unitReward:              10,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 1313,
			fundsSpent:              3000,
			amortizationCapBips:     math.MaxUint64,
			l1BasefeeGwei:           10,
		},
		{
			unitReward:              10,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 31,
			fundsSpent:              3000,
			amortizationCapBips:     math.MaxUint64,
			l1BasefeeGwei:           10,
		},
		{
			unitReward:              10,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 7800,
			fundsSpent:              3000,
			amortizationCapBips:     100,
			l1BasefeeGwei:           10,
		},
		{
			unitReward:              0,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 7800 * params.GWei,
			fundsSpent:              3000 * params.GWei,
			amortizationCapBips:     100,
			l1BasefeeGwei:           10,
		},
	}
	for _, input := range inputs {
		expectedResult := expectedResultsForL1Test(input)
		_testL1PricingFundsDue(t, input, expectedResult)
	}
}

func expectedResultsForL1Test(input *l1PricingTest) *l1TestExpectedResults {
	ret := &l1TestExpectedResults{}
	availableFunds := arbmath.UintToBig(3 * input.fundsCollectedPerSecond)
	uncappedAvailableFunds := availableFunds
	if input.amortizationCapBips != 0 {
		availableFundsCap := arbmath.BigMulByBips(arbmath.BigMulByUint(
			arbmath.UintToBig(input.unitsPerSecond),
			input.l1BasefeeGwei*params.GWei),
			arbmath.SaturatingCastToBips(input.amortizationCapBips),
		)
		if arbmath.BigLessThan(availableFundsCap, availableFunds) {
			availableFunds = availableFundsCap
		}
	}
	fundsWantedForRewards := big.NewInt(int64(input.unitReward * input.unitsPerSecond))
	unitsAllocated := arbmath.UintToBig(input.unitsPerSecond)
	if arbmath.BigLessThan(availableFunds, fundsWantedForRewards) {
		ret.rewardRecipientBalance = availableFunds
	} else {
		ret.rewardRecipientBalance = fundsWantedForRewards
	}
	availableFunds = arbmath.BigSub(availableFunds, ret.rewardRecipientBalance)
	uncappedAvailableFunds = arbmath.BigSub(uncappedAvailableFunds, ret.rewardRecipientBalance)
	ret.unitsRemaining = (3 * input.unitsPerSecond) - unitsAllocated.Uint64()

	maxCollectable := big.NewInt(int64(input.fundsSpent))
	if arbmath.BigLessThan(availableFunds, maxCollectable) {
		maxCollectable = availableFunds
	}
	ret.fundsReceived = maxCollectable
	uncappedAvailableFunds = arbmath.BigSub(uncappedAvailableFunds, maxCollectable)
	ret.fundsStillHeld = uncappedAvailableFunds

	return ret
}

func _testL1PricingFundsDue(t *testing.T, testParams *l1PricingTest, expectedResults *l1TestExpectedResults) {
	evm := newMockEVMForTesting()
	burner := burn.NewSystemBurner(nil, false)
	arbosSt, err := arbosState.OpenArbosState(evm.StateDB, burner)
	Require(t, err)

	l1p := arbosSt.L1PricingState()
	err = l1p.SetPerUnitReward(testParams.unitReward)
	Require(t, err)
	rewardAddress := common.Address{137}
	err = l1p.SetPayRewardsTo(rewardAddress)
	Require(t, err)

	posterTable := l1p.BatchPosterTable()

	// check initial funds state
	rewardsDue, err := l1p.FundsDueForRewards()
	Require(t, err)
	if rewardsDue.Sign() != 0 {
		Fail(t)
	}
	if evm.StateDB.GetBalance(rewardAddress).Sign() != 0 {
		Fail(t)
	}
	posterAddrs, err := posterTable.AllPosters(math.MaxUint64)
	Require(t, err)
	if len(posterAddrs) != 1 {
		Fail(t)
	}
	firstPoster := posterAddrs[0]
	firstPayTo := common.Address{1, 2}
	poster, err := posterTable.OpenPoster(firstPoster, true)
	Require(t, err)
	due, err := poster.FundsDue()
	Require(t, err)
	if due.Sign() != 0 {
		Fail(t)
	}
	err = poster.SetPayTo(firstPayTo)
	Require(t, err)

	// add another poster
	secondPoster := common.Address{3, 4, 5}
	secondPayTo := common.Address{6, 7}
	_, err = posterTable.AddPoster(secondPoster, secondPayTo)
	Require(t, err)

	// create some fake collection
	balanceAdded := big.NewInt(int64(testParams.fundsCollectedPerSecond * 3))
	unitsAdded := testParams.unitsPerSecond * 3
	evm.StateDB.AddBalance(l1pricing.L1PricerFundsPoolAddress, balanceAdded)
	err = l1p.SetL1FeesAvailable(balanceAdded)
	Require(t, err)
	err = l1p.SetUnitsSinceUpdate(unitsAdded)
	Require(t, err)

	// submit a fake spending update, then check that balances are correct
	err = l1p.SetAmortizedCostCapBips(testParams.amortizationCapBips)
	Require(t, err)
	version := arbosSt.ArbOSVersion()
	scenario := util.TracingDuringEVM
	err = l1p.UpdateForBatchPosterSpending(
		evm.StateDB, evm, version, 1, 3, firstPoster, arbmath.UintToBig(testParams.fundsSpent), arbmath.UintToBig(testParams.l1BasefeeGwei*params.GWei), scenario,
	)
	Require(t, err)
	rewardRecipientBalance := evm.StateDB.GetBalance(rewardAddress)
	if !arbmath.BigEquals(rewardRecipientBalance, expectedResults.rewardRecipientBalance) {
		Fail(t, rewardRecipientBalance, expectedResults.rewardRecipientBalance)
	}
	unitsRemaining, err := l1p.UnitsSinceUpdate()
	Require(t, err)
	if unitsRemaining != expectedResults.unitsRemaining {
		Fail(t, unitsRemaining, expectedResults.unitsRemaining)
	}
	fundsReceived := evm.StateDB.GetBalance(firstPayTo)
	if !arbmath.BigEquals(fundsReceived, expectedResults.fundsReceived) {
		Fail(t, fundsReceived, expectedResults.fundsReceived)
	}
	fundsStillHeld := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	if !arbmath.BigEquals(fundsStillHeld, expectedResults.fundsStillHeld) {
		Fail(t, fundsStillHeld, expectedResults.fundsStillHeld)
	}
	fundsAvail, err := l1p.L1FeesAvailable()
	Require(t, err)
	if fundsStillHeld.Cmp(fundsAvail) != 0 {
		Fail(t, fundsStillHeld, fundsAvail)
	}
}

func TestUpdateTimeUpgradeBehavior(t *testing.T) {
	evm := newMockEVMForTesting()
	burner := burn.NewSystemBurner(nil, false)
	arbosSt, err := arbosState.OpenArbosState(evm.StateDB, burner)
	Require(t, err)

	l1p := arbosSt.L1PricingState()
	amount := arbmath.UintToBig(10 * params.GWei)
	poster := common.Address{3, 4, 5}
	_, err = l1p.BatchPosterTable().AddPoster(poster, poster)
	Require(t, err)

	// In the past this would have errored due to an invalid timestamp.
	// We don't want to error since it'd create noise in the console,
	// so instead let's check that nothing happened
	statedb, ok := evm.StateDB.(*state.StateDB)
	if !ok {
		panic("not a statedb")
	}
	stateCheck(t, statedb, false, "uh oh, nothing should have happened", func() {
		Require(t, l1p.UpdateForBatchPosterSpending(
			evm.StateDB, evm, 1, 1, 1, poster, common.Big1, amount, util.TracingDuringEVM,
		))
	})

	Require(t, l1p.UpdateForBatchPosterSpending(
		evm.StateDB, evm, 3, 1, 1, poster, common.Big1, amount, util.TracingDuringEVM,
	))
}

func TestL1PriceEquilibrationUp(t *testing.T) {
	_testL1PriceEquilibration(t, big.NewInt(1_000_000_000), big.NewInt(5_000_000_000))
}

func TestL1PriceEquilibrationDown(t *testing.T) {
	_testL1PriceEquilibration(t, big.NewInt(5_000_000_000), big.NewInt(1_000_000_000))
}

func TestL1PriceEquilibrationConstant(t *testing.T) {
	_testL1PriceEquilibration(t, big.NewInt(2_000_000_000), big.NewInt(2_000_000_000))
}

func _testL1PriceEquilibration(t *testing.T, initialL1BasefeeEstimate *big.Int, equilibriumL1BasefeeEstimate *big.Int) {
	evm := newMockEVMForTesting()
	stateDb := evm.StateDB
	state, err := arbosState.OpenArbosState(stateDb, burn.NewSystemBurner(nil, false))
	Require(t, err)

	l1p := state.L1PricingState()
	Require(t, l1p.SetPerUnitReward(0))
	Require(t, l1p.SetPricePerUnit(initialL1BasefeeEstimate))

	bpAddr := common.Address{3, 4, 5, 6}
	l1PoolAddress := l1pricing.L1PricerFundsPoolAddress
	for i := 0; i < 10; i++ {
		unitsToAdd := l1pricing.InitialEquilibrationUnitsV6.Uint64()
		oldUnits, err := l1p.UnitsSinceUpdate()
		Require(t, err)
		err = l1p.SetUnitsSinceUpdate(oldUnits + unitsToAdd)
		Require(t, err)
		currentPricePerUnit, err := l1p.PricePerUnit()
		Require(t, err)
		feesToAdd := arbmath.BigMulByUint(currentPricePerUnit, unitsToAdd)
		util.MintBalance(&l1PoolAddress, feesToAdd, evm, util.TracingBeforeEVM, "test")
		err = l1p.UpdateForBatchPosterSpending(
			evm.StateDB,
			evm,
			3,
			uint64(10*(i+1)),
			uint64(10*(i+1)+5),
			bpAddr,
			arbmath.BigMulByUint(equilibriumL1BasefeeEstimate, unitsToAdd),
			equilibriumL1BasefeeEstimate,
			util.TracingBeforeEVM,
		)
		Require(t, err)
	}
	expectedMovement := arbmath.BigSub(equilibriumL1BasefeeEstimate, initialL1BasefeeEstimate)
	actualPricePerUnit, err := l1p.PricePerUnit()
	Require(t, err)
	actualMovement := arbmath.BigSub(actualPricePerUnit, initialL1BasefeeEstimate)
	if expectedMovement.Sign() != actualMovement.Sign() {
		Fail(t, "L1 data fee moved in wrong direction", initialL1BasefeeEstimate, equilibriumL1BasefeeEstimate, actualPricePerUnit)
	}
	expectedMovement = new(big.Int).Abs(expectedMovement)
	actualMovement = new(big.Int).Abs(actualMovement)
	if !_withinOnePercent(expectedMovement, actualMovement) {
		Fail(t, "Expected vs actual movement are too far apart", expectedMovement, actualMovement)
	}
}

func _withinOnePercent(v1, v2 *big.Int) bool {
	if arbmath.BigMulByUint(v1, 100).Cmp(arbmath.BigMulByUint(v2, 101)) > 0 {
		return false
	}
	if arbmath.BigMulByUint(v2, 100).Cmp(arbmath.BigMulByUint(v1, 101)) > 0 {
		return false
	}
	return true
}

func newMockEVMForTesting() *vm.EVM {
	chainConfig := params.ArbitrumDevTestChainConfig()
	_, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	context := vm.BlockContext{
		BlockNumber: big.NewInt(0),
		GasLimit:    ^uint64(0),
		Time:        0,
	}
	evm := vm.NewEVM(context, vm.TxContext{}, statedb, chainConfig, vm.Config{})
	evm.ProcessingHook = &TxProcessor{}
	return evm
}
