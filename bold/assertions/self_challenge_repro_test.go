// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package assertions

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/bold/challenge/types"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/state"
	challenge_testing "github.com/offchainlabs/nitro/bold/testing"
	stateprovider "github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
)

// TestSelfChallengeBugWhenStateProviderReturnsTransientWrongRoot is the
// regression gate for the same-hash short-circuit in
// maybePostRivalAssertionAndChallenge. A flaky ExecutionProvider returns a
// wrong EndHistoryRoot on the first call for Y's batch, so
// findCanonicalAssertionBranch disagrees and the rival path fires.
// maybePostRivalAssertion's ErrAlreadyExists fall-through hands Y back as
// the "correct rival"; without the short-circuit HandleCorrectRival would
// then be invoked on the canonical assertion. Passes when the short-circuit
// is in place, fails when removed.
func TestSelfChallengeBugWhenStateProviderReturnsTransientWrongRoot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		// Drive MinAssertionPeriodBlocks to 1 so PostAssertionBasedOnParent's
		// waitToPostIfNeeded does not block on L1 block production.
		setup.WithMinimumAssertionPeriod(1),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)

	chain := cfg.Chains[0]
	backend := cfg.Backend

	genesisHash, err := chain.GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisInfo, err := chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	realProvider, err := stateprovider.NewForSimpleMachine(t, cfg.StateManagerOpts...)
	require.NoError(t, err)

	// Post a single canonical child Y on-chain. This is the assertion that the
	// validator will later see and incorrectly flag as invalid.
	parentGlobalState := protocol.GoGlobalStateFromSolidity(genesisInfo.AfterState.GlobalState)
	yState, err := realProvider.ExecutionStateAfterPreviousState(
		ctx,
		genesisInfo.InboxMaxCount.Uint64(),
		parentGlobalState,
	)
	require.NoError(t, err)
	yAssertion, err := chain.NewStakeOnNewAssertion(ctx, genesisInfo, yState)
	require.NoError(t, err)
	yInfo, err := chain.ReadAssertionCreationInfo(ctx, yAssertion.Id())
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		backend.Commit()
	}

	flaky := &flakyExecutionProvider{
		inner:            realProvider,
		targetInboxCount: genesisInfo.InboxMaxCount.Uint64(),
	}

	manager, err := NewManager(
		chain,
		flaky,
		"self-challenge-repro",
		types.MakeMode,
		WithDangerousReadyToPost(),
		WithMinimumGapToParentAssertion(0),
	)
	require.NoError(t, err)

	// Seed canonicalAssertions with genesis — what syncAssertions does at
	// startup before processing any AssertionCreated events.
	genesisAssertionHash := protocol.AssertionHash{Hash: genesisHash}
	manager.assertionChainData.latestAgreedAssertion = genesisAssertionHash
	manager.assertionChainData.canonicalAssertions[genesisAssertionHash] = genesisInfo

	// Drain the confirmation channel so respondToAnyInvalidAssertions does not
	// block on a full buffer when it pushes the (erroneous) rival.
	go func() {
		for {
			select {
			case <-manager.observedCanonicalAssertions:
			case <-ctx.Done():
				return
			}
		}
	}()

	handler := &recordingRivalHandler{}
	manager.SetRivalHandler(handler)

	assertions := []assertionAndParentCreationInfo{
		{parent: genesisInfo, assertion: yInfo},
	}

	// Phase 1: canonical-branch scan with the flaky provider's first (wrong) call.
	require.NoError(t, manager.findCanonicalAssertionBranch(ctx, assertions))
	_, isCanonical := manager.assertionChainData.canonicalAssertions[yInfo.AssertionHash]
	require.False(
		t,
		isCanonical,
		"Y was not added to canonicalAssertions after the flaky first call — required precondition for the bug",
	)
	require.Equal(
		t,
		1,
		flaky.calls(),
		"findCanonicalAssertionBranch should make exactly one ExecutionStateAfterPreviousState call for Y",
	)

	// Phase 2: rival path fires; without the short-circuit HandleCorrectRival
	// is invoked on the canonical assertion.
	require.NoError(t, manager.respondToAnyInvalidAssertions(ctx, assertions, manager))

	calls := handler.snapshot()

	// Diagnostic log so a failure names the hash that would have been challenged.
	for _, h := range calls {
		t.Logf(
			"self-challenge bug fired: HandleCorrectRival(%s); "+
				"args.invalidAssertion.AssertionHash=%s (equal=%v)",
			h, yInfo.AssertionHash, h == yInfo.AssertionHash,
		)
	}

	// Regression gate: fails if the same-hash short-circuit in
	// maybePostRivalAssertionAndChallenge is removed.
	require.Empty(
		t,
		calls,
		"self-challenge bug reproduced: HandleCorrectRival was invoked when it should "+
			"have been short-circuited. The supposedly-invalid assertion %s has the same "+
			"hash as the 'correct rival' the validator computed, so the rival path is "+
			"trying to challenge a canonical assertion against itself. Captured calls: %v",
		yInfo.AssertionHash,
		calls,
	)
}

// flakyExecutionProvider wraps a real state.ExecutionProvider and, on the
// first call to ExecutionStateAfterPreviousState for a given maxInboxCount,
// returns a state with a deliberately-wrong EndHistoryRoot. Subsequent calls
// delegate to the wrapped provider unmodified.
type flakyExecutionProvider struct {
	inner            state.ExecutionProvider
	targetInboxCount uint64

	mu        sync.Mutex
	triggered bool
	callCount int
}

func (f *flakyExecutionProvider) ExecutionStateAfterPreviousState(
	ctx context.Context,
	maxInboxCount uint64,
	previousGlobalState protocol.GoGlobalState,
) (*protocol.ExecutionState, error) {
	real, err := f.inner.ExecutionStateAfterPreviousState(ctx, maxInboxCount, previousGlobalState)
	if err != nil {
		return real, err
	}

	f.mu.Lock()
	f.callCount++
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

func (f *flakyExecutionProvider) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

type recordingRivalHandler struct {
	mu    sync.Mutex
	calls []protocol.AssertionHash
}

func (r *recordingRivalHandler) HandleCorrectRival(_ context.Context, hash protocol.AssertionHash) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, hash)
	return nil
}

func (r *recordingRivalHandler) snapshot() []protocol.AssertionHash {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]protocol.AssertionHash, len(r.calls))
	copy(out, r.calls)
	return out
}

var _ types.RivalHandler = (*recordingRivalHandler)(nil)
var _ state.ExecutionProvider = (*flakyExecutionProvider)(nil)
