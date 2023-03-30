package solimpl

import (
	"context"
	"time"

	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/pkg/errors"
	"math/big"
)

type SpecEdge struct {
	id         [32]byte
	manager    *SpecChallengeManager
	miniStaker util.Option[common.Address]
	inner      challengeV2gen.ChallengeEdge
}

func (e *SpecEdge) Id() protocol.EdgeId {
	return e.id
}

func (e *SpecEdge) GetType() protocol.EdgeType {
	return protocol.EdgeType(e.inner.EType)
}

func (e *SpecEdge) MiniStaker() util.Option[common.Address] {
	return e.miniStaker
}

func (e *SpecEdge) StartCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.inner.StartHeight.Uint64()), e.inner.StartHistoryRoot
}

func (e *SpecEdge) EndCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.inner.EndHeight.Uint64()), e.inner.EndHistoryRoot
}

func (e *SpecEdge) PresumptiveTimer(ctx context.Context) (uint64, error) {
	timer, err := e.manager.caller.GetCurrentPsTimer(e.manager.callOpts, e.id)
	if err != nil {
		return 0, err
	}
	return timer.Uint64(), nil
}

func (e *SpecEdge) IsPresumptive(ctx context.Context) (bool, error) {
	return e.manager.caller.IsPresumptive(e.manager.callOpts, e.id)
}

func (e *SpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	edge, err := e.manager.caller.GetEdge(e.manager.callOpts, e.id)
	if err != nil {
		return 0, err
	}
	return protocol.EdgeStatus(edge.Status), nil
}

func (e *SpecEdge) IsOneStepForkSource(ctx context.Context) (bool, error) {
	return e.manager.caller.IsAtOneStepFork(e.manager.callOpts, e.id)
}

func (e *SpecEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (protocol.SpecEdge, protocol.SpecEdge, error) {
	_, err := transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.BisectEdge(e.manager.txOpts, e.id, prefixHistoryRoot, prefixProof)
	})
	if err != nil {
		return nil, nil, err
	}
	someEdge, err := e.manager.GetEdge(ctx, e.id)
	if err != nil {
		return nil, nil, err
	}
	if someEdge.IsNone() {
		return nil, nil, errors.New("could not refresh edge after bisecting, got empty result")
	}
	edge, ok := someEdge.Unwrap().(*SpecEdge)
	if !ok {
		return nil, nil, errors.New("not a *SpecEdge")
	}
	// Refresh the edge.
	e = edge
	someLowerChild, err := e.manager.GetEdge(ctx, e.inner.LowerChildId)
	if err != nil {
		return nil, nil, err
	}
	someUpperChild, err := e.manager.GetEdge(ctx, e.inner.UpperChildId)
	if err != nil {
		return nil, nil, err
	}
	if someLowerChild.IsNone() || someUpperChild.IsNone() {
		return nil, nil, errors.New("expected edge to have children post-bisection, but has none")
	}
	return someLowerChild.Unwrap(), someUpperChild.Unwrap(), nil
}

func (e *SpecEdge) ConfirmByTimer(ctx context.Context, ancestorIds []protocol.EdgeId) error {
	ancestors := make([][32]byte, len(ancestorIds))
	for i, r := range ancestorIds {
		ancestors[i] = r
	}
	_, err := transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByTimer(e.manager.txOpts, e.id, ancestors)
	})
	return err
}

func (e *SpecEdge) ConfirmByClaim(ctx context.Context, claimId protocol.ClaimId) error {
	// TODO: Add in fields.
	_, err := transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByClaim(e.manager.txOpts, e.id, claimId)
	})
	return err
}

func (e *SpecEdge) OriginCommitment(ctx context.Context) (protocol.Height, common.Hash, error) {
	return 0, common.Hash{}, nil
}

func (e *SpecEdge) ConfirmByOneStepProof(ctx context.Context) error {
	_, err := transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByOneStepProof(
			e.manager.txOpts,
			e.id,
			// TODO: Fill in.
			challengeV2gen.OneStepData{},
			// TODO: Add pre/post proofs.
			nil,
			nil,
		)
	})
	return err
}

// TopLevelClaimHeight gets the height at the BlockChallenge level that originated a subchallenge.
// For example, if two validators open a subchallenge S at vertex A in a BlockChallenge, the TopLevelClaimHeight of S is the height of A.
// of S is A. If two validators open a subchallenge S' at vertex B in BigStepChallenge, the TopLevelClaimVertex
// is the height of A.
func (e *SpecEdge) TopLevelClaimHeight(ctx context.Context) (protocol.Height, error) {
	switch e.GetType() {
	case protocol.BigStepChallengeEdge:
		blockChallengeOneStepForkSource, err := e.manager.GetEdge(ctx, e.inner.ClaimEdgeId)
		if err != nil {
			return 0, err
		}
		if blockChallengeOneStepForkSource.IsNone() {
			return 0, errors.New("source edge is none")
		}
		startHeight, _ := blockChallengeOneStepForkSource.Unwrap().StartCommitment()
		return startHeight, nil
	case protocol.SmallStepChallengeEdge:
		bigStepChallengeOneStepForkSource, err := e.manager.GetEdge(ctx, e.inner.ClaimEdgeId)
		if err != nil {
			return 0, err
		}
		if bigStepChallengeOneStepForkSource.IsNone() {
			return 0, errors.New("source edge is none")
		}
		bigStepEdge, ok := bigStepChallengeOneStepForkSource.Unwrap().(*SpecEdge)
		if !ok {
			return 0, errors.New("not *SpecEdge")
		}
		blockChallengeOneStepForkSource, err := e.manager.GetEdge(ctx, bigStepEdge.inner.ClaimEdgeId)
		if err != nil {
			return 0, err
		}
		if blockChallengeOneStepForkSource.IsNone() {
			return 0, errors.New("source edge is none")
		}
		startHeight, _ := blockChallengeOneStepForkSource.Unwrap().StartCommitment()
		return startHeight, nil
	default:
		return 0, errors.New("not a subchallenge")
	}
}

// ChallengeManager --
type SpecChallengeManager struct {
	addr           common.Address
	backend        ChainBackend
	assertionChain *AssertionChain
	reader         *headerreader.HeaderReader
	callOpts       *bind.CallOpts
	txOpts         *bind.TransactOpts
	caller         *challengeV2gen.EdgeChallengeManagerCaller
	writer         *challengeV2gen.EdgeChallengeManagerTransactor
	filterer       *challengeV2gen.EdgeChallengeManagerFilterer
}

// CurrentChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func NewSpecCM(
	ctx context.Context,
	addr common.Address,
	assertionChain *AssertionChain,
	backend ChainBackend,
	reader *headerreader.HeaderReader,
	callOpts *bind.CallOpts,
	txOpts *bind.TransactOpts,
) (protocol.SpecChallengeManager, error) {
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(addr, backend)
	if err != nil {
		return nil, err
	}
	return &SpecChallengeManager{
		addr:           common.Address{},
		assertionChain: assertionChain,
		backend:        backend,
		reader:         reader,
		callOpts:       callOpts,
		txOpts:         txOpts,
		caller:         &managerBinding.EdgeChallengeManagerCaller,
		writer:         &managerBinding.EdgeChallengeManagerTransactor,
		filterer:       &managerBinding.EdgeChallengeManagerFilterer,
	}, nil
}

func (cm *SpecChallengeManager) Address() common.Address {
	return cm.addr
}

// Duration of the challenge period.
func (cm *SpecChallengeManager) ChallengePeriodSeconds(
	ctx context.Context,
) (time.Duration, error) {
	return time.Second, nil
}

// Gets an edge by its hash.
func (cm *SpecChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (util.Option[protocol.SpecEdge], error) {
	edge, err := cm.caller.GetEdge(cm.callOpts, edgeId)
	if err != nil {
		return util.None[protocol.SpecEdge](), err
	}
	miniStaker := util.None[common.Address]()
	if edge.Staker != (common.Address{}) {
		miniStaker = util.Some(edge.Staker)
	}
	return util.Some(protocol.SpecEdge(&SpecEdge{
		id:         edgeId,
		manager:    cm,
		inner:      edge,
		miniStaker: miniStaker,
	})), nil
}

// Calculates an edge hash given its challenge id, start history, and end history.
func (cm *SpecChallengeManager) CalculateMutualId(
	ctx context.Context,
	edgeType protocol.EdgeType,
	originId protocol.OriginId,
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
) (protocol.MutualId, error) {
	return cm.caller.CalculateMutualId(
		cm.callOpts,
		uint8(edgeType),
		originId,
		big.NewInt(int64(startHeight)),
		startHistoryRoot,
		big.NewInt(int64(endHeight)),
	)
}

// Calculates an edge hash given its challenge id, start history, and end history.
func (cm *SpecChallengeManager) CalculateEdgeId(
	ctx context.Context,
	edgeType protocol.EdgeType,
	originId protocol.OriginId,
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
	endHistoryRoot common.Hash,
) (protocol.EdgeId, error) {
	return cm.caller.CalculateEdgeId(
		cm.callOpts,
		uint8(edgeType),
		originId,
		big.NewInt(int64(startHeight)),
		startHistoryRoot,
		big.NewInt(int64(endHeight)),
		endHistoryRoot,
	)
}

func (cm *SpecChallengeManager) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
	startCommit util.HistoryCommitment,
	endCommit util.HistoryCommitment,
) (protocol.SpecEdge, error) {
	assertionId, err := cm.assertionChain.GetAssertionId(ctx, assertion.SeqNum())
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not get id for assertion with sequence num %d",
			assertion.SeqNum(),
		)
	}
	_, err = transact(ctx, cm.backend, cm.reader, func() (*types.Transaction, error) {
		return cm.writer.CreateLayerZeroEdge(
			cm.txOpts,
			challengeV2gen.CreateEdgeArgs{
				EdgeType:         uint8(protocol.BlockChallenge),
				StartHistoryRoot: startCommit.Merkle,
				StartHeight:      big.NewInt(int64(startCommit.Height)),
				EndHistoryRoot:   endCommit.Merkle,
				EndHeight:        big.NewInt(int64(endCommit.Height)),
				ClaimId:          assertionId,
			},
			// TODO: Add inclusion proofs.
			make([]byte, 0),
			make([]byte, 0),
		)
	})
	if err != nil {
		return nil, err
	}

	edgeId, err := cm.CalculateEdgeId(
		ctx,
		protocol.BlockChallengeEdge,
		protocol.OriginId(assertionId),
		protocol.Height(startCommit.Height),
		startCommit.Merkle,
		protocol.Height(endCommit.Height),
		endCommit.Merkle,
	)
	if err != nil {
		return nil, err
	}
	someLevelZeroEdge, err := cm.GetEdge(ctx, edgeId)
	if err != nil {
		return nil, err
	}
	if someLevelZeroEdge.IsNone() {
		return nil, errors.New("got empty, newly created level zero edge")
	}
	return someLevelZeroEdge.Unwrap(), nil
}

func (cm *SpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startCommit util.HistoryCommitment,
	endCommit util.HistoryCommitment,
) (protocol.SpecEdge, error) {
	var subChalTyp protocol.EdgeType
	switch challengedEdge.GetType() {
	case protocol.BlockChallengeEdge:
		subChalTyp = protocol.BigStepChallengeEdge
	case protocol.BigStepChallengeEdge:
		subChalTyp = protocol.SmallStepChallengeEdge
	default:
		return nil, fmt.Errorf("cannot open level zero edge beneath small step challenge: %s", challengedEdge.GetType())
	}
	_, err := transact(ctx, cm.backend, cm.reader, func() (*types.Transaction, error) {
		return cm.writer.CreateLayerZeroEdge(
			cm.txOpts,
			challengeV2gen.CreateEdgeArgs{
				EdgeType:         uint8(subChalTyp),
				StartHistoryRoot: startCommit.Merkle,
				StartHeight:      big.NewInt(int64(startCommit.Height)),
				EndHistoryRoot:   endCommit.Merkle,
				EndHeight:        big.NewInt(int64(endCommit.Height)),
				ClaimId:          challengedEdge.Id(),
			},
			// TODO: Add inclusion proofs.
			make([]byte, 0),
			make([]byte, 0),
		)
	})
	challenged, ok := challengedEdge.(*SpecEdge)
	if !ok {
		return nil, errors.New("not a *SpecEdge")
	}
	mutualId, err := cm.CalculateMutualId(
		ctx,
		challengedEdge.GetType(),
		challenged.inner.OriginId,
		protocol.Height(challenged.inner.StartHeight.Uint64()),
		challenged.inner.StartHistoryRoot,
		protocol.Height(challenged.inner.EndHeight.Uint64()),
	)
	if err != nil {
		return nil, err
	}
	edgeId, err := cm.CalculateEdgeId(
		ctx,
		subChalTyp,
		protocol.OriginId(mutualId),
		protocol.Height(startCommit.Height),
		startCommit.Merkle,
		protocol.Height(endCommit.Height),
		endCommit.Merkle,
	)
	if err != nil {
		return nil, err
	}
	someLevelZeroEdge, err := cm.GetEdge(ctx, edgeId)
	if err != nil {
		return nil, err
	}
	if someLevelZeroEdge.IsNone() {
		return nil, errors.New("got empty, newly created level zero edge")
	}
	return someLevelZeroEdge.Unwrap(), nil
}
