package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// submitRetryableNoAutoRedeem submits a retryable ticket via the L1 delayed inbox
// with gasLimit=0 so no auto-redeem is scheduled. The ticket survives for later
// manual redemption. Returns the L1 receipt and the L2 submission tx hash (ticketId).
func submitRetryableNoAutoRedeem(
	t *testing.T,
	p *retryableFilterTestParams,
	l1Sender string,
	destAddr common.Address,
	callValue *big.Int,
	beneficiary common.Address,
	feeRefundAddr common.Address,
	data []byte,
) (*types.Receipt, common.Hash) {
	t.Helper()
	zeroGasLimit := big.NewInt(0)
	return submitRetryableViaL1WithGasLimit(t, p, l1Sender, destAddr, callValue, beneficiary, feeRefundAddr, data, zeroGasLimit)
}

// submitRetryableWithFailingAutoRedeem submits a retryable ticket via the L1 delayed inbox
// with just enough gasLimit for the auto-redeem to be scheduled (>= TxGas) but insufficient
// gas for the inner call to succeed. This results in a failed auto-redeem, leaving the
// ticket with numTries=1 after processing.
func submitRetryableWithFailingAutoRedeem(
	t *testing.T,
	p *retryableFilterTestParams,
	l1Sender string,
	destAddr common.Address,
	callValue *big.Int,
	beneficiary common.Address,
	feeRefundAddr common.Address,
	data []byte,
) (*types.Receipt, common.Hash) {
	t.Helper()

	// Calculate intrinsic gas for the data
	var dataGas uint64
	for _, b := range data {
		if b == 0 {
			dataGas += params.TxDataZeroGas
		} else {
			dataGas += params.TxDataNonZeroGasEIP2028
		}
	}
	// Gas covers intrinsic cost + tiny buffer: enough to schedule auto-redeem
	// (gasLimit >= TxGas) but inner call gets ~100 gas → immediate out-of-gas.
	gasLimit := big.NewInt(int64(params.TxGas + dataGas + 100))

	return submitRetryableViaL1WithGasLimit(t, p, l1Sender, destAddr, callValue, beneficiary, feeRefundAddr, data, gasLimit)
}

// processRetryableWithFailingAutoRedeem advances L1, waits for the retryable
// submission to be processed on L2, then extracts the auto-redeem tx hash from
// the RedeemScheduled event and waits for it to fail. After this call, the
// ticket has numTries=1.
func processRetryableWithFailingAutoRedeem(
	t *testing.T,
	p *retryableFilterTestParams,
	ticketId common.Hash,
) {
	t.Helper()
	advanceL1ForDelayed(t, p.ctx, p.builder)

	// Wait for submission receipt
	submissionReceipt, err := WaitForTx(p.ctx, p.builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, submissionReceipt.Status,
		"retryable submission should succeed for ticket %s", ticketId.Hex())

	// Parse RedeemScheduled event to get auto-redeem tx hash
	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(
		common.HexToAddress("6e"), p.builder.L2.Client)
	require.NoError(t, err)
	var retryTxHash common.Hash
	foundEvent := false
	for _, log := range submissionReceipt.Logs {
		event, err := arbRetryableFilterer.ParseRedeemScheduled(*log)
		if err != nil {
			continue
		}
		retryTxHash = common.Hash(event.RetryTxHash)
		foundEvent = true
		break
	}

	// Wait for auto-redeem tx and verify it failed
	autoRedeemReceipt, err := WaitForTx(p.ctx, p.builder.L2.Client, retryTxHash, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, autoRedeemReceipt.Status,
		"auto-redeem should fail for ticket %s (insufficient gas for inner call)", ticketId.Hex())
	require.True(t, foundEvent, "RedeemScheduled event should be emitted for ticket %s", ticketId.Hex())
}

// processRetryableSubmission advances L1 and waits for the retryable submission
// to be processed on L2. Verifies the submission receipt has the expected status.
func processRetryableSubmission(
	t *testing.T,
	p *retryableFilterTestParams,
	ticketId common.Hash,
	expectedStatus uint64,
) {
	t.Helper()
	advanceL1ForDelayed(t, p.ctx, p.builder)
	receipt, err := WaitForTx(p.ctx, p.builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, receipt.Status,
		"retryable submission status mismatch for ticket %s", ticketId.Hex())
}

// verifyTicketExistsWithNumTries verifies a retryable ticket exists and checks
// its numTries by clearing the filter, performing a manual redeem, and inspecting
// the RedeemScheduled event's SequenceNum field.
// Note: this function clears the address filter as a side effect.
func verifyTicketExistsWithNumTries(
	t *testing.T,
	ctx context.Context,
	builder *NodeBuilder,
	ticketId common.Hash,
	expectedSequenceNum uint64,
) {
	t.Helper()

	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{Context: ctx}, ticketId)
	require.NoError(t, err, "retryable ticket %s should exist", ticketId.Hex())

	// Clear filter for a clean redeem
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(nil)

	redeemerName := "Redeemer_" + ticketId.Hex()[:4]
	builder.L2Info.GenerateAccount(redeemerName)
	builder.L2.TransferBalance(t, "Owner", redeemerName, big.NewInt(1e18), builder.L2Info)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts(redeemerName, ctx)
	redeemTx, err := arbRetryable.Redeem(&redeemOpts, ticketId)
	require.NoError(t, err)
	redeemReceipt, err := builder.L2.EnsureTxSucceeded(redeemTx)
	require.NoError(t, err)

	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	foundEvent := false
	for _, log := range redeemReceipt.Logs {
		event, err := arbRetryableFilterer.ParseRedeemScheduled(*log)
		if err != nil {
			continue
		}
		require.Equal(t, expectedSequenceNum, event.SequenceNum,
			"numTries mismatch for ticket %s", ticketId.Hex())
		foundEvent = true

	}
	require.True(t, foundEvent, "successful redeem should emit RedeemScheduled event for ticket %s", ticketId.Hex())
}

// verifyTicketExists verifies that a retryable ticket exists (has not been deleted).
func verifyTicketExists(t *testing.T, ctx context.Context, builder *NodeBuilder, ticketId common.Hash) {
	t.Helper()
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{Context: ctx}, ticketId)
	require.NoError(t, err, "retryable ticket %s should exist", ticketId.Hex())
}

// verifyTicketDeleted verifies that a retryable ticket has been deleted (redeemed).
func verifyTicketDeleted(t *testing.T, ctx context.Context, builder *NodeBuilder, ticketId common.Hash) {
	t.Helper()
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{Context: ctx}, ticketId)
	require.Error(t, err, "retryable ticket %s should be deleted after successful redeem", ticketId.Hex())
}

// manualRedeemSucceeds clears the filter, performs a manual redeem of ticketA,
// and verifies it succeeds. Used after cascading-filter tests to prove the
// chain works once the filter is cleared.
func manualRedeemSucceeds(t *testing.T, ctx context.Context, builder *NodeBuilder, ticketId common.Hash) {
	t.Helper()
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(nil)

	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)

	builder.L2Info.GenerateAccount("Redeemer_test")
	builder.L2.TransferBalance(t, "Owner", "Redeemer_test", big.NewInt(1e18), builder.L2Info)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer_test", ctx)
	redeemTx, err := arbRetryable.Redeem(&redeemOpts, ticketId)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(redeemTx)
	require.NoError(t, err, "manual redeem of ticket %s should succeed after clearing filter", ticketId.Hex())
}

// ============================================================================
// Part A: Deep Linear Cascade — Auto-Redeem Path (L1 submission)
// ============================================================================

// TestAutoRedeemFilteredDepth1 tests the base case: a single retryable A whose
// auto-redeem directly calls callTarget(filteredTarget). No inner tickets.
// The group reverts, ticket survives, manual redeem succeeds after clearing filter.
func TestRetryableFilteringAutoRedeemFilteredDepth1(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller and filtered target contracts
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)

	// Set filter on filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=callerAddr, data=callTarget(filteredTarget)). Advance L1.
	retryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// A's auto-redeem → callTarget(filteredTarget) → filter → group revert
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// After clearing filter, manual redeem of A succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestAutoRedeemCascadeDepth2 tests A→B→filtered via auto-redeem.
// A's auto-redeem calls redeem(ticketB), B's redeem calls callTarget(filteredTarget).
// The entire group reverts, both tickets survive, manual redeem succeeds after clearing.
func TestRetryableFilteringAutoRedeemCascadeDepth2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller and filtered target contracts
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit B (gasLimit=0, data=callTarget(filteredTarget)). Process submission.
	bRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set filter for filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableViaL1(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// A's auto-redeem → B's redeem → callTarget(filteredTarget) → filter → group revert.
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// B should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)

	// After clearing filter, manual redeem of A succeeds (chains B, both complete)
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestAutoRedeemCascadeDepth3 tests A→B→C→filtered via auto-redeem.
// A's auto-redeem calls redeem(ticketB), B's redeem calls redeem(ticketC),
// C's redeem calls callTarget(filteredTarget). The entire group reverts.
func TestRetryableFilteringAutoRedeemCascadeDepth3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller and filtered target contracts
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit C (gasLimit=0, data=callTarget(filteredTarget)). Process submission.
	cRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	// Submit B (gasLimit=0, destAddr=0x6e, data=redeem(ticketC)). Process submission.
	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set filter for filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)

	gasLimit := big.NewInt(1e6) // enough for A and B to redeem, but C's call to filteredTarget runs out of gas and triggers the filter
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// A's auto-redeem → B's redeem → C's redeem → C touches filtered → group revert.
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// B and C should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)

	// After clearing filter, manual redeem of A succeeds (chains B→C, all complete)
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestAutoRedeemCascadeDepth4 tests A→B→C→D→filtered via auto-redeem.
// Proves arbitrary depth works.
func TestRetryableFilteringAutoRedeemCascadeDepth4(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit D (gasLimit=0, data=callTarget(filteredTarget)). Process.
	dRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdD := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, dRetryData,
	)
	processRetryableSubmission(t, p, ticketIdD, types.ReceiptStatusSuccessful)

	// Submit C (gasLimit=0, destAddr=0x6e, data=redeem(ticketD)). Process.
	cRetryData, err := arbRetryableABI.Pack("redeem", ticketIdD)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	// Submit B (gasLimit=0, destAddr=0x6e, data=redeem(ticketC)). Process.
	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set filter for filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	gasLimit := big.NewInt(1e6) // enough for A, B, C to redeem, but D's call to filteredTarget runs out of gas and triggers the filter
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// B, C, D should all still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)
	verifyTicketExists(t, ctx, builder, ticketIdD)

	// After clearing filter, manual redeem of A chains through all
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// ============================================================================
// Part B: Deep Linear Cascade — L2 Manual Redeem Path
// ============================================================================

// TestL2ManualRedeemCascadeDepth2 tests L2 manual redeem of A → B → filtered.
func TestRetryableFilteringL2ManualRedeemCascadeDepth2(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit B (gasLimit=0, data=callTarget(filteredTarget)). Process.
	bRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Submit A (gasLimit=0, destAddr=0x6e, data=redeem(ticketB)). Process.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)
	processRetryableSubmission(t, p, ticketIdA, types.ReceiptStatusSuccessful)

	// Set filter for filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send L2 manual redeem of A
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	_, err = arbRetryable.Redeem(&redeemOpts, ticketIdA)
	require.ErrorContains(t, err, "cascading redeem filtered",
		"manual redeem should fail with cascading redeem filter error")

	// A and B should still exist
	verifyTicketExists(t, ctx, builder, ticketIdA)
	verifyTicketExists(t, ctx, builder, ticketIdB)

	// After clearing filter, manual redeem succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestL2ManualRedeemCascadeDepth3 tests L2 manual redeem of A → B → C → filtered.
func TestRetryableFilteringL2ManualRedeemCascadeDepth3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit C (gasLimit=0, data=callTarget(filteredTarget)). Process.
	cRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	// Submit B (gasLimit=0, destAddr=0x6e, data=redeem(ticketC)). Process.
	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Submit A (gasLimit=0, destAddr=0x6e, data=redeem(ticketB)). Process.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)
	processRetryableSubmission(t, p, ticketIdA, types.ReceiptStatusSuccessful)

	// Set filter
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send L2 manual redeem of A
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	_, err = arbRetryable.Redeem(&redeemOpts, ticketIdA)
	require.ErrorContains(t, err, "cascading redeem filtered",
		"manual redeem should fail with cascading redeem filter error")

	// A, B, C should still exist
	verifyTicketExists(t, ctx, builder, ticketIdA)
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)

	// After clearing filter, manual redeem of A chains through B→C
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// ============================================================================
// Part C: Deep Linear Cascade — L1 Delayed Manual Redeem Path
// ============================================================================

// TestL1DelayedManualRedeemCascadeDepth2 tests delayed L1 manual redeem of
// A → B → filtered. The group revert fires on the L2 tx hash (NOT ticketA).
func TestRetryableFilteringL1DelayedManualRedeemCascadeDepth2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit B (gasLimit=0, data=callTarget(filteredTarget)). Process.
	bRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Submit A (gasLimit=0, destAddr=0x6e, data=redeem(ticketB)). Process.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)
	processRetryableSubmission(t, p, ticketIdA, types.ReceiptStatusSuccessful)

	// Set filter
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send delayed manual redeem (signed L2 tx via L1 inbox calling ArbRetryableTx.redeem(ticketA))
	redeemCallData, err := arbRetryableABI.Pack("redeem", ticketIdA)
	require.NoError(t, err)
	signedL2Tx := prepareDelayedContractCall(t, builder, "Redeemer", arbRetryableTxAddr, redeemCallData)
	l2TxHash := sendDelayedTx(t, ctx, builder, signedL2Tx)
	advanceL1ForDelayed(t, ctx, builder)

	// Group revert fires on L2 tx hash (NOT ticketA)
	require.NotEqual(t, ticketIdA, l2TxHash, "L2 tx hash must differ from ticketA")
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{l2TxHash}, 10*time.Second)

	// Add l2TxHash to onchain filter + resume
	addTxHashToOnChainFilter(t, ctx, builder, l2TxHash, p.filtererName)
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	// Verify the delayed redeem receipt has failed status
	redeemReceipt, err := WaitForTx(ctx, builder.L2.Client, l2TxHash, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, redeemReceipt.Status,
		"delayed manual redeem should fail (filtered)")

	// A and B should still exist
	verifyTicketExists(t, ctx, builder, ticketIdA)
	verifyTicketExists(t, ctx, builder, ticketIdB)

	// After clearing filter, clean manual redeem succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestL1DelayedManualRedeemCascadeDepth3 tests delayed L1 manual redeem of
// A → B → C → filtered. Depth 3 via L1 delayed.
func TestRetryableFilteringL1DelayedManualRedeemCascadeDepth3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit C, B, A (all gasLimit=0). Process all.
	cRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)
	processRetryableSubmission(t, p, ticketIdA, types.ReceiptStatusSuccessful)

	// Set filter
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send delayed manual redeem of A
	redeemCallData, err := arbRetryableABI.Pack("redeem", ticketIdA)
	require.NoError(t, err)
	signedL2Tx := prepareDelayedContractCall(t, builder, "Redeemer", arbRetryableTxAddr, redeemCallData)
	l2TxHash := sendDelayedTx(t, ctx, builder, signedL2Tx)
	advanceL1ForDelayed(t, ctx, builder)

	// Group revert fires on L2 tx hash
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{l2TxHash}, 10*time.Second)
	addTxHashToOnChainFilter(t, ctx, builder, l2TxHash, p.filtererName)
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	redeemReceipt, err := WaitForTx(ctx, builder.L2.Client, l2TxHash, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, redeemReceipt.Status)

	// A, B, C should all still exist
	verifyTicketExists(t, ctx, builder, ticketIdA)
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)

	// After clearing filter, clean manual redeem succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// ============================================================================
// Part D: Filter Detection Variants
// ============================================================================

// TestAutoRedeemCascadeDepth2_EventFilter tests A→B via auto-redeem, B emits Transfer
// event to filtered address. Event filter catches it.
func TestRetryableFilteringAutoRedeemCascadeDepth2_EventFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector, _, err := eventfilter.CanonicalSelectorFromEvent("Transfer(address,address,uint256)")
	require.NoError(t, err)
	rules := []eventfilter.EventRule{{
		Event:          "Transfer(address,address,uint256)",
		Selector:       selector,
		TopicAddresses: []int{1, 2},
	}}

	p, cleanup := setupRetryableFilterTest(t, ctx, true, rules)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("FilteredTarget")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredTarget")

	contractAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	contractABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit B (gasLimit=0, destAddr=contractAddr, data=emitTransfer(cleanAddr, filteredAddr)). Process.
	bRetryData, err := contractABI.Pack("emitTransfer", cleanBeneficiary, filteredAddr)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", contractAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set address filter
	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(addrFilter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableViaL1(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// B emits Transfer to filtered → event filter triggers group revert
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// B should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)

	// After clearing filter, manual redeem of A succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestAutoRedeemCascadeDepth3_EventFilter tests A→B→C via auto-redeem, C emits Transfer
// event to filtered address. Depth 3 event filter.
func TestRetryableFilteringAutoRedeemCascadeDepth3_EventFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector, _, err := eventfilter.CanonicalSelectorFromEvent("Transfer(address,address,uint256)")
	require.NoError(t, err)
	rules := []eventfilter.EventRule{{
		Event:          "Transfer(address,address,uint256)",
		Selector:       selector,
		TopicAddresses: []int{1, 2},
	}}

	p, cleanup := setupRetryableFilterTest(t, ctx, true, rules)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("FilteredTarget")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredTarget")

	contractAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	contractABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit C (gasLimit=0, destAddr=contractAddr, data=emitTransfer(cleanAddr, filteredAddr)). Process.
	cRetryData, err := contractABI.Pack("emitTransfer", cleanBeneficiary, filteredAddr)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", contractAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	// Submit B (gasLimit=0, destAddr=0x6e, data=redeem(ticketC)). Process.
	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set address filter
	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(addrFilter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	gasLimit := big.NewInt(1e6)
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// B and C should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)

	// After clearing filter, manual redeem of A succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
}

// TestAutoRedeemFilteredDepth1_Create2 tests A's auto-redeem CREATE2s at a
// pre-computed filtered address. Verifies no contract is created after group revert.
func TestRetryableFilteringAutoRedeemFilteredDepth1_Create2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, caller := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Pre-compute CREATE2 address
	salt := [32]byte{42}
	create2Addr, err := caller.ComputeCreate2Address(&bind.CallOpts{Context: ctx}, salt)
	require.NoError(t, err)

	// Set filter for computed address
	filter := newHashedChecker([]common.Address{create2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("create2Contract", salt)
	require.NoError(t, err)

	// Submit A (gasLimit>0, destAddr=callerAddr, data=create2Contract(salt))
	_, ticketIdA := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// Verify no contract code at CREATE2 address
	code, err := builder.L2.Client.CodeAt(ctx, create2Addr, nil)
	require.NoError(t, err)
	require.Empty(t, code, "no contract should exist at filtered CREATE2 address after group revert")

	// After clearing filter + manual redeem, contract IS created.
	// Note: we inline the redeem instead of using manualRedeemSucceeds because
	// we need to verify code != empty at the CREATE2 address after redeem.
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(nil)

	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	redeemTx, err := arbRetryable.Redeem(&redeemOpts, ticketIdA)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(redeemTx)
	require.NoError(t, err)

	code, err = builder.L2.Client.CodeAt(ctx, create2Addr, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code, "contract should be created at CREATE2 address after clearing filter")
}

// ============================================================================
// Part E: CallValue with Cascade
// ============================================================================

// TestAutoRedeemCascadeWithCallValue tests B has callValue > 0, A→B→filtered
// via auto-redeem. Verifies escrow accounting rollback on B.
func TestRetryableFilteringAutoRedeemCascadeWithCallValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	callValue := big.NewInt(1e6)

	// Submit B (gasLimit=0, callValue=1e6, data=callTarget(filteredTarget)). Process.
	bRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, callValue, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set filter
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, callValue=0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	gasLimit := big.NewInt(1e6)
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// Verify B's escrow address still holds callValue
	escrowAddr := retryables.RetryableEscrowAddress(ticketIdB)
	state, err := builder.L2.ExecNode.ArbInterface.BlockChain().State()
	require.NoError(t, err)
	escrowBalance := state.GetBalance(escrowAddr)
	require.Equal(t, callValue, escrowBalance.ToBig(), "B's escrow should hold the call value")

	// B should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)

	// After clearing filter, manual redeem of A succeeds and B's escrow is drained
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
	state, err = builder.L2.ExecNode.ArbInterface.BlockChain().State()
	require.NoError(t, err)
	escrowBalance = state.GetBalance(escrowAddr)
	require.True(t, escrowBalance.IsZero(), "B's escrow should be drained after successful redeem")
}

// TestL2ManualRedeemCascadeWithCallValue tests B has callValue > 0, A→B→filtered
// via L2 manual redeem. Verifies escrow accounting rollback on B.
func TestRetryableFilteringL2ManualRedeemCascadeWithCallValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	callValue := big.NewInt(1e6)

	// Submit B (gasLimit=0, callValue=1e6, data=callTarget(filteredTarget)). Process.
	bRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, callValue, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Submit A (gasLimit=0, callValue=0, destAddr=0x6e, data=redeem(ticketB)). Process.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	_, ticketIdA := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData,
	)
	processRetryableSubmission(t, p, ticketIdA, types.ReceiptStatusSuccessful)

	// Verify B's escrow holds callValue
	escrowAddr := retryables.RetryableEscrowAddress(ticketIdB)
	state, err := builder.L2.ExecNode.ArbInterface.BlockChain().State()
	require.NoError(t, err)
	escrowBalance := state.GetBalance(escrowAddr)
	require.Equal(t, callValue, escrowBalance.ToBig(), "B's escrow should hold the call value before redeem")

	// Set filter
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send L2 manual redeem of A
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	_, err = arbRetryable.Redeem(&redeemOpts, ticketIdA)
	require.ErrorContains(t, err, "cascading redeem filtered",
		"manual redeem should fail with cascading redeem filter error")

	// Verify B's escrow still holds callValue
	state, err = builder.L2.ExecNode.ArbInterface.BlockChain().State()
	require.NoError(t, err)
	escrowBalance = state.GetBalance(escrowAddr)
	require.Equal(t, callValue, escrowBalance.ToBig(), "B's escrow should still hold the call value after failed redeem")

	// B should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)

	// After clearing filter, manual redeem of A succeeds and B's escrow is drained
	manualRedeemSucceeds(t, ctx, builder, ticketIdA)
	state, err = builder.L2.ExecNode.ArbInterface.BlockChain().State()
	require.NoError(t, err)
	escrowBalance = state.GetBalance(escrowAddr)
	require.True(t, escrowBalance.IsZero(), "B's escrow should be drained after successful redeem")
}

// ============================================================================
// Part F: State Preservation and Multi-Retryable
// ============================================================================

// TestStorageRollbackAtIntermediateChainLevel tests A→B→C→filtered.
// B's execution does dummy++ THEN chains into C via incrementDummyThenRedeem(ticketC).
// Verifies B's storage write is rolled back.
func TestRetryableFilteringStorageRollbackAtIntermediateChainLevel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller contract (will be used by B for incrementDummyThenRedeem)
	callerAddr, callerContract := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	// Deploy a separate caller for C's inner execution
	innerCallerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Record dummy value before
	dummyBefore, err := callerContract.Dummy(&bind.CallOpts{})
	require.NoError(t, err)

	// Submit C (gasLimit=0, data=callTarget(filteredTarget)). Process.
	cRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", innerCallerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	// Submit B (gasLimit=0, destAddr=callerAddr, data=incrementDummyThenRedeem(ticketC)). Process.
	bRetryData, err := callerABI.Pack("incrementDummyThenRedeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Set filter
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	gasLimit := big.NewInt(1e6)
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// Verify dummy counter unchanged (B's dummy++ rolled back)
	dummyAfter, err := callerContract.Dummy(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, dummyBefore, dummyAfter,
		"dummy counter should be unchanged after group revert (B's dummy++ was rolled back)")

	// B and C should still exist
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)
}

// TestCleanRetryableBeforeDeepDirtyChain tests that a clean retryable
// (auto-redeem succeeds) processed before a dirty depth-3 chain (A→B→C→filtered)
// is unaffected. The clean retryable's block is committed; the dirty chain's
// revert doesn't affect it.
func TestRetryableFilteringCleanRetryableBeforeDeepDirtyChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	cleanTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit C, B for dirty chain (all gasLimit=0). Process all.
	cRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	processRetryableSubmission(t, p, ticketIdC, types.ReceiptStatusSuccessful)

	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableNoAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	processRetryableSubmission(t, p, ticketIdB, types.ReceiptStatusSuccessful)

	// Submit clean retryable X (gasLimit>0, clean data). Process.
	cleanRetryData, err := callerABI.Pack("callTarget", cleanTarget)
	require.NoError(t, err)
	_, cleanTicketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, cleanRetryData,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// Clean X should succeed (processed first, group finalized before dirty)
	cleanReceipt, err := WaitForTx(ctx, builder.L2.Client, cleanTicketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, cleanReceipt.Status,
		"clean retryable submission should succeed")

	// Clean retryable's auto-redeem should have succeeded (ticket deleted)
	verifyTicketDeleted(t, ctx, builder, cleanTicketId)

	// Set filter for dirty chain
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	gasLimit := big.NewInt(1e6)
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// Dirty A halts via verifyCascadingRedeemFiltered
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA,
		p.filtererName, p.fundsRecipientAddr)

	// B and C survive
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)

	// Clean X still processed (ticket deleted confirms earlier success is preserved)
	verifyTicketDeleted(t, ctx, builder, cleanTicketId)
}

// TestAutoRedeemCascadeDepth3_NumTriesReset verifies numTries restoration at cascade depth 3.
// Inner tickets B and C each get a failed auto-redeem (numTries=1) before the
// cascade. After revert, both must have numTries=1.
// Cascade: A (auto-redeem) → B → C → filteredTarget
func TestRetryableFilteringAutoRedeemCascadeDepth3_NumTriesReset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller and filtered target contracts
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit C with failing auto-redeem (data=callTarget(filteredTarget)).
	cRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketIdC := submitRetryableWithFailingAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, cRetryData,
	)
	// Process C: submission succeeds, auto-redeem fails → C.numTries=1
	processRetryableWithFailingAutoRedeem(t, p, ticketIdC)

	// Submit B with failing auto-redeem (destAddr=0x6e, data=redeem(ticketC)).
	// B's auto-redeem tries redeem(C) but fails due to insufficient gas.
	bRetryData, err := arbRetryableABI.Pack("redeem", ticketIdC)
	require.NoError(t, err)
	_, ticketIdB := submitRetryableWithFailingAutoRedeem(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, bRetryData,
	)
	// Process B: submission succeeds, auto-redeem fails → B.numTries=1
	processRetryableWithFailingAutoRedeem(t, p, ticketIdB)

	// Set filter on filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit A (gasLimit>0, destAddr=0x6e, data=redeem(ticketB)). Advance L1.
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)
	gasLimit := big.NewInt(1e6)
	_, ticketIdA := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, aRetryData, gasLimit,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// A's auto-redeem → redeem(B) → redeem(C) → callTarget(filteredTarget) → filter → group revert
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA, p.filtererName, p.fundsRecipientAddr)

	// B and C should still exist after group revert
	verifyTicketExists(t, ctx, builder, ticketIdB)
	verifyTicketExists(t, ctx, builder, ticketIdC)

	// Verify leaf-to-root: consume leaf first, then work upward.

	// C.numTries restored to 1. Clears filter, redeems C → callTarget(filteredTarget) succeeds → C consumed.
	verifyTicketExistsWithNumTries(t, ctx, builder, ticketIdC, 1)

	// Confirm C and A were consumed
	verifyTicketDeleted(t, ctx, builder, ticketIdC)
}

// ============================================================================
// Part G: L2 Contract Chain → Single Retryable Redeem
// ============================================================================

// TestL2ContractChainToRedeemFiltered tests that filtering works when a Redeem()
// is buried behind multiple levels of L2 contract indirection (not called
// directly from an EOA or from another retryable's inner execution).
// Flow: L2 EOA tx → contractA.forwardCall(contractB, ...) →
//
//	contractB.forwardCall(0x6e, ...) → ArbRetryableTx.Redeem(ticketId) →
//	inner tx touches filteredTarget → entire group should revert.
func TestRetryableFilteringL2ContractChainToRedeemFiltered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy two forwarder contracts and a filtered target
	_, contractA := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	contractBAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")

	// Submit retryable ticket (gasLimit=0) whose inner tx calls callTarget(filteredTarget)
	retryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	_, ticketId := submitRetryableNoAutoRedeem(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)
	processRetryableSubmission(t, p, ticketId, types.ReceiptStatusSuccessful)

	// Set the address filter on filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Build the nested call chain:
	// Inner: ArbRetryableTx.redeem(ticketId) — what contractB will call on 0x6e
	innerCalldata, err := arbRetryableABI.Pack("redeem", ticketId)
	require.NoError(t, err)

	// Middle: contractB.forwardCall(0x6e, innerCalldata) — what contractA will call on contractB
	middleCalldata, err := callerABI.Pack("forwardCall", arbRetryableTxAddr, innerCalldata)
	require.NoError(t, err)

	// Send L2 tx from EOA: contractA.forwardCall(contractB, middleCalldata)
	// This creates: EOA → contractA → contractB → 0x6e.Redeem(ticketId)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	_, err = contractA.ForwardCall(&redeemOpts, contractBAddr, middleCalldata)
	require.ErrorContains(t, err, "cascading redeem filtered",
		"L2 contract chain to redeem should fail with cascading redeem filter error")

	// Verify the retryable ticket still exists (redeem was rolled back)
	verifyTicketExists(t, ctx, builder, ticketId)

	// Clear filter, verify manual redeem succeeds
	manualRedeemSucceeds(t, ctx, builder, ticketId)
}
