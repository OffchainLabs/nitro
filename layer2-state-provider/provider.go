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
	"github.com/OffchainLabs/bold/containers/option"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
)

var ErrChainCatchingUp = errors.New("chain is catching up to the execution state")

// Batch index for an Arbitrum L2 state.
type Batch uint64

// Height for a BOLD history commitment.
type Height uint64

// OpcodeIndex within an Arbitrator machine for an L2 message.
type OpcodeIndex uint64

// StepSize is the number of opcode increments used for stepping through
// machines for BOLD challenges.
type StepSize uint64

// ConfigSnapshot for an assertion on Arbitrum.
type ConfigSnapshot struct {
	RequiredStake           *big.Int
	ChallengeManagerAddress common.Address
	ConfirmPeriodBlocks     uint64
	WasmModuleRoot          [32]byte
	InboxMaxCount           *big.Int
}

// Provider defines an L2 state backend that can provide history commitments, execution
// states, prefix proofs, and more for the BOLD protocol.
type Provider interface {
	ExecutionProvider
	GeneralHistoryCommitter
	GeneralPrefixProver
	OneStepProofProvider
	HistoryChecker
}

type ExecutionProvider interface {
	// Produces the L2 execution state to assert to after the previous assertion state.
	// Returns either the state at the batch count maxInboxCount or the state maxNumberOfBlocks after previousGlobalState,
	// whichever is an earlier state. If previousGlobalState is nil, this function simply returns the state at maxInboxCount batches.
	ExecutionStateAfterPreviousState(ctx context.Context, maxInboxCount uint64, previousGlobalState *protocol.GoGlobalState, maxNumberOfBlocks uint64) (*protocol.ExecutionState, error)
}

type HistoryCommitmentRequest struct {
	// The WasmModuleRoot for the execution of machines. This is a global parameter
	// that is specified in the Rollup contracts.
	WasmModuleRoot common.Hash
	// The batch sequence number at which we want to start computing this history commitment.
	FromBatch Batch
	// The batch sequence number at which we want to end computing this history commitment.
	ToBatch Batch
	// A slice of heights that tells the backend where the subchallenges for the requested
	// history commitment originated from.
	// Each index corresponds to a challenge level. For example,
	// if we have three levels, where lvl 0 is the block challenge level, an input of
	// []Height{12, 3} tells us that that the top-level subchallenge originated at height 12
	// then the next subchallenge originated at height 3 below that.
	UpperChallengeOriginHeights []Height
	// The height at which to start the history commitment.
	FromHeight Height
	// An optional height at which to end the history commitment. If none, the request
	// will commit to all the leaves at the specified challenge level.
	UpToHeight option.Option[Height]
	// ClaimId for the request.
	ClaimId common.Hash
}

type GeneralHistoryCommitter interface {
	HistoryCommitment(
		ctx context.Context,
		req *HistoryCommitmentRequest,
	) (commitments.History, error)
}

type GeneralPrefixProver interface {
	PrefixProof(
		ctx context.Context,
		req *HistoryCommitmentRequest,
		prefixHeight Height,
	) ([]byte, error)
}

type OneStepProofProvider interface {
	OneStepProofData(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		fromBatch,
		toBatch Batch,
		upperChallengeOriginHeights []Height,
		fromHeight,
		upToHeight Height,
	) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error)
}

type History struct {
	Height     uint64
	MerkleRoot common.Hash
}

type HistoryChecker interface {
	AgreesWithHistoryCommitment(
		ctx context.Context,
		challengeLevel protocol.ChallengeLevel,
		historyCommitMetadata *HistoryCommitmentRequest,
		commit History,
	) (bool, error)
}
