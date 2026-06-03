// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/bold/api/db"
	"github.com/offchainlabs/nitro/bold/assertions"
	"github.com/offchainlabs/nitro/bold/challenge"
	modes "github.com/offchainlabs/nitro/bold/challenge/types"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/protocol/sol"
	"github.com/offchainlabs/nitro/bold/state"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

// TestBoldSelfChallengeRepro is the system-level regression gate for the
// same-hash short-circuit in maybePostRivalAssertionAndChallenge. It posts
// one canonical child Y on-chain, then runs a full challenge.Stack with a
// flaky ExecutionProvider that returns a wrong EndHistoryRoot on the first
// call for Y's batch — the prod state-provider race.
//
// The bug's signature is HandleCorrectRival being invoked with Y's own hash
// (matching the production log "correctRivalAssertionHash == detectedAssertionHash").
// Detection is a recording wrapper around the challenge manager's
// HandleCorrectRival. The require.Empty at the bottom passes when the
// short-circuit is in place, fails when it is removed.
//
// Note: this test does not exercise the cursor-downgrade path in
// applyRecordAgreedAssertion — see TestRecordAgreedAssertionDoesNotDowngradeLatestAgreedAssertion
// for that side.
func TestBoldSelfChallengeRepro(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repro := setupBoldSelfChallengeRepro(t, ctx)
	defer requireClose(t, repro.l1stack)
	defer repro.l2node.StopAndWait()

	// Grant sequencer batch-poster rights and post two batches.
	sequencerTxOpts := repro.l1info.GetDefaultTransactOpts("Sequencer", ctx)
	seqInbox := repro.l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInbox, repro.l1client)
	Require(t, err)
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err)
	upgradeExec, err := mocksgen.NewUpgradeExecutorMock(repro.l1info.GetAddress("UpgradeExecutor"), repro.l1client)
	Require(t, err)
	data, err := seqInboxABI.Pack("setIsBatchPoster", sequencerTxOpts.From, true)
	Require(t, err)
	rollupOwnerOpts := repro.l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = upgradeExec.ExecuteCall(&rollupOwnerOpts, seqInbox, data)
	Require(t, err)

	repro.l2info.GenerateAccount("Destination")
	makeBoldBatch(t, repro.l2node, repro.l2info, repro.l1client, &sequencerTxOpts, seqInboxBinding, seqInbox, 5, -1)
	makeBoldBatch(t, repro.l2node, repro.l2info, repro.l1client, &sequencerTxOpts, seqInboxBinding, seqInbox, 5, -1)

	bridgeBinding, err := bridgegen.NewBridge(repro.l1info.GetAddress("Bridge"), repro.l1client)
	Require(t, err)
	totalBatchesBig, err := bridgeBinding.SequencerMessageCount(&bind.CallOpts{Context: ctx})
	Require(t, err)
	totalBatches := totalBatchesBig.Uint64()

	pollUntil(t, ctx, 5*time.Minute, 100*time.Millisecond, "validator to validate batches", func() bool {
		lastInfo, err := repro.blockValidator.ReadLastValidatedInfo()
		if err != nil || lastInfo == nil {
			return false
		}
		return lastInfo.GlobalState.Batch >= totalBatches
	})

	// Post canonical child Y on-chain; this is what the flaky-wrapped sync
	// loop will later misclassify as invalid.
	genesisHash, err := repro.assertionChain.GenesisAssertionHash(ctx)
	Require(t, err)
	genesisInfo, err := repro.assertionChain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	Require(t, err)
	parentGlobalState := protocol.GoGlobalStateFromSolidity(genesisInfo.AfterState.GlobalState)
	yState, err := repro.stateManager.ExecutionStateAfterPreviousState(
		ctx, genesisInfo.InboxMaxCount.Uint64(), parentGlobalState,
	)
	Require(t, err)
	yAssertion, err := repro.assertionChain.NewStakeOnNewAssertion(ctx, genesisInfo, yState)
	Require(t, err)
	yHash := yAssertion.Id()
	t.Logf("posted canonical child Y on-chain: %s", yHash)

	// Flaky provider: wrong EndHistoryRoot on first call for Y's batch,
	// real provider thereafter. Models the prod state-provider race.
	flaky := newFlakySystemExecutionProvider(repro.stateManager, genesisInfo.InboxMaxCount.Uint64())

	// Only the ExecutionProvider slot is wrapped; the other roles use the
	// unwrapped provider.
	provider := state.NewHistoryCommitmentProvider(
		repro.stateManager,
		repro.stateManager,
		repro.stateManager,
		[]state.Height{state.Height(repro.blockChallengeLeafHeight)},
		flaky,
		containers.None[db.Database](),
	)

	// Posting disabled so the poster doesn't consume flaky's first-fire
	// trigger before findCanonicalAssertionBranch sees it.
	asm, err := assertions.NewManager(
		repro.assertionChain,
		provider,
		"self-challenge-repro",
		modes.MakeMode,
		assertions.WithPostingDisabled(),
		assertions.WithPollingInterval(500*time.Millisecond),
		assertions.WithConfirmationInterval(time.Hour),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(0),
		assertions.WithPostingInterval(time.Hour),
		assertions.WithMaxGetLogBlocks(5000),
	)
	Require(t, err)

	manager, err := challenge.NewChallengeStack(
		repro.assertionChain,
		provider,
		challenge.StackWithName("self-challenge-repro"),
		challenge.StackWithMode(modes.MakeMode),
		challenge.StackWithPostingInterval(time.Hour),
		challenge.StackWithPollingInterval(500*time.Millisecond),
		challenge.StackWithConfirmationInterval(time.Hour),
		challenge.StackWithMinimumGapToParentAssertion(0),
		challenge.StackWithAverageBlockCreationTime(time.Second),
		challenge.StackWithHeaderProvider(repro.l2node.L1Reader),
		challenge.StackWithSyncMaxGetLogBlocks(5000),
		challenge.OverrideAssertionManager(asm),
	)
	Require(t, err)

	recorder := &recordingRivalHandler{inner: manager}
	asm.SetRivalHandler(recorder)

	manager.Start(ctx)

	deadline := time.Now().Add(2 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
poll:
	for time.Now().Before(deadline) {
		if len(recorder.snapshot()) > 0 {
			break poll
		}
		select {
		case <-ctx.Done():
			break poll // labeled: bare break exits only the select
		case <-ticker.C:
		}
	}

	calls := recorder.snapshot()
	for _, h := range calls {
		t.Logf("HandleCorrectRival(%s); invalidAssertion=%s (equal=%v)",
			h, yHash, h == yHash)
	}

	// Regression gate: fails if the same-hash short-circuit is removed.
	require.Empty(t, calls,
		"self-challenge bug reproduced: HandleCorrectRival invoked on canonical assertion %s; captured calls=%v",
		yHash, calls)
}

// recordingRivalHandler records every HandleCorrectRival call before
// delegating to the real challenge manager. Lets us assert at the hash level
// without scraping logs while still exercising the production code path.
type recordingRivalHandler struct {
	inner modes.RivalHandler

	mu    sync.Mutex
	calls []protocol.AssertionHash
}

func (r *recordingRivalHandler) HandleCorrectRival(ctx context.Context, hash protocol.AssertionHash) error {
	r.mu.Lock()
	r.calls = append(r.calls, hash)
	r.mu.Unlock()
	if r.inner != nil {
		return r.inner.HandleCorrectRival(ctx, hash)
	}
	return nil
}

func (r *recordingRivalHandler) snapshot() []protocol.AssertionHash {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]protocol.AssertionHash, len(r.calls))
	copy(out, r.calls)
	return out
}

var _ modes.RivalHandler = (*recordingRivalHandler)(nil)

// flakySystemExecutionProvider returns a wrong EndHistoryRoot on the first
// call for targetInboxCount, then delegates unchanged. Self-contained
// system-test copy of the unit-test wrapper.
type flakySystemExecutionProvider struct {
	inner            state.ExecutionProvider
	targetInboxCount uint64

	mu        sync.Mutex
	triggered bool
}

func newFlakySystemExecutionProvider(inner state.ExecutionProvider, targetInboxCount uint64) *flakySystemExecutionProvider {
	return &flakySystemExecutionProvider{inner: inner, targetInboxCount: targetInboxCount}
}

func (f *flakySystemExecutionProvider) ExecutionStateAfterPreviousState(
	ctx context.Context,
	maxInboxCount uint64,
	previousGlobalState protocol.GoGlobalState,
) (*protocol.ExecutionState, error) {
	real, err := f.inner.ExecutionStateAfterPreviousState(ctx, maxInboxCount, previousGlobalState)
	if err != nil {
		return real, err
	}
	f.mu.Lock()
	corrupt := maxInboxCount == f.targetInboxCount && !f.triggered
	if corrupt {
		f.triggered = true
	}
	f.mu.Unlock()
	if !corrupt {
		return real, nil
	}
	corrupted := *real
	corrupted.EndHistoryRoot = common.Hash(crypto.Keccak256Hash([]byte("transient-wrong-end-history-root")))
	return &corrupted, nil
}

var _ state.ExecutionProvider = (*flakySystemExecutionProvider)(nil)

// reproRig bundles what the test needs from the L1+L2 setup.
type reproRig struct {
	l1stack                  *node.Node
	l1client                 *ethclient.Client
	l1info                   info
	l2node                   *arbnode.Node
	l2info                   info
	stateManager             *bold.BOLDStateProvider
	blockValidator           *staker.BlockValidator
	assertionChain           *sol.AssertionChain
	blockChallengeLeafHeight uint64
}

func setupBoldSelfChallengeRepro(t *testing.T, ctx context.Context) *reproRig {
	t.Helper()
	transferGas := util.NormalizeL2GasForL1GasInitial(800_000, params.GWei)
	l2chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)),
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	ownerBal := big.NewInt(params.Ether)
	ownerBal.Mul(ownerBal, big.NewInt(1_000_000))
	l2info.GenerateGenesisAccount("Owner", ownerBal)
	sconf := setup.RollupStackConfig{
		UseMockBridge:          false,
		UseMockOneStepProver:   false,
		UseBlobs:               true,
		MinimumAssertionPeriod: 0,
	}

	l2info, l2node, l2execNode, _, l2stack, l1info, _, l1client, l1stack, assertionChain, _, _, _, _ := createCompleteTestNodeOnL1(
		t, ctx, false, nil, l2chainConfig, nil, sconf, l2info, false, false,
	)

	valnode.TestValidationConfig.UseJit = false
	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	locator, err := server_common.NewMachineLocator(valnode.TestValidationConfig.Wasm.RootPath)
	Require(t, err)
	pcds := l2node.GetParentChainDataSource()
	stateless, err := staker.NewStatelessBlockValidator(
		pcds, pcds, l2node.TxStreamer, l2node.ExecutionRecorder, l2node.ConsensusDB, nil,
		StaticFetcherFrom(t, &blockValidatorConfig), valStack, locator.LatestWasmModuleRoot(),
	)
	Require(t, err)
	Require(t, stateless.Start(ctx))

	blockValidator, err := staker.NewBlockValidator(
		stateless, pcds, l2node.TxStreamer, StaticFetcherFrom(t, &blockValidatorConfig), nil,
	)
	Require(t, err)
	Require(t, blockValidator.Initialize(ctx))
	Require(t, blockValidator.Start(ctx))

	blockChallengeLeafHeight := uint64(1 << 5)
	dir := t.TempDir()
	stateManager, err := bold.NewBOLDStateProvider(
		blockValidator, stateless, state.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          "self-challenge-repro",
			MachineLeavesCachePath: dir,
			CheckBatchFinality:     false,
		},
		dir, pcds, l2node.TxStreamer, pcds, nil,
	)
	Require(t, err)

	_, err = execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, l2stack, l2execNode, l2node)
	Require(t, err)

	return &reproRig{
		l1stack:                  l1stack,
		l1client:                 l1client,
		l1info:                   l1info,
		l2node:                   l2node,
		l2info:                   l2info,
		stateManager:             stateManager,
		blockValidator:           blockValidator,
		assertionChain:           assertionChain,
		blockChallengeLeafHeight: blockChallengeLeafHeight,
	}
}
