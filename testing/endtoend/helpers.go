// Package endtoend includes e2e tests for BOLD challenges and assertion posting.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package endtoend

import (
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/endtoend/internal/backend"
)

// edgeManager fetches the challenge manager contract address from the rollup contract and returns
// a challenge manager instance.
func edgeManager(be backend.Backend) (*challengeV2gen.EdgeChallengeManager, error) {
	rc, err := rollupgen.NewRollupCore(be.ContractAddresses().Rollup, be.Client())
	if err != nil {
		return nil, err
	}
	cmAddr, err := rc.ChallengeManager(nil)
	if err != nil {
		return nil, err
	}
	return challengeV2gen.NewEdgeChallengeManager(cmAddr, be.Client())
}
