// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"testing"

	"github.com/offchainlabs/nitro/arbnode"
)

// buildMultiNodeStack builds a sequencer stack plus a plain non-sequencer follower.
func buildMultiNodeStack(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	t.Helper()

	env, seqCleanup := buildL1L2Node(t, ctx, spec, overrides)
	// Tear down the sequencer+L1 if the follower build fails (Goexit).
	var rb rollbackGuard
	rb.stage(seqCleanup)
	defer rb.run()

	follower, followerCleanup := buildFollowerNode(t, ctx, spec, overrides, env)
	env.L2Followers = append(env.L2Followers, follower)

	rb.commit()
	return env, func() {
		followerCleanup()
		seqCleanup()
	}
}

// buildFollowerNode spins a non-sequencer L2 reusing the sequencer's L1 setup.
func buildFollowerNode(t *testing.T, ctx context.Context, spec Spec, overrides overrides, env *Env) (*L2Handle, func()) {
	t.Helper()

	nodeConfig, chainConfig, execCfg, stackCfg := seedConfigs(t, spec, overrides, arbnode.ConfigDefaultL1NonSequencerTest())
	// The follower must not sequence its own txs; with ForwardingTarget "null"
	// sends to it fail loudly instead of silently forking the chain.
	execCfg.Sequencer.Enable = false

	var rb rollbackGuard
	defer rb.run()

	_, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg, env.L1.initMsg, spec.arbOSInit, overrides.InitData)
	rb.stage(func() { blockchain.Stop(); closeStack("follower", stack) })

	execNode, fatalCh := newExecNode(t, ctx, "follower", stack, executionDB, blockchain, execCfg, env.L1.Client, env.L2.Consensus.ParentChain)
	seqTxOpts, dataSigner := sequencerCredentials(ctx, env.L1.Info)
	consensusNode := newConsensusNode(t, ctx, "follower", stack, execNode, consensusDB, nodeConfig, blockchain.Config(),
		env.L1.Client, env.L2.Consensus.DeployInfo, nil, seqTxOpts, dataSigner, fatalCh, env.L1.blobReader, env.L1.wasmRoot, env.L2.Consensus.ParentChain)

	handle, fullCleanup := startNode(t, ctx, &rb, env, "follower", env.L2.Info, stack, execNode, consensusNode, fatalCh, nil)

	rb.commit()
	return handle, fullCleanup
}
