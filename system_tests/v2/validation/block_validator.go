// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package validation

import (
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var blockValidatorTests = []systest.Scenario{
	systest.Test(testRunBlockValidatorSimple, systest.WithL1(), systest.WithValidation()),
	systest.Test(testRunStakingValidation, systest.WithStakingValidation()),
}

func testRunBlockValidatorSimple(env *systest.Env) {
	contractCode := program.New().
		Op(vm.PUSH0, vm.PUSH0).
		Push(8). // the prelude length
		Op(vm.PUSH0, vm.CODECOPY, vm.PUSH0, vm.BLOBHASH, vm.RETURN).
		Bytes()
	for range 8 {
		gas := env.L2.EstimateGas(ethereum.CallMsg{
			From:  env.L2.Info.GetAddress("Owner"),
			Value: common.Big0,
			Data:  contractCode,
		})
		tx := env.L2.Info.PrepareTxTo("Owner", nil, gas, common.Big0, contractCode)
		env.L2.SendTx(tx)
		env.L2.EnsureTxSucceeded(tx)
	}

	env.L2.Info.GenerateAccount("User2")
	delayedTx := env.L2.Info.PrepareTx("Owner", "User2", 30002, big.NewInt(1e12), nil)
	env.SendSignedTxViaL1(delayedTx)
}

func testRunStakingValidation(env *systest.Env) {
	env.L2.Info.GenerateAccount("User2")
	txs := []*types.Transaction{
		env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil),
		env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil),
	}
	env.L2.SendWaitTestTransactions(txs)

	env.WaitForFollowersSync()
	bal := env.Follower().BalanceAt(env.L2.Info.GetAddress("User2"))
	env.EqualBig(big.NewInt(2e12), bal, "follower balance")
}
