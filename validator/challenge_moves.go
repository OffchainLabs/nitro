package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
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
	var historyCommit util.HistoryCommitment
	switch v.challenge.GetType() {
	case protocol.BlockChallenge:
		historyCommit, err = v.cfg.stateManager.HistoryCommitmentUpTo(ctx, bisectTo)
	case protocol.BigStepChallenge:
		// TODO: Need the block hash of the one-step-fork point.
		fmt.Println("CALLED BIG")
		historyCommit, err = v.cfg.stateManager.BigStepCommitmentUpTo(ctx, 0, common.Hash{}, common.Hash{}, bisectTo)
	case protocol.SmallStepChallenge:
		fmt.Println("CALLED BIG")
		historyCommit, err = v.cfg.stateManager.SmallStepCommitmentUpTo(ctx, 0, common.Hash{}, common.Hash{}, bisectTo)
	}
	if err != nil {
		return util.HistoryCommitment{}, errors.Wrapf(err, "could not rertieve history commitment up to height %d", bisectTo)
	}
	return historyCommit, nil
}

// Performs a bisection move during a BlockChallenge in the assertion protocol given
// a validator challenge vertex. It will create a historical commitment for the vertex
// the validator wants to bisect to and an associated proof for submitting to the goimpl.
func (v *vertexTracker) bisect(
	ctx context.Context,
	validatorChallengeVertex protocol.ChallengeVertex,
) (protocol.ChallengeVertex, error) {
	var bisectedVertex protocol.ChallengeVertex
	var isPresumptive bool

	if err := v.cfg.chain.Tx(func(tx protocol.ActiveTx) error {
		commitment := validatorChallengeVertex.HistoryCommitment()
		toHeight := commitment.Height
		prev, err := validatorChallengeVertex.Prev(ctx, tx)
		if err != nil {
			return err
		}
		prevCommitment := prev.Unwrap().HistoryCommitment()
		parentHeight := prevCommitment.Height

		historyCommit, err := v.determineBisectionPointWithHistory(ctx, parentHeight, toHeight)
		if err != nil {
			return err
		}
		bisectTo := historyCommit.Height
		proof, err := v.cfg.stateManager.PrefixProof(ctx, bisectTo, toHeight)
		if err != nil {
			return errors.Wrapf(err, "generating prefix proof failed from height %d to %d", bisectTo, toHeight)
		}
		// Perform an extra safety check to ensure our proof verifies against the specified commitment
		// before we make an on-chain transaction.
		if err = util.VerifyPrefixProof(historyCommit, commitment, proof); err != nil {
			return errors.Wrapf(
				err,
				"proof failed for height=%d,commit=%s to height%d,commit=%s",
				historyCommit.Height,
				util.Trunc(historyCommit.Merkle.Bytes()),
				commitment.Height,
				util.Trunc(commitment.Merkle.Bytes()),
			)
		}
		bisected, err := validatorChallengeVertex.Bisect(ctx, tx, historyCommit, proof)
		if err != nil {
			return errors.Wrapf(
				err,
				"%s could not bisect height=%d,commit=%s to height%d,commit=%s",
				v.cfg.validatorName,
				bisectTo,
				util.Trunc(validatorChallengeVertex.HistoryCommitment().Merkle.Bytes()),
				historyCommit.Height,
				util.Trunc(historyCommit.Merkle.Bytes()),
			)
		}
		bisectedVertex = bisected
		bisectedVertexIsPresumptiveSuccessor, err := bisectedVertex.IsPresumptiveSuccessor(ctx, tx)
		if err != nil {
			return err
		}
		isPresumptive = bisectedVertexIsPresumptiveSuccessor
		return nil
	}); err != nil {
		return nil, err
	}
	bisectedVertexCommitment := bisectedVertex.HistoryCommitment()
	log.WithFields(logrus.Fields{
		"name":               v.cfg.validatorName,
		"isPs":               isPresumptive,
		"bisectedFrom":       validatorChallengeVertex.HistoryCommitment().Height,
		"bisectedFromMerkle": util.Trunc(validatorChallengeVertex.HistoryCommitment().Merkle.Bytes()),
		"bisectedTo":         bisectedVertexCommitment.Height,
		"bisectedToMerkle":   util.Trunc(bisectedVertexCommitment.Merkle[:]),
	}).Info("Successfully bisected to vertex")
	return bisectedVertex, nil
}

// Performs a merge move during a BlockChallenge in the assertion protocol given
// a challenge vertex and the sequence number we should be merging into. To do this, we
// also need to fetch vertex we are merging to by reading it from the goimpl.
func (v *vertexTracker) merge(
	ctx context.Context,
	challengeCommitHash protocol.ChallengeHash,
	mergingTo protocol.ChallengeVertex,
	mergingFrom protocol.ChallengeVertex,
) (protocol.ChallengeVertex, error) {
	currentCommit := mergingFrom.HistoryCommitment()
	mergingToCommit := mergingTo.HistoryCommitment()
	mergingToHeight := mergingToCommit.Height
	if mergingToHeight >= currentCommit.Height {
		return nil, fmt.Errorf(
			"merging to height %d cannot be >= vertex height %d",
			mergingToHeight,
			currentCommit.Height,
		)
	}
	var historyCommit util.HistoryCommitment
	var err error
	switch v.challenge.GetType() {
	case protocol.BlockChallenge:
		historyCommit, err = v.cfg.stateManager.HistoryCommitmentUpTo(ctx, mergingToHeight)
	case protocol.BigStepChallenge:
		// TODO: Need the block hash of the one-step-fork point.
		historyCommit, err = v.cfg.stateManager.BigStepCommitmentUpTo(ctx, 0, common.Hash{}, common.Hash{}, mergingToHeight)
	case protocol.SmallStepChallenge:
		historyCommit, err = v.cfg.stateManager.SmallStepCommitmentUpTo(ctx, 0, common.Hash{}, common.Hash{}, mergingToHeight)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not rertieve history commitment up to height %d", mergingToHeight)
	}
	proof, err := v.cfg.stateManager.PrefixProof(ctx, mergingToHeight, currentCommit.Height)
	if err != nil {
		return nil, err
	}
	if err = util.VerifyPrefixProof(historyCommit, currentCommit, proof); err != nil {
		return nil, err
	}

	var mergedTo protocol.ChallengeVertex
	if err = v.cfg.chain.Tx(func(tx protocol.ActiveTx) error {
		mergedToV, err2 := mergingFrom.Merge(ctx, tx, historyCommit, proof)
		if err2 != nil {
			return err2
		}
		mergedTo = mergedToV
		return nil
	}); err != nil {
		return nil, errors.Wrapf(
			err,
			"%s could not merge vertex at height=%d,commit=%s to height%d,commit=%s",
			v.cfg.validatorName,
			currentCommit.Height,
			util.Trunc(currentCommit.Merkle.Bytes()),
			mergingToHeight,
			util.Trunc(mergingToCommit.Merkle.Bytes()),
		)
	}
	log.WithFields(logrus.Fields{
		"name":             v.cfg.validatorName,
		"mergedFrom":       mergingFrom.HistoryCommitment().Height,
		"mergedFromMerkle": util.Trunc(mergingFrom.HistoryCommitment().Merkle.Bytes()),
		"mergedTo":         mergingToCommit.Height,
		"mergedToMerkle":   util.Trunc(mergingToCommit.Merkle[:]),
	}).Info("Successfully merged to vertex")
	return mergedTo, nil
}
