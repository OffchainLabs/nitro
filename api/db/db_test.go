package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSqliteDatabase_Updates(t *testing.T) {
	sqlDB, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sqlDB.Close()

	_, err = sqlDB.Exec(schema)
	require.NoError(t, err)

	db := &SqliteDatabase{sqlDB: sqlDB}
	numAssertions := 10
	assertionsToCreate := make([]*api.JsonAssertion, numAssertions)
	for i := 0; i < numAssertions; i++ {
		base := baseAssertion()
		base.Hash = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		base.CreationBlock = uint64(i)
		assertionsToCreate[i] = base
	}
	require.NoError(t, db.InsertAssertions(assertionsToCreate))

	// Get the inserted assertions.
	assertions, err := db.GetAssertions()
	require.NoError(t, err)
	require.Equal(t, numAssertions, len(assertions))

	time.Sleep(time.Second)

	lastAssertion := assertions[len(assertions)-1]
	lastUpdated := lastAssertion.LastUpdatedAt
	lastAssertion.Status = "confirmed"
	require.NoError(t, db.UpdateAssertion(lastAssertion))

	// Check the last updated timestamp gets increased.
	updatedAssertions, err := db.GetAssertions(WithAssertionHash(protocol.AssertionHash{Hash: lastAssertion.Hash}), WithAssertionLimit(1))
	require.NoError(t, err)
	require.Equal(t, 1, len(updatedAssertions))
	require.Equal(t, "confirmed", updatedAssertions[0].Status)
	require.Equal(t, true, lastUpdated.Before(updatedAssertions[0].LastUpdatedAt))

	// Insert an edge, update it, then check the last updated timestamp increased.
	edge := baseEdge()
	edge.AssertionHash = lastAssertion.Hash
	require.NoError(t, db.InsertEdge(edge))

	edges, err := db.GetEdges()
	require.NoError(t, err)
	require.Equal(t, 1, len(edges))
	lastUpdated = edges[0].LastUpdatedAt

	time.Sleep(time.Second)

	edge.Status = "confirmed"
	require.NoError(t, db.UpdateEdge(edge))

	// Check the last updated timestamp gets increased.
	updatedEdges, err := db.GetEdges(WithEdgeAssertionHash(protocol.AssertionHash{Hash: lastAssertion.Hash}), WithLimit(1))
	require.NoError(t, err)
	require.Equal(t, 1, len(updatedEdges))
	require.Equal(t, "confirmed", updatedEdges[0].Status)
	require.Equal(t, true, lastUpdated.Before(updatedEdges[0].LastUpdatedAt))
}

func TestSqliteDatabase_Assertions(t *testing.T) {
	sqlDB, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sqlDB.Close()

	_, err = sqlDB.Exec(schema)
	require.NoError(t, err)

	// Inserting edges that don't have an associated assertion should fail.
	db := &SqliteDatabase{sqlDB: sqlDB}
	err = db.InsertEdges([]*api.JsonEdge{baseEdge()})
	require.ErrorIs(t, err, ErrNoAssertionForEdge)

	numAssertions := 10
	assertionsToCreate := make([]*api.JsonAssertion, numAssertions)
	for i := 0; i < numAssertions; i++ {
		base := baseAssertion()
		base.Hash = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		base.CreationBlock = uint64(i)
		if i == 1 {
			base.ConfirmPeriodBlocks = 20
			base.BeforeStateBlockHash = common.BytesToHash([]byte("block"))
			base.BeforeStateSendRoot = common.BytesToHash([]byte("send"))
			base.BeforeStateBatch = 4
			base.BeforeStatePosInBatch = 0
		}
		if i == 2 || i == 3 {
			base.RequiredStake = "1000"
			base.ChallengeManager = common.BytesToAddress([]byte("foo"))
			b1 := uint64(2)
			b2 := uint64(3)
			base.FirstChildBlock = &b1
			base.SecondChildBlock = &b2
			base.Status = protocol.AssertionConfirmed.String()
		}
		if i == 4 {
			base.ParentAssertionHash = common.BytesToHash([]byte("3"))
			base.InboxMaxCount = "1000"
			base.AfterInboxBatchAcc = common.BytesToHash([]byte("nyan"))
			base.WasmModuleRoot = common.BytesToHash([]byte("nyan"))
			base.TransactionHash = common.BytesToHash([]byte("baz"))
			base.AfterStateBlockHash = common.BytesToHash([]byte("block2"))
			base.AfterStateSendRoot = common.BytesToHash([]byte("send2"))
			base.AfterStateBatch = 6
			base.AfterStatePosInBatch = 2
			base.IsFirstChild = true
		}
		base.CreationBlock = uint64(i)
		assertionsToCreate[i] = base
	}
	require.NoError(t, db.InsertAssertions(assertionsToCreate))

	assertions, err := db.GetAssertions()
	require.NoError(t, err)
	require.Equal(t, numAssertions, len(assertions))

	// There should be no challenged assertions.
	challengedAssertions, err := db.GetChallengedAssertions()
	require.NoError(t, err)
	require.Equal(t, 0, len(challengedAssertions))

	t.Run("query options", func(t *testing.T) {
		assertions, err := db.GetAssertions(WithAssertionHash(protocol.AssertionHash{Hash: common.BytesToHash([]byte("3"))}))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithAssertionHash(protocol.AssertionHash{Hash: common.BytesToHash([]byte("100"))}))
		require.NoError(t, err)
		require.Equal(t, 0, len(assertions))

		assertions, err = db.GetAssertions(WithConfirmPeriodBlocks(20))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithRequiredStake("1000"))
		require.NoError(t, err)
		require.Equal(t, 2, len(assertions))

		assertions, err = db.GetAssertions(WithParentAssertionHash(protocol.AssertionHash{Hash: common.BytesToHash([]byte("3"))}))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithInboxMaxCount("1000"))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithAfterInboxBatchAcc(common.BytesToHash([]byte("nyan"))))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithWasmModuleRoot(common.BytesToHash([]byte("nyan"))))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithChallengeManager(common.BytesToAddress([]byte("foo"))))
		require.NoError(t, err)
		require.Equal(t, 2, len(assertions))

		assertions, err = db.GetAssertions(FromAssertionCreationBlock(5), ToAssertionCreationBlock(6))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(FromAssertionCreationBlock(1), ToAssertionCreationBlock(4))
		require.NoError(t, err)
		require.Equal(t, 3, len(assertions))

		assertions, err = db.GetAssertions(WithTransactionHash(common.BytesToHash([]byte("baz"))))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithBeforeState(&protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  common.BytesToHash([]byte("block")),
				SendRoot:   common.BytesToHash([]byte("send")),
				PosInBatch: 0,
				Batch:      4,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithAfterState(&protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  common.BytesToHash([]byte("block2")),
				SendRoot:   common.BytesToHash([]byte("send2")),
				PosInBatch: 2,
				Batch:      6,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}))
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithFirstChildBlock(2))
		require.NoError(t, err)
		require.Equal(t, 2, len(assertions))

		assertions, err = db.GetAssertions(WithSecondChildBlock(3))
		require.NoError(t, err)
		require.Equal(t, 2, len(assertions))

		assertions, err = db.GetAssertions(WithIsFirstChild())
		require.NoError(t, err)
		require.Equal(t, 1, len(assertions))

		assertions, err = db.GetAssertions(WithAssertionStatus(protocol.AssertionConfirmed))
		require.NoError(t, err)
		require.Equal(t, 2, len(assertions))
	})
	t.Run("orderings limits and offsets", func(t *testing.T) {
		gotIds := make([]protocol.AssertionHash, 0)
		wantIds := make([]protocol.AssertionHash, 0)

		expectedAssertions := assertionsToCreate[2:4]
		for _, a := range expectedAssertions {
			wantIds = append(wantIds, protocol.AssertionHash{Hash: a.Hash})
		}

		assertions, err := db.GetAssertions(WithAssertionLimit(2), WithAssertionOffset(2), WithAssertionOrderBy("CreationBlock ASC"))
		require.NoError(t, err)
		for _, a := range assertions {
			gotIds = append(gotIds, protocol.AssertionHash{Hash: a.Hash})
		}
		require.Equal(t, wantIds, gotIds)
	})
}

func TestSqliteDatabase_Edges(t *testing.T) {
	sqlDB, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sqlDB.Close()

	_, err = sqlDB.Exec(schema)
	require.NoError(t, err)

	// Inserting edges that don't have an associated assertion should fail.
	db := &SqliteDatabase{sqlDB: sqlDB}
	err = db.InsertEdges([]*api.JsonEdge{baseEdge()})
	require.ErrorIs(t, err, ErrNoAssertionForEdge)

	numAssertions := 10
	assertionsToCreate := make([]*api.JsonAssertion, numAssertions)
	for i := 0; i < numAssertions; i++ {
		base := baseAssertion()
		base.Hash = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		base.CreationBlock = uint64(i)
		assertionsToCreate[i] = base
	}
	require.NoError(t, db.InsertAssertions(assertionsToCreate))

	numEdges := 5
	endHeight := uint64(32)
	edgesToCreate := make([]*api.JsonEdge, numEdges)
	for i := 0; i < numEdges; i++ {
		base := baseEdge()
		base.Id = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		base.AssertionHash = common.BytesToHash([]byte("1"))
		base.CreatedAtBlock = uint64(i)
		base.EndHeight = endHeight
		if i == 0 {
			base.OriginId = common.BytesToHash([]byte("foo"))
			base.MutualId = common.BytesToHash([]byte("bar"))
			base.MiniStaker = common.BytesToAddress([]byte("nyan"))
			base.Status = "confirmed"
		}
		if i == 2 || i == 3 {
			base.HasChildren = true
			base.LowerChildId = common.BytesToHash([]byte("0"))
			base.UpperChildId = common.BytesToHash([]byte("1"))
			base.HasRival = true
			base.HasLengthOneRival = true
			base.ClaimId = common.BytesToHash([]byte("1"))
		}
		edgesToCreate[i] = base
		endHeight = endHeight / 2
	}
	require.NoError(t, db.InsertEdges(edgesToCreate))

	// Check the edges retrieved.
	edges, err := db.GetEdges()
	require.NoError(t, err)
	require.Equal(t, numEdges, len(edges))

	// A challenge should have been created for the edges that were inserted
	// for their associated assertion in the database. There should only be one challenged assertion.
	challengedAssertions, err := db.GetChallengedAssertions()
	require.NoError(t, err)
	require.Equal(t, 1, len(challengedAssertions))

	t.Run("query options", func(t *testing.T) {
		edges, err = db.GetEdges(WithId(protocol.EdgeId{Hash: common.BytesToHash([]byte("0"))}))
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(WithChallengeLevel(0))
		require.NoError(t, err)
		require.Equal(t, numEdges, len(edges))

		edges, err = db.GetEdges(WithChallengeLevel(1))
		require.NoError(t, err)
		require.Equal(t, 0, len(edges))

		edges, err = db.GetEdges(WithOriginId(protocol.OriginId(common.BytesToHash([]byte("foo")))))
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(WithStartHistoryCommitment(history.History{
			Height: 0,
			Merkle: common.Hash{},
		}))
		require.NoError(t, err)
		require.Equal(t, 5, len(edges))

		edges, err = db.GetEdges(WithEndHistoryCommitment(history.History{
			Height: 32,
			Merkle: common.Hash{},
		}))
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(
			WithStartHistoryCommitment(history.History{
				Height: 0,
				Merkle: common.Hash{},
			}),
			WithEndHistoryCommitment(history.History{
				Height: 16,
				Merkle: common.Hash{},
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(WithMutualId(protocol.MutualId(common.BytesToHash([]byte("bar")))))
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(HasChildren())
		require.NoError(t, err)
		require.Equal(t, 2, len(edges))

		edges, err = db.GetEdges(WithLowerChildId(protocol.EdgeId{Hash: common.BytesToHash([]byte("0"))}))
		require.NoError(t, err)
		require.Equal(t, 2, len(edges))

		edges, err = db.GetEdges(WithUpperChildId(protocol.EdgeId{Hash: common.BytesToHash([]byte("1"))}))
		require.NoError(t, err)
		require.Equal(t, 2, len(edges))

		edges, err = db.GetEdges(WithMiniStaker(common.BytesToAddress([]byte("nyan"))))
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(WithEdgeAssertionHash(protocol.AssertionHash{Hash: common.BytesToHash([]byte("1"))}))
		require.NoError(t, err)
		require.Equal(t, numEdges, len(edges))

		edges, err = db.GetEdges(WithEdgeAssertionHash(protocol.AssertionHash{Hash: common.BytesToHash([]byte("0"))}))
		require.NoError(t, err)
		require.Equal(t, 0, len(edges))

		edges, err = db.GetEdges(WithRival())
		require.NoError(t, err)
		require.Equal(t, 2, len(edges))

		edges, err = db.GetEdges(WithEdgeStatus(protocol.EdgeConfirmed))
		require.NoError(t, err)
		require.Equal(t, 1, len(edges))

		edges, err = db.GetEdges(WithLengthOneRival())
		require.NoError(t, err)
		require.Equal(t, 2, len(edges))

		edges, err = db.GetEdges(WithClaimId(protocol.ClaimId(common.BytesToHash([]byte("1")))))
		require.NoError(t, err)
		require.Equal(t, 2, len(edges))
	})
	t.Run("orderings limits and offsets", func(t *testing.T) {
		gotIds := make([]protocol.EdgeId, 0)
		wantIds := make([]protocol.EdgeId, 0)

		expectedEdges := edgesToCreate[2:4]
		for _, e := range expectedEdges {
			wantIds = append(wantIds, protocol.EdgeId{Hash: e.Id})
		}

		edges, err = db.GetEdges(WithLimit(2), WithOffset(2), WithOrderBy("CreatedAtBlock ASC"))
		require.NoError(t, err)
		for _, e := range edges {
			gotIds = append(gotIds, protocol.EdgeId{Hash: e.Id})
		}
		require.Equal(t, wantIds, gotIds)
	})
}

func baseAssertion() *api.JsonAssertion {
	return &api.JsonAssertion{
		Hash:                     common.Hash{},
		ConfirmPeriodBlocks:      100,
		RequiredStake:            "1",
		ParentAssertionHash:      common.Hash{},
		InboxMaxCount:            "1",
		AfterInboxBatchAcc:       common.Hash{},
		WasmModuleRoot:           common.Hash{},
		ChallengeManager:         common.Address{},
		CreationBlock:            1,
		TransactionHash:          common.Hash{},
		BeforeStateBlockHash:     common.Hash{},
		BeforeStateSendRoot:      common.Hash{},
		BeforeStateBatch:         0,
		BeforeStatePosInBatch:    0,
		BeforeStateMachineStatus: protocol.MachineStatusFinished,
		AfterStateBlockHash:      common.Hash{},
		AfterStateSendRoot:       common.Hash{},
		AfterStateBatch:          0,
		AfterStatePosInBatch:     0,
		AfterStateMachineStatus:  protocol.MachineStatusFinished,
		FirstChildBlock:          nil,
		SecondChildBlock:         nil,
		IsFirstChild:             false,
		Status:                   protocol.AssertionPending.String(),
	}
}

func baseEdge() *api.JsonEdge {
	return &api.JsonEdge{
		Id:                common.Hash{},
		ChallengeLevel:    0,
		OriginId:          common.Hash{},
		AssertionHash:     common.Hash{},
		StartHistoryRoot:  common.Hash{},
		StartHeight:       0,
		EndHistoryRoot:    common.Hash{},
		EndHeight:         0,
		MutualId:          common.Hash{},
		ClaimId:           common.Hash{},
		HasChildren:       false,
		LowerChildId:      common.Hash{},
		UpperChildId:      common.Hash{},
		MiniStaker:        common.Address{},
		HasRival:          false,
		Status:            "pending",
		HasLengthOneRival: false,
		CreatedAtBlock:    1,
	}
}
