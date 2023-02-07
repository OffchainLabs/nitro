package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (v *vertexTracker) determineBisectionPointWithHistory(
	ctx context.Context,
	parentHeight,
	toHeight uint64,
) (util.HistoryCommitment, error) {
	bisectTo, err := util.BisectionPoint(parentHeight, toHeight)
	if err != nil {
		return util.HistoryCommitment{}, errors.Wrapf(err, "determining bisection point failed for %d and %d", parentHeight, toHeight)
	}
	historyCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, bisectTo)
	if err != nil {
		return util.HistoryCommitment{}, errors.Wrapf(err, "could not rertieve history commitment up to height %d", bisectTo)
	}
	return historyCommit, nil
}

// Performs a bisection move during a BlockChallenge in the assertion protocol given
// a validator challenge vertex. It will create a historical commitment for the vertex
// the validator wants to bisect to and an associated proof for submitting to the protocol.
func (v *vertexTracker) bisect(
	ctx context.Context,
	validatorChallengeVertex protocol.ChallengeVertexInterface,
) (protocol.ChallengeVertexInterface, error) {
	toHeight := validatorChallengeVertex.GetCommitment().Height
	parentHeight := validatorChallengeVertex.GetPrev().Unwrap().GetCommitment().Height

	historyCommit, err := v.determineBisectionPointWithHistory(ctx, parentHeight, toHeight)
	if err != nil {
		return nil, err
	}
	bisectTo := historyCommit.Height
	proof, err := v.stateManager.PrefixProof(ctx, bisectTo, toHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "generating prefix proof failed from height %d to %d", bisectTo, toHeight)
	}
	// Perform an extra safety check to ensure our proof verifies against the specified commitment
	// before we make an on-chain transaction.
	if err = util.VerifyPrefixProof(historyCommit, validatorChallengeVertex.GetCommitment(), proof); err != nil {
		return nil, errors.Wrapf(
			err,
			"prefix proof failed to verify for commit %+v to commit %+v",
			historyCommit,
			validatorChallengeVertex.GetCommitment(),
		)
	}
	var bisectedVertex protocol.ChallengeVertexInterface
	err = v.chain.Tx(func(tx *protocol.ActiveTx) error {
		bisectedVertex, err = validatorChallengeVertex.Bisect(tx, historyCommit, proof, v.validatorAddress)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not bisect vertex with sequence %d and validator %#x to height %d with history %d and %#x",
			validatorChallengeVertex.GetSequenceNum(),
			validatorChallengeVertex.GetValidator(),
			bisectTo,
			historyCommit.Height,
			historyCommit.Merkle,
		)
	}
	log.WithFields(logrus.Fields{
		"name":                   v.validatorName,
		"isPresumptiveSuccessor": bisectedVertex.IsPresumptiveSuccessor(),
		"historyCommitHeight":    bisectedVertex.GetCommitment().Height,
		"historyCommitMerkle":    fmt.Sprintf("%#x", bisectedVertex.GetCommitment().Merkle),
	}).Info("Successfully bisected to vertex")
	return bisectedVertex, nil
}

// Performs a merge move during a BlockChallenge in the assertion protocol given
// a challenge vertex and the sequence number we should be merging into. To do this, we
// also need to fetch vertex we are merging to by reading it from the protocol.
func (v *vertexTracker) merge(
	ctx context.Context,
	challengeCommitHash protocol.ChallengeCommitHash,
	mergingTo protocol.ChallengeVertexInterface,
	mergingFrom protocol.ChallengeVertexInterface,
) (protocol.ChallengeVertexInterface, error) {
	mergingToHeight := mergingTo.GetCommitment().Height
	historyCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, mergingToHeight)
	if err != nil {
		return nil, err
	}
	currentCommit := mergingFrom.GetCommitment()
	proof, err := v.stateManager.PrefixProof(ctx, mergingToHeight, currentCommit.Height)
	if err != nil {
		return nil, err
	}
	if err = util.VerifyPrefixProof(historyCommit, currentCommit, proof); err != nil {
		return nil, err
	}
	if err = v.chain.Tx(func(tx *protocol.ActiveTx) error {
		if err = mergingFrom.Merge(tx, mergingTo, proof, v.validatorAddress); err != nil {
			return err
		}
		// Refresh the mergingTo vertex by reading it from the protocol, as some of its fields may have
		// changed after we made the merge transaction above.
		mergingTo, err = v.chain.ChallengeVertexByCommitHash(tx, challengeCommitHash, protocol.VertexCommitHash(mergingTo.GetCommitment().Hash()))
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, errors.Wrapf(
			err,
			"could not merge vertex with height %d and commit %#x to height %x and commit %#x",
			currentCommit.Height,
			currentCommit.Merkle,
			mergingToHeight,
			mergingTo.GetCommitment().Merkle,
		)
	}
	log.WithFields(logrus.Fields{
		"name": v.validatorName,
	}).Infof(
		"Successfully merged to vertex with height %d and commit %#x",
		mergingTo.GetCommitment().Height,
		mergingTo.GetCommitment().Merkle,
	)
	return mergingTo, nil
}
