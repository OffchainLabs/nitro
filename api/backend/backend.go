// Package backend handles the business logic for API data fetching
// for BOLD challenge information. It is meant to be fairly abstract and
// well-tested.
package backend

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/OffchainLabs/bold/api"
	"github.com/OffchainLabs/bold/api/db"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/ethereum/go-ethereum/common"
)

type BusinessLogicProvider interface {
	GetAssertions(ctx context.Context, opts ...db.AssertionOption) ([]*api.JsonAssertion, error)
	GetCollectMachineHashes(ctx context.Context, opts ...db.CollectMachineHashesOption) ([]*api.JsonCollectMachineHashes, error)
	GetEdges(ctx context.Context, opts ...db.EdgeOption) ([]*api.JsonEdge, error)
	GetTrackedRoyalEdges(ctx context.Context) ([]*api.JsonEdgesByChallengedAssertion, error)
	GetMiniStakes(ctx context.Context, assertionHash protocol.AssertionHash, opts ...db.EdgeOption) (*api.JsonMiniStakes, error)
	LatestConfirmedAssertion(ctx context.Context) (*api.JsonAssertion, error)
}

type EdgeTrackerFetcher interface {
	GetEdgeTracker(edgeId protocol.EdgeId) option.Option[*edgetracker.Tracker]
}

type Backend struct {
	db               db.ReadUpdateDatabase
	chainDataFetcher protocol.AssertionChain
	chainWatcher     *watcher.Watcher
	trackerFetcher   EdgeTrackerFetcher
}

func NewBackend(
	db db.ReadUpdateDatabase,
	chainDataFetcher protocol.AssertionChain,
	chainWatcher *watcher.Watcher,
	trackerFetcher EdgeTrackerFetcher,
) *Backend {
	return &Backend{
		db:               db,
		chainDataFetcher: chainDataFetcher,
		chainWatcher:     chainWatcher,
		trackerFetcher:   trackerFetcher,
	}
}

func (b *Backend) GetAssertions(ctx context.Context, opts ...db.AssertionOption) ([]*api.JsonAssertion, error) {
	query := &db.AssertionQuery{}
	for _, o := range opts {
		o(query)
	}
	assertions, err := b.db.GetAssertions(opts...)
	if err != nil {
		return nil, err
	}
	if query.ShouldForceUpdate() {
		for _, a := range assertions {
			fetchedAssertion, err := b.chainDataFetcher.GetAssertion(ctx, protocol.AssertionHash{Hash: a.Hash})
			if err != nil {
				return nil, err
			}
			status, err := fetchedAssertion.Status(ctx)
			if err != nil {
				return nil, err
			}
			isFirstChild, err := fetchedAssertion.IsFirstChild()
			if err != nil {
				return nil, err
			}
			firstChildBlock, err := fetchedAssertion.FirstChildCreationBlock()
			if err != nil {
				return nil, err
			}
			secondChildBlock, err := fetchedAssertion.SecondChildCreationBlock()
			if err != nil {
				return nil, err
			}
			a.Status = status.String()
			a.IsFirstChild = isFirstChild
			a.FirstChildBlock = &firstChildBlock
			a.SecondChildBlock = &secondChildBlock
		}
		if err := b.db.UpdateAssertions(assertions); err != nil {
			return nil, err
		}
	}
	return assertions, nil
}

func (b *Backend) GetCollectMachineHashes(ctx context.Context, opts ...db.CollectMachineHashesOption) ([]*api.JsonCollectMachineHashes, error) {
	query := &db.CollectMachineHashesQuery{}
	for _, o := range opts {
		o(query)
	}
	collectMachineHashes, err := b.db.GetCollectMachineHashes(opts...)
	for _, cmh := range collectMachineHashes {
		if cmh.RawStepHeights != "" {
			stepHeightsStr := strings.Split(cmh.RawStepHeights, ",")
			stepHeights := make([]uint64, len(stepHeightsStr))
			for i, stepHeightStr := range stepHeightsStr {
				stepHeight := 0
				if stepHeightStr == "" {
					continue
				}
				stepHeight, err = strconv.Atoi(stepHeightStr)
				if err != nil {
					return nil, fmt.Errorf("could not parse step height %s: %w", stepHeightStr, err)
				}
				stepHeights[i] = uint64(stepHeight)
			}
			cmh.StepHeights = stepHeights
		}
	}
	if err != nil {
		return nil, err
	}
	return collectMachineHashes, nil
}

func (b *Backend) GetEdges(ctx context.Context, opts ...db.EdgeOption) ([]*api.JsonEdge, error) {
	query := &db.EdgeQuery{}
	for _, o := range opts {
		o(query)
	}
	edges, err := b.db.GetEdges(opts...)
	if err != nil {
		return nil, err
	}
	if query.ShouldForceUpdate() {
		chalManager, err := b.chainDataFetcher.SpecChallengeManager(ctx)
		if err != nil {
			return nil, err
		}
		for _, e := range edges {
			edgeOpt, err := chalManager.GetEdge(ctx, protocol.EdgeId{Hash: e.Id})
			if err != nil {
				return nil, err
			}
			if edgeOpt.IsNone() {
				return nil, fmt.Errorf("edge with id %#x was nil onchain", e.Id)
			}
			edge := edgeOpt.Unwrap()
			status, err := edge.Status(ctx)
			if err != nil {
				return nil, err
			}
			hasRival, err := edge.HasRival(ctx)
			if err != nil {
				return nil, err
			}
			hasLengthOneRival, err := edge.HasLengthOneRival(ctx)
			if err != nil {
				return nil, err
			}
			timeUnrivaled, err := edge.TimeUnrivaled(ctx)
			if err != nil {
				return nil, err
			}
			var lowerChildId, upperChildId common.Hash
			var hasChildren bool
			lowerChild, err := edge.LowerChild(ctx)
			if err != nil {
				return nil, err
			}
			upperChild, err := edge.UpperChild(ctx)
			if err != nil {
				return nil, err
			}
			assertionHash, err := edge.AssertionHash(ctx)
			if err != nil {
				return nil, err
			}
			if lowerChild.IsSome() {
				hasChildren = true
				lowerChildId = lowerChild.Unwrap().Hash
			}
			if upperChild.IsSome() {
				hasChildren = true
				upperChildId = upperChild.Unwrap().Hash
			}
			e.Status = status.String()
			e.HasRival = hasRival
			e.HasLengthOneRival = hasLengthOneRival
			e.LowerChildId = lowerChildId
			e.UpperChildId = upperChildId
			e.HasChildren = hasChildren
			e.TimeUnrivaled = timeUnrivaled
			isRoyal := b.chainWatcher.IsRoyal(assertionHash, edge.Id())
			if isRoyal {
				inheritedTimer, err := b.chainWatcher.InheritedTimer(ctx, edge.Id())
				if err != nil {
					return nil, err
				}
				e.InheritedTimer = inheritedTimer
			}
			e.IsRoyal = isRoyal
			trackerOpt := b.trackerFetcher.GetEdgeTracker(edge.Id())
			if trackerOpt.IsSome() {
				fsmState := trackerOpt.Unwrap().FSMSummary()
				e.FSMState = fsmState.CurrentState
				if fsmState.Error != nil {
					e.FSMError = fsmState.Error.Error()
				}
			}
		}
		if err := b.db.UpdateEdges(edges); err != nil {
			return nil, err
		}
	}
	return edges, nil
}

func (b *Backend) GetMiniStakes(ctx context.Context, assertionHash protocol.AssertionHash, opts ...db.EdgeOption) (*api.JsonMiniStakes, error) {
	edgeOpts := opts
	edgeOpts = append(
		edgeOpts,
		db.WithMiniStakerDefined(),
		db.WithEdgeAssertionHash(assertionHash),
		db.WithRootEdges(),
	)
	edges, err := b.db.GetEdges(edgeOpts...)
	if err != nil {
		return nil, err
	}
	stakeInfo := &api.JsonMiniStakes{
		ChallengedAssertionHash: assertionHash.Hash,
		StakesByLvlAndOrigin:    make(map[protocol.ChallengeLevel][]*api.JsonMiniStakeInfo),
	}
	edgesByOriginId := make(map[common.Hash][]*api.JsonEdge)
	for _, e := range edges {
		edgesByOriginId[e.OriginId] = append(edgesByOriginId[e.OriginId], e)
	}
	for originId, originDefinedEdges := range edgesByOriginId {
		lvl := protocol.ChallengeLevel(originDefinedEdges[0].ChallengeLevel)
		if stakeInfo.StakesByLvlAndOrigin[lvl] == nil {
			stakeInfo.StakesByLvlAndOrigin[lvl] = make([]*api.JsonMiniStakeInfo, 0)
		}
		info := &api.JsonMiniStakeInfo{
			ChallengeOriginId:  originId,
			StakerAddresses:    []common.Address{},
			NumberOfMiniStakes: 0,
		}
		for _, e := range originDefinedEdges {
			info.StakerAddresses = append(info.StakerAddresses, e.MiniStaker)
			info.NumberOfMiniStakes += 1
		}
		stakeInfo.StakesByLvlAndOrigin[lvl] = append(stakeInfo.StakesByLvlAndOrigin[lvl], info)
	}
	return stakeInfo, nil
}

func (b *Backend) GetTrackedRoyalEdges(ctx context.Context) ([]*api.JsonEdgesByChallengedAssertion, error) {
	resp, err := b.chainWatcher.GetRoyalEdges(ctx)
	if err != nil {
		return nil, err
	}
	edgesByAssertion := make([]*api.JsonEdgesByChallengedAssertion, 0)
	for assertionHash, e := range resp {
		edgesByAssertion = append(edgesByAssertion, &api.JsonEdgesByChallengedAssertion{
			AssertionHash: assertionHash.Hash,
			RoyalEdges:    e,
		})
	}
	return edgesByAssertion, nil
}

func (b *Backend) LatestConfirmedAssertion(ctx context.Context) (*api.JsonAssertion, error) {
	latestConfirmedAssertion, err := b.chainDataFetcher.LatestConfirmed(ctx)
	if err != nil {
		return nil, err
	}
	hash := latestConfirmedAssertion.Id()
	creationInfo, err := b.chainDataFetcher.ReadAssertionCreationInfo(ctx, hash)
	if err != nil {
		return nil, err
	}
	status, err := b.chainDataFetcher.AssertionStatus(ctx, hash)
	if err != nil {
		return nil, err
	}
	fetchedAssertion, err := b.chainDataFetcher.GetAssertion(ctx, hash)
	if err != nil {
		return nil, err
	}
	isFirstChild, err := fetchedAssertion.IsFirstChild()
	if err != nil {
		return nil, err
	}
	firstChildBlock, err := fetchedAssertion.FirstChildCreationBlock()
	if err != nil {
		return nil, err
	}
	secondChildBlock, err := fetchedAssertion.SecondChildCreationBlock()
	if err != nil {
		return nil, err
	}
	beforeState := protocol.GoExecutionStateFromSolidity(creationInfo.BeforeState)
	afterState := protocol.GoExecutionStateFromSolidity(creationInfo.AfterState)
	return &api.JsonAssertion{
		Hash:                     hash.Hash,
		ConfirmPeriodBlocks:      creationInfo.ConfirmPeriodBlocks,
		RequiredStake:            creationInfo.RequiredStake.String(),
		ParentAssertionHash:      creationInfo.ParentAssertionHash,
		InboxMaxCount:            creationInfo.InboxMaxCount.String(),
		AfterInboxBatchAcc:       creationInfo.AfterInboxBatchAcc,
		WasmModuleRoot:           creationInfo.WasmModuleRoot,
		TransactionHash:          creationInfo.TransactionHash,
		CreationBlock:            creationInfo.CreationBlock,
		ChallengeManager:         creationInfo.ChallengeManager,
		AfterStateBlockHash:      afterState.GlobalState.BlockHash,
		AfterStateSendRoot:       afterState.GlobalState.SendRoot,
		AfterStateBatch:          afterState.GlobalState.Batch,
		AfterStatePosInBatch:     afterState.GlobalState.PosInBatch,
		AfterStateMachineStatus:  afterState.MachineStatus,
		BeforeStateBlockHash:     beforeState.GlobalState.BlockHash,
		BeforeStateSendRoot:      beforeState.GlobalState.SendRoot,
		BeforeStateBatch:         beforeState.GlobalState.Batch,
		BeforeStatePosInBatch:    beforeState.GlobalState.PosInBatch,
		BeforeStateMachineStatus: beforeState.MachineStatus,
		IsFirstChild:             isFirstChild,
		FirstChildBlock:          &firstChildBlock,
		SecondChildBlock:         &secondChildBlock,
		Status:                   status.String(),
		LastUpdatedAt:            time.Now(),
	}, nil
}
