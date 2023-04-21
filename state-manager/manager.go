package statemanager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/execution"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	prefixproofs "github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Defines the ABI encoding structure for submission of prefix proofs to the protocol contracts
var (
	b32Arr, _ = abi.NewType("bytes32[]", "", nil)
	// ProofArgs for submission to the protocol.
	ProofArgs = abi.Arguments{
		{Type: b32Arr, Name: "prefixExpansion"},
		{Type: b32Arr, Name: "prefixProof"},
	}
)

// AssertionToCreate defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type AssertionToCreate struct {
	State         *protocol.ExecutionState
	InboxMaxCount *big.Int
}

type Manager interface {
	// Produces the latest assertion data to post to L1 from the local state manager's
	// perspective based on a parent assertion height.
	LatestAssertionCreationData(ctx context.Context) (*AssertionToCreate, error)
	AssertionExecutionState(ctx context.Context, assertionStateHash common.Hash) (*protocol.ExecutionState, error)
	// Checks if a state commitment corresponds to data the state manager has locally.
	HasStateCommitment(ctx context.Context, blockChallengeCommitment util.StateCommitment) bool
	// Produces a block challenge history commitment up to and including a certain height.
	HistoryCommitmentUpTo(ctx context.Context, blockChallengeHeight uint64) (util.HistoryCommitment, error)
	// Produces a block challenge history commitment in a certain inclusive block range,
	// but padding states with duplicates after the first state with a
	// batch count of at least the specified max.
	HistoryCommitmentUpToBatch(
		ctx context.Context,
		blockStart,
		blockEnd,
		batchCount uint64,
	) (util.HistoryCommitment, error)
	// Produces a big step history commitment for all big steps within block
	// challenge heights H to H+1.
	BigStepLeafCommitment(
		ctx context.Context,
		fromBlockChallengeHeight,
		toBlockChallengeHeight uint64,
	) (util.HistoryCommitment, error)
	// Produces a big step history commitment from big step 0 to N within block
	// challenge heights A and B where B = A + 1.
	BigStepCommitmentUpTo(
		ctx context.Context,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		toBigStep uint64,
	) (util.HistoryCommitment, error)
	// Produces a small step history commitment for all small steps between
	// big steps S to S+1 within block challenge heights H to H+1.
	SmallStepLeafCommitment(
		ctx context.Context,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep uint64,
	) (util.HistoryCommitment, error)
	// Produces a small step history commitment from small step 0 to N between
	// big steps S to S+1 within block challenge heights H to H+1.
	SmallStepCommitmentUpTo(
		ctx context.Context,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		toSmallStep uint64,
	) (util.HistoryCommitment, error)
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
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep uint64,
	) ([]byte, error)
	// Produces a small step prefix proof from height A to B for big step S to S+1 and
	// block challenge height heights H to H+1.
	SmallStepPrefixProof(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight,
		fromBigStep,
		toBigStep,
		fromSmallStep,
		toSmallStep uint64,
	) ([]byte, error)
	OneStepProofData(
		ctx context.Context,
		parentAssertionStateHash common.Hash,
		assertionCreationInfo *protocol.AssertionCreatedInfo,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		fromSmallStep,
		toSmallStep uint64,
	) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error)
}

// Simulated defines a very naive state manager that is initialized from a list of predetermined
// state roots. It can produce state and history commitments from those roots.
type Simulated struct {
	stateRoots                []common.Hash
	executionStates           []*protocol.ExecutionState
	inboxMaxCounts            []*big.Int
	maxWavmOpcodes            uint64
	numOpcodesPerBigStep      uint64
	bigStepDivergenceHeight   uint64
	smallStepDivergenceHeight uint64
	malicious                 bool
}

// New simulated manager from a list of predefined state roots, useful for tests and simulations.
func New(stateRoots []common.Hash, opts ...Opt) (*Simulated, error) {
	if len(stateRoots) == 0 {
		return nil, errors.New("no state roots provided")
	}
	s := &Simulated{stateRoots: stateRoots}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

type Opt func(*Simulated)

func WithMaxWavmOpcodesPerBlock(maxOpcodes uint64) Opt {
	return func(s *Simulated) {
		s.maxWavmOpcodes = maxOpcodes
	}
}

func WithNumOpcodesPerBigStep(numOpcodes uint64) Opt {
	return func(s *Simulated) {
		s.numOpcodesPerBigStep = numOpcodes
	}
}

func WithBigStepStateDivergenceHeight(divergenceHeight uint64) Opt {
	return func(s *Simulated) {
		s.bigStepDivergenceHeight = divergenceHeight
	}
}

// The divergence height is relative to the last non-diverging big step.
// E.g. if the big step divergence is set to 2, there are 32 small steps per big steps,
// and the small step divergence is set to 10, then small steps would start diverging at step 42.
// That's because we need to make a divergence before big step 2, but after big step 1.
// We put the divergence 10 small steps into that big step block, as specified by this parameter.
func WithSmallStepStateDivergenceHeight(divergenceHeight uint64) Opt {
	return func(s *Simulated) {
		s.smallStepDivergenceHeight = divergenceHeight
	}
}

func WithMaliciousIntent() Opt {
	return func(s *Simulated) {
		s.malicious = true
	}
}

// NewWithAssertionStates creates a simulated state manager from a list of predefined state roots for
// the top-level assertion chain, useful for tests and simulation purposes in block challenges.
// This also allows for specifying the honest states for big and small step subchallenges along
// with the point at which the state manager should diverge from the honest computation.
func NewWithAssertionStates(
	assertionChainExecutionStates []*protocol.ExecutionState,
	inboxMaxCounts []*big.Int,
	opts ...Opt,
) (*Simulated, error) {
	if len(assertionChainExecutionStates) == 0 {
		return nil, errors.New("must have execution states")
	}
	if len(assertionChainExecutionStates) != len(inboxMaxCounts) {
		return nil, fmt.Errorf(
			"number of exec states %d must match number of inbox max counts %d",
			len(assertionChainExecutionStates),
			len(inboxMaxCounts),
		)
	}
	stateRoots := make([]common.Hash, len(assertionChainExecutionStates))
	var lastBatch uint64 = math.MaxUint64
	var lastPosInBatch uint64 = math.MaxUint64
	for i := 0; i < len(stateRoots); i++ {
		state := assertionChainExecutionStates[i]
		if state.GlobalState.Batch == lastBatch && state.GlobalState.PosInBatch == lastPosInBatch {
			return nil, fmt.Errorf("execution states %v and %v have the same batch %v and position in batch %v", i-1, i, lastBatch, lastPosInBatch)
		}
		lastBatch = state.GlobalState.Batch
		lastPosInBatch = state.GlobalState.PosInBatch
		stateRoots[i] = protocol.ComputeStateHash(state, inboxMaxCounts[i])
	}
	s := &Simulated{
		stateRoots:      stateRoots,
		executionStates: assertionChainExecutionStates,
		inboxMaxCounts:  inboxMaxCounts,
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

// LatestAssertionCreationData gets the state commitment corresponding to the last, local state root the manager has
// and a pre-state based on a height of the previous assertion the validator should build upon.
func (s *Simulated) LatestAssertionCreationData(_ context.Context) (*AssertionToCreate, error) {
	lastState := s.executionStates[len(s.executionStates)-1]
	return &AssertionToCreate{
		State:         lastState,
		InboxMaxCount: big.NewInt(1), // TODO: this should be s.inboxMaxCounts[len(s.inboxMaxCounts)-1] but that breaks other stuff
	}, nil
}

// HasStateCommitment checks if a state commitment is found in our local list of state roots.
func (s *Simulated) HasStateCommitment(_ context.Context, commitment util.StateCommitment) bool {
	for _, r := range s.stateRoots {
		if r == commitment.StateRoot {
			return true
		}
	}
	return false
}

func (s *Simulated) HistoryCommitmentUpTo(_ context.Context, blockChallengeHeight uint64) (util.HistoryCommitment, error) {
	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	size := blockChallengeHeight + 1
	return util.NewHistoryCommitment(
		blockChallengeHeight,
		s.stateRoots[:size],
	)
}

func (s *Simulated) statesUpTo(blockStart, blockEnd, nextBatchCount uint64) ([]common.Hash, error) {
	if blockEnd < blockStart {
		return nil, fmt.Errorf("end block %v is less than start block %v", blockEnd, blockStart)
	}
	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	desiredStatesLen := int(blockEnd - blockStart + 1)
	var states []common.Hash
	var lastState common.Hash
	for i := blockStart; i <= blockEnd; i++ {
		if i >= uint64(len(s.stateRoots)) {
			break
		}
		state := s.stateRoots[i]
		states = append(states, state)
		lastState = state
		if len(s.executionStates) == 0 {
			// should only happen in tests
			continue
		}
		gs := s.executionStates[i].GlobalState
		if gs.Batch >= nextBatchCount {
			if gs.Batch > nextBatchCount || gs.PosInBatch > 0 {
				return nil, fmt.Errorf("overran next batch count %v with global state batch %v position %v", nextBatchCount, gs.Batch, gs.PosInBatch)
			}
			break
		}
	}
	for len(states) < desiredStatesLen {
		states = append(states, lastState)
	}
	return states, nil
}

func (s *Simulated) HistoryCommitmentUpToBatch(_ context.Context, blockStart, blockEnd, nextBatchCount uint64) (util.HistoryCommitment, error) {
	states, err := s.statesUpTo(blockStart, blockEnd, nextBatchCount)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.NewHistoryCommitment(
		blockEnd-blockStart,
		states,
	)
}

func (s *Simulated) BigStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	// Number of big steps between assertion heights A and B will be
	// fixed in this simulated state manager. It is simply the max number of opcodes
	// per block divided by the size of a big step.
	numBigSteps := s.maxWavmOpcodes / s.numOpcodesPerBigStep
	return s.BigStepCommitmentUpTo(
		ctx,
		fromAssertionHeight,
		toAssertionHeight,
		numBigSteps,
	)
}

func (s *Simulated) BigStepCommitmentUpTo(
	_ context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	if fromAssertionHeight+1 != toAssertionHeight {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromAssertionHeight,
			toAssertionHeight,
		)
	}
	engine, err := s.setupEngine(fromAssertionHeight)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	if engine.NumBigSteps() < toBigStep {
		return util.HistoryCommitment{}, errors.New("not enough big steps")
	}
	leaves, err := s.intermediateBigStepLeaves(
		fromAssertionHeight,
		toAssertionHeight,
		0, // from big step.
		toBigStep,
		engine,
	)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.NewHistoryCommitment(toBigStep, leaves)
}

func (s *Simulated) bigStepShouldDiverge(step uint64) bool {
	// Diverge if:
	return s.bigStepDivergenceHeight != 0 && // diverging is enabled, and
		step >= s.bigStepDivergenceHeight && // we're past the divergence point, and
		step != 0 && // we're not at the beginning of a block (otherwise the block->big step subchallenge wouldn't work), and
		step != s.maxWavmOpcodes/s.numOpcodesPerBigStep // we're not at the end of a block (for the same reason)
}

func (s *Simulated) intermediateBigStepLeaves(
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	toBigStep uint64,
	engine execution.EngineAtBlock,
) ([]common.Hash, error) {
	leaves := make([]common.Hash, 0)
	// Up to and including the specified step.
	for i := fromBigStep; i <= toBigStep; i++ {
		start, err := engine.StateAfterBigSteps(i)
		if err != nil {
			return nil, err
		}
		var hash common.Hash

		// For testing purposes, if we want to diverge from the honest
		// hashes starting at a specified hash.
		if s.bigStepShouldDiverge(i) {
			hash = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d:%d:%d", i*s.numOpcodesPerBigStep, fromBlockChallengeHeight, toBlockChallengeHeight)))
		} else {
			hash = start.Hash()
		}
		leaves = append(leaves, hash)
	}
	return leaves, nil
}

func (s *Simulated) SmallStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	fromBigStep,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	return s.SmallStepCommitmentUpTo(
		ctx,
		fromAssertionHeight,
		toAssertionHeight,
		fromBigStep,
		toBigStep,
		s.numOpcodesPerBigStep,
	)
}

func (s *Simulated) SmallStepCommitmentUpTo(
	_ context.Context,
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	toBigStep,
	toSmallStep uint64,
) (util.HistoryCommitment, error) {
	if fromBlockChallengeHeight+1 != toBlockChallengeHeight {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromBlockChallengeHeight,
			toBlockChallengeHeight,
		)
	}
	if fromBigStep+1 != toBigStep {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromBigStep,
			toBigStep,
		)
	}
	engine, err := s.setupEngine(fromBlockChallengeHeight)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	if engine.NumOpcodes() < toSmallStep {
		return util.HistoryCommitment{}, fmt.Errorf("not enough small steps: %d < %d", engine.NumOpcodes(), toSmallStep)
	}

	fromSmall := (fromBigStep * s.numOpcodesPerBigStep)
	toSmall := fromSmall + toSmallStep
	leaves, err := s.intermediateSmallStepLeaves(
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromSmall,
		toSmall,
		engine,
	)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.NewHistoryCommitment(toSmallStep, leaves)
}

func (s *Simulated) intermediateSmallStepLeaves(
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromSmallStep,
	toSmallStep uint64,
	engine execution.EngineAtBlock,
) ([]common.Hash, error) {
	leaves := make([]common.Hash, 0)
	// Up to and including the specified step.
	var divergeAt uint64
	if s.bigStepDivergenceHeight > 0 {
		divergeAt = s.numOpcodesPerBigStep*(s.bigStepDivergenceHeight-1) + s.smallStepDivergenceHeight
	}
	for i := fromSmallStep; i <= toSmallStep; i++ {
		start, err := engine.StateAfterSmallSteps(i)
		if err != nil {
			return nil, err
		}
		var hash common.Hash

		// For testing purposes, if we want to diverge from the honest
		// hashes starting at a specified hash.
		var shouldDiverge bool
		if i%s.numOpcodesPerBigStep == 0 {
			// If we're at a big step point, maintain compatibility so big step -> small step subchallenges work
			shouldDiverge = s.bigStepShouldDiverge(i / s.numOpcodesPerBigStep)
		} else {
			shouldDiverge = divergeAt != 0 && i >= divergeAt
		}
		if shouldDiverge {
			hash = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d:%d:%d", i, fromBlockChallengeHeight, toBlockChallengeHeight)))
		} else {
			hash = start.Hash()
		}
		leaves = append(leaves, hash)
	}
	return leaves, nil
}

// Like abi.NewType but panics if it fails for use in constants
func newStaticType(t string, internalType string, components []abi.ArgumentMarshaling) abi.Type {
	ty, err := abi.NewType(t, internalType, components)
	if err != nil {
		panic(err)
	}
	return ty
}

func (s *Simulated) AssertionExecutionState(
	_ context.Context,
	assertionStateHash common.Hash,
) (*protocol.ExecutionState, error) {
	var stateRootIndex int
	var found bool
	for i, r := range s.stateRoots {
		if r == assertionStateHash {
			stateRootIndex = i
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("assertion state hash %#x not found locally", assertionStateHash)
	}
	return s.executionStates[stateRootIndex], nil
}

var bytes32Type = newStaticType("bytes32", "", nil)
var uint64Type = newStaticType("uint64", "", nil)
var uint8Type = newStaticType("uint8", "", nil)

var WasmModuleProofAbi = abi.Arguments{
	{
		Name: "lastHash",
		Type: bytes32Type,
	},
	{
		Name: "assertionExecHash",
		Type: bytes32Type,
	},
	{
		Name: "inboxAcc",
		Type: bytes32Type,
	},
}

var ExecutionStateAbi = abi.Arguments{
	{
		Name: "b1",
		Type: bytes32Type,
	},
	{
		Name: "b2",
		Type: bytes32Type,
	},
	{
		Name: "u1",
		Type: uint64Type,
	},
	{
		Name: "u2",
		Type: uint64Type,
	},
	{
		Name: "status",
		Type: uint8Type,
	},
}

func (s *Simulated) OneStepProofData(
	ctx context.Context,
	assertionStateHash common.Hash,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	toBigStep,
	fromSmallStep,
	toSmallStep uint64,
) (data *protocol.OneStepData, startLeafInclusionProof, endLeafInclusionProof []common.Hash, err error) {
	assertionExecutionState, getErr := s.AssertionExecutionState(ctx, assertionStateHash)
	if getErr != nil {
		err = getErr
		return
	}
	execState := assertionExecutionState.AsSolidityStruct()
	inboxMaxCountProof, packErr := ExecutionStateAbi.Pack(
		execState.GlobalState.Bytes32Vals[0],
		execState.GlobalState.Bytes32Vals[1],
		execState.GlobalState.U64Vals[0],
		execState.GlobalState.U64Vals[1],
		execState.MachineStatus,
	)
	if packErr != nil {
		err = packErr
		return
	}

	wasmModuleRootProof, packErr := WasmModuleProofAbi.Pack(
		assertionCreationInfo.ParentAssertionHash,
		assertionCreationInfo.ExecutionHash,
		assertionCreationInfo.AfterInboxBatchAcc,
	)
	if packErr != nil {
		err = packErr
		return
	}
	startCommit, commitErr := s.SmallStepCommitmentUpTo(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		fromSmallStep,
	)
	if commitErr != nil {
		err = commitErr
		return
	}
	endCommit, commitErr := s.SmallStepCommitmentUpTo(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		toSmallStep,
	)
	if commitErr != nil {
		err = commitErr
		return
	}
	data = &protocol.OneStepData{
		BeforeHash:             startCommit.LastLeaf,
		Proof:                  make([]byte, 0),
		InboxMsgCountSeen:      assertionCreationInfo.InboxMaxCount,
		InboxMsgCountSeenProof: inboxMaxCountProof,
		WasmModuleRoot:         assertionCreationInfo.WasmModuleRoot,
		WasmModuleRootProof:    wasmModuleRootProof,
	}
	if !s.malicious {
		// Only honest validators can produce a valid one step proof.
		data.Proof = endCommit.LastLeaf[:]
	}
	startLeafInclusionProof = startCommit.LastLeafProof
	endLeafInclusionProof = endCommit.LastLeafProof
	return
}

func (s *Simulated) prefixProofImpl(_ context.Context, start, lo, hi, batchCount uint64) ([]byte, error) {
	states, err := s.statesUpTo(start, hi, batchCount)
	if err != nil {
		return nil, err
	}
	loSize := lo + 1 - start
	hiSize := hi + 1 - start
	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(states[:loSize])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		loSize,
		prefixExpansion,
		states[loSize:hiSize],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, loSize)
	onlyProof := prefixProof[numRead:]
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
}

func (s *Simulated) PrefixProof(ctx context.Context, lo, hi uint64) ([]byte, error) {
	return s.prefixProofImpl(ctx, 0, lo, hi, math.MaxUint64)
}

func (s *Simulated) PrefixProofUpToBatch(ctx context.Context, start, lo, hi, batchCount uint64) ([]byte, error) {
	return s.prefixProofImpl(ctx, start, lo, hi, batchCount)
}

func (s *Simulated) BigStepPrefixProof(
	_ context.Context,
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	toBigStep uint64,
) ([]byte, error) {
	if fromBlockChallengeHeight+1 != toBlockChallengeHeight {
		return nil, fmt.Errorf(
			"fromAssertionHeight=%d is not 1 height apart from toAssertionHeight=%d",
			fromBlockChallengeHeight,
			toBlockChallengeHeight,
		)
	}
	engine, err := s.setupEngine(fromBlockChallengeHeight)
	if err != nil {
		return nil, err
	}
	if engine.NumBigSteps() < toBigStep {
		return nil, errors.New("wrong number of big steps")
	}
	return s.bigStepPrefixProofCalculation(
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		engine,
	)
}

func (s *Simulated) bigStepPrefixProofCalculation(
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	toBigStep uint64,
	engine execution.EngineAtBlock,
) ([]byte, error) {
	loSize := fromBigStep + 1
	hiSize := toBigStep + 1
	prefixLeaves, err := s.intermediateBigStepLeaves(
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		0,
		toBigStep,
		engine,
	)
	if err != nil {
		return nil, err
	}
	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(prefixLeaves[:loSize])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		loSize,
		prefixExpansion,
		prefixLeaves[loSize:hiSize],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, loSize)
	onlyProof := prefixProof[numRead:]
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
}

func (s *Simulated) SmallStepPrefixProof(
	_ context.Context,
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	toBigStep,
	fromSmallStep,
	toSmallStep uint64,
) ([]byte, error) {
	if fromBlockChallengeHeight+1 != toBlockChallengeHeight {
		return nil, fmt.Errorf(
			"fromAssertionHeight=%d is not 1 height apart from toAssertionHeight=%d",
			fromBlockChallengeHeight,
			toBlockChallengeHeight,
		)
	}
	if fromBigStep+1 != toBigStep {
		return nil, fmt.Errorf(
			"fromBigStep=%d is not 1 height apart from toBigStep=%d",
			fromBigStep,
			toBigStep,
		)
	}
	engine, err := s.setupEngine(fromBlockChallengeHeight)
	if err != nil {
		return nil, err
	}
	if engine.NumOpcodes() < toSmallStep {
		return nil, errors.New("wrong number of opcodes")
	}
	return s.smallStepPrefixProofCalculation(
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		fromSmallStep,
		toSmallStep,
		engine,
	)
}

func (s *Simulated) setupEngine(fromHeight uint64) (*execution.Engine, error) {
	machineCfg := execution.DefaultMachineConfig()
	if s.maxWavmOpcodes > 0 {
		machineCfg.MaxInstructionsPerBlock = s.maxWavmOpcodes
	}
	if s.numOpcodesPerBigStep > 0 {
		machineCfg.BigStepSize = s.numOpcodesPerBigStep
	}
	return execution.NewExecutionEngine(
		machineCfg,
		s.stateRoots[fromHeight],
		s.stateRoots[fromHeight+1],
	)
}

func (s *Simulated) smallStepPrefixProofCalculation(
	fromBlockChallengeHeight,
	toBlockChallengeHeight,
	fromBigStep,
	fromSmallStep,
	toSmallStep uint64,
	engine execution.EngineAtBlock,
) ([]byte, error) {
	fromSmall := (fromBigStep * s.numOpcodesPerBigStep)
	toSmall := fromSmall + toSmallStep
	prefixLeaves, err := s.intermediateSmallStepLeaves(
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromSmall,
		toSmall,
		engine,
	)
	if err != nil {
		return nil, err
	}
	loSize := fromSmallStep + 1
	hiSize := toSmallStep + 1
	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(prefixLeaves[:loSize])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		loSize,
		prefixExpansion,
		prefixLeaves[loSize:hiSize],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, loSize)
	onlyProof := prefixProof[numRead:]
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
}
