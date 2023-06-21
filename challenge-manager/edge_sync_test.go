package challengemanager

import (
	"context"
	"math/big"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/stretchr/testify/require"
)

func Test_getEdgeTrackers(t *testing.T) {
	ctx := context.Background()

	v, m, s := setupValidator(t)
	edge := &mocks.MockSpecEdge{}
	edge.On("AssertionId", ctx).Return(protocol.AssertionId{}, nil)
	m.On("ReadAssertionCreationInfo", ctx, protocol.AssertionId{}).Return(&protocol.AssertionCreatedInfo{InboxMaxCount: big.NewInt(100)}, nil)
	s.On("ExecutionStateBlockHeight", ctx, &protocol.ExecutionState{}).Return(uint64(1), true)

	trackers, err := v.getEdgeTrackers(ctx, []protocol.SpecEdge{protocol.SpecEdge(edge)})
	require.NoError(t, err)
	require.Len(t, trackers, 1)

	require.Equal(t, uint64(1), trackers[0].StartBlockHeight())
	require.Equal(t, uint64(0x64), trackers[0].TopLevelClaimEndBatchCount())
}
