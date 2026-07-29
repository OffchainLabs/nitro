// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"testing"

	"github.com/offchainlabs/nitro/arbnode"
)

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
	env.L2Followers = append(env.L2Followers, follower)

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

	return buildGenericNode(t, ctx, env, spec, overrides, "follower",
		nodeConfig, chainConfig, execCfg, stackCfg,
		env.L1.initMsg, env.L2.Info,
		env.L1.Client, env.L1.Info, env.L2.Consensus.ParentChain, env.L2.Consensus.DeployInfo,
		env.L1.blobReader, env.L1.wasmRoot)
}
