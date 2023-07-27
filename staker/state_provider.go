package staker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	prefixproofs "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/prefix-proofs"

	"github.com/offchainlabs/nitro/arbutil"
	challengecache "github.com/offchainlabs/nitro/staker/challenge-cache"
	"github.com/offchainlabs/nitro/validator"
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

var ErrChainCatchingUp = errors.New("chain catching up")

type StateManager struct {
	validator            *StatelessBlockValidator
	blockValidator       *BlockValidator
	numOpcodesPerBigStep uint64
	maxWavmOpcodes       uint64
	historyCache         challengecache.HistoryCommitmentCacher
}

func NewStateManager(val *StatelessBlockValidator, blockValidator *BlockValidator, numOpcodesPerBigStep uint64, maxWavmOpcodes uint64, cacheBaseDir string) (*StateManager, error) {
	historyCache := challengecache.New(cacheBaseDir)
	return &StateManager{
		validator:            val,
		blockValidator:       blockValidator,
		numOpcodesPerBigStep: numOpcodesPerBigStep,
		maxWavmOpcodes:       maxWavmOpcodes,
		historyCache:         historyCache,
	}, nil
}

// ExecutionStateMsgCount If the state manager locally has this validated execution state.
// Returns ErrNoExecutionState if not found, or ErrChainCatchingUp if not yet
// validated / syncing.
func (s *StateManager) ExecutionStateMsgCount(ctx context.Context, state *protocol.ExecutionState) (uint64, error) {
	// if state.MachineStatus != protocol.MachineStatusRunning {
	// 	return 0, errors.New("state is not running")
	// }
	messageCount, err := s.validator.inboxTracker.GetBatchMessageCount(state.GlobalState.Batch)
	if err != nil {
		return 0, err
	}
	validatedExecutionState, err := s.executionStateAtMessageNumberImpl(ctx, uint64(messageCount)-1)
	if err != nil {
		return 0, err
	}
	if validatedExecutionState.GlobalState.Batch < state.GlobalState.Batch ||
		(validatedExecutionState.GlobalState.Batch == state.GlobalState.Batch &&
			validatedExecutionState.GlobalState.PosInBatch < state.GlobalState.PosInBatch) {
		return 0, ErrChainCatchingUp
	}
	var prevBatchMsgCount arbutil.MessageIndex
	if state.GlobalState.Batch > 0 {
		var err error
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(state.GlobalState.Batch - 1)
		if err != nil {
			return 0, err
		}
	}
	count := prevBatchMsgCount
	if state.GlobalState.PosInBatch > 0 {
		count += arbutil.MessageIndex(state.GlobalState.PosInBatch)
	}
	res, err := s.validator.streamer.ResultAtCount(count)
	if err != nil {
		return 0, err
	}
	if res.BlockHash != state.GlobalState.BlockHash || res.SendRoot != state.GlobalState.SendRoot {
		return 0, l2stateprovider.ErrNoExecutionState
	}
	return uint64(count), nil
}

// ExecutionStateAtMessageNumber Produces the l2 state to assert at the message number specified.
// Makes sure that PosInBatch is always 0
func (s *StateManager) ExecutionStateAtMessageNumber(ctx context.Context, messageNumber uint64) (*protocol.ExecutionState, error) {
	executionState, err := s.executionStateAtMessageNumberImpl(ctx, messageNumber)
	if err != nil {
		return nil, err
	}
	if executionState.GlobalState.PosInBatch != 0 {
		executionState.GlobalState.Batch++
		executionState.GlobalState.PosInBatch = 0
	}
	return executionState, nil
}

func (s *StateManager) executionStateAtMessageNumberImpl(ctx context.Context, messageNumber uint64) (*protocol.ExecutionState, error) {
	batch, err := s.findBatchAfterMessageCount(arbutil.MessageIndex(messageNumber))
	if err != nil {
		return &protocol.ExecutionState{}, err
	}
	batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
	if err != nil {
		return &protocol.ExecutionState{}, err
	}
	if batchMsgCount <= arbutil.MessageIndex(messageNumber) {
		batch++
	}
	globalState, err := s.getInfoAtMessageCountAndBatch(arbutil.MessageIndex(messageNumber), batch)
	if err != nil {
		return &protocol.ExecutionState{}, err
	}
	return &protocol.ExecutionState{
		GlobalState:   protocol.GoGlobalState(globalState),
		MachineStatus: protocol.MachineStatusFinished, // TODO: Why hardcode?
	}, nil
}

// HistoryCommitmentAtMessage Produces a block history commitment of messageCount.
func (s *StateManager) HistoryCommitmentAtMessage(ctx context.Context, messageNumber uint64) (commitments.History, error) {
	batch, err := s.findBatchAfterMessageCount(arbutil.MessageIndex(messageNumber))
	if err != nil {
		return commitments.History{}, err
	}
	batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(messageNumber)
	if err != nil {
		return commitments.History{}, err
	}
	if batchMsgCount <= arbutil.MessageIndex(messageNumber) {
		batch++
	}
	stateRoot, err := s.getHashAtMessageCountAndBatch(ctx, arbutil.MessageIndex(messageNumber), batch)
	if err != nil {
		return commitments.History{}, err
	}
	return commitments.New([]common.Hash{stateRoot})
}

// BigStepCommitmentUpTo Produces a big step history commitment from big step 0 to toBigStep within block
// challenge heights blockHeight and blockHeight+1.
func (s *StateManager) BigStepCommitmentUpTo(ctx context.Context, wasmModuleRoot common.Hash, messageNumber uint64, toBigStep uint64) (commitments.History, error) {
	result, err := s.intermediateBigStepLeaves(ctx, wasmModuleRoot, messageNumber, toBigStep)
	if err != nil {
		return commitments.History{}, err
	}
	fmt.Printf("Big step leaves, message %d, to big step %d, total %d\n", messageNumber, toBigStep, len(result))
	return commitments.New(result)
}

// SmallStepCommitmentUpTo Produces a small step history commitment from small step 0 to N between
// big steps bigStep to bigStep+1 within block challenge heights blockHeight to blockHeight+1.
func (s *StateManager) SmallStepCommitmentUpTo(ctx context.Context, wasmModuleRoot common.Hash, messageNumber uint64, bigStep uint64, toSmallStep uint64) (commitments.History, error) {
	result, err := s.intermediateSmallStepLeaves(ctx, wasmModuleRoot, messageNumber, bigStep, toSmallStep)
	if err != nil {
		return commitments.History{}, err
	}
	return commitments.New(result)
}

// HistoryCommitmentUpToBatch Produces a block challenge history commitment in a certain inclusive block range,
// but padding states with duplicates after the first state with a batch count of at least the specified max.
func (s *StateManager) HistoryCommitmentUpToBatch(ctx context.Context, messageNumberStart uint64, messageNumberEnd uint64, nextBatchCount uint64) (commitments.History, error) {
	stateRoots, err := s.statesUpTo(messageNumberStart, messageNumberEnd, nextBatchCount)
	if err != nil {
		return commitments.History{}, err
	}
	return commitments.New(stateRoots)
}

// BigStepLeafCommitment Produces a big step history commitment for all big steps within block
// challenge heights blockHeight to blockHeight+1.
func (s *StateManager) BigStepLeafCommitment(ctx context.Context, wasmModuleRoot common.Hash, messageNumber uint64) (commitments.History, error) {
	// Number of big steps between assertion heights A and B will be
	// fixed. It is simply the max number of opcodes
	// per block divided by the size of a big step.
	numBigSteps := s.maxWavmOpcodes / s.numOpcodesPerBigStep
	return s.BigStepCommitmentUpTo(ctx, wasmModuleRoot, messageNumber, numBigSteps)
}

// SmallStepLeafCommitment Produces a small step history commitment for all small steps between
// big steps bigStep to bigStep+1 within block challenge heights blockHeight to blockHeight+1.
func (s *StateManager) SmallStepLeafCommitment(ctx context.Context, wasmModuleRoot common.Hash, messageNumber uint64, bigStep uint64) (commitments.History, error) {
	return s.SmallStepCommitmentUpTo(
		ctx,
		wasmModuleRoot,
		messageNumber,
		bigStep,
		s.numOpcodesPerBigStep,
	)
}

// PrefixProofUpToBatch Produces a prefix proof in a block challenge from height A to B,
// but padding states with duplicates after the first state with a batch count of at least the specified max.
func (s *StateManager) PrefixProofUpToBatch(
	ctx context.Context,
	startHeight,
	fromMessageNumber,
	toMessageNumber,
	batchCount uint64,
) ([]byte, error) {
	if toMessageNumber > batchCount {
		return nil, errors.New("toMessageNumber should not be greater than batchCount")
	}
	states, err := s.statesUpTo(startHeight, toMessageNumber, batchCount)
	if err != nil {
		return nil, err
	}
	loSize := fromMessageNumber + 1 - startHeight
	hiSize := toMessageNumber + 1 - startHeight
	return s.getPrefixProof(loSize, hiSize, states)
}

// BigStepPrefixProof Produces a big step prefix proof from height A to B for heights fromBlockChallengeHeight to H+1
// within a block challenge.
func (s *StateManager) BigStepPrefixProof(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	messageNumber uint64,
	fromBigStep uint64,
	toBigStep uint64,
) ([]byte, error) {
	prefixLeaves, err := s.intermediateBigStepLeaves(ctx, wasmModuleRoot, messageNumber, toBigStep)
	if err != nil {
		return nil, err
	}
	loSize := fromBigStep + 1
	hiSize := toBigStep + 1
	return s.getPrefixProof(loSize, hiSize, prefixLeaves)
}

// SmallStepPrefixProof Produces a small step prefix proof from height A to B for big step S to S+1 and
// block challenge height heights H to H+1.
func (s *StateManager) SmallStepPrefixProof(ctx context.Context, wasmModuleRoot common.Hash, messageNumber uint64, bigStep uint64, fromSmallStep uint64, toSmallStep uint64) ([]byte, error) {
	prefixLeaves, err := s.intermediateSmallStepLeaves(ctx, wasmModuleRoot, messageNumber, bigStep, toSmallStep)
	if err != nil {
		return nil, err
	}
	loSize := fromSmallStep + 1
	hiSize := toSmallStep + 1
	return s.getPrefixProof(loSize, hiSize, prefixLeaves)
}

// Like abi.NewType but panics if it fails for use in constants
func newStaticType(t string, internalType string, components []abi.ArgumentMarshaling) abi.Type {
	ty, err := abi.NewType(t, internalType, components)
	if err != nil {
		panic(err)
	}
	return ty
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

func (s *StateManager) OneStepProofData(
	ctx context.Context,
	cfgSnapshot *l2stateprovider.ConfigSnapshot,
	postState rollupgen.ExecutionState,
	messageNumber,
	bigStep,
	smallStep uint64,
) (*protocol.OneStepData, []common.Hash, []common.Hash, error) {
	inboxMaxCountProof, err := ExecutionStateAbi.Pack(
		postState.GlobalState.Bytes32Vals[0],
		postState.GlobalState.Bytes32Vals[1],
		postState.GlobalState.U64Vals[0],
		postState.GlobalState.U64Vals[1],
		postState.MachineStatus,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	wasmModuleRootProof, err := WasmModuleProofAbi.Pack(
		cfgSnapshot.RequiredStake,
		cfgSnapshot.ChallengeManagerAddress,
		cfgSnapshot.ConfirmPeriodBlocks,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	startCommit, err := s.SmallStepCommitmentUpTo(
		ctx,
		cfgSnapshot.WasmModuleRoot,
		messageNumber,
		bigStep,
		smallStep,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	endCommit, err := s.SmallStepCommitmentUpTo(
		ctx,
		cfgSnapshot.WasmModuleRoot,
		messageNumber,
		bigStep,
		smallStep,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	step := bigStep*s.numOpcodesPerBigStep + smallStep

	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(messageNumber))
	if err != nil {
		return nil, nil, nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return nil, nil, nil, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(cfgSnapshot.WasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	oneStepProofPromise := execRun.GetProofAt(step)
	oneStepProof, err := oneStepProofPromise.Await(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	machineStepPromise := execRun.GetStepAt(step)
	machineStep, err := machineStepPromise.Await(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	beforeHash := machineStep.Hash
	if beforeHash != startCommit.LastLeaf {
		return nil, nil, nil, fmt.Errorf("machine executed to start step %v hash %v but expected %v", step, beforeHash, startCommit.LastLeaf)
	}

	machineStepPromise = execRun.GetStepAt(step + 1)
	machineStep, err = machineStepPromise.Await(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	afterHash := machineStep.Hash
	if afterHash != endCommit.LastLeaf {
		return nil, nil, nil, fmt.Errorf("machine executed to end step %v hash %v but expected %v", step+1, beforeHash, endCommit.LastLeaf)
	}

	data := &protocol.OneStepData{
		BeforeHash:             startCommit.LastLeaf,
		Proof:                  oneStepProof,
		InboxMsgCountSeen:      cfgSnapshot.InboxMaxCount,
		InboxMsgCountSeenProof: inboxMaxCountProof,
		WasmModuleRoot:         cfgSnapshot.WasmModuleRoot,
		WasmModuleRootProof:    wasmModuleRootProof,
	}
	return data, startCommit.LastLeafProof, endCommit.LastLeafProof, nil
}

func (s *StateManager) AgreesWithHistoryCommitment(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	parentAssertionAfterStateBatch uint64,
	assertionInboxMaxCount uint64,
	edgeType protocol.EdgeType,
	heights protocol.OriginHeights,
	history l2stateprovider.History,
) (bool, error) {
	var localCommit commitments.History
	var err error
	switch edgeType {
	case protocol.BlockChallengeEdge:
		localCommit, err = s.HistoryCommitmentUpToBatch(ctx, parentAssertionAfterStateBatch, parentAssertionAfterStateBatch+history.Height, assertionInboxMaxCount)
		if err != nil {
			return false, err
		}
	case protocol.BigStepChallengeEdge:
		localCommit, err = s.BigStepCommitmentUpTo(
			ctx,
			wasmModuleRoot,
			uint64(heights.BlockChallengeOriginHeight),
			history.Height,
		)
		if err != nil {
			return false, err
		}
	case protocol.SmallStepChallengeEdge:
		localCommit, err = s.SmallStepCommitmentUpTo(
			ctx,
			wasmModuleRoot,
			uint64(heights.BlockChallengeOriginHeight),
			uint64(heights.BigStepChallengeOriginHeight),
			history.Height,
		)
		if err != nil {
			return false, err
		}
	default:
		return false, errors.New("unsupported edge type")
	}
	return localCommit.Height == history.Height && localCommit.Merkle == history.MerkleRoot, nil
}

func (s *StateManager) getPrefixProof(loSize uint64, hiSize uint64, leaves []common.Hash) ([]byte, error) {
	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(leaves[:loSize])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		loSize,
		prefixExpansion,
		leaves[loSize:hiSize],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, loSize)
	onlyProof := prefixProof[numRead:]
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
}

func (s *StateManager) intermediateBigStepLeaves(ctx context.Context, wasmModuleRoot common.Hash, blockHeight uint64, toBigStep uint64) ([]common.Hash, error) {
	cacheKey := &challengecache.Key{
		WavmModuleRoot: wasmModuleRoot,
		MessageHeight:  protocol.Height(blockHeight),
		BigStepHeight:  option.None[protocol.Height](),
	}
	cachedRoots, err := s.historyCache.Get(cacheKey, option.Some(protocol.Height(toBigStep)))
	if err == nil {
		return cachedRoots, nil
	}
	if !errors.Is(err, challengecache.ErrNotFoundInCache) {
		return nil, err
	}
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(blockHeight))
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return nil, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	bigStepLeaves := execRun.GetBigStepLeavesUpTo(toBigStep, s.numOpcodesPerBigStep)
	result, err := bigStepLeaves.Await(ctx)
	if err != nil {
		return nil, err
	}
	// TODO: Hacky workaround to avoid saving a history commitment to height 0.
	if len(result) > 1 {
		if err := s.historyCache.Put(cacheKey, result); err != nil {
			if !errors.Is(err, challengecache.ErrFileAlreadyExists) {
				return nil, err
			}
		}
	}
	return result, nil
}

func (s *StateManager) intermediateSmallStepLeaves(ctx context.Context, wasmModuleRoot common.Hash, blockHeight uint64, bigStep uint64, toSmallStep uint64) ([]common.Hash, error) {
	cacheKey := &challengecache.Key{
		WavmModuleRoot: wasmModuleRoot,
		MessageHeight:  protocol.Height(blockHeight),
		BigStepHeight:  option.Some[protocol.Height](protocol.Height(bigStep)),
	}
	cachedRoots, err := s.historyCache.Get(cacheKey, option.Some(protocol.Height(toSmallStep)))
	if err == nil {
		return cachedRoots, nil
	}
	if !errors.Is(err, challengecache.ErrNotFoundInCache) {
		return nil, err
	}
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(blockHeight))
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return nil, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	smallStepLeaves := execRun.GetSmallStepLeavesUpTo(bigStep, toSmallStep, s.numOpcodesPerBigStep)
	result, err := smallStepLeaves.Await(ctx)
	if err != nil {
		return nil, err
	}
	// TODO: Hacky workaround to avoid saving a history commitment to height 0.
	if len(result) > 1 {
		if err := s.historyCache.Put(cacheKey, result); err != nil {
			if !errors.Is(err, challengecache.ErrFileAlreadyExists) {
				return nil, err
			}
		}
	}
	return result, nil
}

// TODO: Rename block to message.
func (s *StateManager) statesUpTo(blockStart uint64, blockEnd uint64, nextBatchCount uint64) ([]common.Hash, error) {
	if blockEnd < blockStart {
		return nil, fmt.Errorf("end block %v is less than start block %v", blockEnd, blockStart)
	}
	batch, err := s.findBatchAfterMessageCount(arbutil.MessageIndex(blockStart))
	if err != nil {
		return nil, err
	}
	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	desiredStatesLen := int(blockEnd - blockStart + 1)
	var stateRoots []common.Hash
	var lastStateRoot common.Hash
	for i := blockStart; i <= blockEnd; i++ {
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
		if err != nil {
			return nil, err
		}
		if batchMsgCount <= arbutil.MessageIndex(i) {
			batch++
		}
		gs, err := s.getInfoAtMessageCountAndBatch(arbutil.MessageIndex(i), batch)
		if err != nil {
			return nil, err
		}
		stateRoot := gs.Hash()
		stateRoots = append(stateRoots, stateRoot)
		lastStateRoot = stateRoot
		if gs.Batch >= nextBatchCount {
			if gs.Batch > nextBatchCount || gs.PosInBatch > 0 {
				return nil, fmt.Errorf("overran next batch count %v with global state batch %v position %v", nextBatchCount, gs.Batch, gs.PosInBatch)
			}
			break
		}
	}
	for len(stateRoots) < desiredStatesLen {
		stateRoots = append(stateRoots, lastStateRoot)
	}
	return stateRoots, nil
}

func (s *StateManager) findBatchAfterMessageCount(msgCount arbutil.MessageIndex) (uint64, error) {
	if msgCount == 0 {
		return 0, nil
	}
	low := uint64(0)
	batchCount, err := s.validator.inboxTracker.GetBatchCount()
	if err != nil {
		return 0, err
	}
	high := batchCount
	for {
		// Binary search invariants:
		//   - messageCount(high) >= msgCount
		//   - messageCount(low-1) < msgCount
		//   - high >= low
		if high < low {
			return 0, fmt.Errorf("when attempting to find batch for message count %v high %v < low %v", msgCount, high, low)
		}
		mid := (low + high) / 2
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(mid)
		if err != nil {
			// TODO: There is a circular dep with the error in inbox_tracker.go, we
			// should move it somewhere else and use errors.Is.
			if strings.Contains(err.Error(), "accumulator not found") {
				high = mid
			} else {
				return 0, fmt.Errorf("failed to get batch metadata while binary searching: %w", err)
			}
		}
		if batchMsgCount < msgCount {
			low = mid + 1
		} else if batchMsgCount == msgCount {
			return mid + 1, nil
		} else if mid == low { // batchMsgCount > msgCount
			return mid, nil
		} else { // batchMsgCount > msgCount
			high = mid
		}
	}
}

func (s *StateManager) getHashAtMessageCountAndBatch(_ context.Context, messageCount arbutil.MessageIndex, batch uint64) (common.Hash, error) {
	gs, err := s.getInfoAtMessageCountAndBatch(messageCount, batch)
	if err != nil {
		return common.Hash{}, err
	}
	return gs.Hash(), nil
}

func (s *StateManager) getInfoAtMessageCountAndBatch(messageCount arbutil.MessageIndex, batch uint64) (validator.GoGlobalState, error) {
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, batch)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return globalState, nil
}

func (s *StateManager) findGlobalStateFromMessageCountAndBatch(count arbutil.MessageIndex, batch uint64) (validator.GoGlobalState, error) {
	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if batch > 0 {
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if prevBatchMsgCount > count {
			return validator.GoGlobalState{}, errors.New("bad batch provided")
		}
	}
	res, err := s.validator.streamer.ResultAtCount(count)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      batch,
		PosInBatch: uint64(count - prevBatchMsgCount),
	}, nil
}
