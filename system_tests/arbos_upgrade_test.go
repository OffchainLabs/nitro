// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
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

	l2rpc := builder.L2.Stack.Attach()
	var result json.RawMessage
	traceConfig := map[string]interface{}{
		"tracer": "prestateTracer",
		"tracerConfig": map[string]interface{}{
			"diffMode": true,
		},
	}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), traceConfig)
	Require(t, err)

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
		t.Fatalf("%s: expected ArbOS version %v, got %v", scenario, expectedVersion, state.ArbOSVersion())
	}

}

func TestArbos11To32UpgradeWithMcopy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialVersion := uint64(11)
	finalVersion := uint64(32)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(initialVersion)
	cleanup := builder.Build(t)
	defer cleanup()
	seqTestClient := builder.L2

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasLimit = 32000000

	// makes Owner a chain owner
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, seqTestClient.Client)
	Require(t, err)
	tx, err := arbDebug.BecomeChainOwner(&auth)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, seqTestClient.Client, tx)
	Require(t, err)

	// deploys test contract
	_, tx, contract, err := localgen.DeployArbOS11To32UpgradeTest(&auth, seqTestClient.Client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, seqTestClient.Client, tx)
	Require(t, err)

	// build replica node
	replicaConfig := arbnode.ConfigDefaultL1Test()
	replicaConfig.BatchPoster.Enable = false
	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: replicaConfig})
	defer replicaCleanup()

	checkArbOSVersion(t, seqTestClient, initialVersion, "initial sequencer")
	checkArbOSVersion(t, replicaTestClient, initialVersion, "initial replica")

	// mcopy should fail since arbos 11 doesn't support it
	tx, err = contract.Mcopy(&auth)
	Require(t, err)
	_, err = seqTestClient.EnsureTxSucceeded(tx)
	if (err == nil) || !strings.Contains(err.Error(), "invalid opcode: MCOPY") {
		t.Errorf("expected MCOPY to fail, got %v", err)
	}
	_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)

	// upgrade arbos to final version
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, seqTestClient.Client)
	Require(t, err)
	tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, finalVersion, 0)
	Require(t, err)
	_, err = seqTestClient.EnsureTxSucceeded(tx)
	Require(t, err)
	_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)

	// checks upgrade worked
	tx, err = contract.Mcopy(&auth)
	Require(t, err)
	_, err = seqTestClient.EnsureTxSucceeded(tx)
	Require(t, err)
	_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)

	checkArbOSVersion(t, seqTestClient, finalVersion, "final sequencer")
	checkArbOSVersion(t, replicaTestClient, finalVersion, "final replica")

	// generates more blocks
	builder.L2Info.GenerateAccount("User2")
	for i := 0; i < 3; i++ {
		tx = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err = seqTestClient.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = seqTestClient.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
		Require(t, err)
	}

	blockNumberSeq, err := seqTestClient.Client.BlockNumber(ctx)
	Require(t, err)
	blockNumberReplica, err := replicaTestClient.Client.BlockNumber(ctx)
	Require(t, err)
	if blockNumberSeq != blockNumberReplica {
		t.Errorf("expected sequencer and replica to have same block number, got %v and %v", blockNumberSeq, blockNumberReplica)
	}
	// #nosec G115
	blockNumber := big.NewInt(int64(blockNumberSeq))

	blockSeq, err := seqTestClient.Client.BlockByNumber(ctx, blockNumber)
	Require(t, err)
	blockReplica, err := replicaTestClient.Client.BlockByNumber(ctx, blockNumber)
	Require(t, err)
	if blockSeq.Hash() != blockReplica.Hash() {
		t.Errorf("expected sequencer and replica to have same block hash, got %v and %v", blockSeq.Hash(), blockReplica.Hash())
	}
}

func TestArbNativeTokenManagerInArbos32To41Upgrade(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialVersion := uint64(32)
	finalVersion := uint64(41)

	arbOSInit := &params.ArbOSInit{
		NativeTokenSupplyManagementEnabled: true,
	}
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(initialVersion).
		WithArbOSInit(arbOSInit)
	builder.execConfig.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessLikelyCompatible
	cleanup := builder.Build(t)
	defer cleanup()

	authOwner := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	authOwner.GasLimit = 32000000

	// makes Owner a chain owner
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	Require(t, err)
	tx, err := arbDebug.BecomeChainOwner(&authOwner)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	checkArbOSVersion(t, builder.L2, initialVersion, "")

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	callOpts := &bind.CallOpts{Context: ctx}

	nativeTokenOwnerName := "NativeTokenOwner"
	builder.L2Info.GenerateAccount(nativeTokenOwnerName)
	nativeTokenOwnerAddr := builder.L2Info.GetAddress(nativeTokenOwnerName)

	// checks that IsNativeTokenOwner doesn't exist in ArbOwner before upgrade
	_, err = arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	if err == nil || !strings.Contains(err.Error(), "execution reverted") {
		t.Fatalf("expected IsNativeTokenOwner to fail before upgrade, got %v", err)
	}

	// schedule arbos upgrade
	tx, err = arbOwner.ScheduleArbOSUpgrade(&authOwner, finalVersion, 0)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// checks upgrade worked
	var data []byte
	for i := range 10 {
		for range 100 {
			data = append(data, byte(i))
		}
	}
	tx = builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), data)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	checkArbOSVersion(t, builder.L2, finalVersion, "")

	// checks that IsNativeTokenOwner works after upgrade
	_, err = arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	Require(t, err)

	// adds native token owner
	tx, err = arbOwner.AddNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)

	// funds the native token owner
	tx = builder.L2Info.PrepareTx("Owner", nativeTokenOwnerName, builder.L2Info.TransferGas, big.NewInt(500000000000000000), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	arbNativeTokenManager, err := precompilesgen.NewArbNativeTokenManager(types.ArbNativeTokenManagerAddress, builder.L2.Client)
	Require(t, err)

	// checks minting
	nativeTokenOwnerABI, err := precompilesgen.ArbNativeTokenManagerMetaData.GetAbi()
	Require(t, err)
	mintTopic := nativeTokenOwnerABI.Events["NativeTokenMinted"].ID
	authNativeTokenOwner := builder.L2Info.GetDefaultTransactOpts(nativeTokenOwnerName, ctx)
	authNativeTokenOwner.GasLimit = 32000000
	toMint := big.NewInt(100)
	tx, err = arbNativeTokenManager.MintNativeToken(&authNativeTokenOwner, toMint)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	mintLogged := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == mintTopic {
			mintLogged = true
			parsedLog, err := arbNativeTokenManager.ParseNativeTokenMinted(*log)
			Require(t, err)
			if parsedLog.To != nativeTokenOwnerAddr {
				t.Fatal("expected mint to be to", nativeTokenOwnerAddr, "got", parsedLog.To)
			}
			if parsedLog.Amount.Cmp(toMint) != 0 {
				t.Fatal("expected mint amount to be", toMint, "got", parsedLog.Amount)
			}
		}
	}
	if !mintLogged {
		t.Fatal("expected mint event to be logged")
	}
}

func TestArbos11To32UpgradeWithCalldata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialVersion := uint64(11)
	finalVersion := uint64(32)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(initialVersion)
	builder.execConfig.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessLikelyCompatible
	cleanup := builder.Build(t)
	defer cleanup()
	seqTestClient := builder.L2

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasLimit = 32000000

	// makes Owner a chain owner
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, seqTestClient.Client)
	Require(t, err)
	tx, err := arbDebug.BecomeChainOwner(&auth)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, seqTestClient.Client, tx)
	Require(t, err)

	// build replica node
	replicaConfig := arbnode.ConfigDefaultL1Test()
	replicaConfig.BatchPoster.Enable = false
	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: replicaConfig})
	defer replicaCleanup()

	checkArbOSVersion(t, seqTestClient, initialVersion, "initial sequencer")
	checkArbOSVersion(t, replicaTestClient, initialVersion, "initial replica")

	// upgrade arbos to final version
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, seqTestClient.Client)
	Require(t, err)
	tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, finalVersion, 0)
	Require(t, err)
	_, err = seqTestClient.EnsureTxSucceeded(tx)
	Require(t, err)
	_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)

	// checks upgrade worked
	var data []byte
	for i := range 10 {
		for range 100 {
			data = append(data, byte(i))
		}
	}
	tx = builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), data)
	err = seqTestClient.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = seqTestClient.EnsureTxSucceeded(tx)
	Require(t, err)
	_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)

	checkArbOSVersion(t, seqTestClient, finalVersion, "final sequencer")
	checkArbOSVersion(t, replicaTestClient, finalVersion, "final replica")

	blockNumberSeq, err := seqTestClient.Client.BlockNumber(ctx)
	Require(t, err)
	blockNumberReplica, err := replicaTestClient.Client.BlockNumber(ctx)
	Require(t, err)
	if blockNumberSeq != blockNumberReplica {
		t.Errorf("expected sequencer and replica to have same block number, got %v and %v", blockNumberSeq, blockNumberReplica)
	}
	// #nosec G115
	blockNumber := big.NewInt(int64(blockNumberSeq))

	blockSeq, err := seqTestClient.Client.BlockByNumber(ctx, blockNumber)
	Require(t, err)
	blockReplica, err := replicaTestClient.Client.BlockByNumber(ctx, blockNumber)
	Require(t, err)
	if blockSeq.Hash() != blockReplica.Hash() {
		t.Errorf("expected sequencer and replica to have same block hash, got %v and %v", blockSeq.Hash(), blockReplica.Hash())
	}
}
