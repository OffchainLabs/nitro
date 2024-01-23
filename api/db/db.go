// Package db handles the interface to an underlying database of BOLD data
// for easy querying of information used by the BOLD API.
package db

import (
	"os"
	"strings"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var (
	ErrNoAssertionForEdge = errors.New("no matching assertion found for edge")
)

type Database interface {
	ReadOnlyDatabase
	InsertEdges(edges []*api.JsonEdge) error
	InsertEdge(edge *api.JsonEdge) error
	InsertAssertions(assertions []*api.JsonAssertion) error
	InsertAssertion(assertion *api.JsonAssertion) error
}

type ReadOnlyDatabase interface {
	GetAssertions(opts ...AssertionOption) ([]*api.JsonAssertion, error)
	GetChallengedAssertions(opts ...AssertionOption) ([]*api.JsonAssertion, error)
	GetEdges(opts ...EdgeOption) ([]*api.JsonEdge, error)
}

type SqliteDatabase struct {
	sqlDB               *sqlx.DB
	currentTableVersion int
}

func NewDatabase(path string) (*SqliteDatabase, error) {
	//#nosec G304
	if _, err := os.Stat(path); err != nil {
		_, err = os.Create(path)
		if err != nil {
			return nil, err
		}
	}
	db, err := sqlx.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	//#nosec G104
	db.Exec(schema)
	return &SqliteDatabase{
		sqlDB:               db,
		currentTableVersion: -1,
	}, nil
}

type AssertionQuery struct {
	filters           []string
	args              []interface{}
	limit             int
	offset            int
	orderBy           string
	withChallenge     bool
	fromCreationBlock option.Option[uint64]
	toCreationBlock   option.Option[uint64]
	forceUpdate       bool
}

func NewAssertionQuery(opts ...AssertionOption) *AssertionQuery {
	query := &AssertionQuery{
		fromCreationBlock: option.None[uint64](),
		toCreationBlock:   option.None[uint64](),
	}
	for _, opt := range opts {
		opt(query)
	}
	return query
}

type AssertionOption func(*AssertionQuery)

func WithAssertionForceUpdate() AssertionOption {
	return func(q *AssertionQuery) {
		q.forceUpdate = true
	}
}
func WithChallenge() AssertionOption {
	return func(q *AssertionQuery) {
		q.withChallenge = true
	}
}
func WithAssertionHash(hash protocol.AssertionHash) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "Hash = ?")
		q.args = append(q.args, hash.Hash)
	}
}
func WithConfirmPeriodBlocks(confirmPeriodBlocks uint64) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "ConfirmPeriodBlocks = ?")
		q.args = append(q.args, confirmPeriodBlocks)
	}
}
func WithRequiredStake(requiredStake string) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "RequiredStake = ?")
		q.args = append(q.args, requiredStake)
	}
}
func WithParentAssertionHash(hash protocol.AssertionHash) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "ParentAssertionHash = ?")
		q.args = append(q.args, hash.Hash)
	}
}
func WithInboxMaxCount(inboxMaxCount string) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "InboxMaxCount = ?")
		q.args = append(q.args, inboxMaxCount)
	}
}
func WithAfterInboxBatchAcc(afterInboxBatchAcc common.Hash) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "AfterInboxBatchAcc = ?")
		q.args = append(q.args, afterInboxBatchAcc)
	}
}
func WithWasmModuleRoot(wasmModuleRoot common.Hash) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "WasmModuleRoot = ?")
		q.args = append(q.args, wasmModuleRoot)
	}
}
func WithChallengeManager(challengeManager common.Address) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "ChallengeManager = ?")
		q.args = append(q.args, challengeManager)
	}
}
func WithTransactionHash(hash common.Hash) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "TransactionHash = ?")
		q.args = append(q.args, hash)
	}
}
func WithBeforeState(state *protocol.ExecutionState) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "BeforeStateBlockHash = ?")
		q.args = append(q.args, state.GlobalState.BlockHash)
		q.filters = append(q.filters, "BeforeStateSendRoot = ?")
		q.args = append(q.args, state.GlobalState.SendRoot)
		q.filters = append(q.filters, "BeforeStateBatch = ?")
		q.args = append(q.args, state.GlobalState.Batch)
		q.filters = append(q.filters, "BeforeStatePosInBatch = ?")
		q.args = append(q.args, state.GlobalState.PosInBatch)
		q.filters = append(q.filters, "BeforeStateMachineStatus = ?")
		q.args = append(q.args, state.MachineStatus)
	}
}
func WithAfterState(state *protocol.ExecutionState) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "AfterStateBlockHash = ?")
		q.args = append(q.args, state.GlobalState.BlockHash)
		q.filters = append(q.filters, "AfterStateSendRoot = ?")
		q.args = append(q.args, state.GlobalState.SendRoot)
		q.filters = append(q.filters, "AfterStateBatch = ?")
		q.args = append(q.args, state.GlobalState.Batch)
		q.filters = append(q.filters, "AfterStatePosInBatch = ?")
		q.args = append(q.args, state.GlobalState.PosInBatch)
		q.filters = append(q.filters, "AfterStateMachineStatus = ?")
		q.args = append(q.args, state.MachineStatus)
	}
}
func WithFirstChildBlock(n uint64) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "FirstChildBlock = ?")
		q.args = append(q.args, n)
	}
}
func WithSecondChildBlock(n uint64) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "SecondChildBlock = ?")
		q.args = append(q.args, n)
	}
}
func WithIsFirstChild() AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "IsFirstChild = true")
	}
}
func WithAssertionStatus(status protocol.AssertionStatus) AssertionOption {
	return func(q *AssertionQuery) {
		q.filters = append(q.filters, "Status = ?")
		q.args = append(q.args, status.String())
	}
}
func FromAssertionCreationBlock(n uint64) AssertionOption {
	return func(q *AssertionQuery) {
		q.fromCreationBlock = option.Some(n)
	}
}
func ToAssertionCreationBlock(n uint64) AssertionOption {
	return func(q *AssertionQuery) {
		q.toCreationBlock = option.Some(n)
	}
}
func WithAssertionLimit(limit int) AssertionOption {
	return func(q *AssertionQuery) {
		q.limit = limit
	}
}
func WithAssertionOffset(offset int) AssertionOption {
	return func(q *AssertionQuery) {
		q.offset = offset
	}
}
func WithAssertionOrderBy(orderBy string) AssertionOption {
	return func(q *AssertionQuery) {
		q.orderBy = orderBy
	}
}

func (q *AssertionQuery) ToSQL() (string, []interface{}) {
	baseQuery := "SELECT * FROM Assertions a"
	if q.withChallenge {
		baseQuery += " INNER JOIN Challenges c ON a.Hash = c.Hash"
	}
	if q.fromCreationBlock.IsSome() {
		q.filters = append(q.filters, "a.CreationBlock >= ?")
		q.args = append(q.args, q.fromCreationBlock.Unwrap())
	}
	if q.toCreationBlock.IsSome() {
		q.filters = append(q.filters, "a.CreationBlock < ?")
		q.args = append(q.args, q.toCreationBlock.Unwrap())
	}
	if len(q.filters) > 0 {
		baseQuery += " WHERE " + strings.Join(q.filters, " AND ")
	}

	if q.orderBy != "" {
		baseQuery += " ORDER BY " + q.orderBy
	}
	if q.limit > 0 {
		baseQuery += " LIMIT ?"
		q.args = append(q.args, q.limit)
	}
	if q.offset > 0 {
		baseQuery += " OFFSET ?"
		q.args = append(q.args, q.offset)
	}
	return baseQuery, q.args
}

func (d *SqliteDatabase) GetAssertions(opts ...AssertionOption) ([]*api.JsonAssertion, error) {
	query := NewAssertionQuery(opts...)
	sql, args := query.ToSQL()
	assertions := make([]*api.JsonAssertion, 0)
	err := d.sqlDB.Select(&assertions, sql, args...)
	if err != nil {
		return nil, err
	}
	return assertions, nil
}

func (d *SqliteDatabase) GetChallengedAssertions(opts ...AssertionOption) ([]*api.JsonAssertion, error) {
	newOpts := []AssertionOption{
		WithChallenge(),
	}
	newOpts = append(newOpts, opts...)
	return d.GetAssertions(newOpts...)
}

type EdgeQuery struct {
	filters []string
	args    []interface{}
	limit   int
	offset  int
	orderBy string
}

func NewEdgeQuery(opts ...EdgeOption) *EdgeQuery {
	query := &EdgeQuery{}
	for _, opt := range opts {
		opt(query)
	}
	return query
}

type EdgeOption func(e *EdgeQuery)

func WithId(id protocol.EdgeId) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "Id = ?")
		q.args = append(q.args, id)
	}
}
func WithChallengeLevel(level uint8) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "ChallengeLevel = ?")
		q.args = append(q.args, level)
	}
}
func WithOriginId(originId protocol.OriginId) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "OriginId = ?")
		q.args = append(q.args, common.Hash(originId))
	}
}
func WithStartHistoryCommitment(startHistory history.History) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "StartHistoryRoot = ?")
		q.args = append(q.args, startHistory.Merkle)
		q.filters = append(q.filters, "StartHeight = ?")
		q.args = append(q.args, startHistory.Height)
	}
}
func WithEndHistoryCommitment(endHistory history.History) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "EndHistoryRoot = ?")
		q.args = append(q.args, endHistory.Merkle)
		q.filters = append(q.filters, "EndHeight = ?")
		q.args = append(q.args, endHistory.Height)
	}
}
func WithMutualId(mutualId protocol.MutualId) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "MutualId = ?")
		q.args = append(q.args, common.Hash(mutualId))
	}
}
func WithClaimId(claimId protocol.ClaimId) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "ClaimId = ?")
		q.args = append(q.args, common.Hash(claimId))
	}
}
func HasChildren() EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "HasChildren = true")
	}
}
func WithLowerChildId(id protocol.EdgeId) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "LowerChildId = ?")
		q.args = append(q.args, id.Hash)
	}
}
func WithUpperChildId(id protocol.EdgeId) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "UpperChildId = ?")
		q.args = append(q.args, id.Hash)
	}
}
func WithMiniStaker(staker common.Address) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "MiniStaker = ?")
		q.args = append(q.args, staker)
	}
}
func WithEdgeAssertionHash(hash protocol.AssertionHash) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "AssertionHash = ?")
		q.args = append(q.args, hash.Hash)
	}
}
func WithRival() EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "HasRival = true")
	}
}
func WithSubchallenge() EdgeOption {
	return func(q *EdgeQuery) {
	}
}
func WithEdgeStatus(st protocol.EdgeStatus) EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "Status = ?")
		q.args = append(q.args, st.String())
	}
}
func WithHonestEdges() EdgeOption {
	return func(q *EdgeQuery) {
	}
}
func WithEdgeForceUpdate() EdgeOption {
	return func(q *EdgeQuery) {
	}
}
func WithRootEdges() EdgeOption {
	return func(q *EdgeQuery) {
	}
}
func FromEdgeCreationBlock(n uint64) EdgeOption {
	return func(q *EdgeQuery) {
	}
}
func ToEdgeCreationBlock(n uint64) EdgeOption {
	return func(e *EdgeQuery) {
	}
}
func WithLengthOneRival() EdgeOption {
	return func(q *EdgeQuery) {
		q.filters = append(q.filters, "HasLengthOneRival = true")
	}
}
func WithLimit(limit int) EdgeOption {
	return func(q *EdgeQuery) {
		q.limit = limit
	}
}
func WithOffset(offset int) EdgeOption {
	return func(q *EdgeQuery) {
		q.offset = offset
	}
}
func WithOrderBy(orderBy string) EdgeOption {
	return func(q *EdgeQuery) {
		q.orderBy = orderBy
	}
}

func (q *EdgeQuery) ToSQL() (string, []interface{}) {
	baseQuery := "SELECT * FROM Edges"
	if len(q.filters) > 0 {
		baseQuery += " WHERE " + strings.Join(q.filters, " AND ")
	}
	if q.orderBy != "" {
		baseQuery += " ORDER BY " + q.orderBy
	}
	if q.limit > 0 {
		baseQuery += " LIMIT ?"
		q.args = append(q.args, q.limit)
	}
	if q.offset > 0 {
		baseQuery += " OFFSET ?"
		q.args = append(q.args, q.offset)
	}
	return baseQuery, q.args
}

func (d *SqliteDatabase) GetEdges(opts ...EdgeOption) ([]*api.JsonEdge, error) {
	query := NewEdgeQuery(opts...)
	sql, args := query.ToSQL()
	edges := make([]*api.JsonEdge, 0)
	err := d.sqlDB.Select(&edges, sql, args...)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (d *SqliteDatabase) InsertAssertions(assertions []*api.JsonAssertion) error {
	for _, a := range assertions {
		if err := d.InsertAssertion(a); err != nil {
			return err
		}
	}
	return nil
}

func (d *SqliteDatabase) InsertAssertion(a *api.JsonAssertion) error {
	query := `INSERT INTO Assertions (
        Hash, ConfirmPeriodBlocks, RequiredStake, ParentAssertionHash, InboxMaxCount,
        AfterInboxBatchAcc, WasmModuleRoot, ChallengeManager, CreationBlock, TransactionHash,
        BeforeStateBlockHash, BeforeStateSendRoot, BeforeStateBatch, BeforeStatePosInBatch, BeforeStateMachineStatus, AfterStateBlockHash,
        AfterStateSendRoot, AfterStateBatch, AfterStatePosInBatch, AfterStateMachineStatus, FirstChildBlock, SecondChildBlock,
        IsFirstChild, Status
    ) VALUES (
        :Hash, :ConfirmPeriodBlocks, :RequiredStake, :ParentAssertionHash, :InboxMaxCount,
        :AfterInboxBatchAcc, :WasmModuleRoot, :ChallengeManager, :CreationBlock, :TransactionHash,
        :BeforeStateBlockHash, :BeforeStateSendRoot, :BeforeStateBatch, :BeforeStatePosInBatch, :BeforeStateMachineStatus, :AfterStateBlockHash,
        :AfterStateSendRoot,:AfterStateBatch,:AfterStatePosInBatch, :AfterStateMachineStatus, :FirstChildBlock, :SecondChildBlock,
        :IsFirstChild, :Status
    )`
	_, err := d.sqlDB.NamedExec(query, a)
	if err != nil {
		return err
	}
	return nil
}

func (d *SqliteDatabase) InsertEdges(edges []*api.JsonEdge) error {
	for _, e := range edges {
		if err := d.InsertEdge(e); err != nil {
			return err
		}
	}
	return nil
}

func (d *SqliteDatabase) InsertEdge(edge *api.JsonEdge) error {
	tx, err := d.sqlDB.Beginx()
	if err != nil {
		return err
	}
	// Check if the assertion exists
	var assertionExists int
	err = tx.Get(&assertionExists, "SELECT COUNT(*) FROM Assertions WHERE Hash = ?", edge.AssertionHash)
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			return err2
		}
		return err
	}
	if assertionExists == 0 {
		if err2 := tx.Rollback(); err2 != nil {
			return err2
		}
		return errors.Wrapf(ErrNoAssertionForEdge, "edge_id=%#x, assertion_hash=%#x", edge.Id, edge.AssertionHash)
	}
	// Check if a challenge exists for the assertion
	var challengeExists int
	err = tx.Get(&challengeExists, "SELECT COUNT(*) FROM Challenges WHERE Hash = ?", edge.AssertionHash)
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			return err2
		}
		return err
	}
	// If the assertion exists but not the challenge, create the challenge
	if challengeExists == 0 {
		insertChallengeQuery := `INSERT INTO Challenges (Hash) VALUES (?)`
		_, err = tx.Exec(insertChallengeQuery, edge.AssertionHash)
		if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				return err2
			}
			return err
		}
	}
	insertEdgeQuery := `INSERT INTO Edges (
	   Id, ChallengeLevel, OriginId, StartHistoryRoot, StartHeight,
	   EndHistoryRoot, EndHeight, CreatedAtBlock, MutualId, ClaimId,
	   HasChildren, LowerChildId, UpperChildId, MiniStaker, AssertionHash,
	   HasRival, Status, HasLengthOneRival, IsHonest, IsRelevant, CumulativePathTimer
   ) VALUES (
	   :Id, :ChallengeLevel, :OriginId, :StartHistoryRoot, :StartHeight,
	   :EndHistoryRoot, :EndHeight, :CreatedAtBlock, :MutualId, :ClaimId,
	   :HasChildren, :LowerChildId, :UpperChildId, :MiniStaker, :AssertionHash,
	   :HasRival, :Status, :HasLengthOneRival, :IsHonest, :IsRelevant, :CumulativePathTimer
   )`

	if _, err = tx.NamedExec(insertEdgeQuery, edge); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			return err2
		}
		return err
	}
	return tx.Commit()
}

func (d *SqliteDatabase) UpdateEdge(edge *api.JsonEdge) error {
	query := `UPDATE Edges SET 
	 ChallengeLevel = :ChallengeLevel,
	 OriginId = :OriginId,
	 StartHistoryRoot = :StartHistoryRoot,
	 StartHeight = :StartHeight,
	 EndHistoryRoot = :EndHistoryRoot,
	 EndHeight = :EndHeight,
	 CreatedAtBlock = :CreatedAtBlock,
	 MutualId = :MutualId,
	 ClaimId = :ClaimId,
	 MiniStaker = :MiniStaker,
	 AssertionHash = :AssertionHash,
	 HasChildren = :HasChildren,
	 LowerChildId = :LowerChildId,
	 UpperChildId = :UpperChildId,
	 HasRival = :HasRival,
	 Status = :Status,
	 HasLengthOneRival = :HasLengthOneRival,
	 IsHonest = :IsHonest,
	 IsRelevant = :IsRelevant,
	 CumulativePathTimer = :CumulativePathTimer
	 WHERE Id = :Id`
	_, err := d.sqlDB.NamedExec(query, edge)
	if err != nil {
		return err
	}
	return nil
}

func (d *SqliteDatabase) UpdateAssertion(assertion *api.JsonAssertion) error {
	// Construct the query
	query := `UPDATE Assertions SET 
   ConfirmPeriodBlocks = :ConfirmPeriodBlocks,
   RequiredStake = :RequiredStake,
   ParentAssertionHash = :ParentAssertionHash,
   InboxMaxCount = :InboxMaxCount,
   AfterInboxBatchAcc = :AfterInboxBatchAcc,
   WasmModuleRoot = :WasmModuleRoot,
   ChallengeManager = :ChallengeManager,
   CreationBlock = :CreationBlock,
   TransactionHash = :TransactionHash,
   BeforeStateBlockHash = :BeforeStateBlockHash,
   BeforeStateSendRoot = :BeforeStateSendRoot,
   BeforeStateBatch = :BeforeStateBatch,
   BeforeStatePosInBatch = :BeforeStatePosInBatch,
   BeforeStateMachineStatus = :BeforeStateMachineStatus,
   AfterStateBlockHash = :AfterStateBlockHash,
   AfterStateSendRoot = :AfterStateSendRoot,
   AfterStateBatch = :AfterStateBatch,
   AfterStatePosInBatch = :AfterStatePosInBatch,
   AfterStateMachineStatus = :AfterStateMachineStatus,
   FirstChildBlock = :FirstChildBlock,
   SecondChildBlock = :SecondChildBlock,
   IsFirstChild = :IsFirstChild,
   Status = :Status
   WHERE Hash = :Hash`

	// Execute the query with the assertion data
	_, err := d.sqlDB.NamedExec(query, assertion)
	if err != nil {
		return err
	}

	return nil
}
