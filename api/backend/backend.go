// Package backend handles the business logic for API data fetching
// for BOLD challenge information. It is meant to be fairly abstract and
// well-tested.
package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/bold/api"
	"github.com/OffchainLabs/bold/api/db"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	"github.com/ethereum/go-ethereum/common"
)

type BusinessLogicProvider interface {
	GetAssertions(ctx context.Context, opts ...db.AssertionOption) ([]*api.JsonAssertion, error)
	GetEdges(ctx context.Context, opts ...db.EdgeOption) ([]*api.JsonEdge, error)
	GetMiniStakes(ctx context.Context, assertionHash protocol.AssertionHash, opts ...db.EdgeOption) ([]*api.JsonEdge, error)
	LatestConfirmedAssertion(ctx context.Context) (*api.JsonAssertion, error)
}

type Backend struct {
	db               db.ReadUpdateDatabase
	chainDataFetcher protocol.AssertionChain
	chainWatcher     *watcher.Watcher
}

func NewBackend(
	db db.ReadUpdateDatabase,
	chainDataFetcher protocol.AssertionChain,
	chainWatcher *watcher.Watcher,
) *Backend {
	return &Backend{
		db:               db,
		chainDataFetcher: chainDataFetcher,
		chainWatcher:     chainWatcher,
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
				pathTimer, _, _, err := b.chainWatcher.ComputeHonestPathTimer(ctx, assertionHash, edge.Id())
				if err != nil {
					return nil, err
				}
				e.CumulativePathTimer = uint64(pathTimer)
			}
			e.IsRoyal = isRoyal
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
	}
	for _, e := range edges {
		lvl := protocol.ChallengeLevel(e.ChallengeLevel)
		origin := e.OriginId
		if stakeInfo.StakesByLvlAndOrigin[lvl] == nil {
			stakeInfo.StakesByLvlAndOrigin[lvl] = make(map[common.Hash]*api.JsonMiniStakeInfo)
		}
		if stakeInfo.StakesByLvlAndOrigin[lvl][origin] == nil {
			stakeInfo.StakesByLvlAndOrigin[lvl][origin] = &api.JsonMiniStakeInfo{
				StakerAddresses:       []common.Address{},
				NumberOfMiniStakes:    0,
				StartCommitmentHeight: e.StartHeight,
				EndCommitmentHeight:   e.EndHeight,
			}
		}
		stakeInfo.StakesByLvlAndOrigin[lvl][origin].StakerAddresses = append(
			stakeInfo.StakesByLvlAndOrigin[lvl][origin].StakerAddresses,
			e.MiniStaker,
		)
		stakeInfo.StakesByLvlAndOrigin[lvl][origin].NumberOfMiniStakes += 1
	}
	return stakeInfo, nil
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
