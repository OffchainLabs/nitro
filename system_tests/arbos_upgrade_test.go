// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestScheduleArbosUpgrade(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), builder.L2.Client)
	Require(t, err, "could not bind ArbOwner contract")

	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err, "could not bind ArbOwner contract")

	callOpts := &bind.CallOpts{Context: ctx}
	scheduled, err := arbOwnerPublic.GetScheduledUpgrade(callOpts)
	Require(t, err, "failed to call GetScheduledUpgrade before scheduling upgrade")
	if scheduled.ArbosVersion != 0 || scheduled.ScheduledForTimestamp != 0 {
		t.Errorf("expected no upgrade to be scheduled, got version %v timestamp %v", scheduled.ArbosVersion, scheduled.ScheduledForTimestamp)
	}

	// Schedule a noop upgrade, which should test GetScheduledUpgrade in the same way an already completed upgrade would.
	tx, err := arbOwner.ScheduleArbOSUpgrade(&auth, 1, 1)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	scheduled, err = arbOwnerPublic.GetScheduledUpgrade(callOpts)
	Require(t, err, "failed to call GetScheduledUpgrade after scheduling noop upgrade")
	if scheduled.ArbosVersion != 0 || scheduled.ScheduledForTimestamp != 0 {
		t.Errorf("expected completed scheduled upgrade to be ignored, got version %v timestamp %v", scheduled.ArbosVersion, scheduled.ScheduledForTimestamp)
	}

	// TODO: Once we have an ArbOS 30, test a real upgrade with it
	// We can't test 11 -> 20 because 11 doesn't have the GetScheduledUpgrade method we want to test
	var testVersion uint64 = 100
	var testTimestamp uint64 = 1 << 62
	tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, 100, 1<<62)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	scheduled, err = arbOwnerPublic.GetScheduledUpgrade(callOpts)
	Require(t, err, "failed to call GetScheduledUpgrade after scheduling upgrade")
	if scheduled.ArbosVersion != testVersion || scheduled.ScheduledForTimestamp != testTimestamp {
		t.Errorf("expected upgrade to be scheduled for version %v timestamp %v, got version %v timestamp %v", testVersion, testTimestamp, scheduled.ArbosVersion, scheduled.ScheduledForTimestamp)
	}
}

func checkArbOSVersion(t *testing.T, testClient *TestClient, expectedVersion uint64, scenario string) {
	statedb, err := testClient.ExecNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err, "could not get statedb", scenario)
	state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	Require(t, err, "could not open ArbOS state", scenario)
	if state.ArbOSVersion() != expectedVersion {
		t.Errorf("%s: expected ArbOS version %v, got %v", scenario, expectedVersion, state.ArbOSVersion())
	}

}

func TestArbos11To32Upgrade(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialVersion := uint64(11)
	finalVersion := uint64(32)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(initialVersion)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	// deploys test contract
	_, tx, contract, err := mocksgen.DeployArbOS11To32UpgradeTest(&auth, builder.L2.Client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	checkArbOSVersion(t, builder.L2, initialVersion, "initial")

	// mcopy should revert since arbos 11 doesn't support it
	_, err = contract.Mcopy(&auth)
	if (err == nil) || (err.Error() != "invalid opcode: MCOPY") {
		t.Fatalf("expected MCOPY to revert, got %v", err)
	}

	// upgrade arbos to final version
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, finalVersion, 0)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// generates a new block to trigger the upgrade
	builder.L2Info.GenerateAccount("User2")
	tx = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	checkArbOSVersion(t, builder.L2, finalVersion, "final")
	_, err = contract.Mcopy(&auth)
	Require(t, err)
}
