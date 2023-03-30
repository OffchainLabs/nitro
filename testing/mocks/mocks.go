package mocks

import (
	"context"
	"math/big"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

var (
	_ = protocol.SpecChallengeManager(&MockSpecChallengeManager{})
	_ = protocol.SpecEdge(&MockSpecEdge{})
	_ = protocol.AssertionChain(&MockProtocol{})
	_ = protocol.Challenge(&MockChallenge{})
	_ = protocol.ChallengeManager(&MockChallengeManager{})
	_ = protocol.ChallengeVertex(&MockChallengeVertex{})
	_ = statemanager.Manager(&MockStateManager{})
)

type MockChallengeVertex struct {
	mock.Mock
	MockId         [32]byte
	MockSeqNum     protocol.VertexSequenceNumber
	MockStatus     protocol.AssertionState
	MockHistory    util.HistoryCommitment
	MockMiniStaker common.Address
	MockPrev       util.Option[protocol.ChallengeVertex]
	MockSubChal    util.Option[protocol.Challenge]
}

func (m *MockChallengeVertex) Id() [32]byte {
	return m.MockId
}

func (m *MockChallengeVertex) SequenceNum() protocol.VertexSequenceNumber {
	return m.MockSeqNum
}

func (m *MockChallengeVertex) Status(ctx context.Context) (protocol.AssertionState, error) {
	return m.MockStatus, nil
}

func (m *MockChallengeVertex) HistoryCommitment() util.HistoryCommitment {
	return m.MockHistory
}

func (m *MockChallengeVertex) MiniStaker(ctx context.Context) (common.Address, error) {
	return m.MockMiniStaker, nil
}

func (m *MockChallengeVertex) Prev(ctx context.Context) (util.Option[protocol.ChallengeVertex], error) {
	return m.MockPrev, nil
}

func (m *MockChallengeVertex) GetSubChallenge(ctx context.Context) (util.Option[protocol.Challenge], error) {
	return m.MockSubChal, nil
}

func (m *MockChallengeVertex) HasConfirmedSibling(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

// EligibleForNewSuccessor for presumptive status / timer readers.
func (m *MockChallengeVertex) EligibleForNewSuccessor(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallengeVertex) IsPresumptiveSuccessor(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallengeVertex) PresumptiveSuccessor(ctx context.Context) (util.Option[protocol.ChallengeVertex], error) {
	args := m.Called(ctx)
	return args.Get(0).(util.Option[protocol.ChallengeVertex]), args.Error(1)
}

func (m *MockChallengeVertex) PsTimer(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockChallengeVertex) ChessClockExpired(ctx context.Context, challengePeriodSeconds time.Duration) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallengeVertex) ChildrenAreAtOneStepFork(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

// CreateSubChallenge is a mutating calls for challenge moves.
func (m *MockChallengeVertex) CreateSubChallenge(ctx context.Context) (protocol.Challenge, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.Challenge), args.Error(1)
}

func (m *MockChallengeVertex) Bisect(ctx context.Context, history util.HistoryCommitment, proof []byte) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, history, proof)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

func (m *MockChallengeVertex) Merge(ctx context.Context, mergingToHistory util.HistoryCommitment, proof []byte) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, mergingToHistory, proof)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

// ConfirmForPsTimer is a mutating calls for confirmations.
func (m *MockChallengeVertex) ConfirmForPsTimer(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockChallengeVertex) ConfirmForChallengeDeadline(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockChallengeVertex) ConfirmForSubChallengeWin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockAssertion struct {
	Prev           util.Option[*MockAssertion]
	MockHeight     uint64
	MockSeqNum     protocol.AssertionSequenceNumber
	MockPrevSeqNum protocol.AssertionSequenceNumber
	MockStateHash  common.Hash
}

func (m *MockAssertion) Height() (uint64, error) {
	return m.MockHeight, nil

}

func (m *MockAssertion) SeqNum() protocol.AssertionSequenceNumber {
	return m.MockSeqNum
}

func (m *MockAssertion) PrevSeqNum() (protocol.AssertionSequenceNumber, error) {
	return m.MockPrevSeqNum, nil
}

func (m *MockAssertion) StateHash() (common.Hash, error) {
	return m.MockStateHash, nil
}

type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) LatestAssertionCreationData(ctx context.Context, prevHeight uint64) (*statemanager.AssertionToCreate, error) {
	args := m.Called(ctx, prevHeight)
	return args.Get(0).(*statemanager.AssertionToCreate), args.Error(1)
}

func (m *MockStateManager) HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error) {

	args := m.Called(ctx, height)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) PrefixProof(ctx context.Context, from, to uint64) ([]byte, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) BigStepPrefixProof(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	lo,
	hi uint64,
) ([]byte, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight, lo, hi)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) SmallStepPrefixProof(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	lo,
	hi uint64,
) ([]byte, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight, lo, hi)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) HasStateCommitment(ctx context.Context, commit util.StateCommitment) bool {
	args := m.Called(ctx, commit)
	return args.Bool(0)
}

func (m *MockStateManager) BigStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) BigStepCommitmentUpTo(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight, toBigStep)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) SmallStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) SmallStepCommitmentUpTo(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toStep uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight, toStep)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

type MockChallenge struct {
	mock.Mock
	MockID                  protocol.ChallengeHash
	MockType                protocol.ChallengeType
	MockWinningAssertion    util.Option[protocol.AssertionHash]
	MockAssertion           protocol.Assertion
	MockRootVertex          protocol.ChallengeVertex
	MockCreationTime        time.Time
	MockParentStateCommit   util.StateCommitment
	MockWinnerVertex        util.Option[protocol.ChallengeVertex]
	MockChallenger          common.Address
	MockTopLevelClaimVertex protocol.ChallengeVertex
}

// Getters.
func (m *MockChallenge) Id() protocol.ChallengeHash {
	return m.MockID
}

func (m *MockChallenge) GetType() protocol.ChallengeType {
	return m.MockType
}

func (m *MockChallenge) WinningClaim(ctx context.Context) (util.Option[protocol.AssertionHash], error) {
	return m.MockWinningAssertion, nil
}

func (m *MockChallenge) RootAssertion(ctx context.Context) (protocol.Assertion, error) {
	return m.MockAssertion, nil
}

func (m *MockChallenge) TopLevelClaimVertex(ctx context.Context) (protocol.ChallengeVertex, error) {
	return m.MockTopLevelClaimVertex, nil
}

func (m *MockChallenge) RootVertex(ctx context.Context) (protocol.ChallengeVertex, error) {
	return m.MockRootVertex, nil
}

func (m *MockChallenge) GetCreationTime(ctx context.Context) (time.Time, error) {
	return m.MockCreationTime, nil
}

func (m *MockChallenge) ParentStateCommitment(ctx context.Context) (util.StateCommitment, error) {
	return m.MockParentStateCommit, nil
}

func (m *MockChallenge) WinnerVertex(ctx context.Context) (util.Option[protocol.ChallengeVertex], error) {
	return m.MockWinnerVertex, nil
}

func (m *MockChallenge) Completed(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallenge) Challenger() common.Address {
	return m.MockChallenger
}

// Mutating calls.
func (m *MockChallenge) AddBlockChallengeLeaf(ctx context.Context, assertion protocol.Assertion, history util.HistoryCommitment) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, assertion, history)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

func (m *MockChallenge) AddSubChallengeLeaf(ctx context.Context, vertex protocol.ChallengeVertex, history util.HistoryCommitment) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, vertex, history)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

type MockChallengeManager struct {
	mock.Mock
	MockAddr common.Address
}

func (m *MockChallengeManager) ChallengePeriodSeconds(ctx context.Context) (time.Duration, error) {
	args := m.Called(ctx)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockChallengeManager) CalculateChallengeHash(ctx context.Context, itemId common.Hash, challengeType protocol.ChallengeType) (protocol.ChallengeHash, error) {
	args := m.Called(ctx, itemId, challengeType)
	return args.Get(0).(protocol.ChallengeHash), args.Error(1)
}

func (m *MockChallengeManager) CalculateChallengeVertexId(ctx context.Context, challengeId protocol.ChallengeHash, history util.HistoryCommitment) (protocol.VertexHash, error) {
	args := m.Called(ctx, challengeId, history)
	return args.Get(0).(protocol.VertexHash), args.Error(1)
}

func (m *MockChallengeManager) GetVertex(ctx context.Context, vertexId protocol.VertexHash) (util.Option[protocol.ChallengeVertex], error) {
	args := m.Called(ctx, vertexId)
	return args.Get(0).(util.Option[protocol.ChallengeVertex]), args.Error(1)
}

func (m *MockChallengeManager) GetChallenge(ctx context.Context, challengeId protocol.ChallengeHash) (util.Option[protocol.Challenge], error) {
	args := m.Called(ctx, challengeId)
	return args.Get(0).(util.Option[protocol.Challenge]), args.Error(1)
}

func (m *MockChallengeManager) Address() common.Address {
	return m.MockAddr
}

// MockSpecChallengeManager
type MockSpecChallengeManager struct {
	mock.Mock
	MockAddr common.Address
}

func (m *MockSpecChallengeManager) Address() common.Address {
	return m.MockAddr
}

func (m *MockSpecChallengeManager) ChallengePeriodSeconds(ctx context.Context) (time.Duration, error) {
	args := m.Called(ctx)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockSpecChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (util.Option[protocol.SpecEdge], error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(util.Option[protocol.SpecEdge]), args.Error(1)
}

func (m *MockSpecChallengeManager) CalculateMutualId(
	ctx context.Context,
	edgeType protocol.EdgeType,
	originId protocol.OriginId,
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
) (protocol.MutualId, error) {
	args := m.Called(ctx, edgeType, originId, startHeight, startHistoryRoot, endHeight)
	return args.Get(0).(protocol.MutualId), args.Error(1)
}

func (m *MockSpecChallengeManager) CalculateEdgeId(
	ctx context.Context,
	edgeType protocol.EdgeType,
	originId protocol.OriginId,
	startHeight protocol.Height,
	startHistoryRoot common.Hash,
	endHeight protocol.Height,
	endHistoryRoot common.Hash,
) (protocol.EdgeId, error) {
	args := m.Called(ctx, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
	return args.Get(0).(protocol.EdgeId), args.Error(1)
}

func (m *MockSpecChallengeManager) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
	startCommit util.HistoryCommitment,
	endCommit util.HistoryCommitment,
) (protocol.SpecEdge, error) {
	args := m.Called(ctx, assertion, startCommit, endCommit)
	return args.Get(0).(protocol.SpecEdge), args.Error(1)
}

func (m *MockSpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startCommit util.HistoryCommitment,
	endCommit util.HistoryCommitment,
) (protocol.SpecEdge, error) {
	args := m.Called(ctx, challengedEdge, startCommit, endCommit)
	return args.Get(0).(protocol.SpecEdge), args.Error(1)
}

// MockSpecEdge
type MockSpecEdge struct {
	mock.Mock
}

func (m *MockSpecEdge) Id() protocol.EdgeId {
	args := m.Called()
	return args.Get(0).(protocol.EdgeId)
}
func (m *MockSpecEdge) GetType() protocol.EdgeType {
	args := m.Called()
	return args.Get(0).(protocol.EdgeType)
}
func (m *MockSpecEdge) MiniStaker() util.Option[common.Address] {
	args := m.Called()
	return args.Get(0).(util.Option[common.Address])
}
func (m *MockSpecEdge) StartCommitment() (protocol.Height, common.Hash) {
	args := m.Called()
	return args.Get(0).(protocol.Height), args.Get(1).(common.Hash)
}
func (m *MockSpecEdge) EndCommitment() (protocol.Height, common.Hash) {
	args := m.Called()
	return args.Get(0).(protocol.Height), args.Get(1).(common.Hash)
}
func (m *MockSpecEdge) PresumptiveTimer(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}
func (m *MockSpecEdge) IsPresumptive(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}
func (m *MockSpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.EdgeStatus), args.Error(1)
}
func (m *MockSpecEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (protocol.SpecEdge, protocol.SpecEdge, error) {
	args := m.Called(ctx, prefixHistoryRoot, prefixProof)
	return args.Get(0).(protocol.SpecEdge), args.Get(1).(protocol.SpecEdge), args.Error(2)
}
func (m *MockSpecEdge) ConfirmByTimer(ctx context.Context, ancestorIds []protocol.EdgeId) error {
	args := m.Called(ctx, ancestorIds)
	return args.Error(0)
}
func (m *MockSpecEdge) ConfirmByClaim(ctx context.Context, claimId protocol.ClaimId) error {
	args := m.Called(ctx, claimId)
	return args.Error(0)
}
func (m *MockSpecEdge) ConfirmByOneStepProof(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockSpecEdge) OriginCommitment(ctx context.Context) (protocol.Height, common.Hash, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.Height), args.Get(1).(common.Hash), args.Error(2)
}
func (m *MockSpecEdge) IsOneStepForkSource(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

type MockProtocol struct {
	mock.Mock
}

// Read-only methods.
func (m *MockProtocol) NumAssertions(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) AssertionBySequenceNum(ctx context.Context, seqNum protocol.AssertionSequenceNumber) (protocol.Assertion, error) {
	args := m.Called(ctx, seqNum)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) GetAssertionId(ctx context.Context, seqNum protocol.AssertionSequenceNumber) (protocol.AssertionHash, error) {
	args := m.Called(ctx, seqNum)
	return args.Get(0).(protocol.AssertionHash), args.Error(1)
}

func (m *MockProtocol) GetAssertionNum(ctx context.Context, assertionHash protocol.AssertionHash) (protocol.AssertionSequenceNumber, error) {
	args := m.Called(ctx, assertionHash)
	return args.Get(0).(protocol.AssertionSequenceNumber), args.Error(1)
}

func (m *MockProtocol) BlockChallenge(ctx context.Context, assertionSeqNum protocol.AssertionSequenceNumber) (protocol.Challenge, error) {
	args := m.Called(ctx, assertionSeqNum)
	return args.Get(0).(protocol.Challenge), args.Error(1)
}

func (m *MockProtocol) LatestConfirmed(ctx context.Context) (protocol.Assertion, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) CurrentChallengeManager(ctx context.Context) (protocol.ChallengeManager, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.ChallengeManager), args.Error(1)
}

// Mutating methods.
func (m *MockProtocol) CreateAssertion(ctx context.Context, height uint64, prevSeqNum protocol.AssertionSequenceNumber, prevAssertionState *protocol.ExecutionState, postState *protocol.ExecutionState, prevInboxMaxCount *big.Int) (protocol.Assertion, error) {
	args := m.Called(ctx, height, prevSeqNum, prevAssertionState, postState, prevInboxMaxCount)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) CreateSuccessionChallenge(ctx context.Context, seqNum protocol.AssertionSequenceNumber) (protocol.Challenge, error) {
	args := m.Called(ctx, seqNum)
	return args.Get(0).(protocol.Challenge), args.Error(1)
}

func (m *MockProtocol) SpecChallengeManager(ctx context.Context) (protocol.SpecChallengeManager, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.SpecChallengeManager), args.Error(1)
}

func (m *MockProtocol) CreateSpecChallenge(ctx context.Context, seqNum protocol.AssertionSequenceNumber) error {
	args := m.Called(ctx, seqNum)
	return args.Error(0)
}

func (m *MockProtocol) Confirm(ctx context.Context, blockHash, sendRoot common.Hash) error {
	args := m.Called(ctx, blockHash, sendRoot)
	return args.Error(0)
}
