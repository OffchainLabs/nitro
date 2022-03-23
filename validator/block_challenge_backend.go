// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"

	"github.com/pkg/errors"
)

type GoGlobalState struct {
	BlockHash  common.Hash
	SendRoot   common.Hash
	Batch      uint64
	PosInBatch uint64
}

func u64ToBe(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func (s GoGlobalState) Hash() common.Hash {
	data := []byte("Global state:")
	data = append(data, s.BlockHash.Bytes()...)
	data = append(data, s.SendRoot.Bytes()...)
	data = append(data, u64ToBe(s.Batch)...)
	data = append(data, u64ToBe(s.PosInBatch)...)
	return crypto.Keccak256Hash(data)
}

func GoGlobalStateFromSolidity(gs challengegen.GlobalState) GoGlobalState {
	return GoGlobalState{
		BlockHash:  gs.Bytes32Vals[0],
		SendRoot:   gs.Bytes32Vals[1],
		Batch:      gs.U64Vals[0],
		PosInBatch: gs.U64Vals[1],
	}
}

func (s GoGlobalState) AsSolidityStruct() challengegen.GlobalState {
	return challengegen.GlobalState{
		Bytes32Vals: [2][32]byte{s.BlockHash, s.SendRoot},
		U64Vals:     [2]uint64{s.Batch, s.PosInBatch},
	}
}

type BlockChallengeBackend struct {
	bc                     *core.BlockChain
	startBlock             int64
	startPosition          uint64
	endPosition            uint64
	startGs                GoGlobalState
	endGs                  GoGlobalState
	inboxTracker           InboxTrackerInterface
	genesisBlockNumber     uint64
	tooFarStartsAtPosition uint64
}

// Assert that BlockChallengeBackend implements ChallengeBackend
var _ ChallengeBackend = (*BlockChallengeBackend)(nil)

func NewBlockChallengeBackend(
	initialState *challengegen.ChallengeManagerInitiatedChallenge,
	bc *core.BlockChain,
	inboxTracker InboxTrackerInterface,
	genesisBlockNumber uint64,
) (*BlockChallengeBackend, error) {
	startGs := GoGlobalStateFromSolidity(initialState.StartState)
	startBlockNum := arbutil.MessageCountToBlockNumber(0, genesisBlockNumber)
	if startGs.BlockHash != (common.Hash{}) {
		startBlock := bc.GetBlockByHash(startGs.BlockHash)
		if startBlock == nil {
			return nil, errors.New("failed to find start block")
		}
		startBlockNum = int64(startBlock.NumberU64())
	}

	var startMsgCount arbutil.MessageIndex
	if startGs.Batch > 0 {
		var err error
		startMsgCount, err = inboxTracker.GetBatchMessageCount(startGs.Batch - 1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get challenge start batch metadata")
		}
	}
	startMsgCount += arbutil.MessageIndex(startGs.PosInBatch)
	expectedMsgCount := arbutil.SignedBlockNumberToMessageCount(startBlockNum, genesisBlockNumber)
	if startMsgCount != expectedMsgCount {
		return nil, fmt.Errorf("start block %v and start message count %v don't correspond", startBlockNum, startMsgCount)
	}

	endGs := GoGlobalStateFromSolidity(initialState.EndState)
	var endMsgCount arbutil.MessageIndex
	if endGs.Batch > 0 {
		var err error
		endMsgCount, err = inboxTracker.GetBatchMessageCount(endGs.Batch - 1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get challenge end batch metadata")
		}
	}
	endMsgCount += arbutil.MessageIndex(endGs.PosInBatch)

	return &BlockChallengeBackend{
		bc:                     bc,
		startBlock:             startBlockNum,
		startGs:                startGs,
		startPosition:          0,
		endPosition:            math.MaxUint64,
		endGs:                  endGs,
		inboxTracker:           inboxTracker,
		genesisBlockNumber:     genesisBlockNumber,
		tooFarStartsAtPosition: uint64(endMsgCount - startMsgCount + 1),
	}, nil
}

func (b *BlockChallengeBackend) findBatchFromMessageCount(msgCount arbutil.MessageIndex) (uint64, error) {
	if msgCount == 0 {
		return 0, nil
	}
	low := b.startGs.Batch
	high := b.endGs.Batch
	if b.endGs.PosInBatch == 0 {
		if high == 0 {
			return 0, errors.New("end global state at inbox position (0, 0)")
		}
		high--
	}
	for {
		// Binary search invariants:
		//   - messageCount(high) >= msgCount
		//   - messageCount(low-1) < msgCount
		mid := (low + high) / 2
		batchMsgCount, err := b.inboxTracker.GetBatchMessageCount(mid)
		if err != nil {
			return 0, errors.Wrap(err, "failed to get batch metadata while binary searching")
		}
		if batchMsgCount < msgCount {
			low = mid + 1
		} else if batchMsgCount == msgCount {
			return mid, nil
		} else if mid == low { // batchMsgCount > msgCount
			return mid, nil
		} else { // batchMsgCount > msgCount
			high = mid
		}
	}
}

func (b *BlockChallengeBackend) FindGlobalStateFromHeader(header *types.Header) (GoGlobalState, error) {
	if header == nil {
		return GoGlobalState{}, nil
	}
	msgCount := arbutil.BlockNumberToMessageCount(header.Number.Uint64(), b.genesisBlockNumber)
	batch, err := b.findBatchFromMessageCount(msgCount)
	if err != nil {
		return GoGlobalState{}, err
	}
	var batchMsgCount arbutil.MessageIndex
	if batch > 0 {
		batchMsgCount, err = b.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return GoGlobalState{}, err
		}
		if batchMsgCount > msgCount {
			return GoGlobalState{}, errors.New("findBatchFromMessageCount returned bad batch")
		}
	}
	extraInfo, err := types.DeserializeHeaderExtraInformation(header)
	if err != nil {
		return GoGlobalState{}, err
	}
	return GoGlobalState{header.Hash(), extraInfo.SendRoot, batch, uint64(msgCount - batchMsgCount)}, nil
}

const StatusFinished uint8 = 1
const StatusTooFar uint8 = 3

func (b *BlockChallengeBackend) GetBlockNrAtStep(step uint64) (int64, bool) {
	return b.startBlock + int64(step), step >= b.tooFarStartsAtPosition
}

func (b *BlockChallengeBackend) GetInfoAtStep(step uint64) (GoGlobalState, uint8, error) {
	blockNum, tooFar := b.GetBlockNrAtStep(step)
	if tooFar {
		return GoGlobalState{}, StatusTooFar, nil
	}
	var header *types.Header
	if blockNum != -1 {
		header = b.bc.GetHeaderByNumber(uint64(blockNum))
		if header == nil {
			return GoGlobalState{}, 0, fmt.Errorf("failed to get block %v in block challenge", blockNum)
		}
	}
	globalState, err := b.FindGlobalStateFromHeader(header)
	if err != nil {
		return GoGlobalState{}, 0, err
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
	globalStates := [2]GoGlobalState{}
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
