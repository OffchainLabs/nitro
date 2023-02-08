package assertionchain

import (
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
)

// ChallengeVertex is a developer-friendly wrapper around
// the protocol struct with the same name.
type ChallengeVertex struct {
	manager *ChallengeManager
	inner   outgen.ChallengeVertex
}
