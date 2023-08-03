// Package l2stateprovider defines a dependency which provides L2 states and proofs
// needed for the challenge manager to interact with Arbitrum chains' rollup and challenge
// contracts.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package l2stateprovider

import (
	"context"
	"errors"
	"math/big"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNoExecutionState = errors.New("chain does not have execution state")
)

type ConfigSnapshot struct {
	RequiredStake           *big.Int
	ChallengeManagerAddress common.Address
	ConfirmPeriodBlocks     uint64
	WasmModuleRoot          [32]byte
	InboxMaxCount           *big.Int
}

type Provider interface {
	ExecutionProvider
	HistoryCommitter
	HistoryLeafCommitter
	PrefixProver
	OneStepProofProvider
	HistoryChecker
}

type ExecutionProvider interface {
	// Produces the latest state to assert to L1 from the local state manager's perspective.
	ExecutionStateAtMessageNumber(ctx context.Context, messageNumber uint64) (*protocol.ExecutionState, error)
	// If the state manager locally has this execution state, returns its message count and no error.
	// Returns ErrChainCatchingUp if catching up to chain.
	// Returns ErrNoExecutionState if the state manager does not have this execution state.
	ExecutionStateMsgCount(ctx context.Context, state *protocol.ExecutionState) (uint64, error)
}

type HistoryCommitter interface {
	// Produces a block challenge history commitment up to and including a certain message number.
	HistoryCommitmentAtMessage(ctx context.Context, messageNumber uint64) (commitments.History, error)
	// Produces a big step history commitment from big step 0 to N within block
	// challenge heights A and B where B = A + 1.
	BigStepCommitmentUpTo(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		messageNumber,
		bigStep uint64,
	) (commitments.History, error)
	// Produces a small step history commitment from small step 0 to N between
	// big steps S to S+1 within block challenge heights H to H+1.
	SmallStepCommitmentUpTo(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		messageNumber,
		bigStep,
		toSmallStep uint64,
	) (commitments.History, error)
}

type HistoryLeafCommitter interface {
	// Produces a block challenge history commitment in a certain inclusive message number range,
	// but padding states with duplicates after the first state with a
	// batch count of at least the specified max.
	HistoryCommitmentUpToBatch(
		ctx context.Context,
		messageNumberStart,
		messageNumberEnd,
		batchCount uint64,
	) (commitments.History, error)
	// Produces a big step history commitment for all big steps within block
	// challenge heights H to H+1.
	BigStepLeafCommitment(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		messageNumber uint64,
	) (commitments.History, error)
	// Produces a small step history commitment for all small steps between
	// big steps S to S+1 within block challenge heights H to H+1.
	SmallStepLeafCommitment(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		messageNumber,
		bigStep uint64,
	) (commitments.History, error)
}

type PrefixProver interface {
	// Produces a prefix proof in a block challenge from height A to B, but padding states with duplicates after the
	// first state with a batch count of at least the specified max.
	PrefixProofUpToBatch(
		ctx context.Context,
		startHeight,
		fromMessageNumber,
		toMessageNumber,
		maxBatchCount uint64,
	) ([]byte, error)
	// Produces a big step prefix proof from height A to B for heights H to H+1
	// within a block challenge.
	BigStepPrefixProof(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		messageNumber,
		fromBigStep,
		toBigStep uint64,
	) ([]byte, error)
	// Produces a small step prefix proof from height A to B for big step S to S+1 and
	// block challenge height heights H to H+1.
	SmallStepPrefixProof(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		messageNumber,
		bigStep,
		fromSmallStep,
		toSmallStep uint64,
	) ([]byte, error)
}

type OneStepProofProvider interface {
	OneStepProofData(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		postState rollupgen.ExecutionState,
		messageNumber,
		bigStep,
		smallStep uint64,
	) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error)
}

type History struct {
	Height     uint64
	MerkleRoot common.Hash
}

type HistoryChecker interface {
	AgreesWithHistoryCommitment(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		assertionInboxMaxCount uint64,
		parentAssertionAfterStateBatch uint64,
		edgeType protocol.EdgeType,
		heights protocol.OriginHeights,
		history History,
	) (bool, error)
}
