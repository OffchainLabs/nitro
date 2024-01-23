package boostpolicies

import "github.com/offchainlabs/nitro/execution"

var (
	_ execution.BoostPolicyScorer = (*ExpressLaneScorer)(nil)
)
