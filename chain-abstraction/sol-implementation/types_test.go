package solimpl

import protocol "github.com/offchainlabs/bold/chain-abstraction"

var (
	_ = protocol.SpecEdge(&specEdge{})
	_ = protocol.SpecChallengeManager(&specChallengeManager{})
)
