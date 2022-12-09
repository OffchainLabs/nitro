package validator

import (
	"context"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Performs a merge move during a BlockChallenge in the assertion protocol given
// a challenge vertex and the sequence number we should be merging into. To do this, we
// also need to fetch vertex we are are merging to by reading it from the protocol.
func (v *Validator) merge(
	ctx context.Context,
	challenge *protocol.Challenge,
	validatorChallengeVertex *protocol.ChallengeVertex,
	newPrevSeqNum protocol.SequenceNum,
) error {
	var mergingTo *protocol.ChallengeVertex
	var err error
	err = v.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		id := protocol.AssertionStateCommitHash(challenge.ParentStateCommitment().Hash())
		mergingTo, err = p.ChallengeVertexBySequenceNum(tx, id, newPrevSeqNum)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "could not read challenge vertex from protocol")
	}
	mergingToHeight := mergingTo.Commitment.Height
	historyCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, mergingToHeight)
	if err != nil {
		return err
	}
	currentCommit := validatorChallengeVertex.Commitment
	proof, err := v.stateManager.PrefixProof(ctx, mergingToHeight, currentCommit.Height)
	if err != nil {
		return err
	}
	if err := util.VerifyPrefixProof(historyCommit, currentCommit, proof); err != nil {
		return err
	}
	if err := v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		return validatorChallengeVertex.Merge(tx, mergingTo, proof, v.address)
	}); err != nil {
		return errors.Wrapf(
			err,
			"could not merge vertex with height %d and commit %#x to height %x and commit %#x",
			currentCommit.Height,
			currentCommit.Merkle,
			mergingToHeight,
			mergingTo.Commitment.Merkle,
		)
	}
	log.WithFields(logrus.Fields{
		"name": v.name,
	}).Infof(
		"Successfully merged to vertex with height %d and commit %#x",
		mergingTo.Commitment.Height,
		mergingTo.Commitment.Merkle,
	)
	return nil
}
