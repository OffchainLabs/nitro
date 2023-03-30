package solimpl

import (
	"context"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/headerreader"
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
	// TODO: Add real return values from event in the receipt.
	return nil, nil, err
}

func (e *SpecEdge) ConfirmByTimer(ctx context.Context, ancestorIds []protocol.EdgeId) error {
	_, err := transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		// TODO: Needs ancestor ids specified, perhaps by caller?
		return e.manager.writer.ConfirmEdgeByTimer(e.manager.txOpts, e.id, nil) // TODO: Fix
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

// ChallengeManager --
type SpecChallengeManager struct {
	addr     common.Address
	backend  ChainBackend
	reader   *headerreader.HeaderReader
	callOpts *bind.CallOpts
	txOpts   *bind.TransactOpts
	caller   *challengeV2gen.EdgeChallengeManagerCaller
	writer   *challengeV2gen.EdgeChallengeManagerTransactor
	filterer *challengeV2gen.EdgeChallengeManagerFilterer
}

// CurrentChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func NewSpecCM(ctx context.Context) (protocol.SpecChallengeManager, error) {
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(common.Address{}, nil)
	if err != nil {
		return nil, err
	}
	return &SpecChallengeManager{
		addr:     common.Address{},
		caller:   &managerBinding.EdgeChallengeManagerCaller,
		writer:   &managerBinding.EdgeChallengeManagerTransactor,
		filterer: &managerBinding.EdgeChallengeManagerFilterer,
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
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
	endHistoryRoot common.Hash,
) (protocol.SpecEdge, error) {
	_, err := transact(ctx, cm.backend, cm.reader, func() (*types.Transaction, error) {
		stHash, err := assertion.StateHash()
		if err != nil {
			return nil, err
		}
		return cm.writer.CreateLayerZeroEdge(
			cm.txOpts,
			challengeV2gen.CreateEdgeArgs{
				EdgeType:         uint8(protocol.BlockChallenge),
				StartHistoryRoot: startHistoryRoot,
				StartHeight:      big.NewInt(int64(startHeight)),
				EndHistoryRoot:   endHistoryRoot,
				EndHeight:        big.NewInt(int64(endHeight)),
				ClaimId:          stHash,
			},
			nil,
			nil, // TODO: Inclusion args.
		)
	})
	// TODO: Add in
	return nil, err
}

func (cm *SpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
	endHistoryRoot common.Hash,
) (protocol.SpecEdge, error) {
	_, err := transact(ctx, cm.backend, cm.reader, func() (*types.Transaction, error) {
		// TODO: Get the edge type.
		return cm.writer.CreateLayerZeroEdge(
			cm.txOpts,
			challengeV2gen.CreateEdgeArgs{
				EdgeType:         uint8(protocol.BlockChallenge),
				StartHistoryRoot: startHistoryRoot,
				StartHeight:      big.NewInt(int64(startHeight)),
				EndHistoryRoot:   endHistoryRoot,
				EndHeight:        big.NewInt(int64(endHeight)),
				ClaimId:          challengedEdge.Id(),
			},
			nil,
			nil, // TODO: Inclusion args.
		)
	})
	// TODO: Add in
	return nil, err
}
