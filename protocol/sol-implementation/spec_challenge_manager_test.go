package solimpl

import (
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
)

var (
	_ = protocol.SpecChallenge(&SpecChallenge{})
	_ = protocol.SpecEdge(&SpecEdge{})
	_ = protocol.SpecChallengeManager(&SpecChallengeManager{})
)
