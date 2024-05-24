// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/containers"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/ospgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pkg/errors"
)

var (
	errorConfirmingEdgeByOneStepProofCounter = metrics.GetOrRegisterCounter("arb/validator/tracker/error_confirming_edge_by_one_step_proof", nil)
	invalidInclusionProofCounter             = metrics.GetOrRegisterCounter("arb/validator/tracker/invalid_inclusion_proof", nil)
)

const InvalidInclusionProofError = "invalid inclusion proof"

func (e *specEdge) Id() protocol.EdgeId {
	return protocol.EdgeId{Hash: e.id}
}

func (e *specEdge) GetChallengeLevel() protocol.ChallengeLevel {
	return protocol.ChallengeLevel(e.inner.Level)
}

// GetReversedChallengeLevel obtains the challenge level for the edge. The lowest level starts at 0, and goes all way
// up to the max number of levels. The reason we go from the lowest challenge level being 0 instead of 2
// is to make our code a lot more readable. If we flipped the order, we would need to do
// a lot of backwards for loops instead of simple range loops over slices.
func (e *specEdge) GetReversedChallengeLevel() protocol.ChallengeLevel {
	return protocol.ChallengeLevel(e.totalChallengeLevels - 1 - e.inner.Level)
}

func (e *specEdge) GetTotalChallengeLevels(ctx context.Context) uint8 {
	return e.totalChallengeLevels
}

func (e *specEdge) MiniStaker() option.Option[common.Address] {
	return e.miniStaker
}

func (e *specEdge) StartCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.startHeight), e.inner.StartHistoryRoot
}

func (e *specEdge) EndCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.endHeight), e.inner.EndHistoryRoot
}

func (e *specEdge) AssertionHash(_ context.Context) (protocol.AssertionHash, error) {
	return e.assertionHash, nil
}

func (e *specEdge) TimeUnrivaled(ctx context.Context) (uint64, error) {
	if e.hasRival && e.timeUnrivaled.IsSome() {
		return e.timeUnrivaled.Unwrap(), nil
	}
	timer, err := e.manager.caller.TimeUnrivaled(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), e.id)
	if err != nil {
		return 0, err
	}
	if !timer.IsUint64() {
		return 0, fmt.Errorf("received time unrivaled > max uint64 for edge %#x", e.id)
	}
	return timer.Uint64(), nil
}

func (e *specEdge) HasRival(ctx context.Context) (bool, error) {
	if e.hasRival {
		return e.hasRival, nil
	}
	hasRival, err := e.manager.caller.HasRival(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), e.id)
	if err != nil {
		return false, err
	}
	if hasRival {
		e.hasRival = true
	}
	return hasRival, nil
}

func (e *specEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	if e.isConfirmed {
		return protocol.EdgeConfirmed, nil
	}
	edge, err := e.fetchEdge(ctx)
	if err != nil {
		return 0, err
	}
	return protocol.EdgeStatus(edge.Status), nil
}

func (e *specEdge) ConfirmedAtBlock(ctx context.Context) (uint64, error) {
	if e.confirmedAtBlock.IsSome() {
		return e.confirmedAtBlock.Unwrap(), nil
	}
	edge, err := e.fetchEdge(ctx)
	if err != nil {
		return 0, err
	}
	return edge.ConfirmedAtBlock, nil
}

// CreatedAtBlock the  block number the edge was created at.
func (e *specEdge) CreatedAtBlock() (uint64, error) {
	return e.inner.CreatedAtBlock, nil
}

// HasChildren checks if the edge has children.
func (e *specEdge) HasChildren(ctx context.Context) (bool, error) {
	if e.lowerChild.IsSome() && e.upperChild.IsSome() {
		return true, nil
	}
	edge, err := e.fetchEdge(ctx)
	if err != nil {
		return false, err
	}
	return edge.LowerChildId != ([32]byte{}) && edge.UpperChildId != ([32]byte{}), nil
}

// LowerChild of the edge, if any.
func (e *specEdge) LowerChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	if e.lowerChild.IsSome() {
		return e.lowerChild, nil
	}
	edge, err := e.fetchEdge(ctx)
	if err != nil {
		return option.None[protocol.EdgeId](), err
	}
	if edge.LowerChildId == ([32]byte{}) {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId{
		Hash: edge.LowerChildId,
	}), nil
}

// UpperChild of the edge, if any.
func (e *specEdge) UpperChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	if e.upperChild.IsSome() {
		return e.upperChild, nil
	}
	edge, err := e.fetchEdge(ctx)
	if err != nil {
		return option.None[protocol.EdgeId](), err
	}
	if edge.LowerChildId == ([32]byte{}) {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId{
		Hash: edge.UpperChildId,
	}), nil
}

// MutualId of the edge.
func (e *specEdge) MutualId() protocol.MutualId {
	return e.mutualId
}

func (e *specEdge) OriginId() protocol.OriginId {
	return e.inner.OriginId
}

// ClaimId of the edge, if any.
func (e *specEdge) ClaimId() option.Option[protocol.ClaimId] {
	if e.inner.ClaimId == [32]byte{} {
		return option.None[protocol.ClaimId]()
	}
	return option.Some(protocol.ClaimId(e.inner.ClaimId))
}

// HasLengthOneRival returns true if there's a length one rival.
func (e *specEdge) HasLengthOneRival(ctx context.Context) (bool, error) {
	if e.hasLengthOneRival {
		return e.hasLengthOneRival, nil
	}
	ok, err := e.manager.caller.HasLengthOneRival(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), e.id)
	if err != nil {
		errS := err.Error()
		switch {
		case strings.Contains(errS, "not length 1"):
			return false, nil
		case strings.Contains(errS, "is unrivaled"):
			return false, nil
		default:
			return false, err
		}
	}
	if ok {
		e.hasLengthOneRival = true
	}
	return ok, nil
}

// Bisect the edge, returning the upper and lower edges.
// If the upper child exists, both edges will be returned.
// Lower child may optionally exist so the method will bisect regardless.
func (e *specEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (protocol.VerifiedRoyalEdge, protocol.VerifiedRoyalEdge, error) {
	upperId, err := e.UpperChild(ctx)
	if err != nil {
		return nil, nil, err
	}
	var upperEdge option.Option[protocol.SpecEdge]
	var lowerId option.Option[protocol.EdgeId]
	var lowerEdge option.Option[protocol.SpecEdge]
	if !upperId.IsNone() {
		upperEdge, err = e.manager.GetEdge(ctx, upperId.Unwrap())
		if err != nil {
			return nil, nil, err
		}
		if upperEdge.IsNone() {
			return nil, nil, errors.New("could not refresh upper edge after bisecting, got empty result")
		}
		lowerId, err = e.LowerChild(ctx)
		if err != nil {
			return nil, nil, err
		}
		lowerEdge, err = e.manager.GetEdge(ctx, lowerId.Unwrap())
		if err != nil {
			return nil, nil, err
		}
		if lowerEdge.IsNone() {
			return nil, nil, errors.New("could not refresh lower edge after bisecting, got empty result")
		}
		lower := &honestEdge{lowerEdge.Unwrap()}
		upper := &honestEdge{upperEdge.Unwrap()}
		return lower, upper, nil
	}

	_, err = e.manager.assertionChain.transact(ctx, e.manager.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return e.manager.writer.BisectEdge(opts, e.id, prefixHistoryRoot, prefixProof)
	})
	if err != nil {
		return nil, nil, err
	}
	someEdge, err := e.manager.GetEdge(ctx, protocol.EdgeId{Hash: e.id})
	if err != nil {
		return nil, nil, err
	}
	if someEdge.IsNone() {
		return nil, nil, errors.New("could not refresh edge after bisecting, got empty result")
	}
	edge, ok := someEdge.Unwrap().(*specEdge)
	if !ok {
		return nil, nil, errors.New("not a *SpecEdge")
	}
	// Refresh the edge.
	e = edge
	someLowerChild, err := e.manager.GetEdge(ctx, protocol.EdgeId{Hash: e.inner.LowerChildId})
	if err != nil {
		return nil, nil, err
	}
	someUpperChild, err := e.manager.GetEdge(ctx, protocol.EdgeId{Hash: e.inner.UpperChildId})
	if err != nil {
		return nil, nil, err
	}
	if someLowerChild.IsNone() || someUpperChild.IsNone() {
		return nil, nil, errors.New("expected edge to have children post-bisection, but has none")
	}
	lower := &honestEdge{someLowerChild.Unwrap()}
	upper := &honestEdge{someUpperChild.Unwrap()}
	return lower, upper, nil
}

func (e *specEdge) ConfirmByTimer(ctx context.Context) (*types.Transaction, error) {
	s, err := e.Status(ctx)
	if err != nil {
		return nil, err
	}
	if s == protocol.EdgeConfirmed {
		return nil, nil
	}
	if e.GetChallengeLevel() != protocol.NewBlockChallengeLevel() {
		return nil, errors.New("only block challenge edges can be confirmed by time")
	}
	if e.ClaimId().IsNone() {
		return nil, errors.New("only root edges can be confirmed by time")
	}
	assertionHash := protocol.AssertionHash{
		Hash: e.inner.ClaimId,
	}
	assertionCreation, err := e.manager.assertionChain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return nil, err
	}
	// The confirm by timer used to require a list of ancestors, but it has since
	// been refactored to use them. However, the function signature still needs this empty list.
	receipt, err := e.manager.assertionChain.transact(ctx, e.manager.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByTime(opts, e.id, challengeV2gen.AssertionStateData{
			AssertionState: challengeV2gen.AssertionState{
				GlobalState:    challengeV2gen.GlobalState(assertionCreation.AfterState.GlobalState),
				MachineStatus:  assertionCreation.AfterState.MachineStatus,
				EndHistoryRoot: assertionCreation.AfterState.EndHistoryRoot,
			},
			PrevAssertionHash: assertionCreation.ParentAssertionHash,
			InboxAcc:          assertionCreation.AfterInboxBatchAcc,
		})
	})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not confirm edge %s by time with tx",
			containers.Trunc(e.id[:]),
		)
	}
	tx, _, err := e.manager.backend.TransactionByHash(ctx, receipt.TxHash)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get transaction by hash: %#x", receipt.TxHash)
	}
	return tx, nil
}

// TopLevelClaimHeight gets the height at the BlockChallenge level that originated a subchallenge.
// For example, if two validators open a subchallenge S at edge A in a BlockChallenge, the TopLevelClaimHeight of S is the height of A.
// If two validators open a subchallenge S' at edge B in BigStepChallenge, the TopLevelClaimHeight
// is the height of A.
func (e *specEdge) TopLevelClaimHeight(ctx context.Context) (protocol.OriginHeights, error) {
	challengeLevel := e.GetChallengeLevel()
	if challengeLevel == 0 {
		startHeight, _ := e.StartCommitment()
		return protocol.OriginHeights{
			ChallengeOriginHeights: []protocol.Height{startHeight},
		}, nil
	}
	challengeOriginHeights := make([]protocol.Height, challengeLevel)
	originId := e.inner.OriginId
	for challengeLevel > 0 {
		if ctx.Err() != nil {
			return protocol.OriginHeights{}, ctx.Err()
		}
		rivalId, err := e.manager.caller.FirstRival(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), originId)
		if err != nil {
			return protocol.OriginHeights{}, err
		}
		challengeOneStepForkSource, err := e.manager.GetEdge(ctx, protocol.EdgeId{Hash: rivalId})
		if err != nil {
			return protocol.OriginHeights{}, errors.Wrapf(err, "big step challenge one step fork source does not exist: origin id %#x, rival %#x, challenge level %d", originId, rivalId, challengeLevel)
		}
		if challengeOneStepForkSource.IsNone() {
			return protocol.OriginHeights{}, errors.New("source edge is none")
		}
		bigStepEdge, ok := challengeOneStepForkSource.Unwrap().(*specEdge)
		if !ok {
			return protocol.OriginHeights{}, errors.New("not *SpecEdge")
		}
		bigStepStartHeight, _ := bigStepEdge.StartCommitment()

		challengeOriginHeights[challengeLevel-1] = bigStepStartHeight
		originId = bigStepEdge.inner.OriginId

		challengeLevel--
	}
	return protocol.OriginHeights{
		ChallengeOriginHeights: challengeOriginHeights,
	}, nil
}

// Wrapper around the challenge manager contract with developer-friendly methods.
type specChallengeManager struct {
	addr                  common.Address
	backend               protocol.ChainBackend
	assertionChain        *AssertionChain
	txOpts                *bind.TransactOpts
	caller                *challengeV2gen.EdgeChallengeManagerCaller
	writer                *challengeV2gen.EdgeChallengeManagerTransactor
	filterer              *challengeV2gen.EdgeChallengeManagerFilterer
	challengePeriodBlocks uint64
	numBigStepLevel       uint8
}

// NewSpecChallengeManager returns an instance of the spec challenge manager
// used by the assertion chain.
func NewSpecChallengeManager(
	ctx context.Context,
	addr common.Address,
	assertionChain *AssertionChain,
	backend protocol.ChainBackend,
	txOpts *bind.TransactOpts,
) (protocol.SpecChallengeManager, error) {
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(addr, backend)
	if err != nil {
		return nil, err
	}
	numBigStepLevel, err := managerBinding.EdgeChallengeManagerCaller.NUMBIGSTEPLEVEL(assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, err
	}
	challengePeriodBlocks, err := managerBinding.EdgeChallengeManagerCaller.ChallengePeriodBlocks(assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, err
	}
	return &specChallengeManager{
		addr:                  addr,
		assertionChain:        assertionChain,
		backend:               backend,
		txOpts:                txOpts,
		caller:                &managerBinding.EdgeChallengeManagerCaller,
		writer:                &managerBinding.EdgeChallengeManagerTransactor,
		filterer:              &managerBinding.EdgeChallengeManagerFilterer,
		numBigStepLevel:       numBigStepLevel,
		challengePeriodBlocks: challengePeriodBlocks,
	}, nil
}

func (cm *specChallengeManager) Address() common.Address {
	return cm.addr
}

func (cm *specChallengeManager) LayerZeroHeights(ctx context.Context) (*protocol.LayerZeroHeights, error) {
	h, err := cm.caller.LAYERZEROBLOCKEDGEHEIGHT(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, err
	}
	if !h.IsUint64() {
		return nil, errors.New("layer zero block edge height was not a uint64")
	}
	bs, err := cm.caller.LAYERZEROBIGSTEPEDGEHEIGHT(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, err
	}
	if !bs.IsUint64() {
		return nil, errors.New("layer zero big step edge height was not a uint64")
	}
	ss, err := cm.caller.LAYERZEROSMALLSTEPEDGEHEIGHT(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, err
	}
	if !ss.IsUint64() {
		return nil, errors.New("layer zero small step height was not a uint64")
	}
	return &protocol.LayerZeroHeights{
		BlockChallengeHeight:     h.Uint64(),
		BigStepChallengeHeight:   bs.Uint64(),
		SmallStepChallengeHeight: ss.Uint64(),
	}, nil
}

func (cm *specChallengeManager) NumBigSteps(ctx context.Context) (uint8, error) {
	return cm.numBigStepLevel, nil
}

func (cm *specChallengeManager) LevelZeroBlockEdgeHeight(ctx context.Context) (uint64, error) {
	h, err := cm.caller.LAYERZEROBLOCKEDGEHEIGHT(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return 0, err
	}
	if !h.IsUint64() {
		return 0, errors.New("level zero block edge height was not a uint64")
	}
	return h.Uint64(), nil
}

// ChallengePeriodBlocks is the duration of the challenge period in blocks.
func (cm *specChallengeManager) ChallengePeriodBlocks(
	ctx context.Context,
) (uint64, error) {
	return cm.challengePeriodBlocks, nil
}

var uint8Type = newStaticType("uint8", "", nil)
var uint256Type = newStaticType("uint256", "", nil)
var mutualIdAbi = abi.Arguments{
	{Type: uint8Type, Name: "level"},
	{Type: bytes32Type, Name: "originId"},
	{Type: uint256Type, Name: "startHeight"},
	{Type: bytes32Type, Name: "startHistoryRoot"},
	{Type: uint256Type, Name: "endHeight"},
}

func calculateMutualId(level uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) (common.Hash, error) {
	mutualIdByte, err := mutualIdAbi.Pack(
		level,
		originId,
		startHeight,
		startHistoryRoot,
		endHeight,
	)
	if err != nil {
		return common.Hash{}, err
	}
	// Pack stores level(uint8) as 32 bytes, so we need to slice off the first 31 bytes
	return crypto.Keccak256Hash(mutualIdByte[31:]), nil
}

// GetEdge gets an edge by its hash.
func (cm *specChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (option.Option[protocol.SpecEdge], error) {
	edge, err := cm.caller.GetEdge(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), edgeId.Hash)
	if err != nil {
		return option.None[protocol.SpecEdge](), err
	}
	miniStaker := option.None[common.Address]()
	if edge.Staker != (common.Address{}) {
		miniStaker = option.Some(edge.Staker)
	}
	mutual, err := calculateMutualId(
		edge.Level,
		edge.OriginId,
		edge.StartHeight,
		edge.StartHistoryRoot,
		edge.EndHeight,
	)
	if err != nil {
		return option.None[protocol.SpecEdge](), err
	}
	if !edge.StartHeight.IsUint64() {
		return option.None[protocol.SpecEdge](), errors.New("start height not a uint64")
	}
	if !edge.EndHeight.IsUint64() {
		return option.None[protocol.SpecEdge](), errors.New("end height not a uint64")
	}
	numbigsteplevel, err := cm.NumBigSteps(ctx)
	if err != nil {
		return option.Option[protocol.SpecEdge]{}, err
	}
	assertionHash, err := cm.caller.GetPrevAssertionHash(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), edgeId.Hash)
	if err != nil {
		return option.Option[protocol.SpecEdge]{}, err
	}
	return option.Some(protocol.SpecEdge(&specEdge{
		id:                   edgeId.Hash,
		mutualId:             mutual,
		manager:              cm,
		inner:                edge,
		startHeight:          edge.StartHeight.Uint64(),
		endHeight:            edge.EndHeight.Uint64(),
		miniStaker:           miniStaker,
		totalChallengeLevels: numbigsteplevel + 2,
		assertionHash:        protocol.AssertionHash{Hash: common.Hash(assertionHash)},
	})), nil
}

func (e *specEdge) SafeHeadInheritedTimer(ctx context.Context) (protocol.InheritedTimer, error) {
	edge, err := e.manager.caller.GetEdge(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), e.id)
	if err != nil {
		return 0, err
	}
	if edgetracker.IsRootBlockChallengeEdge(e) {
		assertionUnrivaledBlocks, err := e.manager.assertionChain.AssertionUnrivaledBlocks(ctx, protocol.AssertionHash{Hash: common.Hash(e.ClaimId().Unwrap())})
		if err != nil {
			return 0, err
		}
		return protocol.InheritedTimer(edge.TotalTimeUnrivaledCache + assertionUnrivaledBlocks), nil
	}
	return protocol.InheritedTimer(edge.TotalTimeUnrivaledCache), nil
}

func (e *specEdge) LatestInheritedTimer(ctx context.Context) (protocol.InheritedTimer, error) {
	edge, err := e.manager.caller.GetEdge(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), e.id)
	if err != nil {
		return 0, err
	}
	if edgetracker.IsRootBlockChallengeEdge(e) {
		// TODO: Use latest here as well.
		assertionUnrivaledBlocks, err := e.manager.assertionChain.AssertionUnrivaledBlocks(
			ctx,
			protocol.AssertionHash{Hash: common.Hash(e.ClaimId().Unwrap())},
		)
		if err != nil {
			return 0, err
		}
		return protocol.InheritedTimer(edge.TotalTimeUnrivaledCache + assertionUnrivaledBlocks), nil
	}
	return protocol.InheritedTimer(edge.TotalTimeUnrivaledCache), nil
}

func (e *specEdge) fetchEdge(
	ctx context.Context,
) (challengeV2gen.ChallengeEdge, error) {
	edge, err := e.manager.caller.GetEdge(e.manager.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), e.id)
	if err != nil {
		return challengeV2gen.ChallengeEdge{}, err
	}

	// Update the edge with the latest data, if they are in now in constant state.
	if protocol.EdgeStatus(edge.Status) == protocol.EdgeConfirmed {
		e.isConfirmed = true
	}
	if edge.ConfirmedAtBlock != 0 {
		e.confirmedAtBlock = option.Some(edge.ConfirmedAtBlock)
	}
	if edge.LowerChildId != ([32]byte{}) {
		e.lowerChild = option.Some(protocol.EdgeId{Hash: edge.LowerChildId})
	}
	if edge.UpperChildId != ([32]byte{}) {
		e.upperChild = option.Some(protocol.EdgeId{Hash: edge.UpperChildId})
	}
	return edge, nil
}

// CalculateEdgeId calculates an edge hash given its challenge id, start history, and end history.
func (cm *specChallengeManager) CalculateEdgeId(
	ctx context.Context,
	challengeLevel protocol.ChallengeLevel,
	originId protocol.OriginId,
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
	endHistoryRoot common.Hash,
) (protocol.EdgeId, error) {
	id, err := cm.caller.CalculateEdgeId(
		cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}),
		challengeLevel.Uint8(),
		originId,
		big.NewInt(int64(startHeight)),
		startHistoryRoot,
		big.NewInt(int64(endHeight)),
		endHistoryRoot,
	)
	return protocol.EdgeId{Hash: id}, err
}

func (cm *specChallengeManager) MultiUpdateInheritedTimers(
	ctx context.Context,
	challengeBranch []protocol.ReadOnlyEdge,
	desiredNewTimerForLastEdge uint64,
) (*types.Transaction, error) {
	if len(challengeBranch) == 0 {
		return nil, errors.New("no edges to update")
	}
	edgeIds := make([][32]byte, 0)
	var lastReceipt *types.Receipt
	for _, edgeId := range challengeBranch {
		edgeIds = append(edgeIds, edgeId.Id().Hash)
		if challengetree.IsClaimingAnEdge(edgeId) {
			_, err := cm.assertionChain.transact(
				ctx,
				cm.assertionChain.backend,
				func(opts *bind.TransactOpts) (*types.Transaction, error) {
					return cm.writer.MultiUpdateTimeCacheByChildren(
						opts,
						edgeIds,
						new(big.Int).SetUint64(desiredNewTimerForLastEdge),
					)
				},
				withoutSafeWait(),
			)
			if err != nil {
				return nil, errors.Wrap(
					err,
					"could not update inherited timer for multiple edge ids",
				)
			}
			receipt, err := cm.assertionChain.transact(
				ctx,
				cm.assertionChain.backend,
				func(opts *bind.TransactOpts) (*types.Transaction, error) {
					return cm.writer.UpdateTimerCacheByClaim(
						opts,
						edgeId.ClaimId().Unwrap(),
						edgeId.Id().Hash,
						new(big.Int).SetUint64(desiredNewTimerForLastEdge),
					)
				},
				withoutSafeWait(),
			)
			if err != nil {
				return nil, errors.Wrap(
					err,
					"could not update inherited timer for multiple edge ids",
				)
			}
			edgeIds = make([][32]byte, 0)
			lastReceipt = receipt
		}
	}
	if len(edgeIds) > 0 {
		receipt, err := cm.assertionChain.transact(
			ctx,
			cm.assertionChain.backend,
			func(opts *bind.TransactOpts) (*types.Transaction, error) {
				return cm.writer.MultiUpdateTimeCacheByChildren(
					opts,
					edgeIds,
					new(big.Int).SetUint64(desiredNewTimerForLastEdge),
				)
			},
			withoutSafeWait(),
		)
		if err != nil {
			return nil, errors.Wrap(
				err,
				"could not update inherited timer for multiple edge ids",
			)
		}
		lastReceipt = receipt
	}
	tx, _, err := cm.backend.TransactionByHash(ctx, lastReceipt.TxHash)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get transaction by hash: %#x", lastReceipt.TxHash)
	}
	return tx, nil
}

// ConfirmEdgeByOneStepProof checks a one step proof for a tentative winner edge id
// which will mark it as the winning claim of its associated challenge if correct.
// The edges along the winning branch and the corresponding assertion then need to be confirmed
// through separate transactions, if this succeeds.
func (cm *specChallengeManager) ConfirmEdgeByOneStepProof(
	ctx context.Context,
	tentativeWinnerId protocol.EdgeId,
	oneStepData *protocol.OneStepData,
	preHistoryInclusionProof []common.Hash,
	postHistoryInclusionProof []common.Hash,
) error {
	edge, err := cm.GetEdge(ctx, tentativeWinnerId)
	if err != nil {
		return err
	}
	s, err := edge.Unwrap().Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}

	assertionHash, err := edge.Unwrap().AssertionHash(ctx)
	if err != nil {
		return err
	}
	creationInfo, err := cm.assertionChain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	if !creationInfo.InboxMaxCount.IsUint64() {
		return errors.New("inbox max count not a uint64")
	}

	pre := make([][32]byte, len(preHistoryInclusionProof))
	for i, r := range preHistoryInclusionProof {
		pre[i] = r
	}
	post := make([][32]byte, len(postHistoryInclusionProof))
	for i, r := range postHistoryInclusionProof {
		post[i] = r
	}

	machineStep, _ := edge.Unwrap().StartCommitment()
	ospEntryAddr, err := cm.caller.OneStepProofEntry(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return err
	}
	ospBindings, err := ospgen.NewOneStepProofEntryCaller(ospEntryAddr, cm.backend)
	if err != nil {
		return err
	}
	bridgeAddr, err := cm.assertionChain.rollup.Bridge(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return err
	}
	execCtx := ospgen.ExecutionContext{
		MaxInboxMessagesRead:  creationInfo.InboxMaxCount,
		Bridge:                bridgeAddr,
		InitialWasmModuleRoot: creationInfo.WasmModuleRoot,
	}
	result, err := ospBindings.ProveOneStep(
		cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}),
		execCtx,
		big.NewInt(int64(machineStep)),
		oneStepData.BeforeHash,
		oneStepData.Proof,
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not pre-check one step proof at machine step %d: before hash %#x, computed after hash %#x, actual expected after hash %#x",
			machineStep,
			oneStepData.BeforeHash,
			oneStepData.AfterHash,
			result,
		)
	}
	if _, err = cm.assertionChain.transact(
		ctx,
		cm.assertionChain.backend,
		func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return cm.writer.ConfirmEdgeByOneStepProof(
				opts,
				tentativeWinnerId.Hash,
				challengeV2gen.OneStepData{
					BeforeHash: oneStepData.BeforeHash,
					Proof:      oneStepData.Proof,
				},
				challengeV2gen.ConfigData{
					WasmModuleRoot:      creationInfo.WasmModuleRoot,
					RequiredStake:       creationInfo.RequiredStake,
					ChallengeManager:    creationInfo.ChallengeManager,
					ConfirmPeriodBlocks: creationInfo.ConfirmPeriodBlocks,
					NextInboxPosition:   creationInfo.InboxMaxCount.Uint64(),
				},
				pre,
				post,
			)
		}); err != nil {
		errorConfirmingEdgeByOneStepProofCounter.Inc(1)
		return errors.Wrapf(
			err,
			"could not confirm one step proof at machine step %d: before hash %#x, computed after hash %#x, actual expected after hash %#x",
			machineStep,
			oneStepData.BeforeHash,
			oneStepData.AfterHash,
			result,
		)
	}

	return err
}

// Like abi.NewType but panics if it errors for use in constants
func newStaticType(t string, internalType string, components []abi.ArgumentMarshaling) abi.Type {
	ty, err := abi.NewType(t, internalType, components)
	if err != nil {
		panic(err)
	}
	return ty
}

var bytes32Type = newStaticType("bytes32", "", nil)
var bytes32ArrayType = newStaticType("bytes32[]", "", []abi.ArgumentMarshaling{{Type: "bytes32"}})
var assertionStateData = newStaticType("tuple", "AssertionStateData", []abi.ArgumentMarshaling{
	{
		Type:         "tuple",
		InternalType: "AssertionState",
		Name:         "assertionState",
		Components: []abi.ArgumentMarshaling{
			{
				Type:         "tuple",
				InternalType: "GlobalState",
				Name:         "globalState",
				Components: []abi.ArgumentMarshaling{
					{
						Type: "bytes32[2]",
						Components: []abi.ArgumentMarshaling{{
							Type: "bytes32",
						}},
						Name: "bytes32Vals",
					},
					{
						Type: "uint64[2]",
						Components: []abi.ArgumentMarshaling{{
							Type: "uint64",
						}},
						Name: "u64Vals",
					},
				},
			},
			{
				Type:         "uint8",
				InternalType: "MachineStatus",
				Name:         "machineStatus",
			},
			{
				Type: "bytes32",
				Name: "endHistoryRoot",
			},
		},
	},
	{
		Type: "bytes32",
		Name: "prevAssertionHash",
	},
	{
		Type: "bytes32",
		Name: "inboxAcc",
	},
})

var blockEdgeCreateProofAbi = abi.Arguments{
	{
		Name: "inclusionProof",
		Type: bytes32ArrayType,
	},
	{
		Name: "startState",
		Type: assertionStateData,
	},
	{
		Name: "endState",
		Type: assertionStateData,
	},
}

type AssertionStateData struct {
	AssertionState    rollupgen.AssertionState
	PrevAssertionHash [32]byte
	InboxAcc          [32]byte
}

func (cm *specChallengeManager) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
	startCommit,
	endCommit commitments.History,
	startEndPrefixProof []byte,
) (protocol.VerifiedRoyalEdge, error) {
	assertionCreation, err := cm.assertionChain.ReadAssertionCreationInfo(ctx, assertion.Id())
	if err != nil {
		return nil, fmt.Errorf("could not read assertion %#x creation info: %w", assertion.Id(), err)
	}
	prevId, err := assertion.PrevId(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get assertion prev id for assertion %#x", assertion.Id().Hash)
	}
	parentAssertionCreation, err := cm.assertionChain.ReadAssertionCreationInfo(ctx, prevId)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read parent assertion %#x creation info", prevId)
	}
	levelZeroBlockHeight, err := cm.caller.LAYERZEROBLOCKEDGEHEIGHT(cm.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, errors.Wrap(err, "could not get level zero block edge height")
	}
	if !levelZeroBlockHeight.IsUint64() {
		return nil, errors.New("level zero block height not a uint64")
	}
	if endCommit.Height != levelZeroBlockHeight.Uint64() {
		return nil, fmt.Errorf(
			"end commit has unexpected height %v (expected %v)",
			endCommit.Height,
			levelZeroBlockHeight.Uint64(),
		)
	}
	blockEdgeProof, err := blockEdgeCreateProofAbi.Pack(
		endCommit.LastLeafProof,
		AssertionStateData{
			AssertionState:    parentAssertionCreation.AfterState,
			PrevAssertionHash: parentAssertionCreation.ParentAssertionHash,
			InboxAcc:          parentAssertionCreation.AfterInboxBatchAcc,
		},
		AssertionStateData{
			AssertionState:    assertionCreation.AfterState,
			PrevAssertionHash: assertionCreation.ParentAssertionHash,
			InboxAcc:          assertionCreation.AfterInboxBatchAcc,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not serialize block edge proof: %w", err)
	}

	edgeId, err := cm.CalculateEdgeId(
		ctx,
		protocol.NewBlockChallengeLevel(),
		protocol.OriginId(assertionCreation.ParentAssertionHash),
		protocol.Height(startCommit.Height),
		startCommit.Merkle,
		protocol.Height(endCommit.Height),
		endCommit.Merkle,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not calculate edge id")
	}
	someLevelZeroEdge, err := cm.GetEdge(ctx, edgeId)
	if err == nil && !someLevelZeroEdge.IsNone() {
		return &honestEdge{someLevelZeroEdge.Unwrap()}, nil
	}
	args := challengeV2gen.CreateEdgeArgs{
		Level:          protocol.NewBlockChallengeLevel().Uint8(),
		EndHistoryRoot: endCommit.Merkle,
		EndHeight:      big.NewInt(int64(endCommit.Height)),
		ClaimId:        assertionCreation.AssertionHash,
		PrefixProof:    startEndPrefixProof,
		Proof:          blockEdgeProof,
	}
	receipt, err := cm.assertionChain.transact(ctx, cm.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return cm.writer.CreateLayerZeroEdge(
			opts,
			args,
		)
	})
	if err != nil {
		if strings.Contains(err.Error(), InvalidInclusionProofError) {
			invalidInclusionProofCounter.Inc(1)
		}
		return nil, fmt.Errorf("could not create root block challenge edge: %w", err)
	}
	if len(receipt.Logs) == 0 {
		return nil, errors.New("no logs observed from root block challenge edge ")
	}
	var edgeAdded *challengeV2gen.EdgeChallengeManagerEdgeAdded
	var found bool
	for _, log := range receipt.Logs {
		creationEvent, creationErr := cm.filterer.ParseEdgeAdded(*log)
		if creationErr == nil {
			edgeAdded = creationEvent
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("could not find edge added event in logs")
	}
	someLevelZeroEdge, err = cm.GetEdge(ctx, protocol.EdgeId{Hash: edgeAdded.EdgeId})
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created edge by id: %#x", edgeAdded.EdgeId)
	}
	if someLevelZeroEdge.IsNone() {
		return nil, fmt.Errorf("edge with id %#x was not found onchain", edgeAdded.EdgeId)
	}
	return &honestEdge{someLevelZeroEdge.Unwrap()}, nil
}

var subchallengeEdgeProofAbi = abi.Arguments{
	{
		Name: "startState",
		Type: bytes32Type,
	},
	{
		Name: "endState",
		Type: bytes32Type,
	},
	{
		Name: "claimStartInclusionProof",
		Type: bytes32ArrayType,
	},
	{
		Name: "claimEndInclusionProof",
		Type: bytes32ArrayType,
	},
	{
		Name: "edgeInclusionProof",
		Type: bytes32ArrayType,
	},
}

func (cm *specChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startCommit,
	endCommit commitments.History,
	startParentInclusionProof,
	endParentInclusionProof []common.Hash,
	startEndPrefixProof []byte,
) (protocol.VerifiedRoyalEdge, error) {
	chalLevel := challengedEdge.GetChallengeLevel()
	subChalTyp := chalLevel.Next()

	// First check if the edge already exists.
	mutualId := challengedEdge.MutualId()
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
	e, err := cm.GetEdge(ctx, edgeId)
	if err == nil {
		if e.IsNone() {
			return nil, errors.New("got empty, newly created level zero edge")
		}
		return &honestEdge{e.Unwrap()}, nil
	}

	subchallengeEdgeProof, err := subchallengeEdgeProofAbi.Pack(
		startCommit.FirstLeaf,
		endCommit.LastLeaf,
		startParentInclusionProof,
		endParentInclusionProof,
		endCommit.LastLeafProof,
	)
	if err != nil {
		return nil, err
	}
	_, err = cm.assertionChain.transact(ctx, cm.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return cm.writer.CreateLayerZeroEdge(
			opts,
			challengeV2gen.CreateEdgeArgs{
				Level:          subChalTyp.Uint8(),
				EndHistoryRoot: endCommit.Merkle,
				EndHeight:      big.NewInt(int64(endCommit.Height)),
				ClaimId:        challengedEdge.Id().Hash,
				PrefixProof:    startEndPrefixProof,
				Proof:          subchallengeEdgeProof,
			},
		)
	})
	if err != nil {
		if strings.Contains(err.Error(), InvalidInclusionProofError) {
			invalidInclusionProofCounter.Inc(1)
		}
		return nil, err
	}

	e, err = cm.GetEdge(ctx, edgeId)
	if err != nil {
		return nil, err
	}
	if e.IsNone() {
		return nil, errors.New("got empty, newly created level zero edge")
	}
	return &honestEdge{e.Unwrap()}, nil
}
