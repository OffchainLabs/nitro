package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (v *vertexTracker) determineBisectionHistoryWithProof(
	ctx context.Context,
	parentHeight,
	toHeight uint64,
) (util.HistoryCommitment, []byte, error) {
	bisectTo, err := util.BisectionPoint(parentHeight, toHeight)
	if err != nil {
		return util.HistoryCommitment{}, nil, errors.Wrapf(err, "determining bisection point failed for %d and %d", parentHeight, toHeight)
	}

	switch v.challenge.GetType() {
	case protocol.BlockChallenge:
		historyCommit, err := v.cfg.stateManager.HistoryCommitmentUpTo(ctx, bisectTo)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		proof, err := v.cfg.stateManager.PrefixProof(ctx, bisectTo, toHeight)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		return historyCommit, proof, nil
	case protocol.BigStepChallenge:
		var topLevelClaimVertex protocol.ChallengeVertex
		if err = v.cfg.chain.Call(func(tx protocol.ActiveTx) error {
			topLevel, err := v.challenge.TopLevelClaimVertex(ctx, tx)
			if err != nil {
				return err
			}
			topLevelClaimVertex = topLevel
			return nil
		}); err != nil {
			return util.HistoryCommitment{}, nil, err
		}

		fromAssertionHeight := topLevelClaimVertex.HistoryCommitment().Height
		toAssertionHeight := fromAssertionHeight + 1
		log.Infof("Root from %d to %d, with bisection %d to %d", fromAssertionHeight, toAssertionHeight, bisectTo, toHeight)

		// I need the height of the blockchallenge claim vertex! Then, it is just that height + 1.
		historyCommit, err := v.cfg.stateManager.BigStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, bisectTo)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		proof, err := v.cfg.stateManager.BigStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, bisectTo, toHeight)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}

		log.Infof("Verifying bisection from vertex with commit %#x and height %d", v.vertex.HistoryCommitment().Merkle, v.vertex.HistoryCommitment().Height)
		data, err := statemanager.ProofArgs.Unpack(proof)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		preExpansion := data[0].([][32]byte)
		pureProof := data[1].([][32]byte)

		preExpansionHashes := make([]common.Hash, len(preExpansion))
		for i := 0; i < len(preExpansion); i++ {
			preExpansionHashes[i] = preExpansion[i]
		}
		prefixProof := make([]common.Hash, len(pureProof))
		for i := 0; i < len(pureProof); i++ {
			prefixProof[i] = pureProof[i]
		}
		// Do a prefix proof verification for sainty.
		if err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
			PreRoot:      historyCommit.Merkle,
			PreSize:      bisectTo + 1,
			PostRoot:     v.vertex.HistoryCommitment().Merkle,
			PostSize:     toHeight + 1,
			PreExpansion: preExpansionHashes,
			PrefixProof:  prefixProof,
		}); err != nil {
			return util.HistoryCommitment{}, nil, errors.Wrapf(err, "val: %s", v.cfg.validatorName)
		}

		return historyCommit, proof, nil
	case protocol.SmallStepChallenge:
		var topLevelClaimVertex protocol.ChallengeVertex
		if err = v.cfg.chain.Call(func(tx protocol.ActiveTx) error {
			topLevel, err := v.challenge.TopLevelClaimVertex(ctx, tx)
			if err != nil {
				return err
			}
			topLevelClaimVertex = topLevel
			return nil
		}); err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		fromAssertionHeight := topLevelClaimVertex.HistoryCommitment().Height
		toAssertionHeight := fromAssertionHeight + 1
		log.Infof("Root assertion from %d to %d", fromAssertionHeight, toAssertionHeight)

		// I need the height of the blockchallenge claim vertex! Then, it is just that height + 1.
		historyCommit, err := v.cfg.stateManager.SmallStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, bisectTo)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		proof, err := v.cfg.stateManager.SmallStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, bisectTo, toHeight)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}
		return historyCommit, proof, nil
	default:
		return util.HistoryCommitment{}, nil, fmt.Errorf("challenge type not supported: %s", v.challenge.GetType())
	}
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

		historyCommit, proof, err := v.determineBisectionHistoryWithProof(ctx, parentHeight, toHeight)
		if err != nil {
			return err
		}
		bisectTo := historyCommit.Height
		bisected, err := validatorChallengeVertex.Bisect(ctx, tx, historyCommit, proof)
		if err != nil {
			return errors.Wrapf(
				err,
				"%s could not bisect to height=%d,commit=%s from height=%d,commit=%s",
				v.cfg.validatorName,
				bisectTo,
				util.Trunc(historyCommit.Merkle.Bytes()),
				validatorChallengeVertex.HistoryCommitment().Height,
				util.Trunc(validatorChallengeVertex.HistoryCommitment().Merkle.Bytes()),
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
	mergingToCommit util.HistoryCommitment,
	proof []byte,
) (protocol.ChallengeVertex, error) {
	var mergedTo protocol.ChallengeVertex
	if err := v.cfg.chain.Tx(func(tx protocol.ActiveTx) error {
		mergedToV, err2 := v.vertex.Merge(ctx, tx, mergingToCommit, proof)
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
			v.vertex.HistoryCommitment().Height,
			util.Trunc(v.vertex.HistoryCommitment().Merkle.Bytes()),
			mergingToCommit.Height,
			util.Trunc(mergingToCommit.Merkle.Bytes()),
		)
	}
	log.WithFields(logrus.Fields{
		"name":             v.cfg.validatorName,
		"mergedFrom":       v.vertex.HistoryCommitment().Height,
		"mergedFromMerkle": util.Trunc(v.vertex.HistoryCommitment().Merkle.Bytes()),
		"mergedTo":         mergingToCommit.Height,
		"mergedToMerkle":   util.Trunc(mergingToCommit.Merkle[:]),
	}).Info("Successfully merged to vertex")
	return mergedTo, nil
}
