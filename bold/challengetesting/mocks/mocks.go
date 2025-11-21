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

	"github.com/offchainlabs/nitro/bold/api/db"
	"github.com/offchainlabs/nitro/bold/chainabstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2stateprovider"
	"github.com/offchainlabs/nitro/bold/statecommitments/history"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

var (
	_ = chainabstraction.SpecChallengeManager(&MockSpecChallengeManager{})
	_ = chainabstraction.SpecEdge(&MockSpecEdge{})
	_ = chainabstraction.AssertionChain(&MockProtocol{})
	_ = layer2stateprovider.Provider(&MockStateManager{})
)

type MockAssertion struct {
	MockId                chainabstraction.AssertionHash
	MockPrevId            chainabstraction.AssertionHash
	Prev                  option.Option[*MockAssertion]
	MockHeight            uint64
	MockStateHash         common.Hash
	MockInboxMsgCountSeen uint64
	MockCreatedAtBlock    uint64
	MockHasSecondChild    bool
	CreatedAt             uint64
}

func (m *MockAssertion) Id() chainabstraction.AssertionHash {
	return m.MockId
}

func (m *MockAssertion) PrevId(ctx context.Context) (chainabstraction.AssertionHash, error) {
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

func (m *MockAssertion) Status(ctx context.Context) (chainabstraction.AssertionStatus, error) {
	return chainabstraction.AssertionPending, nil
}

type MockStateManager struct {
	mock.Mock
	Agrees   bool
	AgreeErr bool
}

func (m *MockStateManager) HistoryCommitment(
	ctx context.Context,
	req *layer2stateprovider.HistoryCommitmentRequest,
) (history.History, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(history.History), args.Error(1)
}

func (m *MockStateManager) UpdateAPIDatabase(apiDB db.Database) {
	m.Called(apiDB)
}

func (m *MockStateManager) PrefixProof(
	ctx context.Context,
	req *layer2stateprovider.HistoryCommitmentRequest,
	prefixHeight layer2stateprovider.Height,
) ([]byte, error) {
	args := m.Called(ctx, req, prefixHeight)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) AgreesWithHistoryCommitment(
	ctx context.Context,
	challengeLevel chainabstraction.ChallengeLevel,
	historyCommitMetadata *layer2stateprovider.HistoryCommitmentRequest,
	commit layer2stateprovider.History,
) (bool, error) {
	args := m.Called(ctx, challengeLevel, historyCommitMetadata, commit)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockStateManager) ExecutionStateAfterPreviousState(ctx context.Context, maxInboxCount uint64, previousGlobalState chainabstraction.GoGlobalState) (*chainabstraction.ExecutionState, error) {
	args := m.Called(ctx, maxInboxCount, previousGlobalState)
	return args.Get(0).(*chainabstraction.ExecutionState), args.Error(1)
}

func (m *MockStateManager) OneStepProofData(
	ctx context.Context,
	assertionMetadata *layer2stateprovider.AssociatedAssertionMetadata,
	startHeights []layer2stateprovider.Height,
	upToHeight layer2stateprovider.Height,
) (data *chainabstraction.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error) {
	args := m.Called(ctx, assertionMetadata.WasmModuleRoot, startHeights, assertionMetadata.FromState.PosInBatch, upToHeight)
	return args.Get(0).(*chainabstraction.OneStepData), args.Get(1).([]common.Hash), args.Get(2).([]common.Hash), args.Error(3)
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

func (m *MockSpecChallengeManager) LayerZeroHeights() chainabstraction.LayerZeroHeights {
	args := m.Called()
	return args.Get(0).(chainabstraction.LayerZeroHeights)
}

func (m *MockSpecChallengeManager) ChallengePeriodBlocks() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

func (m *MockSpecChallengeManager) MultiUpdateInheritedTimers(ctx context.Context, branch []chainabstraction.ReadOnlyEdge, desiredTimerForLastEdge uint64) (*types.Transaction, error) {
	args := m.Called(ctx, branch, desiredTimerForLastEdge)
	return args.Get(0).(*types.Transaction), args.Error(1)
}

func (m *MockSpecChallengeManager) GetEdge(
	ctx context.Context,
	edgeId chainabstraction.EdgeId,
) (option.Option[chainabstraction.SpecEdge], error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(option.Option[chainabstraction.SpecEdge]), args.Error(1)
}

func (m *MockSpecChallengeManager) CalculateMutualId(
	ctx context.Context,
	edgeType chainabstraction.ChallengeLevel,
	originId chainabstraction.OriginId,
	startHeight chainabstraction.Height,
	startHistoryRoot common.Hash,
	endHeight chainabstraction.Height,
) (chainabstraction.MutualId, error) {
	args := m.Called(ctx, edgeType, originId, startHeight, startHistoryRoot, endHeight)
	return args.Get(0).(chainabstraction.MutualId), args.Error(1)
}

func (m *MockSpecChallengeManager) CalculateEdgeId(
	ctx context.Context,
	edgeType chainabstraction.ChallengeLevel,
	originId chainabstraction.OriginId,
	startHeight chainabstraction.Height,
	startHistoryRoot common.Hash,
	endHeight chainabstraction.Height,
	endHistoryRoot common.Hash,
) (chainabstraction.EdgeId, error) {
	args := m.Called(ctx, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
	return args.Get(0).(chainabstraction.EdgeId), args.Error(1)
}

func (m *MockSpecChallengeManager) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion chainabstraction.Assertion,
	startCommit,
	endCommit history.History,
	startEndPrefixProof []byte,
) (chainabstraction.VerifiedRoyalEdge, error) {
	args := m.Called(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
	return args.Get(0).(chainabstraction.VerifiedRoyalEdge), args.Error(1)
}

func (m *MockSpecChallengeManager) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge chainabstraction.SpecEdge,
	startCommit,
	endCommit history.History,
	startParentInclusionProof []common.Hash,
	endParentInclusionProof []common.Hash,
	startEndPrefixProof []byte,
) (chainabstraction.VerifiedRoyalEdge, error) {
	args := m.Called(ctx, challengedEdge, startCommit, endCommit, startParentInclusionProof, endParentInclusionProof, startEndPrefixProof)
	return args.Get(0).(chainabstraction.VerifiedRoyalEdge), args.Error(1)
}

func (m *MockSpecChallengeManager) ConfirmEdgeByOneStepProof(
	ctx context.Context,
	tentativeWinnerId chainabstraction.EdgeId,
	oneStepData *chainabstraction.OneStepData,
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

func (m *MockSpecEdge) Id() chainabstraction.EdgeId {
	args := m.Called()
	return args.Get(0).(chainabstraction.EdgeId)
}

func (m *MockSpecEdge) GetChallengeLevel() chainabstraction.ChallengeLevel {
	args := m.Called()
	return args.Get(0).(chainabstraction.ChallengeLevel)
}

func (m *MockSpecEdge) GetReversedChallengeLevel() chainabstraction.ChallengeLevel {
	args := m.Called()
	return args.Get(0).(chainabstraction.ChallengeLevel)
}

func (m *MockSpecEdge) GetTotalChallengeLevels(ctx context.Context) uint8 {
	args := m.Called(ctx)
	return args.Get(0).(uint8)
}

func (m *MockSpecEdge) MiniStaker() option.Option[common.Address] {
	args := m.Called()
	return args.Get(0).(option.Option[common.Address])
}

func (m *MockSpecEdge) StartCommitment() (chainabstraction.Height, common.Hash) {
	args := m.Called()
	return args.Get(0).(chainabstraction.Height), args.Get(1).(common.Hash)
}

func (m *MockSpecEdge) EndCommitment() (chainabstraction.Height, common.Hash) {
	args := m.Called()
	return args.Get(0).(chainabstraction.Height), args.Get(1).(common.Hash)
}

func (m *MockSpecEdge) TopLevelClaimHeight(ctx context.Context) (chainabstraction.OriginHeights, error) {
	args := m.Called(ctx)
	return args.Get(0).(chainabstraction.OriginHeights), args.Error(1)
}

func (m *MockSpecEdge) AssertionHash(ctx context.Context) (chainabstraction.AssertionHash, error) {
	args := m.Called(ctx)
	return args.Get(0).(chainabstraction.AssertionHash), args.Error(1)
}

func (m *MockSpecEdge) TimeUnrivaled(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSpecEdge) LatestInheritedTimer(ctx context.Context) (chainabstraction.InheritedTimer, error) {
	args := m.Called(ctx)
	return args.Get(0).(chainabstraction.InheritedTimer), args.Error(1)
}

func (m *MockSpecEdge) HasRival(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockSpecEdge) Status(ctx context.Context) (chainabstraction.EdgeStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(chainabstraction.EdgeStatus), args.Error(1)
}

func (m *MockSpecEdge) ConfirmedAtBlock(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSpecEdge) CreatedAtBlock() (uint64, error) {
	args := m.Called()
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSpecEdge) MutualId() chainabstraction.MutualId {
	args := m.Called()
	return args.Get(0).(chainabstraction.MutualId)
}

func (m *MockSpecEdge) OriginId() chainabstraction.OriginId {
	args := m.Called()
	return args.Get(0).(chainabstraction.OriginId)
}

func (m *MockSpecEdge) ClaimId() option.Option[chainabstraction.ClaimId] {
	args := m.Called()
	return args.Get(0).(option.Option[chainabstraction.ClaimId])
}

func (m *MockSpecEdge) LowerChild(ctx context.Context) (option.Option[chainabstraction.EdgeId], error) {
	args := m.Called(ctx)
	return args.Get(0).(option.Option[chainabstraction.EdgeId]), args.Error(1)
}

func (m *MockSpecEdge) UpperChild(ctx context.Context) (option.Option[chainabstraction.EdgeId], error) {
	args := m.Called(ctx)
	return args.Get(0).(option.Option[chainabstraction.EdgeId]), args.Error(1)
}

func (m *MockSpecEdge) HasChildren(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockSpecEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (chainabstraction.VerifiedRoyalEdge, chainabstraction.VerifiedRoyalEdge, error) {
	args := m.Called(ctx, prefixHistoryRoot, prefixProof)
	return args.Get(0).(chainabstraction.VerifiedRoyalEdge), args.Get(1).(chainabstraction.VerifiedRoyalEdge), args.Error(2)
}
func (m *MockSpecEdge) ConfirmByTimer(ctx context.Context, assertionHash chainabstraction.AssertionHash) (*types.Transaction, error) {
	args := m.Called(ctx, assertionHash)
	return args.Get(0).(*types.Transaction), args.Error(1)
}

func (m *MockSpecEdge) ConfirmByClaim(ctx context.Context, claimId chainabstraction.ClaimId) error {
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
func (m *MockSpecEdge) AsVerifiedHonest() (chainabstraction.VerifiedRoyalEdge, bool) {
	args := m.Called()
	return args.Get(0).(chainabstraction.VerifiedRoyalEdge), args.Get(1).(bool)
}

type MockHonestEdge struct {
	*MockSpecEdge
}

func (m *MockHonestEdge) Honest() {}

type MockEdgeTracker struct {
	mock.Mock
}

func (m *MockEdgeTracker) TrackEdge(ctx context.Context, edge chainabstraction.VerifiedRoyalEdge) error {
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

func (m *MockProtocol) Backend() chainabstraction.ChainBackend {
	args := m.Called()
	return args.Get(0).(chainabstraction.ChainBackend)
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

func (m *MockProtocol) IsChallengeComplete(ctx context.Context, challengeParentAssertionHash chainabstraction.AssertionHash) (bool, error) {
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

func (m *MockProtocol) GetAssertion(ctx context.Context, opts *bind.CallOpts, id chainabstraction.AssertionHash) (chainabstraction.Assertion, error) {
	args := m.Called(ctx, opts, id)
	return args.Get(0).(chainabstraction.Assertion), args.Error(1)
}

func (m *MockProtocol) AssertionStatus(ctx context.Context, id chainabstraction.AssertionHash) (chainabstraction.AssertionStatus, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(chainabstraction.AssertionStatus), args.Error(1)
}

func (m *MockProtocol) AssertionUnrivaledBlocks(ctx context.Context, assertionHash chainabstraction.AssertionHash) (uint64, error) {
	args := m.Called(ctx, assertionHash)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) TopLevelAssertion(ctx context.Context, edgeId chainabstraction.EdgeId) (chainabstraction.AssertionHash, error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(chainabstraction.AssertionHash), args.Error(1)
}

func (m *MockProtocol) TopLevelClaimHeights(ctx context.Context, edgeId chainabstraction.EdgeId) (chainabstraction.OriginHeights, error) {
	args := m.Called(ctx, edgeId)
	return args.Get(0).(chainabstraction.OriginHeights), args.Error(1)
}

func (m *MockProtocol) LatestCreatedAssertion(ctx context.Context) (chainabstraction.Assertion, error) {
	args := m.Called(ctx)
	return args.Get(0).(chainabstraction.Assertion), args.Error(1)
}

func (m *MockProtocol) LatestConfirmed(ctx context.Context, opts *bind.CallOpts) (chainabstraction.Assertion, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(chainabstraction.Assertion), args.Error(1)
}

func (m *MockProtocol) ReadAssertionCreationInfo(
	ctx context.Context, id chainabstraction.AssertionHash,
) (*chainabstraction.AssertionCreatedInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*chainabstraction.AssertionCreatedInfo), args.Error(1)
}

func (m *MockProtocol) LatestCreatedAssertionHashes(ctx context.Context) ([]chainabstraction.AssertionHash, error) {
	args := m.Called(ctx)
	return args.Get(0).([]chainabstraction.AssertionHash), args.Error(1)
}

// Mutating methods.
func (m *MockProtocol) ConfirmAssertionByTime(
	ctx context.Context,
	assertionHash chainabstraction.AssertionHash,
) error {
	args := m.Called(ctx, assertionHash)
	return args.Error(0)
}

func (m *MockProtocol) ConfirmAssertionByChallengeWinner(
	ctx context.Context,
	assertionHash chainabstraction.AssertionHash,
	winningEdgeId chainabstraction.EdgeId,
) error {
	args := m.Called(ctx, assertionHash, winningEdgeId)
	return args.Error(0)
}

func (m *MockProtocol) FastConfirmAssertion(
	ctx context.Context,
	assertionCreationInfo *chainabstraction.AssertionCreatedInfo,
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
	assertionCreationInfo *chainabstraction.AssertionCreatedInfo,
	postState *chainabstraction.ExecutionState,
) (chainabstraction.Assertion, error) {
	args := m.Called(ctx, assertionCreationInfo, postState)
	return args.Get(0).(chainabstraction.Assertion), args.Error(1)
}

func (m *MockProtocol) StakeOnNewAssertion(
	ctx context.Context,
	assertionCreationInfo *chainabstraction.AssertionCreatedInfo,
	postState *chainabstraction.ExecutionState,
) (chainabstraction.Assertion, error) {
	args := m.Called(ctx, assertionCreationInfo, postState)
	return args.Get(0).(chainabstraction.Assertion), args.Error(1)
}

func (m *MockProtocol) SpecChallengeManager() chainabstraction.SpecChallengeManager {
	args := m.Called()
	return args.Get(0).(chainabstraction.SpecChallengeManager)
}

func (m *MockProtocol) Confirm(ctx context.Context, blockHash, sendRoot common.Hash) error {
	args := m.Called(ctx, blockHash, sendRoot)
	return args.Error(0)
}
