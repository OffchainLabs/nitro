package l2stateprovider

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	inprogresscache "github.com/OffchainLabs/bold/containers/in-progress-cache"
	prefixproofs "github.com/OffchainLabs/bold/state-commitments/prefix-proofs"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/OffchainLabs/bold/api"
	"github.com/OffchainLabs/bold/api/db"
	"github.com/OffchainLabs/bold/containers/option"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
)

// MachineHashCollector defines an interface which can collect hashes from an Arbitrator machine
// at a block height, starting at a specific opcode index in the machine and stepping through it
// in increments of custom size. Along the way, it computes each machine hash at each step
// and outputs a list of these hashes at the end. This is a computationally expensive process
// that is best performed if machine hashes are cached after runs.
type MachineHashCollector interface {
	CollectMachineHashes(ctx context.Context, cfg *HashCollectorConfig) ([]common.Hash, error)
}

// ProofCollector defines an interface which can collect proof from an Arbitrator machine
// at a block height, at a specific opcode index.
type ProofCollector interface {
	CollectProof(
		ctx context.Context,
		wasmModuleRoot common.Hash,
		fromBatch Batch,
		blockChallengeHeight Height,
		machineIndex OpcodeIndex,
	) ([]byte, error)
}

// HashCollectorConfig defines configuration options for a machine hash collector to
// step through an Arbitrator machine at a specific L2 message number, step through it in
// increments of `StepSize`, and collect the machine hashes at those steps into an output slice.
type HashCollectorConfig struct {
	// The WASM module root the machines should be a part of.
	WasmModuleRoot common.Hash
	// The batch at which to start computation.
	FromBatch Batch
	// The block challenge height for hash collection.
	BlockChallengeHeight Height
	// Defines the heights at which we want to collect machine hashes for each challenge level.
	// An index in this slice represents a challenge level, and a value represents a height within that
	// challenge level.
	StepHeights []Height
	// The number of desired hashes to be collected.
	NumDesiredHashes uint64
	// The opcode index at which to start stepping through the machine at a message number.
	MachineStartIndex OpcodeIndex
	// The step size for stepping through the machine in order to collect its hashes.
	StepSize StepSize
}

func (h *HashCollectorConfig) String() string {
	str := ""
	str += h.WasmModuleRoot.String()
	str += "/"
	str += fmt.Sprintf("%d", h.FromBatch)
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

// L2MessageStateCollector defines an interface which can obtain the machine hashes at each L2 message
// in a specified message range for a given batch index on Arbitrum.
type L2MessageStateCollector interface {
	L2MessageStatesUpTo(
		ctx context.Context,
		fromHeight Height,
		toHeight option.Option[Height],
		fromBatch,
		toBatch Batch,
	) ([]common.Hash, error)
}

// HistoryCommitmentProvider computes history commitments from input parameters
// by loading Arbitrator machines for L2 state transitions. It can compute history commitments
// over ranges of opcodes at specified increments used for the BOLD protocol.
type HistoryCommitmentProvider struct {
	l2MessageStateCollector L2MessageStateCollector
	machineHashCollector    MachineHashCollector
	proofCollector          ProofCollector
	challengeLeafHeights    []Height
	inFlightRequestCache    *inprogresscache.Cache[string, []common.Hash]
	apiDB                   db.Database
	ExecutionProvider
}

// NewHistoryCommitmentProvider creates an instance of a struct which can compute history commitments
// over any number of challenge levels for BOLD.
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
		inFlightRequestCache:    inprogresscache.New[string, []common.Hash](),
		apiDB:                   apiDB,
	}
}

// A list of heights that have been validated to be non-empty
// and to be less than the total number of challenge levels in the protocol.
type validatedStartHeights []Height

// HistoryCommitment computes a Merklelized commitment over a set of hashes
// at specified challenge levels. For block challenges, for example, this is a set
// of machine hashes corresponding each message in a range N to M.
func (p *HistoryCommitmentProvider) HistoryCommitment(
	ctx context.Context,
	req *HistoryCommitmentRequest,
) (commitments.History, error) {
	hashes, err := p.historyCommitmentImpl(ctx, req)
	if err != nil {
		return commitments.History{}, err
	}
	return commitments.New(hashes)
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
			req.FromHeight,
			req.UpToHeight,
			req.FromBatch,
			req.ToBatch,
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

	// Compute the exact start point of where we need to execute
	// the machine from the inputs, and figure out, in what increments, we need to do so.
	machineStartIndex, err := p.computeMachineStartIndex(validatedHeights, req.FromHeight)
	if err != nil {
		return nil, err
	}

	// We compute the stepwise increments we need for stepping through the machine.
	stepSize, err := p.computeStepSize(desiredChallengeLevel)
	if err != nil {
		return nil, err
	}

	// Compute how many machine hashes we need to collect at the desired challenge level.
	numHashes, err := p.computeRequiredNumberOfHashes(desiredChallengeLevel, req.FromHeight, req.UpToHeight)
	if err != nil {
		return nil, err
	}

	// Collect the machine hashes at the specified challenge level based on the values we computed.
	cfg := &HashCollectorConfig{
		WasmModuleRoot:       req.WasmModuleRoot,
		FromBatch:            req.FromBatch,
		BlockChallengeHeight: fromBlockChallengeHeight,
		// We drop the first index of the validated heights, because the first index is for the block challenge level,
		// which is over blocks and not over individual machine WASM opcodes. Starting from the second index, we are now
		// dealing with challenges over ranges of opcodes which are what we care about for our implementation of machine hash collection.
		StepHeights:       validatedHeights[1:],
		NumDesiredHashes:  numHashes,
		MachineStartIndex: machineStartIndex,
		StepSize:          stepSize,
	}
	// Requests collecting machine hashes for the specified config, and uses an in-flight
	// request cache to make sure the same request is not spawned twice, but rather
	// the second request would wait for the in-flight request to complete and use its result.
	return p.inFlightRequestCache.Compute(cfg.String(), func() ([]common.Hash, error) {
		if !api.IsNil(p.apiDB) {
			var rawStepHeights string
			for i, stepHeight := range cfg.StepHeights {
				rawStepHeights += strconv.Itoa(int(stepHeight))
				if i != len(rawStepHeights)-1 {
					rawStepHeights += ","
				}
			}
			collectMachineHashes := api.JsonCollectMachineHashes{
				WasmModuleRoot:       cfg.WasmModuleRoot,
				FromBatch:            uint64(cfg.FromBatch),
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
		return p.machineHashCollector.CollectMachineHashes(ctx, cfg)
	})
}

// AgreesWithHistoryCommitment checks if the l2 state provider agrees with a specified start and end
// history commitment for a type of edge under a specified assertion challenge. It returns an agreement struct
// which informs the caller whether (a) we agree with the start commitment, and whether (b) the edge is honest, meaning
// that we also agree with the end commitment.
func (p *HistoryCommitmentProvider) AgreesWithHistoryCommitment(
	ctx context.Context,
	challengeLevel protocol.ChallengeLevel,
	historyCommitMetadata *HistoryCommitmentRequest,
	commit History,
) (bool, error) {
	var localCommit commitments.History
	var err error
	switch challengeLevel {
	case protocol.NewBlockChallengeLevel():
		localCommit, err = p.HistoryCommitment(
			ctx,
			&HistoryCommitmentRequest{
				WasmModuleRoot:              historyCommitMetadata.WasmModuleRoot,
				FromBatch:                   historyCommitMetadata.FromBatch,
				ToBatch:                     historyCommitMetadata.ToBatch,
				UpperChallengeOriginHeights: []Height{},
				FromHeight:                  historyCommitMetadata.FromHeight,
				UpToHeight:                  option.Some[Height](historyCommitMetadata.FromHeight + Height(commit.Height)),
			},
		)
		if err != nil {
			return false, err
		}
	default:
		localCommit, err = p.HistoryCommitment(
			ctx,
			&HistoryCommitmentRequest{
				WasmModuleRoot:              historyCommitMetadata.WasmModuleRoot,
				FromBatch:                   historyCommitMetadata.FromBatch,
				ToBatch:                     historyCommitMetadata.ToBatch,
				UpperChallengeOriginHeights: historyCommitMetadata.UpperChallengeOriginHeights,
				FromHeight:                  0,
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
// that the history commitment for height N is a Merkle prefix of the commitment at height M.
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
// is a prefix of the history commitment at height 24. Each index in the []Height{} slice
// represents a challenge level. For example, this call wants us to use the history commitment
// at the very first challenge level, over blocks.
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
	// If no upToHeight is provided, we want to use the max number of leaves in our computation.
	lowCommitmentNumLeaves := uint64(prefixHeight + 1)
	var highCommitmentNumLeaves uint64
	if req.UpToHeight.IsNone() {
		highCommitmentNumLeaves = uint64(len(leaves))
	} else {
		// Else if it is provided, we expect the number of leaves to be the difference
		// between the to and from height + 1.
		upTo := req.UpToHeight.Unwrap()
		if upTo < req.FromHeight {
			return nil, fmt.Errorf("invalid range: end %d was < start %d", upTo, req.FromHeight)
		}
		highCommitmentNumLeaves = uint64(upTo) - uint64(req.FromHeight) + 1
	}

	// Validate we are within bounds of the leaves slice.
	if highCommitmentNumLeaves > uint64(len(leaves)) {
		return nil, fmt.Errorf("high prefix size out of bounds, got %d, leaves length %d", highCommitmentNumLeaves, len(leaves))
	}

	// Validate low vs high commitment.
	if lowCommitmentNumLeaves > highCommitmentNumLeaves {
		return nil, fmt.Errorf("low prefix size %d was greater than high prefix size %d", lowCommitmentNumLeaves, highCommitmentNumLeaves)
	}

	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(leaves[:lowCommitmentNumLeaves])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		lowCommitmentNumLeaves,
		prefixExpansion,
		leaves[lowCommitmentNumLeaves:highCommitmentNumLeaves],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	bigCommit, err := commitments.New(leaves[:highCommitmentNumLeaves])
	if err != nil {
		return nil, err
	}

	prefixCommit, err := commitments.New(leaves[:lowCommitmentNumLeaves])
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, lowCommitmentNumLeaves)
	onlyProof := prefixProof[numRead:]

	// We verify our prefix proof before an onchain submission as an extra safety-check.
	if err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      prefixCommit.Merkle,
		PreSize:      lowCommitmentNumLeaves,
		PostRoot:     bigCommit.Merkle,
		PostSize:     highCommitmentNumLeaves,
		PreExpansion: prefixExpansion,
		PrefixProof:  onlyProof,
	}); err != nil {
		return nil, fmt.Errorf("could not verify prefix proof locally: %w", err)
	}
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
}

func (p *HistoryCommitmentProvider) OneStepProofData(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	fromBatch,
	toBatch Batch,
	startHeights []Height,
	fromHeight,
	upToHeight Height,
) (*protocol.OneStepData, []common.Hash, []common.Hash, error) {
	// Start heights must reflect at least two challenge levels to produce one step proofs.
	if len(startHeights) < 1 {
		return nil, nil, nil, fmt.Errorf("upper challenge origin heights must have at least length 1, got %d", len(startHeights))
	}
	endCommit, err := p.HistoryCommitment(
		ctx,
		&HistoryCommitmentRequest{
			WasmModuleRoot:              wasmModuleRoot,
			FromBatch:                   fromBatch,
			ToBatch:                     toBatch,
			UpperChallengeOriginHeights: startHeights,
			FromHeight:                  fromHeight,
			UpToHeight:                  option.Some(upToHeight + 1),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	startCommit, err := p.HistoryCommitment(
		ctx,
		&HistoryCommitmentRequest{
			WasmModuleRoot:              wasmModuleRoot,
			FromBatch:                   fromBatch,
			ToBatch:                     toBatch,
			UpperChallengeOriginHeights: startHeights,
			FromHeight:                  fromHeight,
			UpToHeight:                  option.Some(upToHeight),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// Compute the exact start point of where we need to execute
	// the machine from the inputs, and figure out, in what increments, we need to do so.
	machineIndex, err := p.computeMachineStartIndex(startHeights, fromHeight)
	if err != nil {
		return nil, nil, nil, err
	}
	machineIndex += OpcodeIndex(upToHeight)

	osp, err := p.proofCollector.CollectProof(ctx, wasmModuleRoot, fromBatch, startHeights[0], machineIndex)
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
// for a leaf commitment at each challenge level is a constant, so we can determine
// the desired challenge level from the input params and compute the total
// from there.
func (p *HistoryCommitmentProvider) computeRequiredNumberOfHashes(
	challengeLevel uint64,
	fromHeight Height,
	upToHeight option.Option[Height],
) (uint64, error) {
	maxHeightForLevel, err := p.leafHeightAtChallengeLevel(challengeLevel)
	if err != nil {
		return 0, err
	}

	// Get the requested history commitment height we need at our
	// desired challenge level.
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
	if end < fromHeight {
		return 0, fmt.Errorf("invalid range: end %d was < start %d", end, fromHeight)
	}
	// The number of hashes is the difference between the start and end
	// requested heights, plus 1.
	return uint64(end-fromHeight) + 1, nil
}

// Figures out the actual opcode index we should move the machine to
// when we compute the history commitment. As there are different levels of challenge
// granularity, we have to do some math to figure out the correct index.
//
// Take, for example, that we have 4 challenge kinds:
//
// block_challenge    => over a range of L2 message hashes
// megastep_challenge => over ranges of 1048576 (2^20) opcodes at a time.
// kilostep_challenge => over ranges of 1024 (2^10) opcodes at a time
// step_challenge     => over a range of individual WASM opcodes
//
// We only directly step through WASM machines when in a subchallenge (starting at megastep),
// so we can ignore block challenges for this calculation.
//
// Let's say we want to figure out the machine start opcode index for the following inputs:
//
// megastep=4, kilostep=5, step=10
//
// We can compute the opcode index using the following algorithm for the example above.
//
//	  4 * (1048576)
//	+ 5 * (1024)
//	+ 10
//	= 4,199,434
//
// This generalizes for any number of subchallenge levels into the algorithm below.
// It works by taking the sum of (each input * product of all challenge level height constants beneath its level).
// This means we need to start executing our machine exactly at opcode index 4,199,434.
func (p *HistoryCommitmentProvider) computeMachineStartIndex(
	upperChallengeOriginHeights validatedStartHeights,
	fromHeight Height,
) (OpcodeIndex, error) {
	// For the block challenge level, the machine start opcode index is always 0.
	if len(upperChallengeOriginHeights) == 0 {
		return 0, nil
	}
	// The first position in the start heights slice is the block challenge level, which is over ranges of L2 messages
	// and not over individual opcodes. We ignore this level and start at the next level when it comes to dealing with
	// machines.
	heights := upperChallengeOriginHeights[1:]
	heights = append(heights, fromHeight)
	leafHeights := p.challengeLeafHeights[1:]

	// Next, we compute the opcode index. We use big ints to make sure we do not overflow uint64
	// as this computation depends on external user inputs.
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

// Computes the number of individual opcodes we need to step through a machine at a time.
// Each challenge level has a different amount of ranges of opcodes, so the overall step size can be computed
// as a multiplication of all the next challenge levels needed.
//
// As an example, this function helps answer questions such as: "How many individual opcodes are there in a single step of a
// Megastep challenge?"
func (p *HistoryCommitmentProvider) computeStepSize(challengeLevel uint64) (StepSize, error) {
	// The last challenge level is over individual opcodes, so the step size is always 1 opcode at a time.
	if challengeLevel+1 == p.numberOfChallengeLevels() {
		return 1, nil
	}
	// Otherwise, it is the multiplication of all the challenge leaf heights at the next
	// challenge levels.
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
	// Length cannot be greater than the total number of challenge levels in the protocol - 1.
	if len(upperChallengeOriginHeights) > len(p.challengeLeafHeights)-1 {
		return nil, fmt.Errorf(
			"challenge level %d is out of range for challenge leaf heights %v",
			len(upperChallengeOriginHeights),
			p.challengeLeafHeights,
		)
	}
	return upperChallengeOriginHeights, nil
}

// A caller specifies a request for a history commitment at challenge level N. It specifies a list of
// heights at which to compute the history commitment at each challenge level on the way to level N
// as a list of heights, where each position represents a challenge level.
// The length of this list cannot be greater than the total number of challenge levels in the protocol.
// Takes in an input type that has already been validated for correctness.
func deepestRequestedChallengeLevel(requestedHeights validatedStartHeights) uint64 {
	return uint64(len(requestedHeights))
}

// Gets the required leaf height at a specified challenge level. This is a protocol constant.
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
