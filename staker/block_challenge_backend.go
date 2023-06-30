// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/validator"
)

type BlockChallengeBackend struct {
	bc                     *core.BlockChain
	startBlock             int64
	startPosition          uint64
	endPosition            uint64
	startGs                validator.GoGlobalState
	endGs                  validator.GoGlobalState
	inboxTracker           InboxTrackerInterface
	genesisBlockNumber     uint64
	tooFarStartsAtPosition uint64
}

// Assert that BlockChallengeBackend implements ChallengeBackend
var _ ChallengeBackend = (*BlockChallengeBackend)(nil)

func NewBlockChallengeBackend(
	initialState *challengegen.ChallengeManagerInitiatedChallenge,
	maxBatchesRead uint64,
	bc *core.BlockChain,
	inboxTracker InboxTrackerInterface,
	genesisBlockNumber uint64,
) (*BlockChallengeBackend, error) {
	startGs := validator.GoGlobalStateFromSolidity(initialState.StartState)
	startBlockNum := arbutil.MessageCountToBlockNumber(0, genesisBlockNumber)
	if startGs.BlockHash != (common.Hash{}) {
		startBlock := bc.GetBlockByHash(startGs.BlockHash)
		if startBlock == nil {
			return nil, fmt.Errorf("failed to find start block %v", startGs.BlockHash)
		}
		startBlockNum = int64(startBlock.NumberU64())
	}

	var startMsgCount arbutil.MessageIndex
	if startGs.Batch > 0 {
		var err error
		startMsgCount, err = inboxTracker.GetBatchMessageCount(startGs.Batch - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get challenge start batch metadata: %w", err)
		}
	}
	startMsgCount += arbutil.MessageIndex(startGs.PosInBatch)
	expectedMsgCount := arbutil.SignedBlockNumberToMessageCount(startBlockNum, genesisBlockNumber)
	if startMsgCount != expectedMsgCount {
		return nil, fmt.Errorf("start block %v and start message count %v don't correspond", startBlockNum, startMsgCount)
	}

	var endMsgCount arbutil.MessageIndex
	if maxBatchesRead > 0 {
		var err error
		endMsgCount, err = inboxTracker.GetBatchMessageCount(maxBatchesRead - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get challenge end batch metadata: %w", err)
		}
	}

	return &BlockChallengeBackend{
		bc:                     bc,
		startBlock:             startBlockNum,
		startGs:                startGs,
		startPosition:          0,
		endPosition:            math.MaxUint64,
		endGs:                  validator.GoGlobalStateFromSolidity(initialState.EndState),
		inboxTracker:           inboxTracker,
		genesisBlockNumber:     genesisBlockNumber,
		tooFarStartsAtPosition: uint64(endMsgCount - startMsgCount + 1),
	}, nil
}

func (b *BlockChallengeBackend) findBatchFromMessageIndex(msgCount arbutil.MessageIndex) (uint64, error) {
	if msgCount == 0 {
		return 0, nil
	}
	low := b.startGs.Batch
	high := b.endGs.Batch
	for {
		// Binary search invariants:
		//   - messageCount(high) >= msgCount
		//   - messageCount(low-1) < msgCount
		//   - high >= low
		if high < low {
			return 0, fmt.Errorf("when attempting to find batch for message count %v high %v < low %v", msgCount, high, low)
		}
		mid := (low + high) / 2
		batchMsgCount, err := b.inboxTracker.GetBatchMessageCount(mid)
		if err != nil {
			return 0, fmt.Errorf("failed to get batch metadata while binary searching: %w", err)
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

func (b *BlockChallengeBackend) FindGlobalStateFromHeader(header *types.Header) (validator.GoGlobalState, error) {
	if header == nil {
		return validator.GoGlobalState{}, nil
	}
	msgCount := arbutil.BlockNumberToMessageCount(header.Number.Uint64(), b.genesisBlockNumber)
	batch, err := b.findBatchFromMessageIndex(msgCount)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	var batchMsgCount arbutil.MessageIndex
	if batch > 0 {
		batchMsgCount, err = b.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if batchMsgCount > msgCount {
			return validator.GoGlobalState{}, errors.New("findBatchFromMessageCount returned bad batch")
		}
	}
	extraInfo := types.DeserializeHeaderExtraInformation(header)
	return validator.GoGlobalState{
		BlockHash:  header.Hash(),
		SendRoot:   extraInfo.SendRoot,
		Batch:      batch,
		PosInBatch: uint64(msgCount - batchMsgCount),
	}, nil
}

const StatusFinished uint8 = 1
const StatusTooFar uint8 = 3

func (b *BlockChallengeBackend) GetBlockNrAtStep(step uint64) int64 {
	return b.startBlock + int64(step)
}

func (b *BlockChallengeBackend) GetInfoAtStep(step uint64) (validator.GoGlobalState, uint8, error) {
	blockNum := b.GetBlockNrAtStep(step)
	if step >= b.tooFarStartsAtPosition {
		return validator.GoGlobalState{}, StatusTooFar, nil
	}
	var header *types.Header
	if blockNum != -1 {
		header = b.bc.GetHeaderByNumber(uint64(blockNum))
		if header == nil {
			return validator.GoGlobalState{}, 0, fmt.Errorf("failed to get block %v in block challenge", blockNum)
		}
	}
	globalState, err := b.FindGlobalStateFromHeader(header)
	if err != nil {
		return validator.GoGlobalState{}, 0, err
	}
	return globalState, StatusFinished, nil
}

func (b *BlockChallengeBackend) SetRange(_ context.Context, start uint64, end uint64) error {
	if b.startPosition == start && b.endPosition == end {
		return nil
	}
	newStartGs, _, err := b.GetInfoAtStep(start)
	if err != nil {
		return err
	}
	newEndGs, endStatus, err := b.GetInfoAtStep(end)
	if err != nil {
		return err
	}
	if b.startPosition == start && b.startGs != newStartGs {
		return fmt.Errorf("challenge start position remains at %v but global state changed from %v to %v", start, b.startGs, newStartGs)
	}
	b.startGs = newStartGs
	if endStatus == StatusFinished {
		b.endGs = newEndGs
	}
	return nil
}

func (b *BlockChallengeBackend) GetHashAtStep(_ context.Context, position uint64) (common.Hash, error) {
	gs, status, err := b.GetInfoAtStep(position)
	if err != nil {
		return common.Hash{}, err
	}
	if status == StatusFinished {
		data := []byte("Block state:")
		data = append(data, gs.Hash().Bytes()...)
		return crypto.Keccak256Hash(data), nil
	} else if status == StatusTooFar {
		return crypto.Keccak256Hash([]byte("Block state, too far:")), nil
	} else {
		panic(fmt.Sprintf("Unknown block status: %v", status))
	}
}

func (b *BlockChallengeBackend) IssueExecChallenge(
	core *challengeCore,
	oldState *ChallengeState,
	startSegment int,
	numsteps uint64,
) (*types.Transaction, error) {
	position := oldState.Segments[startSegment].Position
	machineStatuses := [2]uint8{}
	globalStates := [2]validator.GoGlobalState{}
	var err error
	globalStates[0], machineStatuses[0], err = b.GetInfoAtStep(position)
	if err != nil {
		return nil, err
	}
	globalStates[1], machineStatuses[1], err = b.GetInfoAtStep(position + 1)
	if err != nil {
		return nil, err
	}
	globalStateHashes := [2][32]byte{
		globalStates[0].Hash(),
		globalStates[1].Hash(),
	}
	return core.con.ChallengeExecution(
		core.auth,
		core.challengeIndex,
		challengegen.ChallengeLibSegmentSelection{
			OldSegmentsStart:  oldState.Start,
			OldSegmentsLength: new(big.Int).Sub(oldState.End, oldState.Start),
			OldSegments:       oldState.RawSegments,
			ChallengePosition: big.NewInt(int64(startSegment)),
		},
		machineStatuses,
		globalStateHashes,
		big.NewInt(int64(numsteps)),
	)
}
