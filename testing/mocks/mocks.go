// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package mocks

import (
	"context"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

var (
	_ = protocol.SpecChallengeManager(&MockSpecChallengeManager{})
	_ = protocol.SpecEdge(&MockSpecEdge{})
	_ = protocol.AssertionChain(&MockProtocol{})
	_ = l2stateprovider.Provider(&MockStateManager{})
)

type MockAssertion struct {
	MockId                protocol.AssertionHash
	MockPrevId            protocol.AssertionHash
	Prev                  option.Option[*MockAssertion]
	MockHeight            uint64
	MockStateHash         common.Hash
	MockInboxMsgCountSeen uint64
	MockCreatedAtBlock    uint64
	MockHasSecondChild    bool
	CreatedAt             uint64
}

func (m *MockAssertion) Id() protocol.AssertionHash {
	return m.MockId
}

func (m *MockAssertion) PrevId(ctx context.Context) (protocol.AssertionHash, error) {
	return m.MockPrevId, nil
}

func (m *MockAssertion) StateHash() (common.Hash, error) {
	return m.MockStateHash, nil
}

func (m *MockAssertion) HasSecondChild() (bool, error) {
	return m.MockHasSecondChild, nil
}

func (m *MockAssertion) InboxMsgCountSeen() (uint64, error) {
	return m.MockInboxMsgCountSeen, nil
}
func (m *MockAssertion) CreatedAtBlock() (uint64, error) {
	return m.CreatedAt, nil
}

type MockStateManager struct {
	mock.Mock
	Agrees   bool
	AgreeErr bool
}

func (m *MockStateManager) ExecutionStateAtMessageNumber(ctx context.Context, messageNumber uint64) (*protocol.ExecutionState, error) {
	args := m.Called(ctx, messageNumber)
	return args.Get(0).(*protocol.ExecutionState), args.Error(1)
}

func (m *MockStateManager) HistoryCommitmentUpTo(ctx context.Context, height uint64) (commitments.History, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(commitments.History), args.Error(1)
}

func (m *MockStateManager) AgreesWithHistoryCommitment(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	prevInboxMaxCount uint64,
	edgeType protocol.EdgeType,
	originHeights protocol.OriginHeights,
	history l2stateprovider.History,
) (bool, error) {
	args := m.Called(ctx, wasmModuleRoot, prevInboxMaxCount, edgeType, originHeights, history)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockStateManager) HistoryCommitmentUpToBatch(ctx context.Context, startBlock, endBlock, batchCount uint64) (commitments.History, error) {
	args := m.Called(ctx, startBlock, endBlock, batchCount)
	return args.Get(0).(commitments.History), args.Error(1)
}

func (m *MockStateManager) PrefixProofUpToBatch(ctx context.Context, start, from, to, batchCount uint64) ([]byte, error) {
	args := m.Called(ctx, start, from, to, batchCount)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) BigStepPrefixProof(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	blockHeight,
	fromBigStep,
	toBigStep uint64,
) ([]byte, error) {
	args := m.Called(ctx, wasmModuleRoot, blockHeight, fromBigStep, toBigStep)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) SmallStepPrefixProof(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	blockHeight,
	bigStep,
	fromSmallStep,
	toSmallStep uint64,
) ([]byte, error) {
	args := m.Called(ctx, wasmModuleRoot, blockHeight, bigStep, fromSmallStep, toSmallStep)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) ExecutionStateMsgCount(ctx context.Context, state *protocol.ExecutionState) (uint64, error) {
	args := m.Called(ctx, state)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockStateManager) BigStepLeafCommitment(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	blockHeight uint64,
) (commitments.History, error) {
	args := m.Called(ctx, wasmModuleRoot, blockHeight)
	return args.Get(0).(commitments.History), args.Error(1)
}

func (m *MockStateManager) BigStepCommitmentUpTo(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	blockHeight,
	toBigStep uint64,
) (commitments.History, error) {
	args := m.Called(ctx, wasmModuleRoot, blockHeight, toBigStep)
	return args.Get(0).(commitments.History), args.Error(1)
}

func (m *MockStateManager) SmallStepLeafCommitment(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	blockHeight,
	bigStep uint64,
) (commitments.History, error) {
	args := m.Called(ctx, wasmModuleRoot, blockHeight, bigStep)
	return args.Get(0).(commitments.History), args.Error(1)
}

func (m *MockStateManager) SmallStepCommitmentUpTo(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	blockHeight,
	bigStep,
	toSmallStep uint64,
) (commitments.History, error) {
	args := m.Called(ctx, wasmModuleRoot, blockHeight, bigStep, toSmallStep)
	return args.Get(0).(commitments.History), args.Error(1)
}

func (m *MockStateManager) OneStepProofData(
	ctx context.Context,
	cfgSnapshot *l2stateprovider.ConfigSnapshot,
	postState rollupgen.ExecutionState,
	blockHeight,
	bigStep,
	smallStep uint64,
) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error) {
	args := m.Called(ctx, cfgSnapshot, postState, blockHeight, bigStep, smallStep)
	return args.Get(0).(*protocol.OneStepData), args.Get(1).([]common.Hash), args.Get(2).([]common.Hash), args.Error(3)
}

type MockChallengeManager struct {
	mock.Mock
	MockAddr common.Address
}

func (m *MockChallengeManager) ChallengePeriodBlocks(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockChallengeManager) Address() common.Address {
	return m.MockAddr
}

func (m *MockChallengeManager) LevelZeroBlockEdgeHeight(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

// MockSpecChallengeManager is a mock implementation of the SpecChallengeManager interface.
type MockSpecChallengeManager struct {
	mock.Mock
	MockAddr common.Address
}

func (m *MockSpecChallengeManager) Address() common.Address {
	return m.MockAddr
}

func (m *MockSpecChallengeManager) LevelZeroBlockEdgeHeight(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSpecChallengeManager) ChallengePeriodBlocks(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSpecChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (option.Option[protocol.SpecEdge], error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(option.Option[protocol.SpecEdge]), args.Error(1)
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
	startCommit,
	endCommit commitments.History,
	startEndPrefixProof []byte,
) (protocol.SpecEdge, error) {
	args := m.Called(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
	return args.Get(0).(protocol.SpecEdge), args.Error(1)
}

func (m *MockSpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startCommit,
	endCommit commitments.History,
	startParentInclusionProof []common.Hash,
	endParentInclusionProof []common.Hash,
	startEndPrefixProof []byte,
) (protocol.SpecEdge, error) {
	args := m.Called(ctx, challengedEdge, startCommit, endCommit, startParentInclusionProof, endParentInclusionProof, startEndPrefixProof)
	return args.Get(0).(protocol.SpecEdge), args.Error(1)
}
func (m *MockSpecChallengeManager) ConfirmEdgeByOneStepProof(
	ctx context.Context,
	tentativeWinnerId protocol.EdgeId,
	oneStepData *protocol.OneStepData,
	preHistoryInclusionProof []common.Hash,
	postHistoryInclusionProof []common.Hash,
) error {
	args := m.Called(ctx, tentativeWinnerId, oneStepData, preHistoryInclusionProof, postHistoryInclusionProof)
	return args.Error(0)
}

// MockSpecEdge is a mock implementation of the SpecEdge interface.
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
func (m *MockSpecEdge) MiniStaker() option.Option[common.Address] {
	args := m.Called()
	return args.Get(0).(option.Option[common.Address])
}
func (m *MockSpecEdge) StartCommitment() (protocol.Height, common.Hash) {
	args := m.Called()
	return args.Get(0).(protocol.Height), args.Get(1).(common.Hash)
}
func (m *MockSpecEdge) EndCommitment() (protocol.Height, common.Hash) {
	args := m.Called()
	return args.Get(0).(protocol.Height), args.Get(1).(common.Hash)
}
func (m *MockSpecEdge) TopLevelClaimHeight(ctx context.Context) (protocol.OriginHeights, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.OriginHeights), args.Error(1)
}
func (m *MockSpecEdge) AssertionHash(ctx context.Context) (protocol.AssertionHash, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.AssertionHash), args.Error(1)
}
func (m *MockSpecEdge) TimeUnrivaled(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}
func (m *MockSpecEdge) HasRival(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}
func (m *MockSpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.EdgeStatus), args.Error(1)
}
func (m *MockSpecEdge) CreatedAtBlock() (uint64, error) {
	args := m.Called()
	return args.Get(0).(uint64), args.Error(1)
}
func (m *MockSpecEdge) MutualId() protocol.MutualId {
	args := m.Called()
	return args.Get(0).(protocol.MutualId)
}
func (m *MockSpecEdge) OriginId() protocol.OriginId {
	args := m.Called()
	return args.Get(0).(protocol.OriginId)
}
func (m *MockSpecEdge) ClaimId() option.Option[protocol.ClaimId] {
	args := m.Called()
	return args.Get(0).(option.Option[protocol.ClaimId])
}
func (m *MockSpecEdge) LowerChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	args := m.Called(ctx)
	return args.Get(0).(option.Option[protocol.EdgeId]), args.Error(1)
}
func (m *MockSpecEdge) UpperChild(ctx context.Context) (option.Option[protocol.EdgeId], error) {
	args := m.Called(ctx)
	return args.Get(0).(option.Option[protocol.EdgeId]), args.Error(1)
}
func (m *MockSpecEdge) HasChildren(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
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
func (m *MockSpecEdge) ConfirmByChildren(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockSpecEdge) HasLengthOneRival(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

type MockEdgeTracker struct {
	mock.Mock
}

func (m *MockEdgeTracker) TrackEdge(ctx context.Context, edge protocol.SpecEdge) error {
	args := m.Called(ctx, edge)
	return args.Error(0)
}

type MockProtocol struct {
	mock.Mock
}

// Read-only methods.
func (m *MockProtocol) NumAssertions(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) GetAssertion(ctx context.Context, id protocol.AssertionHash) (protocol.Assertion, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) AssertionUnrivaledBlocks(ctx context.Context, assertionHash protocol.AssertionHash) (uint64, error) {
	args := m.Called(ctx, assertionHash)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) TopLevelAssertion(ctx context.Context, edgeId protocol.EdgeId) (protocol.AssertionHash, error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(protocol.AssertionHash), args.Error(1)
}

func (m *MockProtocol) TopLevelClaimHeights(ctx context.Context, edgeId protocol.EdgeId) (protocol.OriginHeights, error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(protocol.OriginHeights), args.Error(1)
}

func (m *MockProtocol) LatestCreatedAssertion(ctx context.Context) (protocol.Assertion, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) LatestConfirmed(ctx context.Context) (protocol.Assertion, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) ReadAssertionCreationInfo(
	ctx context.Context, id protocol.AssertionHash,
) (*protocol.AssertionCreatedInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*protocol.AssertionCreatedInfo), args.Error(1)
}

// Mutating methods.
func (m *MockProtocol) ConfirmAssertionByChallengeWinner(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
	winningEdgeId protocol.EdgeId,
) error {
	args := m.Called(ctx, assertionHash)
	return args.Error(0)
}

func (m *MockProtocol) CreateAssertion(
	ctx context.Context,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
) (protocol.Assertion, error) {
	args := m.Called(ctx, assertionCreationInfo, postState)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) SpecChallengeManager(ctx context.Context) (protocol.SpecChallengeManager, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.SpecChallengeManager), args.Error(1)
}

func (m *MockProtocol) Confirm(ctx context.Context, blockHash, sendRoot common.Hash) error {
	args := m.Called(ctx, blockHash, sendRoot)
	return args.Error(0)
}
