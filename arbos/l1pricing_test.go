// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/arbmath"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/burn"
)

type l1PricingTest struct {
	unitReward              uint64
	unitsPerSecond          uint64
	fundsCollectedPerSecond uint64
	fundsSpent              uint64
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
		},
		{
			unitReward:              10,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 1313,
			fundsSpent:              3000,
		},
		{
			unitReward:              10,
			unitsPerSecond:          78,
			fundsCollectedPerSecond: 31,
			fundsSpent:              3000,
		},
	}
	for _, input := range inputs {
		expectedResult := expectedResultsForL1Test(input)
		_testL1PricingFundsDue(t, input, expectedResult)
	}
}

func expectedResultsForL1Test(input *l1PricingTest) *l1TestExpectedResults {
	ret := &l1TestExpectedResults{}
	availableFunds := arbmath.UintToBig(input.fundsCollectedPerSecond)
	fundsWantedForRewards := big.NewInt(int64(input.unitReward * input.unitsPerSecond))
	unitsAllocated := arbmath.UintToBig(input.unitsPerSecond)
	if arbmath.BigLessThan(availableFunds, fundsWantedForRewards) {
		ret.rewardRecipientBalance = availableFunds
	} else {
		ret.rewardRecipientBalance = fundsWantedForRewards
	}
	availableFunds = arbmath.BigSub(availableFunds, ret.rewardRecipientBalance)
	ret.unitsRemaining = (3 * input.unitsPerSecond) - unitsAllocated.Uint64()

	maxCollectable := big.NewInt(int64(input.fundsSpent))
	if arbmath.BigLessThan(availableFunds, maxCollectable) {
		maxCollectable = availableFunds
	}
	ret.fundsReceived = maxCollectable
	availableFunds = arbmath.BigSub(availableFunds, maxCollectable)
	ret.fundsStillHeld = arbmath.BigAdd(arbmath.UintToBig(2*input.fundsCollectedPerSecond), availableFunds)

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
		t.Fatal()
	}
	if evm.StateDB.GetBalance(rewardAddress).Sign() != 0 {
		t.Fatal()
	}
	posterAddrs, err := posterTable.AllPosters()
	Require(t, err)
	if len(posterAddrs) != 1 {
		t.Fatal()
	}
	firstPoster := posterAddrs[0]
	firstPayTo := common.Address{1, 2}
	poster, err := posterTable.OpenPoster(firstPoster)
	Require(t, err)
	due, err := poster.FundsDue()
	Require(t, err)
	if due.Sign() != 0 {
		t.Fatal()
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
	unitsAdded := uint64(testParams.unitsPerSecond * 3)
	evm.StateDB.AddBalance(l1pricing.L1PricerFundsPoolAddress, balanceAdded)
	err = l1p.SetUnitsSinceUpdate(unitsAdded)
	Require(t, err)

	// submit a fake spending update, then check that balances are correct
	err = l1p.UpdateForBatchPosterSpending(evm.StateDB, evm, 1, 3, firstPoster, new(big.Int).SetUint64(testParams.fundsSpent))
	Require(t, err)
	rewardRecipientBalance := evm.StateDB.GetBalance(rewardAddress)
	if !arbmath.BigEquals(rewardRecipientBalance, expectedResults.rewardRecipientBalance) {
		t.Fatal(rewardRecipientBalance, expectedResults.rewardRecipientBalance)
	}
	unitsRemaining, err := l1p.UnitsSinceUpdate()
	Require(t, err)
	if unitsRemaining != expectedResults.unitsRemaining {
		t.Fatal(unitsRemaining, expectedResults.unitsRemaining)
	}
	fundsReceived := evm.StateDB.GetBalance(firstPayTo)
	if !arbmath.BigEquals(fundsReceived, expectedResults.fundsReceived) {
		t.Fatal(fundsReceived, expectedResults.fundsReceived)
	}
	fundsStillHeld := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	if !arbmath.BigEquals(fundsStillHeld, expectedResults.fundsStillHeld) {
		t.Fatal()
	}
}

func newMockEVMForTesting() *vm.EVM {
	chainConfig := params.ArbitrumDevTestChainConfig()
	_, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	context := vm.BlockContext{
		BlockNumber: big.NewInt(0),
		GasLimit:    ^uint64(0),
		Time:        big.NewInt(0),
	}
	evm := vm.NewEVM(context, vm.TxContext{}, statedb, chainConfig, vm.Config{})
	evm.ProcessingHook = &TxProcessor{}
	return evm
}
