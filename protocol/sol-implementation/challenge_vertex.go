package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

func (v *ChallengeVertex) Id() [32]byte {
	return v.id
}

func (v *ChallengeVertex) Prev(ctx context.Context) (util.Option[protocol.ChallengeVertex], error) {
	// Refreshes the vertex.
	manager, err := v.manager(ctx)
	if err != nil {
		return util.None[protocol.ChallengeVertex](), err
	}
	vertex, err := manager.GetVertex(ctx, v.id)
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
	unwrappedVertexInner, err := unwrappedVertex.inner(ctx)
	if err != nil {
		return util.None[protocol.ChallengeVertex](), err
	}
	return manager.GetVertex(ctx, unwrappedVertexInner.PredecessorId)
}

func (v *ChallengeVertex) Status(ctx context.Context) (protocol.AssertionState, error) {
	// TODO: Should be vertex status.
	inner, err := v.inner(ctx)
	if err != nil {
		return 0, err
	}
	return protocol.AssertionState(inner.Status), nil
}

func (v *ChallengeVertex) HistoryCommitment() util.HistoryCommitment {
	return util.HistoryCommitment{
		Height: v.height,
		Merkle: v.historyCommit,
	}
}

func (v *ChallengeVertex) MiniStaker(ctx context.Context) (common.Address, error) {
	inner, err := v.inner(ctx)
	if err != nil {
		return common.Address{}, err
	}
	return inner.Staker, nil
}

func (v *ChallengeVertex) GetSubChallenge(ctx context.Context) (util.Option[protocol.Challenge], error) {
	return util.None[protocol.Challenge](), errors.New("unimplemented")
}

func (v *ChallengeVertex) PsTimer(ctx context.Context) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (v *ChallengeVertex) ConfirmForSubChallengeWin(ctx context.Context) error {
	return errors.New("unimplemented")
}

// HasConfirmedSibling checks if the vertex has a confirmed sibling in the protocol.
func (v *ChallengeVertex) HasConfirmedSibling(ctx context.Context) (bool, error) {
	manager, err := v.manager(ctx)
	if err != nil {
		return false, err
	}
	return manager.caller.HasConfirmedSibling(v.chain.callOpts, v.id)
}

// IsPresumptiveSuccessor checks if a vertex is the presumptive successor
// within its challenge.
func (v *ChallengeVertex) IsPresumptiveSuccessor(ctx context.Context) (bool, error) {
	manager, err := v.manager(ctx)
	if err != nil {
		return false, err
	}
	return manager.caller.IsPresumptiveSuccessor(v.chain.callOpts, v.id)
}

// ChildrenAreAtOneStepFork checks if child vertices are at a one-step-fork in the challenge
// it is contained in.
func (v *ChallengeVertex) ChildrenAreAtOneStepFork(ctx context.Context) (bool, error) {
	manager, err := v.manager(ctx)
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

// Bisect a challenge vertex by providing a history commitment.
func (v *ChallengeVertex) Bisect(ctx context.Context, history util.HistoryCommitment, proof []byte) (protocol.ChallengeVertex, error) {
	manager, err := v.manager(ctx)
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
	bisectedTo, err := manager.GetVertex(ctx, bisection.ToId)
	if err != nil {
		return nil, err
	}
	if bisectedTo.IsNone() {
		return nil, ErrNotFound
	}
	return bisectedTo.Unwrap(), nil
}

func getVertexFromComponents(
	ctx context.Context,
	manager *ChallengeManager,
	challengeId [32]byte,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	vertexId, err := manager.caller.CalculateChallengeVertexId(
		manager.assertionChain.callOpts,
		challengeId,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	vOpt, err := manager.GetVertex(
		ctx,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	if vOpt.IsNone() {
		return nil, ErrNotFound
	}
	return vOpt.Unwrap(), nil
}

func (v *ChallengeVertex) ConfirmForPsTimer(ctx context.Context) error {
	manager, err := v.manager(ctx)
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

func (v *ChallengeVertex) CreateSubChallenge(ctx context.Context) (protocol.Challenge, error) {
	manager, err := v.manager(ctx)
	if err != nil {
		return nil, err
	}
	inner, err := v.inner(ctx)
	if err != nil {
		return nil, err
	}
	currentChallenge, err := manager.GetChallenge(ctx, inner.ChallengeId)
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

	challengeId, err := manager.CalculateChallengeHash(ctx, v.id, subChallengeType)
	if err != nil {
		return nil, err
	}
	chal, err := manager.GetChallenge(ctx, challengeId)
	if err != nil {
		return nil, err
	}
	if chal.IsNone() {
		return nil, errors.New("no challenge found after subchallenge creation")
	}
	return chal.Unwrap(), nil
}

func (v *ChallengeVertex) inner(ctx context.Context) (challengeV2gen.ChallengeVertex, error) {
	manager, err := v.manager(ctx)
	if err != nil {
		return challengeV2gen.ChallengeVertex{}, err
	}
	vertexInner, err := manager.caller.GetVertex(v.chain.callOpts, v.id)
	if err != nil {
		return challengeV2gen.ChallengeVertex{}, err
	}
	return vertexInner, nil
}

func (v *ChallengeVertex) manager(ctx context.Context) (*ChallengeManager, error) {
	manager, err := v.chain.CurrentChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	challengeManager, ok := manager.(*ChallengeManager)
	if !ok {
		return nil, errors.New("challengemanager is not expected concrete type")
	}
	return challengeManager, nil
}
