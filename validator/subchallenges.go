package validator

import (
	"context"

	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/pkg/errors"
)

func (v *vertexTracker) submitSubChallenge(ctx context.Context) error {
	if v.challenge.GetType() == protocol.SmallStepChallenge {
		return errors.New("cannot create subchallenge on small step challenge")
	}
	// Produce a Merkle commitment of big steps from height v.prev.height to v.height.
	var subChalLeaf protocol.ChallengeVertex
	var subChal protocol.Challenge
	if err := v.chain.Tx(func(tx protocol.ActiveTx) error {
		// TODO(RJ): What happens if subchal creation works, but the rest of this function fails?
		// in this case, we need to make sure we keep retrying, otherwise
		// we do not have another chance to do so.
		prevVertex, err := v.vertex.Prev(ctx, tx)
		if err != nil {
			return err
		}
		if prevVertex.IsNone() {
			return errors.New("no previous vertex found")
		}
		prev := prevVertex.Unwrap()

		manager, err := v.chain.CurrentChallengeManager(ctx, tx)
		if err != nil {
			return err
		}

		var subChalToCreate protocol.ChallengeType
		switch v.challenge.GetType() {
		case protocol.BlockChallenge:
			subChalToCreate = protocol.BigStepChallenge
		case protocol.BigStepChallenge:
			subChalToCreate = protocol.SmallStepChallenge
		default:
			errors.New("unsupported challenge type to create")
		}

		var subChalCreated protocol.Challenge
		subChalCreated, err = prev.CreateSubChallenge(ctx, tx)
		if err != nil {
			switch {
			case errors.Is(err, solimpl.ErrAlreadyExists):
				subChalHash, calcErr := manager.CalculateChallengeHash(ctx, tx, prev.Id(), subChalToCreate)
				if calcErr != nil {
					return calcErr
				}
				fetchedSubChal, fetchErr := manager.GetChallenge(ctx, tx, subChalHash)
				if fetchErr != nil {
					return fetchErr
				}
				if fetchedSubChal.IsNone() {
					return fmt.Errorf("no subchallenge found on-chain for id %#x", subChalHash)
				}
				subChalCreated = fetchedSubChal.Unwrap()
			default:
				return errors.Wrap(err, "subchallenge creation failed")
			}
		}

		fromHeight := prev.HistoryCommitment().Height
		toHeight := v.vertex.HistoryCommitment().Height

		// Next we ask our state manager to produce an initial leaf commitment
		// for the subchallenge we just created.
		var history util.HistoryCommitment
		switch subChalCreated.GetType() {
		case protocol.BigStepChallenge:
			history, err = v.stateManager.BigStepLeafCommitment(ctx, fromHeight, toHeight)
		case protocol.SmallStepChallenge:
			history, err = v.stateManager.SmallStepLeafCommitment(ctx, fromHeight, toHeight)
		default:
			return errors.New("unsupported subchallenge type for creating leaf commitment")
		}
		if err != nil {
			return err
		}
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
