// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package l2stateprovider defines a dependency which provides L2 states and
// proofs needed for the challenge manager to interact with an Arbitrum chain's
// rollup and challenge contracts.
package l2stateprovider

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/api/db"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/state-commitments/history"
)

var ErrChainCatchingUp = errors.New("chain is catching up to the execution state")

// Batch index for an Arbitrum L2 state.
type Batch uint64

// Height for a BoLD history commitment.
type Height uint64

// OpcodeIndex within an Arbitrator machine for an L2 message.
type OpcodeIndex uint64

// StepSize is the number of opcode increments used for stepping through
// machines for BoLD challenges.
type StepSize uint64

// ConfigSnapshot for an assertion on Arbitrum.
type ConfigSnapshot struct {
	RequiredStake           *big.Int
	ChallengeManagerAddress common.Address
	ConfirmPeriodBlocks     uint64
	WasmModuleRoot          [32]byte
	InboxMaxCount           *big.Int
}

type History struct {
	Height     uint64
	MerkleRoot common.Hash
}

// Provider defines an L2 state backend that can provide history commitments,
// execution states, prefix proofs, and more for the BoLD protocol.
type Provider interface {
	ExecutionProvider
	GeneralHistoryCommitter
	GeneralPrefixProver
	OneStepProofProvider
	HistoryChecker
}

type ExecutionProvider interface {
	// Produces the L2 execution state to assert to after the previous assertion
	// state.
	// Returns either the state at the batch count maxInboxCount (PosInBatch=0) or
	// the state LayerZeroHeights.BlockChallengeHeight blokcs after
	// previousGlobalState, whichever is an earlier state.
	ExecutionStateAfterPreviousState(ctx context.Context, maxInboxCount uint64, previousGlobalState protocol.GoGlobalState) (*protocol.ExecutionState, error)
}

// AssociatedAssertionMetadata for the tracked edge.
type AssociatedAssertionMetadata struct {
	FromState protocol.GoGlobalState
	// This assertion may not read this batch.
	// Unless it hits the block limit, its last state in position 0 of this batch.
	BatchLimit           Batch
	WasmModuleRoot       common.Hash
	ClaimedAssertionHash protocol.AssertionHash
}

// HistoryCommitmentRequest for a BoLD history commitment.
//
// The request specifies the metadata for the assertion which is being
// challenged in the block level challenge, and the heights at which the
// challenges at higher challenge levels originated.
//
// HistoryCommitment requestors can also specify an optional height at which to
// end the history commitment. If none, the request will commit to all the
// leaves at the current challenge level.
//
// NOTE: It is NOT possible to request a history commitment which starts at
// some height other than 0 for the current challenge level. This is because
// the edge tracker only needs to be able to provide history commitments for
// all machine state hases at the current challenge level, or sets of leaves
// which are prefixes to that full set of leaves. In all cases, the first leaf
// is the one in relative position 0 for the challenge level.
type HistoryCommitmentRequest struct {
	// Miscellaneous metadata for assertion the commitment is being made for.
	// Includes the WasmModuleRoot and the start and end states.
	AssertionMetadata *AssociatedAssertionMetadata
	// A slice of heights that tells the backend where the subchallenges for the
	// requested history commitment originated from.
	// Each index corresponds to a challenge level. For example,
	// if we have three levels, where lvl 0 is the block challenge level, an
	// input of []Height{12, 3} tells us that that the top-level subchallenge
	// originated at height 12 then the next subchallenge originated at height
	// 3 below that.
	UpperChallengeOriginHeights []Height
	// An optional height at which to end the history commitment. If none, the
	// request will commit to all the leaves at the specified challenge level.
	UpToHeight option.Option[Height]
}

type GeneralHistoryCommitter interface {
	// Request a history commitment for the machine state hashes at the current
	// challenge level. See the HistoryCommitmentRequest struct for details.
	HistoryCommitment(
		ctx context.Context,
		req *HistoryCommitmentRequest,
	) (history.History, error)
	UpdateAPIDatabase(db.Database)
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
		assertionMetadata *AssociatedAssertionMetadata,
		upperChallengeOriginHeights []Height,
		upToHeight Height,
	) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error)
}

type HistoryChecker interface {
	AgreesWithHistoryCommitment(
		ctx context.Context,
		challengeLevel protocol.ChallengeLevel,
		historyCommitMetadata *HistoryCommitmentRequest,
		commit History,
	) (bool, error)
}
