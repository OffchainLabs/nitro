package challengemanager

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	edgetracker "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/edge-tracker"
	retry "github.com/OffchainLabs/challenge-protocol-v2/runtime"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/pkg/errors"
)

// Sync edges from challenges from confirmed block height to latest block height.
// - Get all edges from watcher (retry on fail)
// - Build edge trackers for every edge (retry on fail)
// - Spin of all the edge trackers as part of go routine.
func (m *Manager) syncEdges(ctx context.Context) error {
	if err := m.waitForSync(ctx); err != nil {
		return err
	}
	edges := m.watcher.GetEdges()
	trackers, err := m.getEdgeTrackers(ctx, edges)
	if err != nil {
		return err
	}

	log.WithField(
		"count", len(edges),
	).Infof("Syncing edges for %v", m.name)

	// Spin off all the edge trackers in the background.
	for _, tracker := range trackers {
		go tracker.Spawn(ctx)
	}

	return nil
}

func (m *Manager) getExecutionStateBlockHeight(ctx context.Context, st rollupgen.ExecutionState) (uint64, error) {
	height, ok := m.stateManager.ExecutionStateBlockHeight(ctx, protocol.GoExecutionStateFromSolidity(st))
	if !ok {
		return 0, fmt.Errorf("missing previous assertion after execution %+v in local state manager", st)
	}
	return height, nil
}

// getEdgeTrackers builds edge trackers for every edge.
// If fails on getting assertion number or creation info, it'll retry until it succeeds.
func (m *Manager) getEdgeTrackers(ctx context.Context, edges []protocol.SpecEdge) ([]*edgetracker.Tracker, error) {
	var assertionIdMap = make(map[protocol.AssertionId][2]uint64)
	edgeTrackers := make([]*edgetracker.Tracker, len(edges))
	var err error
	var assertionId protocol.AssertionId
	for i, edge := range edges {
		// Retry until you get the previous assertion ID.
		assertionId, err = retry.UntilSucceeds(ctx, func() (protocol.AssertionId, error) {
			return edge.AssertionId(ctx)
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
			// Retry until you get the assertion creation info.
			assertionCreationInfo, creationErr := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
				return m.chain.ReadAssertionCreationInfo(ctx, assertionId)
			})
			if creationErr != nil {
				return nil, creationErr
			}
			if !assertionCreationInfo.InboxMaxCount.IsUint64() {
				return nil, errors.New("assertion creation info inbox max count was not a uint64")
			}

			// Retry until you get the execution state block height.
			height, heightErr := retry.UntilSucceeds(ctx, func() (uint64, error) {
				return m.getExecutionStateBlockHeight(ctx, assertionCreationInfo.AfterState)
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
		tracker, trackErr := retry.UntilSucceeds(ctx, func() (*edgetracker.Tracker, error) {
			return edgetracker.New(
				edge,
				m.chain,
				m.stateManager,
				m.watcher,
				m,
				edgetracker.HeightConfig{
					StartBlockHeight:           assertionHeight,
					TopLevelClaimEndBatchCount: inboxMsgCount,
				},
				edgetracker.WithActInterval(m.edgeTrackerWakeInterval),
				edgetracker.WithTimeReference(m.timeRef),
				edgetracker.WithValidatorAddress(m.address),
				edgetracker.WithValidatorName(m.name),
			)
		})
		if trackErr != nil {
			return nil, trackErr
		}
		edgeTrackers[i] = tracker
	}
	return edgeTrackers, nil
}
