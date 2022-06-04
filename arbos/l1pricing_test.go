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

const (
	unitReward_test              = 10
	unitsPerSecond_test          = 78
	fundsCollectedPerSecond_test = 7800
	fundsSpent_test              = 3000
)

func TestL1PricingFundsDue(t *testing.T) {
	evm := newMockEVMForTesting()
	burner := burn.NewSystemBurner(nil, false)
	arbosSt, err := arbosState.OpenArbosState(evm.StateDB, burner)
	Require(t, err)

	l1p := arbosSt.L1PricingState()
	err = l1p.SetPerUnitReward(unitReward_test)
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
	balanceAdded := big.NewInt(fundsCollectedPerSecond_test * 3)
	unitsAdded := uint64(unitsPerSecond_test * 3)
	evm.StateDB.AddBalance(l1pricing.L1PricerFundsPoolAddress, balanceAdded)
	err = l1p.SetUnitsSinceUpdate(unitsAdded)
	Require(t, err)

	// submit a fake spending update, then check that balances are correct
	err = l1p.UpdateForBatchPosterSpending(evm.StateDB, evm, 1, 3, firstPoster, big.NewInt(fundsSpent_test))
	Require(t, err)
	rewardRecipientBalance := evm.StateDB.GetBalance(rewardAddress)
	expectedUnitsCollected := unitsPerSecond_test
	if !arbmath.BigEquals(rewardRecipientBalance, big.NewInt(int64(unitReward_test*expectedUnitsCollected))) {
		t.Fatal(rewardRecipientBalance, unitReward_test*expectedUnitsCollected)
	}
	unitsRemaining, err := l1p.UnitsSinceUpdate()
	Require(t, err)
	if unitsRemaining != (3*unitsPerSecond_test)-uint64(expectedUnitsCollected) {
		t.Fatal(unitsRemaining, (3*unitsPerSecond_test)-uint64(expectedUnitsCollected))
	}
	remainingFunds := arbmath.BigSub(balanceAdded, rewardRecipientBalance)
	maxCollectable := big.NewInt(fundsSpent_test)
	if arbmath.BigLessThan(remainingFunds, maxCollectable) {
		maxCollectable = remainingFunds
	}
	fundsReceived := evm.StateDB.GetBalance(firstPayTo)
	if !arbmath.BigEquals(fundsReceived, maxCollectable) {
		t.Fatal(fundsReceived, maxCollectable)
	}
	fundsStillHeld := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	if !arbmath.BigEquals(fundsStillHeld, arbmath.BigSub(remainingFunds, maxCollectable)) {
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
