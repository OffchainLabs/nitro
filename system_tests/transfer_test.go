// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode"
	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
)

func testTransfer(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx := t.Context()

	// For External/Comparison modes, we need L1 for the replica
	withL1 := executionClientMode != ExecutionClientModeInternal

	builder := NewNodeBuilder(ctx).DefaultConfig(t, withL1)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	// For Internal mode, test on primary node only
	if executionClientMode == ExecutionClientModeInternal {
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)

		bal, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), nil)
		Require(t, err)
		fmt.Println("Owner balance is: ", bal)

		bal2, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
		Require(t, err)
		if bal2.Cmp(big.NewInt(1e12)) != 0 {
			Fatal(t, "Unexpected recipient balance: ", bal2)
		}
		return
	}

	// For External/Comparison modes, test on replica
	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}

	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()
	replicaClient := replicaTestClient.Client

	// Wait for replica to initialize and sync
	time.Sleep(time.Second * 2)

	primaryBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	// Wait for replica to catch up
	for i := 0; i < 30; i++ {
		replicaBlock, err := replicaClient.BlockNumber(ctx)
		Require(t, err)
		if replicaBlock >= primaryBlock {
			break
		}
		time.Sleep(time.Second)
	}

	// Send transaction on primary
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Wait for transaction to sync to replica
	_, err = WaitForTx(ctx, replicaClient, tx.Hash(), time.Second*30)
	Require(t, err)

	// Verify balances on replica
	bal, err := replicaClient.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), nil)
	Require(t, err)
	fmt.Println("Replica owner balance is: ", bal)

	bal2, err := replicaClient.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected replica balance: ", bal2)
	}
}

func TestTransfer(t *testing.T) {
	testTransfer(t, ExecutionClientModeInternal)
}

func TestTransferExternal(t *testing.T) {
	testTransfer(t, ExecutionClientModeExternal)
}

func TestTransferComparison(t *testing.T) {
	testTransfer(t, ExecutionClientModeExternal)
}

// getExpectedP256Result returns the expected result for P256Verify based on ArbOS version
// P256VERIFY precompile was introduced in ArbOS 30
func getExpectedP256Result(version uint64) []byte {
	// P256VERIFY is not available in ArbOS versions < 30
	if version < 30 {
		return nil
	}
	// P256VERIFY is available in ArbOS 30 and later
	return common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
}

func testP256Verify(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx := t.Context()

	// Use the version from the flag
	initialVersion := *testflag.ArbOSVersionFlag
	want := getExpectedP256Result(initialVersion)

	if want == nil {
		t.Logf("Testing P256Verify on ArbOS %d - expecting precompile to not be enabled (nil response)", initialVersion)
	} else {
		t.Logf("Testing P256Verify on ArbOS %d - expecting precompile to be enabled (success response)", initialVersion)
	}

	withL1 := executionClientMode != ExecutionClientModeInternal

	builder := NewNodeBuilder(ctx).DefaultConfig(t, withL1).DontParalellise()
	cleanup := builder.Build(t)
	defer cleanup()

	addr := common.BytesToAddress([]byte{0x01, 0x00})
	callMsg := ethereum.CallMsg{
		From:  builder.L2Info.GetAddress("Owner"),
		To:    &addr,
		Gas:   builder.L2Info.TransferGas,
		Data:  common.Hex2Bytes("4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e"),
		Value: big.NewInt(1e12),
	}

	if executionClientMode == ExecutionClientModeInternal {
		got, err := builder.L2.Client.CallContract(ctx, callMsg, nil)
		if err != nil {
			t.Fatalf("CallContract() unexpected error: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("P256Verify() = %v, want: %v (testing ArbOS version %d)", got, want, initialVersion)
		}
		return
	}

	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}

	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()
	replicaClient := replicaTestClient.Client

	time.Sleep(time.Second * 3)

	got, err := replicaClient.CallContract(ctx, callMsg, nil)
	if err != nil {
		t.Fatalf("CallContract() unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("P256Verify() = %v, want: %v (testing ArbOS version %d)", got, want, initialVersion)
	}
}

func TestP256VerifyInternal(t *testing.T) {
	testP256Verify(t, ExecutionClientModeInternal)
}

func TestP256VerifyExternal(t *testing.T) {
	testP256Verify(t, ExecutionClientModeExternal)
}

func TestP256VerifyComparison(t *testing.T) {
	testP256Verify(t, ExecutionClientModeComparison)
}
