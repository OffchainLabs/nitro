package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

// Sync edges from challenges from confirmed block height to latest block height.
// - Get all edges from challenges (retry on fail)
// - Build edge trackers for every edge (retry on fail)
// - Given block still advances while building all the edges trackers. At the end, it checks if it's on the latest block, or loop from the start
// - Once gathered all the sync edges from all the blocks, spin of all the edge trackers as part of go routine.
// nolint:unused
func (v *Validator) syncEdges(ctx context.Context) error {
	latestBlockNum, err := v.getLatestBlockNum(ctx)
	if err != nil {
		return err
	}

	currentBlockNum, err := v.getConfirmedBlockNum(ctx)
	if err != nil {
		return err
	}

	var edgeTrackers []*edgeTracker
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Retry until you get the filterer from the edge challenge manager to filter edge added event.
		filterer, err := util.RetryUntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
			return v.getFilterer(ctx)
		})
		if err != nil {
			return err
		}
		it, err := filterer.FilterEdgeAdded(&bind.FilterOpts{
			Start:   currentBlockNum,
			End:     &latestBlockNum,
			Context: ctx,
		}, nil, nil, nil)
		if err != nil {
			return err
		}
		cm, err := v.chain.SpecChallengeManager(ctx)
		if err != nil {
			return err
		}
		edges, err := v.getEdges(ctx, cm, it)
		if err != nil {
			return err
		}
		trackers, err := v.getEdgeTrackers(ctx, edges)
		if err != nil {
			return err
		}

		edgeTrackers = append(edgeTrackers, trackers...)

		// latest block will keep advance. We shouldn't be done until we've processed all blocks.
		lbn, err := v.getLatestBlockNum(ctx)
		if err != nil {
			return err
		}
		if latestBlockNum == lbn {
			break
		}
		currentBlockNum = latestBlockNum
		latestBlockNum = lbn
	}

	// Spin off all the edge trackers as part of go routine.
	for _, tracker := range edgeTrackers {
		go tracker.spawn(ctx)
	}

	return nil
}

// nolint:unused
func (v *Validator) getFilterer(ctx context.Context) (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
	cm, err := v.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	return challengeV2gen.NewEdgeChallengeManagerFilterer(cm.Address(), v.backend)
}

// get latest block number from the chain.
// nolint:unused
func (v *Validator) getLatestBlockNum(ctx context.Context) (uint64, error) {
	// Retry until you get the latest block number.
	latestBlock, err := util.RetryUntilSucceeds(ctx, func() (*types.Header, error) {
		return v.backend.HeaderByNumber(ctx, nil)
	})
	if err != nil {
		return 0, err
	}
	return latestBlock.Number.Uint64(), nil
}

// get confirmed block number from the chain.
// nolint:unused
func (v *Validator) getConfirmedBlockNum(ctx context.Context) (uint64, error) {
	// Retry until you get the latest confirmed assertion.
	assertion, err := util.RetryUntilSucceeds(ctx, func() (protocol.Assertion, error) {
		return v.chain.LatestConfirmed(ctx)
	})
	if err != nil {
		return 0, err
	}
	return assertion.CreatedAtBlock()
}

// getEdges gets all the edges from edge added events.
// If fails to get an edge given the edge ID, it'll retry until it succeeds.
// nolint:unused
func (v *Validator) getEdges(ctx context.Context, cm protocol.SpecChallengeManager, it *challengeV2gen.EdgeChallengeManagerEdgeAddedIterator) ([]util.Option[protocol.SpecEdge], error) {
	edges := make([]util.Option[protocol.SpecEdge], 0)
	for it.Next() {
		// Retry until you get the edge.
		edge, err := util.RetryUntilSucceeds(ctx, func() (util.Option[protocol.SpecEdge], error) {
			return cm.GetEdge(ctx, it.Event.EdgeId)
		})
		if err != nil {
			return []util.Option[protocol.SpecEdge]{}, err
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

// nolint:unused
func (v *Validator) getExecutionStateBlockHeight(ctx context.Context, st rollupgen.ExecutionState) (uint64, error) {
	height, ok := v.stateManager.ExecutionStateBlockHeight(ctx, protocol.GoExecutionStateFromSolidity(st))
	if !ok {
		return 0, fmt.Errorf("missing previous assertion after execution %+v in local state manager", st)
	}
	return height, nil
}

// getEdgeTrackers builds edge trackers for every edge.
// If fails on getting assertion number or creation info, it'll retry until it succeeds.
// nolint:unused
func (v *Validator) getEdgeTrackers(ctx context.Context, edges []util.Option[protocol.SpecEdge]) ([]*edgeTracker, error) {
	var assertionIdMap = make(map[protocol.AssertionId][2]uint64)
	edgeTrackers := make([]*edgeTracker, len(edges))
	var err error
	var assertionId protocol.AssertionId
	for i, edge := range edges {
		// Retry until you get the previous assertion ID.
		assertionId, err = util.RetryUntilSucceeds(ctx, func() (protocol.AssertionId, error) {
			return edge.Unwrap().PrevAssertionId(ctx)
		})
		if err != nil {
			return nil, err
		}

		// Smart caching to avoid querying the same assertion number and creation info multiple times.
		// Edges in the same challenge should have the same creation info.
		cachedHeightAndInboxMsgCount, ok := assertionIdMap[assertionId]
		var assertionHeight uint64
		var inboxMsgCount uint64
		if !ok {
			// Retry until you get the assertion number.
			assertionNum, assertionErr := util.RetryUntilSucceeds(ctx, func() (protocol.AssertionSequenceNumber, error) {
				return v.chain.GetAssertionNum(ctx, assertionId)
			})
			if assertionErr != nil {
				return nil, assertionErr
			}

			// Retry until you get the assertion creation info.
			assertionCreationInfo, creationErr := util.RetryUntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
				return v.chain.ReadAssertionCreationInfo(ctx, assertionNum)
			})
			if creationErr != nil {
				return nil, creationErr
			}

			// Retry until you get the execution state block height.
			height, heightErr := util.RetryUntilSucceeds(ctx, func() (uint64, error) {
				return v.getExecutionStateBlockHeight(ctx, assertionCreationInfo.AfterState)
			})
			if heightErr != nil {
				return nil, heightErr
			}
			assertionHeight = height
			inboxMsgCount = assertionCreationInfo.InboxMaxCount.Uint64()
			assertionIdMap[assertionId] = [2]uint64{assertionHeight, inboxMsgCount}
		} else {
			assertionHeight, inboxMsgCount = cachedHeightAndInboxMsgCount[0], cachedHeightAndInboxMsgCount[1]
		}
		edgeTrackers[i], err = newEdgeTracker(
			ctx,
			&edgeTrackerConfig{
				timeRef:          v.timeRef,
				actEveryNSeconds: v.edgeTrackerWakeInterval,
				chain:            v.chain,
				stateManager:     v.stateManager,
				validatorName:    v.name,
				validatorAddress: v.address,
			},
			edge.Unwrap(),
			assertionHeight,
			inboxMsgCount,
		)
		if err != nil {
			log.WithError(err).Error("error creating edge tracker")
			continue
		}
	}
	return edgeTrackers, nil
}
