package protocol

import (
	"fmt"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/pkg/errors"
)

var (
	ErrNotBlockChallenge         = errors.New("can only create big step subchallenge on block challenge")
	ErrNoChallenge               = errors.New("no challenge corresponds to vertex")
	ErrChallengeNotRunning       = errors.New("challenge is not ongoing")
	ErrSubchallengeAlreadyExists = errors.New("subchallenge already exists on vertex")
	ErrNotEnoughValidChildren    = errors.New("vertex needs at least two unexpired children")
)

type SubChallenge struct {
	Parent       *ChallengeVertex
	Winner       *ChallengeVertex
	kind         ChallengeKind
	creationTime time.Time
}

// CreateBigStepChallenge creates a BigStep subchallenge for the vertex.
func (v *ChallengeVertex) CreateBigStepChallenge(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.Challenge.IsNone() {
		// Should not normally occur.
		return ErrNoChallenge
	}
	chal := v.Challenge.Unwrap()

	// Can only create a big step challenge if the vertex is
	// part of a block challenge.
	if chal.kind != Block {
		return ErrNotBlockChallenge
	}
	// The overall challenge must be ongoing.
	if !isStillOngoing(chal) {
		return ErrChallengeNotRunning
	}
	if !v.SubChallenge.IsNone() {
		return ErrSubchallengeAlreadyExists
	}
	if v.Status == ConfirmedAssertionState {
		return errors.Wrap(ErrWrongState, "vertex already confirmed")
	}

	// The vertex must have at least two children with unexpired
	// chess clocks in order to create a big step challenge.
	chain := chal.rootAssertion.Unwrap().chain
	ok, err := hasUnexpiredChildren(chain, v)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotEnoughValidChildren
	}
	bigStep := &SubChallenge{
		// Set the creation time of the subchallenge to be
		// the same as the top-level challenge, as they should
		// expire at the same timestamp.
		creationTime: chal.creationTime,
		kind:         BigStep,
	}
	v.SubChallenge = util.Some(bigStep)
	return nil
}

// Checks if a challenge vertex has at least two children with
// unexpired chess-clocks. It does this by filtering out vertices from the chain
// that are the specified vertex's children and checking that at least two in this
// filtered list have unexpired chess clocks.
func hasUnexpiredChildren(chain *AssertionChain, v *ChallengeVertex) (bool, error) {
	challengeCommit := v.Challenge.Unwrap().ParentStateCommitment()
	challengeHash := ChallengeCommitHash(challengeCommit.Hash())
	vertices, ok := chain.challengeVerticesByCommitHash[challengeHash]
	if !ok {
		return false, fmt.Errorf("vertices not found for challenge with hash: %#x", challengeHash)
	}
	vertexCommitHash := v.Commitment.Hash()
	unexpiredChildrenTotal := 0
	for _, otherVertex := range vertices {
		if otherVertex.Prev.IsNone() {
			continue
		}
		parentCommitHash := otherVertex.Prev.Unwrap().Commitment.Hash()
		isChild := parentCommitHash == vertexCommitHash

		if isChild && !chessClockExpired(otherVertex) {
			unexpiredChildrenTotal++
		}
	}
	return unexpiredChildrenTotal > 1, nil
}

// Checks if a challenge is still ongoing by making sure the current
// timestamp is within the challenge's creation time + challenge period.
func isStillOngoing(challenge *Challenge) bool {
	chain := challenge.rootAssertion.Unwrap().chain
	now := chain.timeReference.Get()
	return now.Unix() < challenge.creationTime.Add(challenge.challengePeriod).Unix()
}

// Checks if a vertex's chess-clock has expired according
// to the challenge period length.
func chessClockExpired(v *ChallengeVertex) bool {
	chain := v.Challenge.Unwrap().rootAssertion.Unwrap().chain
	return v.PsTimer.Get() > chain.challengePeriod
}
