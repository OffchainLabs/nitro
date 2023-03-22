package solimpl

import (
	"context"
	"math/big"
	"strings"

	"fmt"
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

func (v *ChallengeVertex) SequenceNum() protocol.VertexSequenceNumber {
	return 0
}

func (v *ChallengeVertex) Prev(ctx context.Context) (util.Option[protocol.ChallengeVertex], error) {
	// Refreshes the vertex.
	vertex, err := v.manager.GetVertex(ctx, v.id)
	if err != nil {
		return util.None[protocol.ChallengeVertex](), err
	}
	if vertex.IsNone() {
		return util.None[protocol.ChallengeVertex](), ErrNotFound
	}
	innerV, ok := vertex.Unwrap().(*ChallengeVertex)
	if !ok {
		return util.None[protocol.ChallengeVertex](), ErrNotFound
	}
	v.inner = innerV.inner
	return v.manager.GetVertex(ctx, v.inner.PredecessorId)
}

func (v *ChallengeVertex) Status() protocol.AssertionState {
	// TODO: Should be vertex status.
	return protocol.AssertionState(v.inner.Status)
}

func (v *ChallengeVertex) HistoryCommitment() util.HistoryCommitment {
	return util.HistoryCommitment{
		Height: v.inner.Height.Uint64(),
		Merkle: v.inner.HistoryRoot,
	}
}

func (v *ChallengeVertex) MiniStaker() common.Address {
	return v.inner.Staker
}

func (v *ChallengeVertex) GetSubChallenge(ctx context.Context) (util.Option[protocol.Challenge], error) {
	return util.None[protocol.Challenge](), errors.New("unimplemented")
}

func (v *ChallengeVertex) EligibleForNewSuccessor(ctx context.Context) (bool, error) {
	return false, errors.New("unimplemented")
}

func (v *ChallengeVertex) PresumptiveSuccessor(ctx context.Context) (util.Option[protocol.ChallengeVertex], error) {
	return util.None[protocol.ChallengeVertex](), errors.New("unimplemented")
}

func (v *ChallengeVertex) PsTimer(ctx context.Context) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (v *ChallengeVertex) ChessClockExpired(ctx context.Context, challengePeriodSeconds time.Duration) (bool, error) {
	return false, errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForChallengeDeadline(ctx context.Context) error {
	return errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForSubChallengeWin(ctx context.Context) error {
	return errors.New("unimplemented")
}

// HasConfirmedSibling checks if the vertex has a confirmed sibling in the protocol.
func (v *ChallengeVertex) HasConfirmedSibling(ctx context.Context) (bool, error) {
	return v.manager.caller.HasConfirmedSibling(v.manager.assertionChain.callOpts, v.id)
}

// IsPresumptiveSuccessor checks if a vertex is the presumptive successor
// within its challenge.
func (v *ChallengeVertex) IsPresumptiveSuccessor(ctx context.Context) (bool, error) {
	return v.manager.caller.IsPresumptiveSuccessor(v.manager.assertionChain.callOpts, v.id)
}

// ChildrenAreAtOneStepFork checks if child vertices are at a one-step-fork in the challenge
// it is contained in.
func (v *ChallengeVertex) ChildrenAreAtOneStepFork(ctx context.Context) (bool, error) {
	atFork, err := v.manager.caller.ChildrenAreAtOneStepFork(v.manager.assertionChain.callOpts, v.id)
	if err != nil {
		errS := err.Error()
		switch {
		case strings.Contains(errS, "Lowest height not one above"):
			return false, nil
		case strings.Contains(errS, "Has presumptive successor"):
			return false, nil
		default:
			return false, err
		}
	}
	return atFork, nil
}

// Merge a challenge vertex to another by providing its history
// commitment and a prefix proof.
func (v *ChallengeVertex) Merge(ctx context.Context, mergingToHistory util.HistoryCommitment, proof []byte) (protocol.ChallengeVertex, error) {
	_, err := transact(ctx, v.manager.assertionChain.backend, v.manager.assertionChain.headerReader, func() (*types.Transaction, error) {
		return v.manager.writer.Merge(
			v.manager.assertionChain.txOpts,
			v.id,
			mergingToHistory.Merkle,
			proof,
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
func (v *ChallengeVertex) Bisect(ctx context.Context, history util.HistoryCommitment, proof []byte) (protocol.ChallengeVertex, error) {
	receipt, err := transact(
		ctx,
		v.manager.assertionChain.backend,
		v.manager.assertionChain.headerReader,
		func() (*types.Transaction, error) {
			return v.manager.writer.Bisect(
				v.manager.assertionChain.txOpts,
				v.id,
				history.Merkle,
				proof,
			)
		})
	if err != nil {
		errS := err.Error()
		switch {
		case strings.Contains(errS, "Bisection vertex already exists"):
			return nil, ErrAlreadyExists
		default:
			return nil, err
		}
	}
	if len(receipt.Logs) == 0 {
		return nil, errors.New("no logs observed from assertion confirmation")
	}
	bisection, err := v.manager.filterer.ParseBisected(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, errors.Wrap(err, "could not parse bisection log")
	}
	bisectedTo, err := v.manager.GetVertex(ctx, bisection.ToId)
	if err != nil {
		return nil, err
	}
	if bisectedTo.IsNone() {
		return nil, ErrNotFound
	}
	return bisectedTo.Unwrap(), nil
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

func (v *ChallengeVertex) ConfirmForPsTimer(ctx context.Context) error {
	_, err := transact(ctx, v.manager.assertionChain.backend, v.manager.assertionChain.headerReader, func() (*types.Transaction, error) {
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

func (v *ChallengeVertex) CreateSubChallenge(ctx context.Context) (protocol.Challenge, error) {
	currentChallenge, err := v.manager.GetChallenge(ctx, v.inner.ChallengeId)
	if err != nil {
		return nil, err
	}
	if currentChallenge.IsNone() {
		return nil, errors.New("no challenge exists found for vertex")
	}
	challenge := currentChallenge.Unwrap()
	var subChallengeType protocol.ChallengeType
	switch challenge.GetType() {
	case protocol.BlockChallenge:
		subChallengeType = protocol.BigStepChallenge
	case protocol.BigStepChallenge:
		subChallengeType = protocol.SmallStepChallenge
	default:
		return nil, fmt.Errorf("cannot make subchallenge for challenge type %d", challenge.GetType())
	}

	if _, err = transact(ctx, v.manager.assertionChain.backend, v.manager.assertionChain.headerReader, func() (*types.Transaction, error) {
		return v.manager.writer.CreateSubChallenge(
			v.manager.assertionChain.txOpts,
			v.id,
		)
	}); err != nil {
		return nil, err
	}

	challengeId, err := v.manager.CalculateChallengeHash(ctx, v.id, subChallengeType)
	if err != nil {
		return nil, err
	}
	chal, err := v.manager.GetChallenge(ctx, challengeId)
	if err != nil {
		return nil, err
	}
	if chal.IsNone() {
		return nil, errors.New("no challenge found after subchallenge creation")
	}
	return chal.Unwrap(), nil
}
