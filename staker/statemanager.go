package staker

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"

	"github.com/OffchainLabs/challenge-protocol-v2/util"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
)

var AccumulatorNotFoundErr = errors.New("accumulator not found")

type StateManager struct {
	validator            *StatelessBlockValidator
	numOpcodesPerBigStep uint64
}

func NewStateManager(val *StatelessBlockValidator, numOpcodesPerBigStep uint64) (*StateManager, error) {
	return &StateManager{
		validator:            val,
		numOpcodesPerBigStep: numOpcodesPerBigStep,
	}, nil
}

// HistoryCommitmentUpTo Produces a block history commitment up to and including messageCount.
func (s *StateManager) HistoryCommitmentUpTo(ctx context.Context, messageCount uint64) (util.HistoryCommitment, error) {
	batch, err := s.findBatchAfterMessageCount(0)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	var stateRoots []common.Hash
	for i := arbutil.MessageIndex(0); i <= arbutil.MessageIndex(messageCount); i++ {
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		if batchMsgCount <= i {
			batch++
		}
		root, err := s.getHashAtMessageCountAndBatch(ctx, i, batch)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		stateRoots = append(stateRoots, root)
	}
	return util.NewHistoryCommitment(messageCount, stateRoots)
}

// BigStepCommitmentUpTo Produces a big step history commitment from big step 0 to toBigStep within block
// challenge heights blockHeight and blockHeight+1.
func (s *StateManager) BigStepCommitmentUpTo(ctx context.Context, wasmModuleRoot common.Hash, blockHeight uint64, toBigStep uint64) (util.HistoryCommitment, error) {
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(blockHeight))
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	bigStepCommitment := execRun.GetBigStepCommitmentUpTo(toBigStep, s.numOpcodesPerBigStep)
	result, err := bigStepCommitment.Await(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return result, nil
}

// SmallStepCommitmentUpTo Produces a small step history commitment from small step 0 to N between
// big steps bigStep to bigStep+1 within block challenge heights blockHeight to blockHeight+1.
func (s *StateManager) SmallStepCommitmentUpTo(ctx context.Context, wasmModuleRoot common.Hash, blockHeight uint64, bigStep uint64, toSmallStep uint64) (util.HistoryCommitment, error) {
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(blockHeight))
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	smallStepCommitment := execRun.GetSmallStepCommitmentUpTo(bigStep, toSmallStep, s.numOpcodesPerBigStep)
	result, err := smallStepCommitment.Await(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return result, nil
}

// HistoryCommitmentUpToBatch Produces a block challenge history commitment in a certain inclusive block range,
// but padding states with duplicates after the first state with a batch count of at least the specified max.
func (s *StateManager) HistoryCommitmentUpToBatch(ctx context.Context, blockStart uint64, blockEnd uint64, nextBatchCount uint64) (util.HistoryCommitment, error) {
	if blockEnd < blockStart {
		return util.HistoryCommitment{}, fmt.Errorf("end block %v is less than start block %v", blockEnd, blockStart)
	}
	batch, err := s.findBatchAfterMessageCount(arbutil.MessageIndex(blockStart))
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	desiredStatesLen := int(blockEnd - blockStart + 1)
	var stateRoots []common.Hash
	var lastStateRoot common.Hash
	for i := blockStart; i <= blockEnd; i++ {
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		if batchMsgCount <= arbutil.MessageIndex(i) {
			batch++
		}
		gs, err := s.getInfoAtMessageCountAndBatch(arbutil.MessageIndex(i), batch)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		stateRoot := gs.Hash()
		stateRoots = append(stateRoots, stateRoot)
		lastStateRoot = stateRoot
		if gs.Batch >= nextBatchCount {
			if gs.Batch > nextBatchCount || gs.PosInBatch > 0 {
				return util.HistoryCommitment{}, fmt.Errorf("overran next batch count %v with global state batch %v position %v", nextBatchCount, gs.Batch, gs.PosInBatch)
			}
			break
		}
	}
	for len(stateRoots) < desiredStatesLen {
		stateRoots = append(stateRoots, lastStateRoot)
	}
	return util.NewHistoryCommitment(blockEnd-blockStart, stateRoots)
}

func (s *StateManager) findBatchAfterMessageCount(msgCount arbutil.MessageIndex) (uint64, error) {
	if msgCount == 0 {
		return 0, nil
	}
	low := uint64(0)
	high := uint64(math.MaxUint64)
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
			if errors.Is(err, AccumulatorNotFoundErr) {
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
