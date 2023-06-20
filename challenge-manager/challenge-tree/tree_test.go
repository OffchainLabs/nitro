package challengetree

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/threadsafe"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAddEdge(t *testing.T) {
	ht := &HonestChallengeTree{
		edges:                         threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
	}
	ht.topLevelAssertionId = protocol.AssertionId(common.BytesToHash([]byte("foo")))
	ctx := context.Background()
	edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 1})

	t.Run("getting top level assertion fails", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr: errors.New("bad request"),
		}
		err := ht.AddEdge(ctx, edge)
		require.ErrorContains(t, err, "could not get top level assertion for edge")
	})
	t.Run("ignores if disagrees with top level assertion id of edge", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr: nil,
			assertionId:  protocol.AssertionId(common.BytesToHash([]byte("bar"))),
		}
		err := ht.AddEdge(ctx, edge)
		require.NoError(t, err)
	})
	t.Run("getting claim heights fails", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:    nil,
			assertionId:     ht.topLevelAssertionId,
			claimHeightsErr: errors.New("bad request"),
		}
		err := ht.AddEdge(ctx, edge)
		require.ErrorContains(t, err, "could not get claim heights for edge")
	})
	t.Run("checking if agrees with commit fails", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr: nil,
			assertionId:  ht.topLevelAssertionId,
		}
		ht.histChecker = &mocks.MockStateManager{
			AgreeErr: true,
		}
		err := ht.AddEdge(ctx, edge)
		require.ErrorContains(t, err, "could not check if agrees with")
	})
	t.Run("fully disagrees with edge", func(t *testing.T) {
		ht.histChecker = &mocks.MockStateManager{
			Agreement: protocol.Agreement{
				IsHonestEdge:          false,
				AgreesWithStartCommit: false,
			},
		}
		badEdge := newEdge(&newCfg{t: t, edgeId: "blk-0.f-16.a", createdAt: 1})
		err := ht.AddEdge(ctx, badEdge)
		require.NoError(t, err)

		// Check the edge is not kept track of anywhere.
		_, ok := ht.edges.TryGet(badEdge.Id())
		require.Equal(t, false, ok)
		_, ok = ht.mutualIds.TryGet(badEdge.MutualId())
		require.Equal(t, false, ok)
	})
	t.Run("agrees with edge but is not a level zero edge", func(t *testing.T) {
		ht.histChecker = &mocks.MockStateManager{
			Agreement: protocol.Agreement{
				IsHonestEdge: true,
			},
		}
		edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 1})
		err := ht.AddEdge(ctx, edge)
		require.NoError(t, err)

		// Exists.
		_, ok := ht.edges.TryGet(edge.Id())
		require.Equal(t, true, ok)
		// Exists in the mutual ids mapping.
		_, ok = ht.mutualIds.TryGet(edge.MutualId())
		require.Equal(t, true, ok)

		// However, we should not have a level zero edge being tracked yet.
		require.Equal(t, true, ht.honestBlockChalLevelZeroEdge.IsNone())
		require.Equal(t, true, ht.honestBigStepLevelZeroEdges.Len() == 0)
		require.Equal(t, true, ht.honestSmallStepLevelZeroEdges.Len() == 0)
	})
	t.Run("agrees with edge and is a level zero edge", func(t *testing.T) {
		edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a", createdAt: 1, claimId: "foo"})
		err := ht.AddEdge(ctx, edge)
		require.NoError(t, err)

		// Exists.
		_, ok := ht.edges.TryGet(edge.Id())
		require.Equal(t, true, ok)
		// Exists in the mutual ids mapping.
		_, ok = ht.mutualIds.TryGet(edge.MutualId())
		require.Equal(t, true, ok)

		// We should have a level zero edge being tracked.
		require.Equal(t, false, ht.honestBlockChalLevelZeroEdge.IsNone())
	})
	t.Run("edge is not honest but we agree with start commit and keep it as a rival", func(t *testing.T) {
		ht.histChecker = &mocks.MockStateManager{
			Agreement: protocol.Agreement{
				IsHonestEdge:          false,
				AgreesWithStartCommit: true,
			},
		}
		edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.b", createdAt: 1, claimId: "bar"})
		err := ht.AddEdge(ctx, edge)
		require.NoError(t, err)

		// Is not being tracked by the honest challenge tree.
		_, ok := ht.edges.TryGet(edge.Id())
		require.Equal(t, false, ok)
		// Exists in the mutual ids mapping.
		mutuals, ok := ht.mutualIds.TryGet(edge.MutualId())
		require.Equal(t, true, ok)
		require.Equal(t, true, mutuals.Has(edge.Id()))
		require.Equal(t, true, mutuals.NumItems() > 0)
	})
}

type mockMetadataReader struct {
	assertionId     protocol.AssertionId
	assertionErr    error
	claimHeights    *protocol.OriginHeights
	claimHeightsErr error
}

func (m *mockMetadataReader) TopLevelAssertion(
	_ context.Context, _ protocol.EdgeId,
) (protocol.AssertionId, error) {
	return m.assertionId, m.assertionErr
}

func (*mockMetadataReader) AssertionUnrivaledTime(
	_ context.Context, _ protocol.AssertionId,
) (uint64, error) {
	return 0, nil
}

func (m *mockMetadataReader) TopLevelClaimHeights(
	_ context.Context, _ protocol.EdgeId,
) (*protocol.OriginHeights, error) {
	return m.claimHeights, m.claimHeightsErr
}

func (m *mockMetadataReader) SpecChallengeManager(_ context.Context) (protocol.SpecChallengeManager, error) {
	return nil, nil
}
func (m *mockMetadataReader) ReadAssertionCreationInfo(
	_ context.Context, _ protocol.AssertionId,
) (*protocol.AssertionCreatedInfo, error) {
	return &protocol.AssertionCreatedInfo{InboxMaxCount: big.NewInt(1)}, nil
}

var _ = protocol.ReadOnlyEdge(&edge{})

type edgeId string
type commit string
type originId string

// Mock edge for challenge tree specific tests, making it easier for test ergonomics.
type edge struct {
	id            edgeId
	edgeType      protocol.EdgeType
	startHeight   uint64
	startCommit   commit
	endHeight     uint64
	endCommit     commit
	originId      originId
	claimId       string
	lowerChildId  edgeId
	upperChildId  edgeId
	creationBlock uint64
}

func (e *edge) Id() protocol.EdgeId {
	return protocol.EdgeId(common.BytesToHash([]byte(e.id)))
}

func (e *edge) GetType() protocol.EdgeType {
	return e.edgeType
}

func (e *edge) StartCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.startHeight), common.BytesToHash([]byte(e.startCommit))
}

func (e *edge) EndCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.endHeight), common.BytesToHash([]byte(e.endCommit))
}

func (e *edge) CreatedAtBlock() (uint64, error) {
	return e.creationBlock, nil
}

func (e *edge) OriginId() protocol.OriginId {
	return protocol.OriginId(common.BytesToHash([]byte(e.originId)))
}

func (e *edge) MutualId() protocol.MutualId {
	return protocol.MutualId(common.BytesToHash([]byte(e.computeMutualId())))
}

func (e *edge) computeMutualId() string {
	return fmt.Sprintf(
		"%d-%s-%d-%s-%d",
		e.edgeType,
		e.originId,
		e.startHeight,
		e.startCommit,
		e.endHeight,
	)
}

// The claim id of the edge, if any
func (e *edge) ClaimId() option.Option[protocol.ClaimId] {
	if e.claimId == "" {
		return option.None[protocol.ClaimId]()
	}
	return option.Some(protocol.ClaimId(common.BytesToHash([]byte(e.claimId))))
}

// The lower child of the edge, if any.
func (e *edge) LowerChild(_ context.Context) (option.Option[protocol.EdgeId], error) {
	if e.lowerChildId == "" {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId(common.BytesToHash([]byte(e.lowerChildId)))), nil
}

// The upper child of the edge, if any.
func (e *edge) UpperChild(_ context.Context) (option.Option[protocol.EdgeId], error) {
	if e.upperChildId == "" {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId(common.BytesToHash([]byte(e.upperChildId)))), nil
}

func (e *edge) HasChildren(ctx context.Context) (bool, error) {
	return e.lowerChildId != "" && e.upperChildId != "", nil
}

// The ministaker of an edge. Only existing for level zero edges.
func (*edge) MiniStaker() option.Option[common.Address] {
	return option.None[common.Address]()
}

// The assertion id of the parent assertion that originated the challenge
// at the top-level.
func (*edge) AssertionId(_ context.Context) (protocol.AssertionId, error) {
	return protocol.AssertionId{}, errors.New("unimplemented")
}

// The time in seconds an edge has been unrivaled.
func (*edge) TimeUnrivaled(_ context.Context) (uint64, error) {
	return 0, errors.New("unimplemented")
}

// The status of an edge.
func (*edge) Status(_ context.Context) (protocol.EdgeStatus, error) {
	return 0, errors.New("unimplemented")
}

// Whether or not an edge has rivals.
func (*edge) HasRival(_ context.Context) (bool, error) {
	return false, errors.New("unimplemented")
}

// Checks if an edge has a length one rival.
func (*edge) HasLengthOneRival(_ context.Context) (bool, error) {
	return false, errors.New("unimplemented")
}

// The history commitment for the top-level edge the current edge's challenge is made upon.
// This is used at subchallenge creation boundaries.
func (*edge) TopLevelClaimHeight(_ context.Context) (*protocol.OriginHeights, error) {
	return nil, errors.New("unimplemented")
}

func (*edge) Bisect(
	_ context.Context,
	_ common.Hash,
	_ []byte,
) (protocol.SpecEdge, protocol.SpecEdge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (*edge) ConfirmByTimer(_ context.Context, _ []protocol.EdgeId) error {
	return errors.New("unimplemented")
}

func (*edge) ConfirmByClaim(_ context.Context, _ protocol.ClaimId) error {
	return errors.New("unimplemented")
}

func (*edge) ConfirmByChildren(_ context.Context) error {
	return errors.New("unimplemented")
}

type newCfg struct {
	t         *testing.T
	originId  originId
	edgeId    edgeId
	claimId   string
	createdAt uint64
}

func newEdge(cfg *newCfg) *edge {
	cfg.t.Helper()
	items := strings.Split(string(cfg.edgeId), "-")
	var typ protocol.EdgeType
	switch items[0] {
	case "blk":
		typ = protocol.BlockChallengeEdge
	case "big":
		typ = protocol.BigStepChallengeEdge
	case "smol":
		typ = protocol.SmallStepChallengeEdge
	}
	startData := strings.Split(items[1], ".")
	startHeight, err := strconv.ParseUint(startData[0], 10, 64)
	require.NoError(cfg.t, err)
	startCommit := startData[1]

	endData := strings.Split(items[2], ".")
	endHeight, err := strconv.ParseUint(endData[0], 10, 64)
	require.NoError(cfg.t, err)
	endCommit := endData[1]

	return &edge{
		edgeType:      typ,
		originId:      cfg.originId,
		id:            cfg.edgeId,
		startHeight:   startHeight,
		claimId:       cfg.claimId,
		startCommit:   commit(startCommit),
		endHeight:     endHeight,
		endCommit:     commit(endCommit),
		lowerChildId:  "",
		upperChildId:  "",
		creationBlock: cfg.createdAt,
	}
}
