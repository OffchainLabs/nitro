// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
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
	"github.com/pkg/errors"
)

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

func (e *specEdge) AssertionHash(ctx context.Context) (protocol.AssertionHash, error) {
	h, err := e.manager.caller.GetPrevAssertionHash(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	return protocol.AssertionHash{Hash: common.Hash(h)}, nil
}

func (e *specEdge) TimeUnrivaled(ctx context.Context) (uint64, error) {
	timer, err := e.manager.caller.TimeUnrivaled(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return 0, err
	}
	return timer, nil
}

func (e *specEdge) HasConfirmedRival(ctx context.Context) (bool, error) {
	mutualId, err := e.manager.caller.CalculateMutualId(
		&bind.CallOpts{Context: ctx},
		e.inner.Level,
		e.inner.OriginId,
		e.inner.StartHeight,
		e.inner.StartHistoryRoot,
		e.inner.EndHeight,
	)
	if err != nil {
		return false, err
	}
	confirmedRival, err := e.manager.caller.ConfirmedRival(&bind.CallOpts{Context: ctx}, mutualId)
	if err != nil {
		return false, err
	}
	return confirmedRival != ([32]byte{}), nil
}

func (e *specEdge) HasRival(ctx context.Context) (bool, error) {
	return e.manager.caller.HasRival(&bind.CallOpts{Context: ctx}, e.id)
}

func (e *specEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return 0, err
	}
	return protocol.EdgeStatus(edge.Status), nil
}

// CreatedAtBlock the  block number the edge was created at.
func (e *specEdge) CreatedAtBlock() (uint64, error) {
	return e.inner.CreatedAtBlock, nil
}

// HasChildren checks if the edge has children.
func (e *specEdge) HasChildren(ctx context.Context) (bool, error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return false, err
	}
	return edge.LowerChildId != ([32]byte{}) && edge.UpperChildId != ([32]byte{}), nil
}

// LowerChild of the edge, if any.
func (e *specEdge) LowerChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
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
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
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
	ok, err := e.manager.caller.HasLengthOneRival(&bind.CallOpts{Context: ctx}, e.id)
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
	return ok, nil
}

// Bisect the edge, returning the upper and lower edges.
// If the upper child exists, both edges will be returned.
// Lower child may optionally exist so the method will bisect regardless.
func (e *specEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (protocol.VerifiedHonestEdge, protocol.VerifiedHonestEdge, error) {
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

func (e *specEdge) ConfirmByTimer(ctx context.Context, ancestorIds []protocol.EdgeId) error {
	s, err := e.Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}
	var assertionHash protocol.AssertionHash
	if len(ancestorIds) != 0 {
		topLevelAncestorId := ancestorIds[len(ancestorIds)-1]
		topLevelAncestor, topLevelErr := e.manager.GetEdge(ctx, topLevelAncestorId)
		if topLevelErr != nil {
			return topLevelErr
		}
		if topLevelAncestor.IsNone() {
			return fmt.Errorf("did not find edge with id %#x for specified top level ancestor", topLevelAncestorId)
		}
		topEdge := topLevelAncestor.Unwrap()
		challengeLevel := topEdge.GetChallengeLevel()
		if !challengeLevel.IsBlockChallengeLevel() {
			return errors.New("top level ancestor must be a block challenge edge")
		}
		assertionHash = protocol.AssertionHash{
			Hash: common.Hash(topEdge.ClaimId().Unwrap()),
		}
	} else {
		assertionHash = protocol.AssertionHash{
			Hash: e.inner.ClaimId,
		}
	}
	assertionCreation, err := e.manager.assertionChain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	ancestors := make([][32]byte, len(ancestorIds))
	for i, r := range ancestorIds {
		ancestors[i] = r.Hash
	}
	_, err = e.manager.assertionChain.transact(ctx, e.manager.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByTime(opts, e.id, ancestors, challengeV2gen.ExecutionStateData{
			ExecutionState: challengeV2gen.ExecutionState{
				GlobalState:   challengeV2gen.GlobalState(assertionCreation.AfterState.GlobalState),
				MachineStatus: assertionCreation.AfterState.MachineStatus,
			},
			PrevAssertionHash: assertionCreation.ParentAssertionHash,
			InboxAcc:          assertionCreation.AfterInboxBatchAcc,
		})
	})
	ancestorStrings := make([]string, len(ancestorIds))
	for i, r := range ancestorIds {
		ancestorStrings[i] = containers.Trunc(r.Hash[:])
	}
	return errors.Wrapf(
		err,
		"could not confirm edge %s with tx and %d ancestors %v",
		containers.Trunc(e.id[:]),
		len(ancestorIds),
		strings.Join(ancestorStrings, ", "),
	)
}

func (e *specEdge) ConfirmByChildren(ctx context.Context) error {
	s, err := e.Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}

	_, err = e.manager.assertionChain.transact(ctx, e.manager.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByChildren(opts, e.id)
	})
	return err
}

func (e *specEdge) ConfirmByClaim(ctx context.Context, claimId protocol.ClaimId) error {
	s, err := e.Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}

	_, err = e.manager.assertionChain.transact(ctx, e.manager.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByClaim(opts, e.id, claimId)
	})
	return err
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
		rivalId, err := e.manager.caller.FirstRival(&bind.CallOpts{Context: ctx}, originId)
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
	addr           common.Address
	backend        protocol.ChainBackend
	assertionChain *AssertionChain
	txOpts         *bind.TransactOpts
	caller         *challengeV2gen.EdgeChallengeManagerCaller
	writer         *challengeV2gen.EdgeChallengeManagerTransactor
	filterer       *challengeV2gen.EdgeChallengeManagerFilterer
}

// NewSpecChallengeManager returns an instance of the spec challenge manager
// used by the assertion chain.
func NewSpecChallengeManager(
	_ context.Context,
	addr common.Address,
	assertionChain *AssertionChain,
	backend protocol.ChainBackend,
	txOpts *bind.TransactOpts,
) (protocol.SpecChallengeManager, error) {
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(addr, backend)
	if err != nil {
		return nil, err
	}
	return &specChallengeManager{
		addr:           addr,
		assertionChain: assertionChain,
		backend:        backend,
		txOpts:         txOpts,
		caller:         &managerBinding.EdgeChallengeManagerCaller,
		writer:         &managerBinding.EdgeChallengeManagerTransactor,
		filterer:       &managerBinding.EdgeChallengeManagerFilterer,
	}, nil
}

func (cm *specChallengeManager) Address() common.Address {
	return cm.addr
}

func (cm *specChallengeManager) LayerZeroHeights(ctx context.Context) (*protocol.LayerZeroHeights, error) {
	h, err := cm.caller.LAYERZEROBLOCKEDGEHEIGHT(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}
	if !h.IsUint64() {
		return nil, errors.New("layer zero block edge height was not a uint64")
	}
	bs, err := cm.caller.LAYERZEROBIGSTEPEDGEHEIGHT(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}
	if !bs.IsUint64() {
		return nil, errors.New("layer zero big step edge height was not a uint64")
	}
	ss, err := cm.caller.LAYERZEROSMALLSTEPEDGEHEIGHT(&bind.CallOpts{Context: ctx})
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
	n, err := cm.caller.NUMBIGSTEPLEVEL(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return uint8(n), nil
}

func (cm *specChallengeManager) LevelZeroBlockEdgeHeight(ctx context.Context) (uint64, error) {
	h, err := cm.caller.LAYERZEROBLOCKEDGEHEIGHT(&bind.CallOpts{Context: ctx})
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
	res, err := cm.caller.ChallengePeriodBlocks(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return res, nil
}

// GetEdge gets an edge by its hash.
func (cm *specChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (option.Option[protocol.SpecEdge], error) {
	edge, err := cm.caller.GetEdge(&bind.CallOpts{Context: ctx}, edgeId.Hash)
	if err != nil {
		return option.None[protocol.SpecEdge](), err
	}
	miniStaker := option.None[common.Address]()
	if edge.Staker != (common.Address{}) {
		miniStaker = option.Some(edge.Staker)
	}
	mutual, err := cm.caller.CalculateMutualId(
		&bind.CallOpts{Context: ctx},
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
	numbigsteplevel, err := cm.caller.NUMBIGSTEPLEVEL(&bind.CallOpts{Context: ctx})
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
	})), nil
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
		&bind.CallOpts{Context: ctx},
		challengeLevel.Uint8(),
		originId,
		big.NewInt(int64(startHeight)),
		startHistoryRoot,
		big.NewInt(int64(endHeight)),
		endHistoryRoot,
	)
	return protocol.EdgeId{Hash: id}, err
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
	ospEntryAddr, err := cm.caller.OneStepProofEntry(&bind.CallOpts{Context: ctx})
	if err != nil {
		return err
	}
	ospBindings, err := ospgen.NewOneStepProofEntryCaller(ospEntryAddr, cm.backend)
	if err != nil {
		return err
	}
	bridgeAddr, err := cm.assertionChain.rollup.Bridge(&bind.CallOpts{Context: ctx})
	if err != nil {
		return err
	}
	execCtx := ospgen.ExecutionContext{
		MaxInboxMessagesRead:  creationInfo.InboxMaxCount,
		Bridge:                bridgeAddr,
		InitialWasmModuleRoot: creationInfo.WasmModuleRoot,
	}
	result, err := ospBindings.ProveOneStep(
		&bind.CallOpts{Context: ctx},
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
var executionStateData = newStaticType("tuple", "ExecutionStateData", []abi.ArgumentMarshaling{
	{
		Type:         "tuple",
		InternalType: "ExecutionState",
		Name:         "executionState",
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
		Type: executionStateData,
	},
	{
		Name: "endState",
		Type: executionStateData,
	},
}

type ExecutionStateData struct {
	ExecutionState    rollupgen.ExecutionState
	PrevAssertionHash [32]byte
	InboxAcc          [32]byte
}

func (cm *specChallengeManager) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
	startCommit,
	endCommit commitments.History,
	startEndPrefixProof []byte,
) (protocol.VerifiedHonestEdge, error) {
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
	levelZeroBlockHeight, err := cm.caller.LAYERZEROBLOCKEDGEHEIGHT(&bind.CallOpts{Context: ctx})
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
		ExecutionStateData{
			ExecutionState:    parentAssertionCreation.AfterState,
			PrevAssertionHash: parentAssertionCreation.ParentAssertionHash,
			InboxAcc:          parentAssertionCreation.AfterInboxBatchAcc,
		},
		ExecutionStateData{
			ExecutionState:    assertionCreation.AfterState,
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
) (protocol.VerifiedHonestEdge, error) {
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
