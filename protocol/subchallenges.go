package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/pkg/errors"
)

var (
	ErrWrongChallengeKind        = errors.New("wrong top-level kind for subchallenge creation")
	ErrNoChallenge               = errors.New("no challenge corresponds to vertex")
	ErrChallengeNotRunning       = errors.New("challenge is not ongoing")
	ErrSubchallengeAlreadyExists = errors.New("subchallenge already exists on vertex")
	ErrNotEnoughValidChildren    = errors.New("vertex needs at least two unexpired children")
)

// CreateBigStepChallenge creates a BigStep subchallenge on a vertex.
func (v *ChallengeVertex) CreateBigStepChallenge(ctx context.Context, tx *ActiveTx) error {
	tx.verifyReadWrite()
	if err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge); err != nil {
		return err
	}
	// TODO: Add all other required challenge fields.
	challengeGetCreationTime, _ := v.Challenge.Unwrap().GetCreationTime(ctx, tx)
	v.SubChallenge = util.Some(ChallengeInterface(&Challenge{
		// Set the creation time of the subchallenge to be
		// the same as the top-level challenge, as they should
		// expire at the same timestamp.
		creationTime:  challengeGetCreationTime,
		challengeType: BigStepChallenge,
	}))
	// TODO: Add the challenge to the chain under a key that does not
	// collide with top-level challenges and fire events.
	return nil
}

// CreateSmallStepChallenge creates a SmallStep subchallenge on a vertex.
func (v *ChallengeVertex) CreateSmallStepChallenge(ctx context.Context, tx *ActiveTx) error {
	tx.verifyReadWrite()
	if err := v.canCreateSubChallenge(ctx, tx, SmallStepChallenge); err != nil {
		return err
	}
	// TODO: Add all other required challenge fields.
	challengeGetCreationTime, _ := v.Challenge.Unwrap().GetCreationTime(ctx, tx)
	v.SubChallenge = util.Some(ChallengeInterface(&Challenge{
		creationTime:  challengeGetCreationTime,
		challengeType: SmallStepChallenge,
	}))
	// TODO: Add the challenge to the chain under a key that does not
	// collide with top-level challenges and fire events.
	return nil
}

// Verifies the a subchallenge can be created on a challenge vertex
// based on specification validity conditions below:
//
//	A subchallenge can be created at a vertex P in a “parent” BlockChallenge if:
//	  - P’s challenge has not reached its end time
//	  - P’s has at least two children with unexpired chess clocks
//	The end time of the new challenge is set equal to the end time of P’s challenge.
func (v *ChallengeVertex) canCreateSubChallenge(
	ctx context.Context, tx *ActiveTx, subChallengeType ChallengeType,
) error {
	if v.Challenge.IsNone() {
		return ErrNoChallenge
	}
	chal := v.Challenge.Unwrap()
	challengeType, _ := chal.GetChallengeType(ctx, tx)
	// Can only create a subchallenge if the vertex is
	// part of a challenge of a specified kind.
	switch subChallengeType {
	case NoChallengeType:
		return ErrWrongChallengeKind
	case BlockChallenge:
		return ErrWrongChallengeKind
	case BigStepChallenge:
		if challengeType != BlockChallenge {
			return ErrWrongChallengeKind
		}
	case SmallStepChallenge:
		if challengeType != BigStepChallenge {
			return ErrWrongChallengeKind
		}
	}
	// The challenge must be ongoing.
	rootAssertion, _ := chal.RootAssertion(ctx, tx)
	chain := rootAssertion.chain
	hasEnded, _ := chal.HasEnded(ctx, tx, chain)
	if hasEnded {
		return ErrChallengeNotRunning
	}
	// There must not exist a subchallenge.
	if !v.SubChallenge.IsNone() {
		return ErrSubchallengeAlreadyExists
	}
	// The vertex must not be confirmed.
	if v.Status == ConfirmedAssertionState {
		return errors.Wrap(ErrWrongState, "vertex already confirmed")
	}
	// The vertex must have at least two children with unexpired
	// chess clocks in order to create a big step challenge.
	ok, err := hasUnexpiredChildren(ctx, tx, chain, v)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotEnoughValidChildren
	}
	return nil
}

// Checks if a challenge vertex has at least two children with
// unexpired chess-clocks. It does this by filtering out vertices from the chain
// that are the specified vertex's children and checking that at least two in this
// filtered list have unexpired chess clocks and are one-step away from the parent.
func hasUnexpiredChildren(ctx context.Context, tx *ActiveTx, chain *AssertionChain, v *ChallengeVertex) (bool, error) {
	if v.Challenge.IsNone() {
		return false, ErrNoChallenge
	}
	chal := v.Challenge.Unwrap()
	challengeCommit, _ := chal.ParentStateCommitment(ctx, tx)
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
		prev := otherVertex.Prev.Unwrap()
		prevCommitment, _ := prev.GetCommitment(ctx, tx)
		parentCommitHash := prevCommitment.Hash()
		isOneStepAway := otherVertex.Commitment.Height == prevCommitment.Height+1
		isChild := parentCommitHash == vertexCommitHash
		if isOneStepAway && isChild && !otherVertex.ChessClockExpired(chain.challengePeriod) {
			unexpiredChildrenTotal++
			if unexpiredChildrenTotal > 1 {
				return true, nil
			}
		}
	}
	return false, nil
}

// Checks if a challenge is still ongoing by making sure the current
// timestamp is within the challenge's creation time + challenge period.
func (c *Challenge) HasEnded(ctx context.Context, tx *ActiveTx, chain *AssertionChain) (bool, error) {
	challengeEndTime := c.creationTime.Add(chain.challengePeriod).Unix()
	now := chain.timeReference.Get().Unix()
	return now > challengeEndTime, nil
}

// Checks if a vertex's chess-clock has expired according
// to the challenge period length.
func (v *ChallengeVertex) ChessClockExpired(challengePeriod time.Duration) bool {
	return v.PsTimer.Get() > challengePeriod
}
