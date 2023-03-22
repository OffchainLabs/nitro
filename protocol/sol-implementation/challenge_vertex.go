package solimpl

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
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

func (v *ChallengeVertex) Prev(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.ChallengeVertex], error) {
	// Refreshes the vertex.
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return util.None[protocol.ChallengeVertex](), err
	}
	vertex, err := manager.GetVertex(ctx, tx, v.id)
	if err != nil {
		return util.None[protocol.ChallengeVertex](), err
	}
	if vertex.IsNone() {
		return util.None[protocol.ChallengeVertex](), ErrNotFound
	}
	unwrappedVertex, ok := vertex.Unwrap().(*ChallengeVertex)
	if !ok {
		return util.None[protocol.ChallengeVertex](), ErrNotFound
	}
	unwrappedVertexInner, err := unwrappedVertex.inner(ctx, tx)
	if err != nil {
		return util.None[protocol.ChallengeVertex](), err
	}
	return manager.GetVertex(ctx, tx, unwrappedVertexInner.PredecessorId)
}

func (v *ChallengeVertex) Status(ctx context.Context, tx protocol.ActiveTx) (protocol.AssertionState, error) {
	// TODO: Should be vertex status.
	inner, err := v.inner(ctx, tx)
	if err != nil {
		return 0, err
	}
	return protocol.AssertionState(inner.Status), nil
}

func (v *ChallengeVertex) HistoryCommitment(ctx context.Context, tx protocol.ActiveTx) (util.HistoryCommitment, error) {
	inner, err := v.inner(ctx, tx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.HistoryCommitment{
		Height: inner.Height.Uint64(),
		Merkle: inner.HistoryRoot,
	}, nil
}

func (v *ChallengeVertex) MiniStaker(ctx context.Context, tx protocol.ActiveTx) (common.Address, error) {
	inner, err := v.inner(ctx, tx)
	if err != nil {
		return common.Address{}, err
	}
	return inner.Staker, nil
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

func (v *ChallengeVertex) ConfirmForChallengeDeadline(ctx context.Context, tx protocol.ActiveTx) error {
	return errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForSubChallengeWin(ctx context.Context, tx protocol.ActiveTx) error {
	return errors.New("unimplemented")
}

// HasConfirmedSibling checks if the vertex has a confirmed sibling in the protocol.
func (v *ChallengeVertex) HasConfirmedSibling(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return false, err
	}
	return manager.caller.HasConfirmedSibling(v.chain.callOpts, v.id)
}

// IsPresumptiveSuccessor checks if a vertex is the presumptive successor
// within its challenge.
func (v *ChallengeVertex) IsPresumptiveSuccessor(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return false, err
	}
	return manager.caller.IsPresumptiveSuccessor(v.chain.callOpts, v.id)
}

// ChildrenAreAtOneStepFork checks if child vertices are at a one-step-fork in the challenge
// it is contained in.
func (v *ChallengeVertex) ChildrenAreAtOneStepFork(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return false, err
	}
	atFork, err := manager.caller.ChildrenAreAtOneStepFork(v.chain.callOpts, v.id)
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
func (v *ChallengeVertex) Merge(
	ctx context.Context,
	tx protocol.ActiveTx,
	mergingToHistory util.HistoryCommitment,
	proof []byte,
) (protocol.ChallengeVertex, error) {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	_, err = transact(ctx, v.chain.backend, v.chain.headerReader, func() (*types.Transaction, error) {
		return manager.writer.Merge(
			v.chain.txOpts,
			v.id,
			mergingToHistory.Merkle,
			proof,
		)
	})
	if err != nil {
		return nil, err
	}
	inner, err := v.inner(ctx, tx)
	if err != nil {
		return nil, err
	}
	return getVertexFromComponents(
		manager,
		v.chain.callOpts,
		inner.ChallengeId,
		mergingToHistory,
	)
}

// Bisect a challenge vertex by providing a history commitment.
func (v *ChallengeVertex) Bisect(
	ctx context.Context,
	tx protocol.ActiveTx,
	history util.HistoryCommitment,
	proof []byte,
) (protocol.ChallengeVertex, error) {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	receipt, err := transact(
		ctx,
		v.chain.backend,
		v.chain.headerReader,
		func() (*types.Transaction, error) {
			return manager.writer.Bisect(
				v.chain.txOpts,
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
	bisection, err := manager.filterer.ParseBisected(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, errors.Wrap(err, "could not parse bisection log")
	}
	bisectedTo, err := manager.GetVertex(ctx, tx, bisection.ToId)
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
	_, err = manager.caller.GetVertex(
		opts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:    vertexId,
		chain: manager.assertionChain,
	}, nil
}

func (v *ChallengeVertex) ConfirmForPsTimer(ctx context.Context, tx protocol.ActiveTx) error {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return err
	}
	_, err = transact(ctx, v.chain.backend, v.chain.headerReader, func() (*types.Transaction, error) {
		return manager.writer.ConfirmForPsTimer(
			v.chain.txOpts,
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
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	inner, err := v.inner(ctx, tx)
	if err != nil {
		return nil, err
	}
	currentChallenge, err := manager.GetChallenge(ctx, tx, inner.ChallengeId)
	if err != nil {
		return nil, err
	}
	if currentChallenge.IsNone() {
		return nil, errors.New("no challenge exists found for vertex")
	}
	challenge := currentChallenge.Unwrap()
	var subChallengeType protocol.ChallengeType
	challengeType, err := challenge.GetType(ctx, tx)
	if err != nil {
		return nil, err
	}
	switch challengeType {
	case protocol.BlockChallenge:
		subChallengeType = protocol.BigStepChallenge
	case protocol.BigStepChallenge:
		subChallengeType = protocol.SmallStepChallenge
	default:
		return nil, fmt.Errorf("cannot make subchallenge for challenge type %d", challengeType)
	}

	if _, err = transact(ctx, v.chain.backend, v.chain.headerReader, func() (*types.Transaction, error) {
		return manager.writer.CreateSubChallenge(
			v.chain.txOpts,
			v.id,
		)
	}); err != nil {
		if strings.Contains(err.Error(), "Challenge already exists") {
			return nil, ErrAlreadyExists
		}
		return nil, errors.Wrap(err, "submitting subchallenge to chain failed")
	}

	challengeId, err := manager.CalculateChallengeHash(ctx, tx, v.id, subChallengeType)
	if err != nil {
		return nil, err
	}
	chal, err := manager.GetChallenge(ctx, tx, challengeId)
	if err != nil {
		return nil, err
	}
	if chal.IsNone() {
		return nil, errors.New("no challenge found after subchallenge creation")
	}
	return chal.Unwrap(), nil
}

func (v *ChallengeVertex) inner(ctx context.Context, tx protocol.ActiveTx) (challengeV2gen.ChallengeVertex, error) {
	manager, err := v.manager(ctx, tx)
	if err != nil {
		return challengeV2gen.ChallengeVertex{}, err
	}
	vertexInner, err := manager.caller.GetVertex(v.chain.callOpts, v.id)
	if err != nil {
		return challengeV2gen.ChallengeVertex{}, err
	}
	return vertexInner, nil
}

func (v *ChallengeVertex) manager(ctx context.Context, tx protocol.ActiveTx) (*ChallengeManager, error) {
	manager, err := v.chain.CurrentChallengeManager(ctx, tx)
	if err != nil {
		return nil, err
	}
	challengeManager, ok := manager.(*ChallengeManager)
	if !ok {
		return nil, errors.New("challengemanager is not expected concrete type")
	}
	return challengeManager, nil
}
