package solimpl

import (
	"context"
	"math/big"
	"strings"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"time"
)

func (v *ChallengeVertex) Id() [32]byte {
	return v.id
}

func (v *ChallengeVertex) SequenceNum(ctx context.Context, tx protocol.ActiveTx) (protocol.VertexSequenceNumber, error) {
	return 0, errors.New("unimplemented")
}

func (v *ChallengeVertex) Prev(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.ChallengeVertex], error) {
	return util.None[protocol.ChallengeVertex](), errors.New("unimplemented")
}

func (v *ChallengeVertex) Status(ctx context.Context, tx protocol.ActiveTx) (protocol.AssertionState, error) {
	return 0, errors.New("unimplemented")
}

func (v *ChallengeVertex) HistoryCommitment(ctx context.Context, tx protocol.ActiveTx) (util.HistoryCommitment, error) {
	return util.HistoryCommitment{}, errors.New("unimplemented")
}

func (v *ChallengeVertex) MiniStaker(ctx context.Context, tx protocol.ActiveTx) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (v *ChallengeVertex) GetSubChallenge(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.Challenge], error) {
	return util.None[protocol.Challenge](), errors.New("unimplemented")
}

func (v *ChallengeVertex) EligibleForNewSuccessor(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	return false, errors.New("unimplemented")
}

func (v *ChallengeVertex) PresumptiveSuccessor(
	ctx context.Context, tx protocol.ActiveTx,
) (util.Option[protocol.ChallengeVertex], error) {
	return util.None[protocol.ChallengeVertex](), errors.New("unimplemented")
}

func (v *ChallengeVertex) PsTimer(ctx context.Context, tx protocol.ActiveTx) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (v *ChallengeVertex) ChessClockExpired(
	ctx context.Context,
	tx protocol.ActiveTx,
	challengePeriodSeconds time.Duration,
) (bool, error) {
	return false, errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForPsTimer(ctx context.Context, tx protocol.ActiveTx) error {
	return errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForChallengeDeadline(ctx context.Context, tx protocol.ActiveTx) error {
	return errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForSubChallengeWin(ctx context.Context, tx protocol.ActiveTx) error {
	return errors.New("unimplemented")
}

// HasConfirmedSibling checks if the vertex has a confirmed sibling in the protocol.
func (v *ChallengeVertex) HasConfirmedSibling(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	return v.manager.caller.HasConfirmedSibling(v.manager.assertionChain.callOpts, v.id)
}

// IsPresumptiveSuccessor checks if a vertex is the presumptive successor
// within its challenge.
func (v *ChallengeVertex) IsPresumptiveSuccessor(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	return v.manager.caller.IsPresumptiveSuccessor(v.manager.assertionChain.callOpts, v.id)
}

// ChildrenAreAtOneStepFork checks if child vertices are at a one-step-fork in the challenge
// it is contained in.
func (v *ChallengeVertex) ChildrenAreAtOneStepFork(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	atFork, err := v.manager.caller.ChildrenAreAtOneStepFork(v.manager.assertionChain.callOpts, v.id)
	if err != nil {
		errS := err.Error()
		switch {
		case strings.Contains(errS, "Lowest height not one above"):
			return false, nil
		default:
			return false, err
		}
	}
	return atFork, nil
}

// Merge a challenge vertex to another by providing its history
// commitment and a prefix proof.
func (v *ChallengeVertex) Merge(
	ctx context.Context,
	tx protocol.ActiveTx,
	mergingToHistory util.HistoryCommitment,
	proof []common.Hash,
) (protocol.ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	flatProof := make([]byte, 0)
	for _, h := range proof {
		flatProof = append(flatProof, h[:]...)
	}
	_, err := transact(ctx, v.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return v.manager.writer.Merge(
			v.manager.assertionChain.txOpts,
			v.id,
			mergingToHistory.Merkle,
			flatProof,
		)
	})
	if err != nil {
		return nil, err
	}
	return getVertexFromComponents(
		v.manager,
		v.manager.assertionChain.callOpts,
		v.inner.ChallengeId,
		mergingToHistory,
	)
}

// Bisect a challenge vertex by providing a history commitment.
func (v *ChallengeVertex) Bisect(
	ctx context.Context,
	tx protocol.ActiveTx,
	history util.HistoryCommitment,
	proof []common.Hash,
) (protocol.ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	flatProof := make([]byte, 0)
	for _, h := range proof {
		flatProof = append(flatProof, h[:]...)
	}
	_, err := transact(ctx, v.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return v.manager.writer.Bisect(
			v.manager.assertionChain.txOpts,
			v.id,
			history.Merkle,
			flatProof,
		)
	})
	if err != nil {
		return nil, err
	}
	return getVertexFromComponents(
		v.manager,
		v.manager.assertionChain.callOpts,
		v.inner.ChallengeId,
		history,
	)
}

func getVertexFromComponents(
	manager *ChallengeManager,
	opts *bind.CallOpts,
	challengeId [32]byte,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	vertexId, err := manager.caller.CalculateChallengeVertexId(
		opts,
		challengeId,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	vertex, err := manager.caller.GetVertex(
		opts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:      vertexId,
		inner:   vertex,
		manager: manager,
	}, nil
}

func (v *ChallengeVertex) ConfirmPsTimer(ctx context.Context) error {
	_, err := transact(ctx, v.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return v.manager.writer.ConfirmForPsTimer(
			v.manager.assertionChain.txOpts,
			v.id,
		)
	})
	if err == nil {
		return nil
	}
	switch {
	case strings.Contains(err.Error(), "PsTimer not greater than challenge period"):
		return errors.Wrapf(ErrPsTimerNotYet, "vertex id %#v", v.id)
	default:
		return err
	}
}

func (v *ChallengeVertex) CreateSubChallenge(ctx context.Context, tx protocol.ActiveTx) (protocol.Challenge, error) {
	_, err := transact(ctx, v.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return v.manager.writer.CreateSubChallenge(
			v.manager.assertionChain.txOpts,
			v.id,
		)
	})
	if err != nil {
		return nil, err
	}
	// TODO: DO not use empty assertion
	challengeId, err := v.manager.CalculateChallengeHash(ctx, tx, v.id, protocol.BigStepChallenge)
	if err != nil {
		return nil, err
	}
	chal, err := v.manager.GetChallenge(ctx, tx, challengeId)
	if err != nil {
		return nil, err
	}
	if chal.IsNone() {
		return nil, errors.New("no challenge found after subchallenge creation")
	}
	return chal.Unwrap(), nil
}
