// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// newEnv seeds the runtime Env for a node build.
func newEnv(t testing.TB, ctx context.Context, spec Spec) *Env {
	return &Env{t: t, Ctx: ctx, Spec: spec}
}

// Env is the runtime handle passed to a Scenario.
type Env struct {
	t   testing.TB
	Ctx context.Context
	// L2 is the sequencer handle. Always populated.
	L2 *L2Handle
	// L2Followers are the non-sequencer follower handles. Empty unless TopologyMultiNode.
	L2Followers []*L2Handle
	// L1 is the parent chain handle. Nil for TopologyL2Only scenarios.
	L1   *L1Handle
	Spec Spec

	goWG sync.WaitGroup
	// running counts in-flight env.Go goroutines, reported on a Wait timeout.
	running atomic.Int64
	// asyncMu guards writes to T. dead flips true after Wait returns; later
	// writes are dropped so a leaked goroutine can't hit a finished subtest
	// (which panics the test binary in the worker-pool model).
	asyncMu sync.Mutex
	dead    bool
}

func (e *Env) Require(err error, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.NoError(e.t, err, msgAndArgs...) })
}

func (e *Env) Equal(expected, actual any, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.Equal(e.t, expected, actual, msgAndArgs...) })
}

func (e *Env) True(value bool, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.True(e.t, value, msgAndArgs...) })
}

func (e *Env) EqualBig(expected, actual *big.Int, msgAndArgs ...any) {
	e.t.Helper()
	if expected == nil || actual == nil || expected.Cmp(actual) != 0 {
		e.guarded(func() {
			require.Fail(e.t, fmt.Sprintf("Not equal: expected %s, actual %s", expected, actual), msgAndArgs...)
		})
	}
}

func (e *Env) Len(object any, length int, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.Len(e.t, object, length, msgAndArgs...) })
}

func (e *Env) Zero(i any, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.Zero(e.t, i, msgAndArgs...) })
}

func (e *Env) Empty(object any, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.Empty(e.t, object, msgAndArgs...) })
}

func (e *Env) NotEmpty(object any, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.NotEmpty(e.t, object, msgAndArgs...) })
}

func (e *Env) NotNil(object any, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.NotNil(e.t, object, msgAndArgs...) })
}

func (e *Env) ErrorContains(err error, contains string, msgAndArgs ...any) {
	e.t.Helper()
	e.guarded(func() { require.ErrorContains(e.t, err, contains, msgAndArgs...) })
}

// Logf logs to the test log.
func (e *Env) Logf(format string, args ...any) {
	e.t.Helper()
	e.guarded(func() { e.t.Logf(format, args...) })
}

// WaitFor polls fn until true or env.Ctx cancels. Fails the test with a
// descriptive message on timeout.
func (e *Env) WaitFor(desc string, fn func() bool) {
	e.t.Helper()
	e.Require(waitFor(e.Ctx, desc, fn))
}

// Follower returns the first non-sequencer follower handle.
func (e *Env) Follower() *L2Handle {
	e.t.Helper()
	e.requireFollower()
	return e.L2Followers[0]
}

// Followers returns all non-sequencer follower handles.
func (e *Env) Followers() []*L2Handle {
	e.t.Helper()
	e.requireFollower()
	return e.L2Followers
}

// WaitForFollowersSync blocks until every follower catches up to the sequencer.
// Fails the test if there is no follower.
func (e *Env) WaitForFollowersSync() {
	e.t.Helper()
	e.requireFollower()
	e.Require(e.waitFollowersSynced())
}

// requireFollower fails the scenario if it has no follower node.
func (e *Env) requireFollower() {
	e.t.Helper()
	e.NotEmpty(e.L2Followers, "follower helper called on a non-multi-node scenario; register it with systest.WithMultiNode()")
}

// waitFollowersSynced blocks until every follower executes the sequencer's
// message count, mining an L1 block each poll so batches post and heads advance.
func (e *Env) waitFollowersSynced() error {
	target, err := e.L2.Consensus.TxStreamer.GetMessageCount()
	if err != nil {
		return err
	}
	for _, f := range e.L2Followers {
		var lastErr error
		var lastGot uint64
		werr := waitFor(e.Ctx, "follower to execute sequencer message count", func() bool {
			// The follower has no feed: it syncs from the L1 inbox, and the
			// simulated L1 only mines on demand.
			e.L1.AdvanceBlocks(1)
			got, err := f.Consensus.TxStreamer.GetProcessedMessageCount()
			lastErr = err
			lastGot = uint64(got)
			return err == nil && got >= target
		})
		if werr == nil {
			continue
		}
		if lastErr != nil {
			return fmt.Errorf("%s: %w (last poll error: %w)", f.name, werr, lastErr)
		}
		return fmt.Errorf("%s: %w (at %d, want %d)", f.name, werr, lastGot, uint64(target))
	}
	return nil
}

// SendSignedTxViaL1 posts a signed L2 tx through the L1 delayed inbox, advances
// L1 past the delay, and waits for it to land on L2. Returns the L2 receipt.
func (e *Env) SendSignedTxViaL1(delayedTx *types.Transaction) *types.Receipt {
	e.t.Helper()
	e.requireL1()
	opts := e.L1.Info.GetDefaultTransactOpts("User", e.Ctx)
	txbytes, err := delayedTx.MarshalBinary()
	e.Require(err, "MarshalBinary")
	wrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	l1tx, err := e.L1.DelayedInbox().SendL2Message(&opts, wrapped)
	e.Require(err, "SendL2Message")
	e.L1.EnsureTxSucceeded(l1tx)
	e.L1.WaitForDelayBlocks()
	return e.L2.EnsureTxSucceeded(delayedTx)
}

// SendSignedTxBatchViaL1 posts a batch of signed L2 txs through the delayed
// inbox in one L1 message, advances L1, and waits for each on L2.
func (e *Env) SendSignedTxBatchViaL1(txes types.Transactions) types.Receipts {
	e.t.Helper()
	e.requireL1()
	opts := e.L1.Info.GetDefaultTransactOpts("User", e.Ctx)
	l1tx, err := e.L1.DelayedInbox().SendL2Message(&opts, e.l2MessageBatchData(txes))
	e.Require(err, "SendL2Message batch")
	e.L1.EnsureTxSucceeded(l1tx)
	e.L1.WaitForDelayBlocks()
	receipts := make(types.Receipts, 0, len(txes))
	for _, tx := range txes {
		receipts = append(receipts, e.L2.EnsureTxSucceeded(tx))
	}
	return receipts
}

// LookupL2Tx finds the single L2 submission tx generated by the L1 transaction
// behind l1Receipt (deposit / retryable / contract tx). Fails if not exactly one.
func (e *Env) LookupL2Tx(l1Receipt *types.Receipt) *types.Transaction {
	e.t.Helper()
	e.requireL1()
	bridge, err := arbnode.NewDelayedBridge(e.L1.Client, e.L1.Info.GetAddress("Bridge"), 0)
	e.Require(err, "NewDelayedBridge")
	messages, err := bridge.LookupMessagesInRange(e.Ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber, nil)
	e.Require(err, "LookupMessagesInRange")
	e.NotEmpty(messages, "LookupL2Tx: no message for submission")
	msgTypes := map[uint8]bool{
		arbostypes.L1MessageType_SubmitRetryable: true,
		arbostypes.L1MessageType_EthDeposit:      true,
		arbostypes.L1MessageType_L2Message:       true,
	}
	txTypes := map[uint8]bool{
		types.ArbitrumSubmitRetryableTxType: true,
		types.ArbitrumDepositTxType:         true,
		types.ArbitrumContractTxType:        true,
	}
	var submissionTxs []*types.Transaction
	chainID := e.L2.ChainID()
	for _, message := range messages {
		if !msgTypes[message.Message.Header.Kind] {
			continue
		}
		txs, err := arbos.ParseL2Transactions(message.Message, chainID, params.MaxDebugArbosVersionSupported)
		e.Require(err, "ParseL2Transactions")
		for _, tx := range txs {
			if txTypes[tx.Type()] {
				submissionTxs = append(submissionTxs, tx)
			}
		}
	}
	e.Len(submissionTxs, 1, "LookupL2Tx: expected exactly 1 submission tx")
	return submissionTxs[0]
}

// requireL1 fails the scenario with a clear message if it lacks a parent chain.
func (e *Env) requireL1() {
	e.t.Helper()
	e.NotNil(e.L1, "L1 helper called on a non-L1 scenario; register it with systest.WithL1()")
}

func (e *Env) l2MessageBatchData(txes types.Transactions) []byte {
	e.t.Helper()
	l2Message := []byte{arbos.L2MessageKind_Batch}
	sizeBuf := make([]byte, 8)
	for _, tx := range txes {
		txBytes, err := tx.MarshalBinary()
		e.Require(err, "MarshalBinary")
		binary.BigEndian.PutUint64(sizeBuf, uint64(len(txBytes))+1)
		l2Message = append(l2Message, sizeBuf...)
		l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
		l2Message = append(l2Message, txBytes...)
	}
	return l2Message
}

// Go spawns fn in a goroutine. Errors and panics surface via t.Errorf;
// context.Canceled / DeadlineExceeded are ignored once env.Ctx is done. Joined
// by the runner via wait() after env.Ctx is cancelled, so fn must observe ctx to exit.
func (e *Env) Go(fn func() error) {
	e.running.Add(1)
	e.goWG.Go(func() {
		defer e.running.Add(-1)
		defer func() {
			if r := recover(); r != nil {
				e.guarded(func() { e.t.Errorf("env.Go panic: %v\n%s", r, debug.Stack()) })
			}
		}()
		err := fn()
		if err == nil {
			return
		}
		// Suppress ctx errors only when our own ctx is done (teardown/timeout); a
		// wrapped deadline from an unrelated call mid-test is a real failure.
		if suppressedAtShutdown(e.Ctx, err) {
			return
		}
		e.guarded(func() { e.t.Errorf("env.Go: %v", err) })
	})
}

// envWaitTimeout bounds how long env.wait() will block.
var envWaitTimeout = 30 * time.Second

// wait joins Go goroutines, aborting on ctx cancel or envWaitTimeout; reports an
// error if any are still in flight, then marks the env dead to silence late writes.
func (e *Env) wait(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		e.goWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		e.asyncMu.Lock()
		e.dead = true
		e.asyncMu.Unlock()
	case <-ctx.Done():
		e.asyncMu.Lock()
		if n := e.running.Load(); n > 0 {
			e.t.Errorf("env.Wait aborted (%v): %d env.Go goroutine(s) still running; their pending failures are now suppressed", context.Cause(ctx), n)
		}
		e.dead = true
		e.asyncMu.Unlock()
	case <-time.After(envWaitTimeout):
		// Record the timeout and mark dead in one critical section so the
		// failure is logged before late goroutines are silenced. A goroutine
		// that ignores ctx leaks (Go can't force-kill it) but can no longer
		// write to the finished subtest.
		e.asyncMu.Lock()
		e.t.Errorf("env.Wait timed out after %v: %d env.Go goroutine(s) still running; their pending failures are now suppressed — raise envWaitTimeout and rerun to surface the real error", envWaitTimeout, e.running.Load())
		e.dead = true
		e.asyncMu.Unlock()
	}
}

// guarded runs fn under the dead guard: once wait has marked the env dead, fn
// is dropped and the calling goroutine stopped instead of hitting a finished subtest.
func (e *Env) guarded(fn func()) {
	e.asyncMu.Lock()
	defer e.asyncMu.Unlock()
	if e.dead {
		if e.Spec.Name != "" {
			log.Printf("systest: %q: dropped a test write after teardown (leaked goroutine)", e.Spec.Name)
		}
		runtime.Goexit()
	}
	fn()
}

// suppressedAtShutdown reports whether err is a ctx error to swallow because
// ctx itself is already done.
func suppressedAtShutdown(ctx context.Context, err error) bool {
	return ctx != nil && ctx.Err() != nil &&
		(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}
