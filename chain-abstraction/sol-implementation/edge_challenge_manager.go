package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/pkg/errors"
)

func (e *SpecEdge) Id() protocol.EdgeId {
	return e.id
}

func (e *SpecEdge) GetType() protocol.EdgeType {
	return protocol.EdgeType(e.inner.EType)
}

func (e *SpecEdge) MiniStaker() option.Option[common.Address] {
	return e.miniStaker
}

func (e *SpecEdge) StartCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.startHeight), e.inner.StartHistoryRoot
}

func (e *SpecEdge) EndCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.endHeight), e.inner.EndHistoryRoot
}

func (e *SpecEdge) AssertionId(ctx context.Context) (protocol.AssertionId, error) {
	return e.manager.caller.GetPrevAssertionId(&bind.CallOpts{Context: ctx}, e.id)
}

func (e *SpecEdge) TimeUnrivaled(ctx context.Context) (uint64, error) {
	timer, err := e.manager.caller.TimeUnrivaled(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return 0, err
	}
	if !timer.IsInt64() {
		return 0, errors.New("time unrivaled result was not a uint64")
	}
	return timer.Uint64(), nil
}

func (e *SpecEdge) HasRival(ctx context.Context) (bool, error) {
	return e.manager.caller.HasRival(&bind.CallOpts{Context: ctx}, e.id)
}

func (e *SpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return 0, err
	}
	return protocol.EdgeStatus(edge.Status), nil
}

// The block number the edge was created at.
func (e *SpecEdge) CreatedAtBlock() (uint64, error) {
	if !e.inner.CreatedAtBlock.IsUint64() {
		return 0, errors.New("edge created at block was not a uint64")
	}
	return e.inner.CreatedAtBlock.Uint64(), nil
}

// Checks if the edge has children.
func (e *SpecEdge) HasChildren(ctx context.Context) (bool, error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return false, err
	}
	return edge.LowerChildId != ([32]byte{}) && edge.UpperChildId != ([32]byte{}), nil
}

// The lower child of the edge, if any.
func (e *SpecEdge) LowerChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return option.None[protocol.EdgeId](), err
	}
	if edge.LowerChildId == ([32]byte{}) {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId(edge.LowerChildId)), nil
}

// The upper child of the edge, if any.
func (e *SpecEdge) UpperChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	edge, err := e.manager.caller.GetEdge(&bind.CallOpts{Context: ctx}, e.id)
	if err != nil {
		return option.None[protocol.EdgeId](), err
	}
	if edge.LowerChildId == ([32]byte{}) {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId(edge.UpperChildId)), nil
}

// The mutual id of the edge.
func (e *SpecEdge) MutualId() protocol.MutualId {
	return e.mutualId
}

func (e *SpecEdge) OriginId() protocol.OriginId {
	return e.inner.OriginId
}

// The claim id of the edge, if any.
func (e *SpecEdge) ClaimId() option.Option[protocol.ClaimId] {
	if e.inner.ClaimId == [32]byte{} {
		return option.None[protocol.ClaimId]()
	}
	return option.Some(protocol.ClaimId(e.inner.ClaimId))
}

func (e *SpecEdge) HasLengthOneRival(ctx context.Context) (bool, error) {
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
func (e *SpecEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (protocol.SpecEdge, protocol.SpecEdge, error) {
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
		return lowerEdge.Unwrap(), upperEdge.Unwrap(), nil
	}

	_, err = transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
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
	s, err := e.Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}
	var assertionId protocol.AssertionId
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
		if topEdge.GetType() != protocol.BlockChallengeEdge {
			return errors.New("top level ancestor must be a block challenge edge")
		}
		assertionId = protocol.AssertionId(topEdge.ClaimId().Unwrap())
	} else {
		assertionId = e.inner.ClaimId
	}
	assertionCreation, err := e.manager.assertionChain.ReadAssertionCreationInfo(ctx, assertionId)
	if err != nil {
		return err
	}
	ancestors := make([][32]byte, len(ancestorIds))
	for i, r := range ancestorIds {
		ancestors[i] = r
	}
	_, err = transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByTime(e.manager.txOpts, e.id, ancestors, challengeV2gen.ExecutionStateData{
			ExecutionState: challengeV2gen.ExecutionState{
				GlobalState:   challengeV2gen.GlobalState(assertionCreation.AfterState.GlobalState),
				MachineStatus: assertionCreation.AfterState.MachineStatus,
			},
			PrevAssertionHash: assertionCreation.ParentAssertionHash,
			InboxAcc:          assertionCreation.AfterInboxBatchAcc,
		})
	})
	return err
}

func (e *SpecEdge) ConfirmByChildren(ctx context.Context) error {
	s, err := e.Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}

	_, err = transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByChildren(e.manager.txOpts, e.id)
	})
	return err
}

func (e *SpecEdge) ConfirmByClaim(ctx context.Context, claimId protocol.ClaimId) error {
	s, err := e.Status(ctx)
	if err != nil {
		return err
	}
	if s == protocol.EdgeConfirmed {
		return nil
	}

	// TODO: Add in fields.
	_, err = transact(ctx, e.manager.backend, e.manager.reader, func() (*types.Transaction, error) {
		return e.manager.writer.ConfirmEdgeByClaim(e.manager.txOpts, e.id, claimId)
	})
	return err
}

// TopLevelClaimHeight gets the height at the BlockChallenge level that originated a subchallenge.
// For example, if two validators open a subchallenge S at edge A in a BlockChallenge, the TopLevelClaimHeight of S is the height of A.
// If two validators open a subchallenge S' at edge B in BigStepChallenge, the TopLevelClaimHeight
// is the height of A.
func (e *SpecEdge) TopLevelClaimHeight(ctx context.Context) (*protocol.OriginHeights, error) {
	switch e.GetType() {
	case protocol.BigStepChallengeEdge:
		rivalId, err := e.manager.caller.FirstRival(&bind.CallOpts{Context: ctx}, e.inner.OriginId)
		if err != nil {
			return nil, err
		}
		blockChallengeOneStepForkSource, err := e.manager.GetEdge(ctx, rivalId)
		if err != nil {
			return nil, errors.Wrapf(err, "block challenge one step fork source does not exist for rival id %#x", rivalId)
		}
		if blockChallengeOneStepForkSource.IsNone() {
			return nil, errors.New("source edge is none")
		}
		startHeight, _ := blockChallengeOneStepForkSource.Unwrap().StartCommitment()
		return &protocol.OriginHeights{
			BlockChallengeOriginHeight: startHeight,
		}, nil
	case protocol.SmallStepChallengeEdge:
		rivalId, err := e.manager.caller.FirstRival(&bind.CallOpts{Context: ctx}, e.inner.OriginId)
		if err != nil {
			return nil, err
		}
		bigStepChallengeOneStepForkSource, err := e.manager.GetEdge(ctx, rivalId)
		if err != nil {
			return nil, errors.Wrap(err, "big step challenge one step fork source does not exist")
		}
		if bigStepChallengeOneStepForkSource.IsNone() {
			return nil, errors.New("source edge is none")
		}
		bigStepEdge, ok := bigStepChallengeOneStepForkSource.Unwrap().(*SpecEdge)
		if !ok {
			return nil, errors.New("not *SpecEdge")
		}
		rivalId, err = e.manager.caller.FirstRival(&bind.CallOpts{Context: ctx}, bigStepEdge.inner.OriginId)
		if err != nil {
			return nil, err
		}
		blockChallengeOneStepForkSource, err := e.manager.GetEdge(ctx, rivalId)
		if err != nil {
			return nil, errors.Wrap(err, "block challenge one step fork source does not exist")
		}
		if blockChallengeOneStepForkSource.IsNone() {
			return nil, errors.New("source edge is none")
		}
		bigStepStartHeight, _ := bigStepEdge.StartCommitment()
		blockChallengeStartHeight, _ := blockChallengeOneStepForkSource.Unwrap().StartCommitment()
		return &protocol.OriginHeights{
			BlockChallengeOriginHeight:   blockChallengeStartHeight,
			BigStepChallengeOriginHeight: bigStepStartHeight,
		}, nil
	default:
		startHeight, _ := e.StartCommitment()
		return &protocol.OriginHeights{
			BlockChallengeOriginHeight: startHeight,
		}, nil
	}
}

// SpecChallengeManager is a wrapper around the challenge manager contract.
type SpecChallengeManager struct {
	addr           common.Address
	backend        ChainBackend
	assertionChain *AssertionChain
	reader         *headerreader.HeaderReader
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
	backend ChainBackend,
	reader *headerreader.HeaderReader,
	txOpts *bind.TransactOpts,
) (protocol.SpecChallengeManager, error) {
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(addr, backend)
	if err != nil {
		return nil, err
	}
	return &SpecChallengeManager{
		addr:           addr,
		assertionChain: assertionChain,
		backend:        backend,
		reader:         reader,
		txOpts:         txOpts,
		caller:         &managerBinding.EdgeChallengeManagerCaller,
		writer:         &managerBinding.EdgeChallengeManagerTransactor,
		filterer:       &managerBinding.EdgeChallengeManagerFilterer,
	}, nil
}

func (cm *SpecChallengeManager) Address() common.Address {
	return cm.addr
}

// Duration of the challenge period in blocks.
func (cm *SpecChallengeManager) ChallengePeriodBlocks(
	ctx context.Context,
) (uint64, error) {
	res, err := cm.caller.ChallengePeriodBlocks(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	if !res.IsUint64() {
		return 0, errors.New("challenge period blocks response was not a uint64")
	}
	return res.Uint64(), nil
}

// Gets an edge by its hash.
func (cm *SpecChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (option.Option[protocol.SpecEdge], error) {
	edge, err := cm.caller.GetEdge(&bind.CallOpts{Context: ctx}, edgeId)
	if err != nil {
		return option.None[protocol.SpecEdge](), err
	}
	miniStaker := option.None[common.Address]()
	if edge.Staker != (common.Address{}) {
		miniStaker = option.Some(edge.Staker)
	}
	mutual, err := cm.caller.CalculateMutualId(
		&bind.CallOpts{Context: ctx},
		edge.EType,
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
	return option.Some(protocol.SpecEdge(&SpecEdge{
		id:          edgeId,
		mutualId:    mutual,
		manager:     cm,
		inner:       edge,
		startHeight: edge.StartHeight.Uint64(),
		endHeight:   edge.EndHeight.Uint64(),
		miniStaker:  miniStaker,
	})), nil
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
		&bind.CallOpts{Context: ctx},
		uint8(edgeType),
		originId,
		big.NewInt(int64(startHeight)),
		startHistoryRoot,
		big.NewInt(int64(endHeight)),
		endHistoryRoot,
	)
}

// ConfirmEdgeByOneStepProof checks a one step proof for a tentative winner edge id
// which will mark it as the winning claim of its associated challenge if correct.
// The edges along the winning branch and the corresponding assertion then need to be confirmed
// through separate transactions, if this succeeds.
func (cm *SpecChallengeManager) ConfirmEdgeByOneStepProof(
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

	assertionId, err := edge.Unwrap().AssertionId(ctx)
	if err != nil {
		return err
	}
	creationInfo, err := cm.assertionChain.ReadAssertionCreationInfo(ctx, assertionId)
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
	_, err = transact(
		ctx,
		cm.assertionChain.backend,
		cm.assertionChain.headerReader,
		func() (*types.Transaction, error) {
			return cm.writer.ConfirmEdgeByOneStepProof(
				cm.assertionChain.txOpts,
				tentativeWinnerId,
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
		})
	// TODO: Handle receipt.
	return err
}

// Like abi.NewType but panics if it fails for use in constants
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

func (cm *SpecChallengeManager) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
	startCommit,
	endCommit commitments.History,
	startEndPrefixProof []byte,
) (protocol.SpecEdge, error) {
	assertionCreation, err := cm.assertionChain.ReadAssertionCreationInfo(ctx, assertion.Id())
	if err != nil {
		return nil, fmt.Errorf("failed to read assertion %#x creation info: %w", assertion.Id(), err)
	}
	prevId, err := assertion.PrevId(ctx)
	if err != nil {
		return nil, err
	}
	parentAssertionCreation, err := cm.assertionChain.ReadAssertionCreationInfo(ctx, prevId)
	if err != nil {
		return nil, fmt.Errorf("failed to read parent assertion %#x creation info: %w", prevId, err)
	}
	if endCommit.Height != protocol.LevelZeroBlockEdgeHeight {
		return nil, fmt.Errorf(
			"end commit has unexpected height %v (expected %v)",
			endCommit.Height,
			protocol.LevelZeroBlockEdgeHeight,
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
		return nil, fmt.Errorf("failed to serialize block edge proof: %w", err)
	}

	edgeId, err := cm.CalculateEdgeId(
		ctx,
		protocol.BlockChallengeEdge,
		protocol.OriginId(assertionCreation.ParentAssertionHash),
		protocol.Height(startCommit.Height),
		startCommit.Merkle,
		protocol.Height(endCommit.Height),
		endCommit.Merkle,
	)
	if err != nil {
		return nil, err
	}
	someLevelZeroEdge, err := cm.GetEdge(ctx, edgeId)
	if err == nil && !someLevelZeroEdge.IsNone() {
		return someLevelZeroEdge.Unwrap(), nil
	}
	_, err = transact(ctx, cm.backend, cm.reader, func() (*types.Transaction, error) {
		return cm.writer.CreateLayerZeroEdge(
			cm.txOpts,
			challengeV2gen.CreateEdgeArgs{
				EdgeType:       uint8(protocol.BlockChallengeEdge),
				EndHistoryRoot: endCommit.Merkle,
				EndHeight:      big.NewInt(int64(endCommit.Height)),
				ClaimId:        assertionCreation.AssertionHash,
				PrefixProof:    startEndPrefixProof,
				Proof:          blockEdgeProof,
			},
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create layer zero edge: %w", err)
	}
	someLevelZeroEdge, err = cm.GetEdge(ctx, edgeId)
	if err != nil {
		return nil, err
	}
	if someLevelZeroEdge.IsNone() {
		return nil, fmt.Errorf("edge with id %#x was not found onchain", edgeId)
	}
	return someLevelZeroEdge.Unwrap(), nil
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

func (cm *SpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startCommit,
	endCommit commitments.History,
	startParentInclusionProof,
	endParentInclusionProof []common.Hash,
	startEndPrefixProof []byte,
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

	// First check if the edge already exists.
	challenged, ok := challengedEdge.(*SpecEdge)
	if !ok {
		return nil, errors.New("not a *SpecEdge")
	}
	mutualId := challenged.MutualId()
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
		return e.Unwrap(), nil
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
	_, err = transact(ctx, cm.backend, cm.reader, func() (*types.Transaction, error) {
		return cm.writer.CreateLayerZeroEdge(
			cm.txOpts,
			challengeV2gen.CreateEdgeArgs{
				EdgeType:       uint8(subChalTyp),
				EndHistoryRoot: endCommit.Merkle,
				EndHeight:      big.NewInt(int64(endCommit.Height)),
				ClaimId:        challengedEdge.Id(),
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
	return e.Unwrap(), nil
}
