package types

import (
	"context"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
)

// ChallengeManager defines an offchain, challenge manager, which will be
// an active participant in interacting with the on-chain contracts.
type ChallengeManager interface {
	ChallengeCreator
	ChallengeModeReader
}

// ChallengeCreator defines a struct which can initiate a challenge on an assertion hash
// by creating a level zero, block challenge edge onchain.
type ChallengeCreator interface {
	ChallengeAssertion(ctx context.Context, id protocol.AssertionHash) error
}

// ChallengeModeReader defines a struct which can read the challenge mode of a challenge manager.
type ChallengeModeReader interface {
	Mode() Mode
}
