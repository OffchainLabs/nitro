package solimpl

import protocol "github.com/OffchainLabs/bold/chain-abstraction"

var (
	_ = protocol.SpecEdge(&specEdge{})
	_ = protocol.SpecChallengeManager(&specChallengeManager{})
)
