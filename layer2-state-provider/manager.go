package l2stateprovider

import (
	"context"
	"errors"
	"math/big"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
)

var ErrChainCatchingUp = errors.New("chain catching up")

type ConfigSnapshot struct {
	RequiredStake           *big.Int
	ChallengeManagerAddress common.Address
	ConfirmPeriodBlocks     uint64
	WasmModuleRoot          [32]byte
	InboxMaxCount           *big.Int
}

type Provider interface {
	// Produces the latest state to assert to L1 from the local state manager's perspective.
	LatestExecutionState(ctx context.Context) (*protocol.ExecutionState, error)
	// If the state manager locally has this execution state, returns its message count and true.
	// Otherwise, returns false.
	// Returns ErrChainCatchingUp if catching up to chain.
	ExecutionStateMsgCount(ctx context.Context, state *protocol.ExecutionState) (uint64, bool, error)
	// Produces a block challenge history commitment up to and including a certain height.
	HistoryCommitmentUpTo(ctx context.Context, blockChallengeHeight uint64) (commitments.History, error)
	// Produces a block challenge history commitment in a certain inclusive block range,
	// but padding states with duplicates after the first state with a
	// batch count of at least the specified max.
	HistoryCommitmentUpToBatch(
		ctx context.Context,
		blockStart,
		blockEnd,
		batchCount uint64,
	) (commitments.History, error)
	// Produces a big step history commitment for all big steps within block
	// challenge heights H to H+1.
	BigStepLeafCommitment(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		blockHeight uint64,
	) (commitments.History, error)
	// Produces a big step history commitment from big step 0 to N within block
	// challenge heights A and B where B = A + 1.
	BigStepCommitmentUpTo(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		blockHeight,
		toBigStep uint64,
	) (commitments.History, error)
	// Produces a small step history commitment for all small steps between
	// big steps S to S+1 within block challenge heights H to H+1.
	SmallStepLeafCommitment(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		blockHeight,
		bigStep uint64,
	) (commitments.History, error)
	// Produces a small step history commitment from small step 0 to N between
	// big steps S to S+1 within block challenge heights H to H+1.
	SmallStepCommitmentUpTo(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		blockHeight,
		bigStep,
		toSmallStep uint64,
	) (commitments.History, error)
	// Produces a prefix proof in a block challenge from height A to B.
	PrefixProof(
		ctx context.Context,
		fromBlockChallengeHeight,
		toBlockChallengeHeight uint64,
	) ([]byte, error)
	// Produces a prefix proof in a block challenge from height A to B, but padding states with duplicates after the first state with a batch count of at least the specified max.
	PrefixProofUpToBatch(
		ctx context.Context,
		startHeight,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		batchCount uint64,
	) ([]byte, error)
	// Produces a big step prefix proof from height A to B for heights H to H+1
	// within a block challenge.
	BigStepPrefixProof(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		blockHeight,
		fromBigStep,
		toBigStep uint64,
	) ([]byte, error)
	// Produces a small step prefix proof from height A to B for big step S to S+1 and
	// block challenge height heights H to H+1.
	SmallStepPrefixProof(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		blockHeight,
		bigStep,
		fromSmallStep,
		toSmallStep uint64,
	) ([]byte, error)
	OneStepProofData(
		ctx context.Context,
		cfgSnapshot *ConfigSnapshot,
		postState rollupgen.ExecutionState,
		blockHeight,
		bigStep,
		fromSmallStep,
		toSmallStep uint64,
	) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error)
	HistoryChecker
}

type HistoryChecker interface {
	AgreesWithHistoryCommitment(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		edgeType protocol.EdgeType,
		prevAssertionInboxMaxCount uint64,
		heights *protocol.OriginHeights,
		startCommit,
		endCommit commitments.History,
	) (protocol.Agreement, error)
}
