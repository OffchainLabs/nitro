package arbtest

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func setupStylusForFilterTest(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	t.Helper()
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	require.NoError(t, err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	tx, err := arbDebug.BecomeChainOwner(&auth)
	require.NoError(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	require.NoError(t, err)

	tx, err = arbOwner.SetInkPrice(&auth, 10000)
	require.NoError(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	require.NoError(t, err)
}

// deployStylusStorageContract deploys and activates the Stylus storage test
// contract. Returns the deployed address.
func deployStylusStorageContract(
	t *testing.T, ctx context.Context, builder *NodeBuilder,
) common.Address {
	t.Helper()
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	return deployWasm(t, ctx, auth, builder.L2.Client, rustFile("storage"))
}

// TestRetryableFilteringStylusSandwichRollback verifies that a group rollback
// of a Stylus redeem chain does not affect neighboring transactions in the same
// block. Three L2 transactions are forced into one block via sequencer
// Pause/Activate:
//   - TX1: writes keyBefore to multicall's storage
//   - TX2: manual redeem that triggers a Stylus chain (multicall writes
//     keyRedeem + CALLs filtered Stylus contract) → group rollback
//   - TX3: writes keyAfter to multicall's storage
//
// After the block: TX1 and TX3's writes must persist, TX2's write must be gone.
func TestRetryableFilteringStylusSandwichRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder
	setupStylusForFilterTest(t, ctx, builder)

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy and activate Stylus multicall contract M (holds the shared storage)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	multicallAddr := deployWasm(t, ctx, auth, builder.L2.Client, rustFile("multicall"))

	// Deploy and activate another Stylus contract B (will be the filtered address)
	filteredStylusAddr := deployStylusStorageContract(t, ctx, builder)

	// Storage keys and values
	keyBefore := common.HexToHash("0x0001")
	valueBefore := common.HexToHash("0xaaaa")
	keyRedeem := common.HexToHash("0x0002")
	valueRedeem := common.HexToHash("0xbbbb")
	keyAfter := common.HexToHash("0x0003")
	valueAfter := common.HexToHash("0xcccc")

	// --- Step 1: Submit retryable (no auto-redeem) ---
	// Inner call: multicall M stores keyRedeem=valueRedeem, then CALLs filteredStylusAddr.
	redeemArgs := multicallEmptyArgs()
	redeemArgs = multicallAppendStore(redeemArgs, keyRedeem, valueRedeem, false, false)
	redeemArgs = multicallAppend(redeemArgs, vm.CALL, filteredStylusAddr, multicallEmptyArgs())

	_, ticketId := submitRetryableNoAutoRedeem(
		t, p, "Faucet", multicallAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, redeemArgs,
	)
	processRetryableSubmission(t, p, ticketId, types.ReceiptStatusSuccessful)

	// --- Step 2: Set filter and prepare sandwich txns ---
	filter := newHashedChecker([]common.Address{filteredStylusAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	sequencer := builder.L2.ExecNode.Sequencer

	// Prepare TX1: write keyBefore=valueBefore to multicall M
	tx1Args := multicallEmptyArgs()
	tx1Args = multicallAppendStore(tx1Args, keyBefore, valueBefore, false, false)
	builder.L2Info.GenerateAccount("Sender1")
	builder.L2.TransferBalance(t, "Owner", "Sender1", big.NewInt(1e18), builder.L2Info)
	tx1 := builder.L2Info.PrepareTxTo("Sender1", &multicallAddr, 1e9, nil, tx1Args)

	// Prepare TX2: manual redeem of the retryable
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	redeemCalldata, err := arbRetryableABI.Pack("redeem", ticketId)
	require.NoError(t, err)
	arbRetryableTxAddr := common.HexToAddress("6e")
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	tx2 := builder.L2Info.PrepareTxTo("Redeemer", &arbRetryableTxAddr, 1e7, nil, redeemCalldata)

	// Prepare TX3: write keyAfter=valueAfter to multicall M
	builder.L2Info.GenerateAccount("Sender2")
	builder.L2.TransferBalance(t, "Owner", "Sender2", big.NewInt(1e18), builder.L2Info)
	tx3Args := multicallEmptyArgs()
	tx3Args = multicallAppendStore(tx3Args, keyAfter, valueAfter, false, false)
	tx3 := builder.L2Info.PrepareTxTo("Sender2", &multicallAddr, 1e9, nil, tx3Args)

	// --- Pause sequencer, queue all 3, resume → same block ---
	sequencer.Pause()

	var wg sync.WaitGroup
	var tx1Err, tx2Err, tx3Err error
	wg.Add(3)

	go func() {
		defer wg.Done()
		tx1Err = sequencer.PublishTransaction(ctx, tx1, nil)
	}()
	go func() {
		defer wg.Done()
		tx2Err = sequencer.PublishTransaction(ctx, tx2, nil)
	}()
	go func() {
		defer wg.Done()
		tx3Err = sequencer.PublishTransaction(ctx, tx3, nil)
	}()

	sequencer.Activate()
	wg.Wait()

	require.NoError(t, tx1Err, "TX1 should have been published successfully")
	require.NoError(t, tx3Err, "TX3 should have been published successfully")

	// TX2 should have failed (cascading redeem filtered)
	require.Error(t, tx2Err, "redeem should have been rejected by filter")
	require.ErrorContains(t, tx2Err, "cascading redeem filtered")

	// Wait for TX1 and TX3 receipts
	tx1Receipt, err := builder.L2.EnsureTxSucceeded(tx1)
	require.NoError(t, err)
	tx3Receipt, err := builder.L2.EnsureTxSucceeded(tx3)
	require.NoError(t, err)

	// Verify same block
	require.Equal(t, tx1Receipt.BlockNumber.Uint64(), tx3Receipt.BlockNumber.Uint64(),
		"TX1 and TX3 should be in the same block")

	// --- Step 3: Validation ---
	// TX1's write preserved
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyBefore, valueBefore)

	// TX2's write rolled back
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyRedeem, common.Hash{})

	// TX3's write preserved
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyAfter, valueAfter)
}

// TestRetryableFilteringStylusDelayedSandwichRollback is the L1 version of the
// sandwich test. Three retryables with auto-redeem are submitted via the delayed
// inbox. TX2's auto-redeem triggers the filter, causing the delayed sequencer to
// halt. After the operator adds TX2 to the onchain filter, the delayed sequencer
// resumes, TX2 replays as filtered (no auto-redeem), and TX3 processes normally.
//
// Validates:
//   - TX1's storage write persists (committed in a prior block before TX2 halted)
//   - TX2's storage write never happens (rolled back, then replayed without auto-redeem)
//   - TX3's storage write persists (processed after resume)
//   - Delayed sequencer halt/resume cycle works correctly with Stylus contracts
func TestRetryableFilteringStylusDelayedSandwichRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder
	setupStylusForFilterTest(t, ctx, builder)

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy and activate Stylus multicall contract M (holds the shared storage)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	multicallAddr := deployWasm(t, ctx, auth, builder.L2.Client, rustFile("multicall"))

	// Deploy and activate another Stylus contract B (will be the filtered address)
	filteredStylusAddr := deployStylusStorageContract(t, ctx, builder)

	// Storage keys and values
	keyBefore := common.HexToHash("0x0001")
	valueBefore := common.HexToHash("0xaaaa")
	keyRedeem := common.HexToHash("0x0002")
	valueRedeem := common.HexToHash("0xbbbb")
	keyAfter := common.HexToHash("0x0003")
	valueAfter := common.HexToHash("0xcccc")

	// Set filter on B's address BEFORE submitting any retryables
	filter := newHashedChecker([]common.Address{filteredStylusAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// --- Submit 3 retryables via L1 (all with auto-redeem) ---

	// TX1: multicall M stores keyBefore=valueBefore (clean, no call to filtered)
	tx1Args := multicallEmptyArgs()
	tx1Args = multicallAppendStore(tx1Args, keyBefore, valueBefore, false, false)
	_, ticketId1 := submitRetryableViaL1(
		t, p, "Faucet", multicallAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, tx1Args,
	)

	// TX2: multicall M stores keyRedeem=valueRedeem + CALLs filteredStylusAddr (triggers filter)
	tx2Args := multicallEmptyArgs()
	tx2Args = multicallAppendStore(tx2Args, keyRedeem, valueRedeem, false, false)
	tx2Args = multicallAppend(tx2Args, vm.CALL, filteredStylusAddr, multicallEmptyArgs())
	_, ticketId2 := submitRetryableViaL1(
		t, p, "Faucet", multicallAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, tx2Args,
	)

	// TX3: multicall M stores keyAfter=valueAfter (clean, no call to filtered)
	tx3Args := multicallEmptyArgs()
	tx3Args = multicallAppendStore(tx3Args, keyAfter, valueAfter, false, false)
	_, ticketId3 := submitRetryableViaL1(
		t, p, "Faucet", multicallAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, tx3Args,
	)

	// --- Advance L1 to trigger delayed sequencer processing ---
	advanceL1ForDelayed(t, ctx, builder)

	// TX1 processes OK. TX2's auto-redeem triggers filter → delayed sequencer halts.
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId2}, 30*time.Second)

	// Verify TX1 already committed (its block is done)
	_, err := WaitForTx(ctx, builder.L2.Client, ticketId1, 10*time.Second)
	require.NoError(t, err)
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyBefore, valueBefore)

	// Verify TX2's write was rolled back
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyRedeem, common.Hash{})

	// Verify TX3 not yet processed (sequencer halted before reaching it)
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyAfter, common.Hash{})

	// --- Resolve the halt: add TX2 to onchain filter, resume ---
	addTxHashToOnChainFilter(t, ctx, builder, ticketId2, p.filtererName)
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	// Wait for TX3 to process
	_, err = WaitForTx(ctx, builder.L2.Client, ticketId3, 30*time.Second)
	require.NoError(t, err)

	// --- Final validation ---
	// TX1's write preserved
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyBefore, valueBefore)

	// TX2's write still absent (replayed as filtered, no auto-redeem executed)
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyRedeem, common.Hash{})

	// TX3's write succeeded
	assertStorageAt(t, ctx, builder.L2.Client, multicallAddr, keyAfter, valueAfter)
}
