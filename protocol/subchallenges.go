package protocol

import (
	"github.com/pkg/errors"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"time"
)

type BigStepChallenge struct {
	creationTime time.Time
}

// CreateBigStepChallenge creates a BigStep subchallenge for the vertex.
func (v *ChallengeVertex) CreateBigStepChallenge(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.Challenge.IsNone() {
		return ErrInvalidOp
	}
	// The overall challenge must be ongoing.
	chal := v.Challenge.Unwrap()
	if !isStillOngoing(chal) {
		return ErrInvalidOp
	}
	if !v.SubChallenge.IsNone() {
		return ErrVertexAlreadyExists
	}
	if v.Status == ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongState, "status: %d", v.Status)
	}
	bigStep := &BigStepChallenge{
		creationTime: chal.creationTime,
	}
	_ = bigStep
	return nil
}

func (b *BigStepChallenge) AddLeaf(v *ChallengeVertex) error {
	if v.Prev.IsNone() {
		return ErrInvalidOp
	}
	if v.Prev.Unwrap().SubChallenge
	return nil
}

type SubChallenge struct {
	parent *ChallengeVertex
	Winner *ChallengeVertex
}

// SetWinner sets the winner of the sub-challenge.
func (sc *SubChallenge) SetWinner(tx *ActiveTx, winner *ChallengeVertex) error {
	tx.verifyReadWrite()
	if sc.Winner != nil {
		return ErrInvalidOp
	}
	if winner.Prev.Unwrap() != sc.parent {
		return ErrInvalidOp
	}
	sc.Winner = winner
	return nil
}

// Checks if a challenge is still ongoing by making sure the current timestamp is within
// the challenge's creation time + challenge period.
func isStillOngoing(challenge *Challenge) bool {
	return time.Now().Unix() < challenge.creationTime.Add(challenge.challengePeriod).Unix()
}
