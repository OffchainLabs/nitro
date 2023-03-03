package validator

import (
	"context"
	"errors"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

type historyProducer interface {
	bigStepCommitment(from, to uint64) util.HistoryCommitment
}

func (v *vertexTracker) submitSubChallenge(ctx context.Context, producer historyProducer) error {
	if v.challenge.GetType() == protocol.SmallStepChallenge {
		return errors.New("cannot create subchallenge on small step challenge")
	}
	// Produce a Merkle commitment of big steps from height v.prev.height to v.height.
	var subChalLeaf protocol.ChallengeVertex
	var subChal protocol.Challenge
	if err := v.chain.Tx(func(tx protocol.ActiveTx) error {
		subChalCreated, err := v.vertex.CreateSubChallenge(ctx, tx)
		if err != nil {
			return err
		}
		// TODO(RJ): What happens if subchal creation works, but the rest of this function fails?
		// in this case, we need to make sure we keep retrying, otherwise
		// we do not have another chance to do so.
		prev, err := v.vertex.Prev(ctx, tx)
		if err != nil {
			return err
		}
		if prev.IsNone() {
			return errors.New("no previous vertex found")
		}
		fromHeight := prev.Unwrap().HistoryCommitment().Height
		toHeight := v.vertex.HistoryCommitment().Height
		history := producer.bigStepCommitment(fromHeight, toHeight)
		subChalLeafV, err := subChalCreated.AddSubChallengeLeaf(ctx, tx, v.vertex, history)
		if err != nil {
			return err
		}
		subChalLeaf = subChalLeafV
		subChal = subChalCreated
		return nil
	}); err != nil {
		return err
	}
	go newVertexTracker(
		v.timeRef,
		v.actEveryNSeconds,
		subChal,
		subChalLeaf,
		v.chain,
		v.stateManager,
		v.validatorName,
		v.validatorAddress,
	).track(ctx)
	return nil
}
