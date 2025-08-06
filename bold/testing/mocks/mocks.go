// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package mocks includes simple mocks for unit testing BOLD.
// nolint:errcheck
package mocks

import (
	"context"
	"math/big"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/api/db"
	protocol "github.com/offchainlabs/bold/chain-abstraction"
	"github.com/offchainlabs/bold/containers/option"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/state-commitments/history"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
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

func (m *MockAssertion) CreatedAtBlock() uint64 {
	return m.CreatedAt
}

func (m *MockAssertion) FirstChildCreationBlock() (uint64, error) {
	return 0, nil
}

func (m *MockAssertion) SecondChildCreationBlock() (uint64, error) {
	return 0, nil
}

func (m *MockAssertion) IsFirstChild() (bool, error) {
	return false, nil
}

func (m *MockAssertion) Status(ctx context.Context) (protocol.AssertionStatus, error) {
	return protocol.AssertionPending, nil
}

type MockStateManager struct {
	mock.Mock
	Agrees   bool
	AgreeErr bool
}

func (m *MockStateManager) HistoryCommitment(
	ctx context.Context,
	req *l2stateprovider.HistoryCommitmentRequest,
) (history.History, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(history.History), args.Error(1)
}

func (m *MockStateManager) UpdateAPIDatabase(apiDB db.Database) {
	m.Called(apiDB)
}

func (m *MockStateManager) PrefixProof(
	ctx context.Context,
	req *l2stateprovider.HistoryCommitmentRequest,
	prefixHeight l2stateprovider.Height,
) ([]byte, error) {
	args := m.Called(ctx, req, prefixHeight)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) AgreesWithHistoryCommitment(
	ctx context.Context,
	challengeLevel protocol.ChallengeLevel,
	historyCommitMetadata *l2stateprovider.HistoryCommitmentRequest,
	commit l2stateprovider.History,
) (bool, error) {
	args := m.Called(ctx, challengeLevel, historyCommitMetadata, commit)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockStateManager) ExecutionStateAfterPreviousState(ctx context.Context, maxInboxCount uint64, previousGlobalState protocol.GoGlobalState) (*protocol.ExecutionState, error) {
	args := m.Called(ctx, maxInboxCount, previousGlobalState)
	return args.Get(0).(*protocol.ExecutionState), args.Error(1)
}

func (m *MockStateManager) OneStepProofData(
	ctx context.Context,
	assertionMetadata *l2stateprovider.AssociatedAssertionMetadata,
	startHeights []l2stateprovider.Height,
	upToHeight l2stateprovider.Height,
) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error) {
	args := m.Called(ctx, assertionMetadata.WasmModuleRoot, startHeights, assertionMetadata.FromState.PosInBatch, upToHeight)
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

func (m *MockSpecChallengeManager) NumBigSteps() uint8 {
	args := m.Called()
	return args.Get(0).(uint8)
}

func (m *MockSpecChallengeManager) LayerZeroHeights() protocol.LayerZeroHeights {
	args := m.Called()
	return args.Get(0).(protocol.LayerZeroHeights)
}

func (m *MockSpecChallengeManager) ChallengePeriodBlocks() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

func (m *MockSpecChallengeManager) MultiUpdateInheritedTimers(ctx context.Context, branch []protocol.ReadOnlyEdge, desiredTimerForLastEdge uint64) (*types.Transaction, error) {
	args := m.Called(ctx, branch, desiredTimerForLastEdge)
	return args.Get(0).(*types.Transaction), args.Error(1)
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
	edgeType protocol.ChallengeLevel,
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
	edgeType protocol.ChallengeLevel,
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
	endCommit history.History,
	startEndPrefixProof []byte,
) (protocol.VerifiedRoyalEdge, error) {
	args := m.Called(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
	return args.Get(0).(protocol.VerifiedRoyalEdge), args.Error(1)
}

func (m *MockSpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	startCommit,
	endCommit history.History,
	startParentInclusionProof []common.Hash,
	endParentInclusionProof []common.Hash,
	startEndPrefixProof []byte,
) (protocol.VerifiedRoyalEdge, error) {
	args := m.Called(ctx, challengedEdge, startCommit, endCommit, startParentInclusionProof, endParentInclusionProof, startEndPrefixProof)
	return args.Get(0).(protocol.VerifiedRoyalEdge), args.Error(1)
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

func (m *MockSpecEdge) GetChallengeLevel() protocol.ChallengeLevel {
	args := m.Called()
	return args.Get(0).(protocol.ChallengeLevel)
}

func (m *MockSpecEdge) GetReversedChallengeLevel() protocol.ChallengeLevel {
	args := m.Called()
	return args.Get(0).(protocol.ChallengeLevel)
}

func (m *MockSpecEdge) GetTotalChallengeLevels(ctx context.Context) uint8 {
	args := m.Called(ctx)
	return args.Get(0).(uint8)
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

func (m *MockSpecEdge) LatestInheritedTimer(ctx context.Context) (protocol.InheritedTimer, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.InheritedTimer), args.Error(1)
}

func (m *MockSpecEdge) HasRival(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockSpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.EdgeStatus), args.Error(1)
}

func (m *MockSpecEdge) ConfirmedAtBlock(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
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
) (protocol.VerifiedRoyalEdge, protocol.VerifiedRoyalEdge, error) {
	args := m.Called(ctx, prefixHistoryRoot, prefixProof)
	return args.Get(0).(protocol.VerifiedRoyalEdge), args.Get(1).(protocol.VerifiedRoyalEdge), args.Error(2)
}
func (m *MockSpecEdge) ConfirmByTimer(ctx context.Context, assertionHash protocol.AssertionHash) (*types.Transaction, error) {
	args := m.Called(ctx, assertionHash)
	return args.Get(0).(*types.Transaction), args.Error(1)
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
func (m *MockSpecEdge) MarkAsHonest() {
	m.Called()
}
func (m *MockSpecEdge) AsVerifiedHonest() (protocol.VerifiedRoyalEdge, bool) {
	args := m.Called()
	return args.Get(0).(protocol.VerifiedRoyalEdge), args.Get(1).(bool)
}

type MockHonestEdge struct {
	*MockSpecEdge
}

func (m *MockHonestEdge) Honest() {}

type MockEdgeTracker struct {
	mock.Mock
}

func (m *MockEdgeTracker) TrackEdge(ctx context.Context, edge protocol.VerifiedRoyalEdge) error {
	args := m.Called(ctx, edge)
	return args.Error(0)
}

type MockProtocol struct {
	mock.Mock
}

func (m *MockProtocol) GetCallOptsWithDesiredRpcHeadBlockNumber(opts *bind.CallOpts) *bind.CallOpts {
	if opts == nil {
		opts = &bind.CallOpts{}
	}
	return opts
}

func (m *MockProtocol) GetAssertionCreationParentBlock(ctx context.Context, assertionHash common.Hash) (uint64, error) {
	args := m.Called(ctx, assertionHash)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) GetDesiredRpcHeadBlockNumber() rpc.BlockNumber {
	return rpc.LatestBlockNumber
}

// Read-only methods.
func (m *MockProtocol) DesiredHeaderU64(ctx context.Context) (uint64, error) {
	args := m.Called()
	return args.Get(0).(uint64), args.Error(1)
}

// Read-only methods.
func (m *MockProtocol) DesiredL1HeaderU64(ctx context.Context) (uint64, error) {
	args := m.Called()
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) Backend() protocol.ChainBackend {
	args := m.Called()
	return args.Get(0).(protocol.ChainBackend)
}

func (m *MockProtocol) RollupAddress() common.Address {
	args := m.Called()
	return args.Get(0).(common.Address)
}

func (m *MockProtocol) StakerAddress() common.Address {
	args := m.Called()
	return args.Get(0).(common.Address)
}

func (m *MockProtocol) RollupUserLogic() *rollupgen.RollupUserLogic {
	args := m.Called()
	return args.Get(0).(*rollupgen.RollupUserLogic)
}

func (m *MockProtocol) IsChallengeComplete(ctx context.Context, challengeParentAssertionHash protocol.AssertionHash) (bool, error) {
	args := m.Called(ctx, challengeParentAssertionHash)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockProtocol) NumAssertions(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) MinAssertionPeriodBlocks() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

func (m *MockProtocol) MaxAssertionsPerChallengePeriod() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

func (m *MockProtocol) GetAssertion(ctx context.Context, opts *bind.CallOpts, id protocol.AssertionHash) (protocol.Assertion, error) {
	args := m.Called(ctx, opts, id)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) AssertionStatus(ctx context.Context, id protocol.AssertionHash) (protocol.AssertionStatus, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(protocol.AssertionStatus), args.Error(1)
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

func (m *MockProtocol) LatestConfirmed(ctx context.Context, opts *bind.CallOpts) (protocol.Assertion, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) ReadAssertionCreationInfo(
	ctx context.Context, id protocol.AssertionHash,
) (*protocol.AssertionCreatedInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*protocol.AssertionCreatedInfo), args.Error(1)
}

func (m *MockProtocol) LatestCreatedAssertionHashes(ctx context.Context) ([]protocol.AssertionHash, error) {
	args := m.Called(ctx)
	return args.Get(0).([]protocol.AssertionHash), args.Error(1)
}

// Mutating methods.
func (m *MockProtocol) ConfirmAssertionByTime(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
) error {
	args := m.Called(ctx, assertionHash)
	return args.Error(0)
}

func (m *MockProtocol) ConfirmAssertionByChallengeWinner(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
	winningEdgeId protocol.EdgeId,
) error {
	args := m.Called(ctx, assertionHash, winningEdgeId)
	return args.Error(0)
}

func (m *MockProtocol) FastConfirmAssertion(
	ctx context.Context,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
) (bool, error) {
	args := m.Called(ctx, assertionCreationInfo)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockProtocol) IsStaked(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockProtocol) AutoDepositTokenForStaking(
	ctx context.Context,
	amount *big.Int,
) error {
	args := m.Called(ctx, amount)
	return args.Error(0)
}

func (m *MockProtocol) ApproveAllowances(
	ctx context.Context,
) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProtocol) NewStake(
	ctx context.Context,
) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProtocol) NewStakeOnNewAssertion(
	ctx context.Context,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
) (protocol.Assertion, error) {
	args := m.Called(ctx, assertionCreationInfo, postState)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) StakeOnNewAssertion(
	ctx context.Context,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
) (protocol.Assertion, error) {
	args := m.Called(ctx, assertionCreationInfo, postState)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) SpecChallengeManager() protocol.SpecChallengeManager {
	args := m.Called()
	return args.Get(0).(protocol.SpecChallengeManager)
}

func (m *MockProtocol) Confirm(ctx context.Context, blockHash, sendRoot common.Hash) error {
	args := m.Called(ctx, blockHash, sendRoot)
	return args.Error(0)
}
