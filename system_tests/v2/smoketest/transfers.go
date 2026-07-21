// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package smoketest holds foundational L2 tests: transfers, deploys, etc.
package smoketest

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var transferTests = []systest.Scenario{
	systest.Test(testRunTransfer),
	systest.Test(testRunSelfDestruct),
}

func testRunTransfer(env *systest.Env) {
	env.L2.Info.GenerateAccount("User2")

	tx := env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil)
	env.L2.SendTx(tx)
	env.L2.EnsureTxSucceeded(tx)

	bal := env.L2.BalanceAt(env.L2.Info.GetAddress("Owner"))
	env.Logf("Owner balance is: %s", bal)

	bal2 := env.L2.BalanceAt(env.L2.Info.GetAddress("User2"))
	env.EqualBig(big.NewInt(1e12), bal2, "recipient balance")
}

func testRunSelfDestruct(env *systest.Env) {
	env.L2.Info.GenerateAccount("Destination")
	destination := env.L2.Info.GetAddress("Destination")
	env.L2.Info.GenerateAccount("SelfDestruct")
	auth := env.L2.TransactOpts("SelfDestruct")
	auth.GasLimit = 32000000
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	env.L2.SendWaitTxs("Faucet", "SelfDestruct", 1, balance)

	// Test self-destruct with recipient same as the contract (contract is created and destroyed in the same transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, _, err := localgen.DeploySelfDestructInConstructorWithoutDestination(&auth, env.L2.Client)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// Test self-destruct with recipient different from the contract (contract is created and destroyed in the same transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, _, err = localgen.DeploySelfDestructInConstructorWithDestination(&auth, env.L2.Client, destination)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// Test self-destruct with recipient same as the contract (contract is created and destroyed in different transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, selfDestructOutsideConstructor, err := localgen.DeploySelfDestructOutsideConstructor(&auth, env.L2.Client)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
	auth.Value = nil
	tx, err = selfDestructOutsideConstructor.SelfDestructWithoutDestination(&auth)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// Test self-destruct with recipient different from the contract (contract is created and destroyed in different transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, selfDestructOutsideConstructor, err = localgen.DeploySelfDestructOutsideConstructor(&auth, env.L2.Client)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
	auth.Value = nil
	tx, err = selfDestructOutsideConstructor.SelfDestructWithDestination(&auth, destination)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
}
