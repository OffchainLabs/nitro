// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package l2stateprovider

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"time"

	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/bold/api"
	"github.com/offchainlabs/nitro/bold/api/db"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/state-commitments/history"
	"github.com/offchainlabs/nitro/bold/state-commitments/prefix-proofs"
)

// MachineHashCollector defines an interface which collects hashes of the state
// of Arbitrator machines according to a provided `cfg` argument.
// See the documentation of the HashCollectorConfig for the details of how the
// configuration affects the collection of machine hashes.
type MachineHashCollector interface {
	CollectMachineHashes(ctx context.Context, cfg *HashCollectorConfig) ([]common.Hash, error)
}

// ProofCollector defines an interface which can collect a one-step proof from
// an Arbitrator machine at a block height offset from some global state, and
// at a specific opcode index.
type ProofCollector interface {
	CollectProof(
		ctx context.Context,
		assertionMetadata *AssociatedAssertionMetadata,
		blockChallengeHeight Height,
		machineIndex OpcodeIndex,
	) ([]byte, error)
}

// HashCollectorConfig configures the behavior the CollectMachineHashes method.
//
// The goal of CollectMachineHashes is to gather Arbitrator machine hashes for
// a specific arbitrum block in the context of a BoLD challenge which has been
// created to deterimine which assertion is correct.
//
// Depending on the challenge level, the set of machine hashes the collector
// needs to collect can vary. But, it will always be some set of machine hashes
// which represent states of the Arbitrator machine when executing a specific
// "challenged" block.  The "challenged" block is the block within the range
// of an assertion where the rival assertion and this staker's assertions
// diverge.
//
// To determine the exact block from which to collect the machine hashes, the
// collector needs to know the `FromState` which contains the batch and position
// within that batch of the first messsage to which the rival assertions are
// committing. In addiiton, the collector needs to know the
// `BlockChallengeHeight` (which is a relative index within the range of blocks
// to which the rival assertions are committing where they first diverge.)
//
// The collector also needs to know the `BatchLimit` to deal with scenarios
// where the `BlockChallengeHeight` is greater than the number of blocks in the
// assertion.
//
// Most of this information is configured using an `AssociatedAssertionMetadata`
// instance.
//
// The collector then starts collecting hashes at a specific `MachineStartIndex`
// which is an opcode index within the execution the block which corresponds to
// the first machine state hash to be returned. It then steps through the
// Arbitrator machine in increments of `StepSize` until it has collected the
// `NumDesiredHashes` machine hashes.
type HashCollectorConfig struct {
	// Miscellaneous metadata for assertion the commitment is being made for.
	// Includes the WasmModuleRoot and the start and end states.
	AssertionMetadata *AssociatedAssertionMetadata
	// The block challenge height is the height of the block at which the rival
	// assertions diverge.
	BlockChallengeHeight Height
	// Defines the heights at which the collector collects machine hashes for
	// each challenge level.
	// An index in this slice represents a challenge level, and a value
	// represents a height within that challenge level.
	StepHeights []Height
	// The number of desired hashes to be collected.
	NumDesiredHashes uint64
	// The opcode index at which to start stepping through the machine.
	MachineStartIndex OpcodeIndex
	// The step size for stepping through the machine to collect its hashes.
	StepSize StepSize
}

func (h *HashCollectorConfig) String() string {
	str := ""
	str += h.AssertionMetadata.WasmModuleRoot.String()
	str += "/"
	str += fmt.Sprintf("%d", h.AssertionMetadata.FromState.Batch)
	str += "/"
	str += fmt.Sprintf("%d", h.AssertionMetadata.FromState.PosInBatch)
	str += "/"
	str += fmt.Sprintf("%d", h.BlockChallengeHeight)
	str += "/"
	for _, height := range h.StepHeights {
		str += fmt.Sprintf("%d", height)
		str += "/"
	}
	str += fmt.Sprintf("%d", h.NumDesiredHashes)
	str += "/"
	str += fmt.Sprintf("%d", h.MachineStartIndex)
	str += "/"
	str += fmt.Sprintf("%d", h.StepSize)
	return str
}

// L2MessageStateCollector defines an interface which can obtain the machine
// hashes at each L2 message from fromState to batchLimit, ending at
// batch=batchLimit posInBatch=0 unless toHeight+1 states are produced first,
// in which case it ends there.
type L2MessageStateCollector interface {
	L2MessageStatesUpTo(
		ctx context.Context,
		fromState protocol.GoGlobalState,
		batchLimit Batch,
		toHeight option.Option[Height],
	) ([]common.Hash, error)
}

// HistoryCommitmentProvider computes history commitments from input parameters
// by loading Arbitrator machines for L2 state transitions. It can compute
// history commitments over ranges of opcodes at specified increments used for
// the BoLD protocol.
type HistoryCommitmentProvider struct {
	l2MessageStateCollector L2MessageStateCollector
	machineHashCollector    MachineHashCollector
	proofCollector          ProofCollector
	challengeLeafHeights    []Height
	apiDB                   db.Database
	ExecutionProvider
}

// NewHistoryCommitmentProvider creates an instance of a struct which can
// compute history commitments over any number of challenge levels for BoLD.
func NewHistoryCommitmentProvider(
	l2MessageStateCollector L2MessageStateCollector,
	machineHashCollector MachineHashCollector,
	proofCollector ProofCollector,
	challengeLeafHeights []Height,
	executionProvider ExecutionProvider,
	apiDB db.Database,
) *HistoryCommitmentProvider {
	return &HistoryCommitmentProvider{
		l2MessageStateCollector: l2MessageStateCollector,
		machineHashCollector:    machineHashCollector,
		proofCollector:          proofCollector,
		challengeLeafHeights:    challengeLeafHeights,
		ExecutionProvider:       executionProvider,
		apiDB:                   apiDB,
	}
}

// A list of heights that have been validated to be non-empty
// and to be less than the total number of challenge levels in the protocol.
type validatedStartHeights []Height

func (p *HistoryCommitmentProvider) UpdateAPIDatabase(apiDB db.Database) {
	p.apiDB = apiDB
}

// virtualFrom computes the virtual value for a history commitment
//
// I the optional h value is None, then based on the challenge level, and given
// slice of challenge origin heights (coh) determine the maximum number of
// leaves for that level and return it as virtual.
func (p *HistoryCommitmentProvider) virtualFrom(h option.Option[Height], coh []Height) (uint64, error) {
	var virtual uint64
	if h.IsNone() {
		validatedHeights, err := p.validateOriginHeights(coh)
		if err != nil {
			return 0, err
		}
		if len(validatedHeights) == 0 {
			virtual = uint64(p.challengeLeafHeights[0]) + 1
		} else {
			lvl := deepestRequestedChallengeLevel(validatedHeights)
			virtual = uint64(p.challengeLeafHeights[lvl]) + 1
		}
	} else {
		virtual = uint64(h.Unwrap()) + 1
	}
	return virtual, nil
}

// HistoryCommitment computes a Merklelized commitment over a set of hashes
// at specified challenge levels. For block challenges, for example, this is a
// set of machine hashes corresponding each message in a range N to M.
func (p *HistoryCommitmentProvider) HistoryCommitment(
	ctx context.Context,
	req *HistoryCommitmentRequest,
) (history.History, error) {
	hashes, err := p.historyCommitmentImpl(ctx, req)
	if err != nil {
		return history.History{}, err
	}
	virtual, err := p.virtualFrom(req.UpToHeight, req.UpperChallengeOriginHeights)
	if err != nil {
		return history.History{}, err
	}
	return history.NewCommitment(hashes, virtual)
}

func (p *HistoryCommitmentProvider) historyCommitmentImpl(
	ctx context.Context,
	req *HistoryCommitmentRequest,
) ([]common.Hash, error) {
	// Validate the input heights for correctness.
	validatedHeights, err := p.validateOriginHeights(req.UpperChallengeOriginHeights)
	if err != nil {
		return nil, err
	}
	// If the call is for message number ranges only, we get the hashes for
	// those states and return a commitment for them.
	var fromBlockChallengeHeight Height
	if len(validatedHeights) == 0 {
		hashes, hashesErr := p.l2MessageStateCollector.L2MessageStatesUpTo(
			ctx,
			req.AssertionMetadata.FromState,
			req.AssertionMetadata.BatchLimit,
			req.UpToHeight,
		)
		if hashesErr != nil {
			return nil, hashesErr
		}
		return hashes, nil
	} else {
		fromBlockChallengeHeight = validatedHeights[0]
	}

	// Computes the desired challenge level this history commitment is for.
	desiredChallengeLevel := deepestRequestedChallengeLevel(validatedHeights)

	// At each challenge level, the history commitment always starts from the
	// state just before the first opcode within the range of opcodes to which
	// the challenge has been narrowed.
	startIdx := Height(0)

	// Compute the exact start point of where we need to execute
	// the machine from the inputs, and figure out, in what increments, we need
	// to do so.
	machineStartIndex, err := p.computeMachineStartIndex(validatedHeights, startIdx)
	if err != nil {
		return nil, err
	}

	// We compute the stepwise increments we need for stepping through the
	// machine.
	stepSize, err := p.computeStepSize(desiredChallengeLevel)
	if err != nil {
		return nil, err
	}

	// Compute the maximum number of machine hashes we need to collect at the
	// desired challenge level.
	maxHashes, err := p.computeRequiredNumberOfHashes(desiredChallengeLevel, req.UpToHeight)
	if err != nil {
		return nil, err
	}

	// Collect the machine hashes at the specified challenge level based on the
	// values we computed.
	cfg := &HashCollectorConfig{
		AssertionMetadata:    req.AssertionMetadata,
		BlockChallengeHeight: fromBlockChallengeHeight,
		// We drop the first index of the validated heights, because the first
		// index is for the block challenge level, which is over blocks and not
		// over individual machine WASM opcodes. Starting from the second index,
		// we are now dealing with challenges over ranges of opcodes which are
		// what we care about for our implementation of machine hash collection.
		StepHeights:       validatedHeights[1:],
		NumDesiredHashes:  maxHashes,
		MachineStartIndex: machineStartIndex,
		StepSize:          stepSize,
	}
	// Requests collecting machine hashes for the specified config.
	if !api.IsNil(p.apiDB) {
		var rawStepHeights string
		for i, stepHeight := range cfg.StepHeights {
			hInt, err := safecast.ToInt(stepHeight)
			if err != nil {
				return nil, err
			}
			rawStepHeights += strconv.Itoa(hInt)
			if i != len(rawStepHeights)-1 {
				rawStepHeights += ","
			}
		}
		collectMachineHashes := api.JsonCollectMachineHashes{
			WasmModuleRoot:       cfg.AssertionMetadata.WasmModuleRoot,
			FromBatch:            cfg.AssertionMetadata.FromState.Batch,
			PositionInBatch:      cfg.AssertionMetadata.FromState.PosInBatch,
			BatchLimit:           uint64(cfg.AssertionMetadata.BatchLimit),
			BlockChallengeHeight: uint64(cfg.BlockChallengeHeight),
			RawStepHeights:       rawStepHeights,
			NumDesiredHashes:     cfg.NumDesiredHashes,
			MachineStartIndex:    uint64(cfg.MachineStartIndex),
			StepSize:             uint64(cfg.StepSize),
			StartTime:            time.Now().UTC(),
		}
		err := p.apiDB.InsertCollectMachineHash(&collectMachineHashes)
		if err != nil {
			return nil, err
		}
		defer func() {
			finishTime := time.Now().UTC()
			collectMachineHashes.FinishTime = &finishTime
			err := p.apiDB.UpdateCollectMachineHash(&collectMachineHashes)
			if err != nil {
				return
			}
		}()
	}
	startTime := time.Now()
	defer func() {
		// TODO: Replace NewUniformSample(100) with
		// NewBoundedHistogramSample(), once offchainlabs geth is merged in
		// bold.
		// Eg https://github.com/offchainlabs/nitro/blob/ab6790a9e33884c3b4e81de2a97dae5bf904266e/das/restful_server.go#L30
		sizeInt, err := safecast.ToInt(stepSize)
		if err != nil {
			return
		}
		metrics.GetOrRegisterHistogram("arb/state_provider/collect_machine_hashes/step_size_"+strconv.Itoa(sizeInt)+"/duration", nil, metrics.NewUniformSample(100)).Update(time.Since(startTime).Nanoseconds())
	}()
	return p.machineHashCollector.CollectMachineHashes(ctx, cfg)
}

// AgreesWithHistoryCommitment checks if the l2 state provider agrees with a
// specified start and end history commitment for a type of edge under a
// specified assertion challenge. It returns an agreement struct which informs
// the caller whether (a) we agree with the start commitment, and whether (b)
// the edge is honest, meaning that we also agree with the end commitment.
func (p *HistoryCommitmentProvider) AgreesWithHistoryCommitment(
	ctx context.Context,
	challengeLevel protocol.ChallengeLevel,
	historyCommitMetadata *HistoryCommitmentRequest,
	commit History,
) (bool, error) {
	var localCommit history.History
	var err error
	switch challengeLevel {
	case protocol.NewBlockChallengeLevel():
		localCommit, err = p.HistoryCommitment(
			ctx,
			&HistoryCommitmentRequest{
				AssertionMetadata:           historyCommitMetadata.AssertionMetadata,
				UpperChallengeOriginHeights: []Height{},
				UpToHeight:                  option.Some(Height(commit.Height)),
			},
		)
		if err != nil {
			return false, err
		}
	default:
		localCommit, err = p.HistoryCommitment(
			ctx,
			&HistoryCommitmentRequest{
				AssertionMetadata:           historyCommitMetadata.AssertionMetadata,
				UpperChallengeOriginHeights: historyCommitMetadata.UpperChallengeOriginHeights,
				UpToHeight:                  option.Some(Height(commit.Height)),
			},
		)
		if err != nil {
			return false, err
		}
	}
	return localCommit.Height == commit.Height && localCommit.Merkle == commit.MerkleRoot, nil
}

var (
	b32Arr, _ = abi.NewType("bytes32[]", "", nil)
	// ProofArgs for submission to the protocol.
	ProofArgs = abi.Arguments{
		{Type: b32Arr, Name: "prefixExpansion"},
		{Type: b32Arr, Name: "prefixProof"},
	}
)

// PrefixProof allows a caller to provide a proof that, given heights N < M,
// that the history commitment for height N is a Merkle prefix of the
// commitment at height M.
//
// Here's how one would use it:
//
//	fromMessageNumber := 1000
//
//	PrefixProof(
//	  wasmModuleRoot,
//	  batch,
//	  []Height{16},
//	  fromMessageNumber,
//	  upToHeight(Height(24)),
//	)
//
// This means that we want a proof that the history commitment at height 16
// is a prefix of the history commitment at height 24. Each index in the
// []Height{} slice represents a challenge level. For example, this call wants
// us to use the history commitment at the very first challenge level, over
// blocks.
func (p *HistoryCommitmentProvider) PrefixProof(
	ctx context.Context,
	req *HistoryCommitmentRequest,
	prefixHeight Height,
) ([]byte, error) {
	// Obtain the leaves we need to produce our Merkle expansion.
	leaves, err := p.historyCommitmentImpl(
		ctx,
		req,
	)
	if err != nil {
		return nil, err
	}
	virtual, err := p.virtualFrom(req.UpToHeight, req.UpperChallengeOriginHeights)
	if err != nil {
		return nil, err
	}
	// If no upToHeight is provided, we want to use the max number of leaves in
	// our computation.
	lowCommitmentNumLeaves := uint64(prefixHeight + 1)
	// The prefix proof may be over a range of leaves that include virtual ones.
	prefixLen := min(lowCommitmentNumLeaves, uint64(len(leaves)))
	prefixHashes := slices.Clone(leaves[:prefixLen])
	prefixRoot, err := history.ComputeRoot(prefixHashes, lowCommitmentNumLeaves)
	if err != nil {
		return nil, err
	}
	fullTreeHashes := slices.Clone(leaves)
	fullTreeRoot, err := history.ComputeRoot(fullTreeHashes, virtual)
	if err != nil {
		return nil, err
	}
	hashesForProof := make([]common.Hash, len(leaves))
	for i := uint64(0); i < uint64(len(leaves)); i++ {
		hashesForProof[i] = leaves[i]
	}
	prefixExp, proof, err := history.GeneratePrefixProof(uint64(prefixHeight), hashesForProof, virtual)
	if err != nil {
		return nil, err
	}
	// We verify our prefix proof before an onchain submission as an extra
	// safety-check.
	if err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      prefixRoot,
		PreSize:      lowCommitmentNumLeaves,
		PostRoot:     fullTreeRoot,
		PostSize:     virtual,
		PreExpansion: prefixExp,
		PrefixProof:  proof,
	}); err != nil {
		return nil, fmt.Errorf("could not verify prefix proof locally: %w", err)
	}
	return ProofArgs.Pack(&prefixExp, &proof)
}

func (p *HistoryCommitmentProvider) OneStepProofData(
	ctx context.Context,
	assertionMetadata *AssociatedAssertionMetadata,
	startHeights []Height,
	upToHeight Height,
) (*protocol.OneStepData, []common.Hash, []common.Hash, error) {
	// Start heights must reflect at least one challenge level to produce one
	// step proofs.
	if len(startHeights) < 1 {
		return nil, nil, nil, fmt.Errorf("upper challenge origin heights must have at least length 1, got %d", len(startHeights))
	}
	endCommit, err := p.HistoryCommitment(
		ctx,
		&HistoryCommitmentRequest{
			AssertionMetadata:           assertionMetadata,
			UpperChallengeOriginHeights: startHeights,
			UpToHeight:                  option.Some(upToHeight + 1),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	startCommit, err := p.HistoryCommitment(
		ctx,
		&HistoryCommitmentRequest{
			AssertionMetadata:           assertionMetadata,
			UpperChallengeOriginHeights: startHeights,
			UpToHeight:                  option.Some(upToHeight),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// Compute the exact start point of where we need to execute the machine
	// from the inputs, and figure out, in what increments, we need to do so.
	machineIndex, err := p.computeMachineStartIndex(startHeights, upToHeight)
	if err != nil {
		return nil, nil, nil, err
	}

	osp, err := p.proofCollector.CollectProof(ctx, assertionMetadata, startHeights[0], machineIndex)
	if err != nil {
		return nil, nil, nil, err
	}

	data := &protocol.OneStepData{
		BeforeHash: startCommit.LastLeaf,
		AfterHash:  endCommit.LastLeaf,
		Proof:      osp,
	}
	return data, startCommit.LastLeafProof, endCommit.LastLeafProof, nil
}

// Computes the required number of hashes for a history commitment
// based on the requested challenge level. The required number of hashes
// for a leaf commitment at each challenge level is a constant, so we can
// determine the desired challenge level from the input params and compute the
// total from there.
func (p *HistoryCommitmentProvider) computeRequiredNumberOfHashes(
	challengeLevel uint64,
	upToHeight option.Option[Height],
) (uint64, error) {
	maxHeightForLevel, err := p.leafHeightAtChallengeLevel(challengeLevel)
	if err != nil {
		return 0, err
	}

	// Get the requested history commitment height we need at our desired
	// challenge level.
	var end Height
	if upToHeight.IsNone() {
		end = maxHeightForLevel
	} else {
		end = upToHeight.Unwrap()
		// If the end height is more than the allowed max, we return an error.
		// This scenario should not happen, and instead of silently truncating,
		// surfacing an error is the safest way of warning the operator
		// they are committing something invalid.
		if end > maxHeightForLevel {
			return 0, fmt.Errorf(
				"end %d was greater than max height for level %d",
				end,
				maxHeightForLevel,
			)
		}
	}
	// The number of hashes is the difference between the start and end
	// requested heights, plus 1. But, since we always start at 0, it's just
	// the end height + 1.
	return uint64(end) + 1, nil
}

// Figures out the actual opcode index we should move the machine to
// when we compute the history commitment. As there are different levels of
// challenge granularity, we have to do some math to figure out the correct
// index.
//
// Take, for example, that we have 4 challenge kinds:
//
// block_challenge    => over a range of L2 message hashes
// megastep_challenge => over ranges of 1048576 (2^20) opcodes at a time.
// kilostep_challenge => over ranges of 1024 (2^10) opcodes at a time
// step_challenge     => over a range of individual WASM opcodes
//
// We only directly step through WASM machines when in a subchallenge (starting
// at megastep), so we can ignore block challenges for this calculation.
//
// Let's say we want to figure out the machine start opcode index for the
// following inputs:
//
// megastep=4, kilostep=5, step=10
//
// We can compute the opcode index using the following algorithm for the example
// above.
//
//	  4 * (1048576)
//	+ 5 * (1024)
//	+ 10
//	= 4,199,434
//
// This generalizes for any number of subchallenge levels into the algorithm
// below.
// It works by taking the sum of (each input * product of all challenge level
// height constants beneath its level).
// This means we start executing our machine exactly at opcode index 4,199,434.
func (p *HistoryCommitmentProvider) computeMachineStartIndex(
	upperChallengeOriginHeights validatedStartHeights,
	fromHeight Height,
) (OpcodeIndex, error) {
	// For the block challenge level, the machine start opcode index is 0.
	if len(upperChallengeOriginHeights) == 0 {
		return 0, nil
	}
	// The first position in the start heights slice is the block challenge
	// level, which is over ranges of L2 messages and not over individual
	// opcodes. We ignore this level and start at the next level when it comes
	// to dealing with machines.
	heights := upperChallengeOriginHeights[1:]
	heights = append(heights, fromHeight)
	leafHeights := p.challengeLeafHeights[1:]

	// Next, we compute the opcode index. We use big ints to make sure we do not
	// overflow uint64 as this computation depends on external user inputs.
	opcodeIndex := new(big.Int).SetUint64(0)
	idx := 1
	for _, height := range heights {
		total := new(big.Int).SetUint64(1)
		for i := idx; i < len(leafHeights); i++ {
			total = new(big.Int).Mul(total, new(big.Int).SetUint64(uint64(leafHeights[i])))
		}
		increase := new(big.Int).Mul(total, new(big.Int).SetUint64(uint64(height)))
		opcodeIndex = new(big.Int).Add(opcodeIndex, increase)
		idx += 1
	}
	if !opcodeIndex.IsUint64() {
		return 0, fmt.Errorf("computed machine start index overflows uint64: %s", opcodeIndex.String())
	}
	return OpcodeIndex(opcodeIndex.Uint64()), nil
}

// Computes the number of individual opcodes we need to step through a machine
// at a time.
// Each challenge level has a different amount of ranges of opcodes, so the
// overall step size can be computed as a multiplication of all the next
// challenge levels needed.
//
// As an example, this function helps answer questions such as: "How many
// individual opcodes are there in a single step of a Megastep challenge?"
func (p *HistoryCommitmentProvider) computeStepSize(challengeLevel uint64) (StepSize, error) {
	// The last challenge level is over individual opcodes, so the step size is
	// always 1 opcode at a time.
	if challengeLevel+1 == p.numberOfChallengeLevels() {
		return 1, nil
	}
	// Otherwise, it is the multiplication of all the challenge leaf heights at
	// the next challenge levels.
	levels := p.challengeLeafHeights[challengeLevel+1:]
	total := uint64(1)
	for _, h := range levels {
		total *= uint64(h)
	}
	return StepSize(total), nil
}

func (p *HistoryCommitmentProvider) validateOriginHeights(
	upperChallengeOriginHeights []Height,
) (validatedStartHeights, error) {
	// Length cannot be greater than the total number of challenge levels in
	// the protocol - 1.
	if len(upperChallengeOriginHeights) > len(p.challengeLeafHeights)-1 {
		return nil, fmt.Errorf(
			"challenge level %d is out of range for challenge leaf heights %v",
			len(upperChallengeOriginHeights),
			p.challengeLeafHeights,
		)
	}
	return upperChallengeOriginHeights, nil
}

// A caller specifies a request for a history commitment at challenge level N.
// It specifies a list of heights at which to compute the history commitment at
// each challenge level on the way to level N as a list of heights, where each
// position represents a challenge level.
// The length of this list cannot be greater than the total number of challenge
// levels in the protocol.
// Takes in an input type that has already been validated for correctness.
func deepestRequestedChallengeLevel(requestedHeights validatedStartHeights) uint64 {
	return uint64(len(requestedHeights))
}

// Gets the required leaf height at a specified challenge level. This is a
// protocol constant.
func (p *HistoryCommitmentProvider) leafHeightAtChallengeLevel(challengeLevel uint64) (Height, error) {
	if challengeLevel >= uint64(len(p.challengeLeafHeights)) {
		return 0, fmt.Errorf(
			"challenge level %d is out of range for challenge leaf heights %v",
			challengeLevel,
			p.challengeLeafHeights,
		)
	}
	return p.challengeLeafHeights[challengeLevel], nil
}

// The total number of challenge levels in the protocol.
func (p *HistoryCommitmentProvider) numberOfChallengeLevels() uint64 {
	return uint64(len(p.challengeLeafHeights))
}
