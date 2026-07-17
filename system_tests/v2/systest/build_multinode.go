// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/signature"
)

// WithMultiNode builds an L1 + sequencer L2 plus a non-sequencer L2 syncing
// via L1 (env.L2Follower).
func WithMultiNode() TestOption {
	return func(b *builder) {
		setTopology(b, TopologyMultiNode, "WithMultiNode")
	}
}

// Follower returns the non-sequencer follower handle.
func (e *Env) Follower() *L2Handle {
	e.t.Helper()
	e.requireFollower()
	return e.L2Follower
}

// WaitForFollowerSync blocks until the follower catches up to the sequencer.
// Fails the test if there is no follower.
func (e *Env) WaitForFollowerSync() {
	e.t.Helper()
	e.requireFollower()
	e.Require(e.waitFollowerSynced())
}

// requireFollower fails the scenario if it has no follower node.
func (e *Env) requireFollower() {
	e.t.Helper()
	e.NotNil(e.L2Follower, "follower helper called on a non-multi-node scenario; register it with systest.WithMultiNode()")
}

// waitFollowerSynced blocks until the follower executes the sequencer's message
// count, mining an L1 block each poll so batches post and heads advance.
func (e *Env) waitFollowerSynced() error {
	target, err := e.L2.Consensus.TxStreamer.GetMessageCount()
	if err != nil {
		return err
	}
	var lastErr error
	var lastGot uint64
	werr := waitFor(e.Ctx, "follower to execute sequencer message count", func() bool {
		e.AdvanceL1(1)
		got, err := e.L2Follower.Consensus.TxStreamer.GetProcessedMessageCount()
		lastErr = err
		lastGot = uint64(got)
		return err == nil && got >= target
	})
	if werr == nil {
		return nil
	}
	if lastErr != nil {
		return joinPollErr(werr, lastErr)
	}
	return fmt.Errorf("%w (follower at %d, want %d)", werr, lastGot, uint64(target))
}

// buildMultiNode builds a sequencer stack plus a plain non-sequencer follower.
func buildMultiNode(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	return buildMultiNodeStack(t, ctx, spec, overrides, false)
}

func buildMultiNodeStack(t *testing.T, ctx context.Context, spec Spec, overrides overrides, staker bool) (*Env, func()) {
	t.Helper()

	env, seqCleanup := buildL1Stack(t, ctx, spec, overrides, staker)
	// Tear down the sequencer+L1 if the follower build fails (Goexit).
	var rb rollbackGuard
	rb.stage(seqCleanup)
	defer rb.run()

	follower, followerCleanup := buildFollowerNode(t, ctx, spec, overrides, env, staker)
	env.L2Follower = follower

	rb.commit()
	return env, func() {
		followerCleanup()
		seqCleanup()
	}
}

// buildFollowerNode spins a non-sequencer L2 reusing the sequencer's L1 setup.
// When staker is set it also runs block validation and a staker.
func buildFollowerNode(t *testing.T, ctx context.Context, spec Spec, overrides overrides, env *Env, staker bool) (*L2Handle, func()) {
	t.Helper()

	nodeConfig := cloneConfig(arbnode.ConfigDefaultL1NonSequencerTest())
	chainConfig, execCfg, stackCfg := seedConfigs(t, spec, overrides, nodeConfig)

	var validatorTxOpts *bind.TransactOpts

	l1Client := env.L1.Client
	addresses := env.L2.Consensus.DeployInfo
	parentChain := env.L2.Consensus.ParentChain

	var rb rollbackGuard
	defer rb.run()

	_, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg, env.L1.initMsg, spec.arbOSInit, overrides.InitData)
	rb.stage(func() { blockchain.Stop(); closeStack("l2-follower", stack) })

	fatalCh := make(chan error, 10)
	execFetcher := newConfigFetcher(execCfg)
	execNode, err := gethexec.CreateExecutionNode(ctx, stack, executionDB, blockchain,
		containers.Some(l1Client), execFetcher, 0, parentChain, fatalCh)
	if err != nil {
		t.Fatalf("follower CreateExecutionNode: %v", err)
	}

	nodeFetcher := newConfigFetcher(nodeConfig)
	seqTxOpts := env.L1.Info.GetDefaultTransactOpts("Sequencer", ctx)
	dataSigner := signature.DataSignerFromPrivateKey(env.L1.Info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	consensusNode, err := arbnode.CreateConsensusNode(
		ctx, stack, execNode, consensusDB, nodeFetcher, blockchain.Config(), l1Client,
		addresses, validatorTxOpts, &seqTxOpts, dataSigner, fatalCh, env.L1.blobReader, env.L1.wasmRoot, parentChain)
	if err != nil {
		t.Fatalf("follower CreateConsensusNode: %v", err)
	}

	handle := &L2Handle{
		ChainHandle: ChainHandle{Info: env.L2.Info, e: env, name: "follower"},
		Stack:       stack,
		ExecNode:    execNode,
		Consensus:   consensusNode,
	}
	fullCleanup := startL2Node(t, ctx, &rb, handle, fatalCh, nil)

	rb.commit()
	return handle, fullCleanup
}
