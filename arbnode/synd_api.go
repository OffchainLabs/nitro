package arbnode

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

type SyndAPI struct {
	recorder      *gethexec.BlockRecorder
	inboxStreamer *TransactionStreamer
}

type ValidationData struct {
	BatchStartBlockNum uint64
	BatchEndBlockNum   uint64
	BatchStartIndex    uint64
	BatchEndIndex      uint64
	DelayedMessages    [][]byte
	StartDelayedAcc    common.Hash
	PreimageData       [][]byte
}

func (a *SyndAPI) BatchMetadata(batch uint64) (BatchMetadata, error) {
	return a.inboxStreamer.inboxReader.Tracker().GetBatchMetadata(batch)
}

func (a *SyndAPI) BatchFromAcc(ctx context.Context, acc common.Hash) (uint64, error) {
	count, err := a.inboxStreamer.inboxReader.Tracker().GetBatchCount()
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, errors.New("no batches found")
	}
	for count > 0 {
		count--
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		batchAcc, err := a.inboxStreamer.inboxReader.Tracker().GetBatchAcc(count)
		if err != nil {
			return 0, err
		}
		if batchAcc == acc {
			return count, nil
		}
	}
	return 0, errors.New("acc not found")
}

func (a *SyndAPI) getDelayedMessage(seqNum uint64) ([]byte, error) {
	data, err := a.inboxStreamer.inboxReader.Tracker().db.Get(dbKey(rlpDelayedMessagePrefix, seqNum))
	if err != nil {
		return nil, err
	}
	if len(data) < 32 {
		return nil, errors.New("delayed message new entry missing accumulator")
	}
	var msg *arbostypes.L1IncomingMessage
	err = rlp.DecodeBytes(data[32:], &msg)
	if err != nil {
		return nil, err
	}
	if msg.Header.RequestId.Big().Uint64() != seqNum {
		return nil, fmt.Errorf("unexpected request id: got %d, expected %d", msg.Header.RequestId, seqNum)
	}
	return msg.Serialize()
}

func (a *SyndAPI) ValidationData(ctx context.Context, startBlock uint64, endBatch uint64, isRelative bool) (*ValidationData, error) {
	startBatch, found, err := a.inboxStreamer.inboxReader.Tracker().FindInboxBatchContainingMessage(arbutil.MessageIndex(startBlock))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("start batch for block %d not found", startBlock)
	}

	if isRelative {
		endBatch += startBatch
	}

	if startBatch > endBatch {
		return nil, errors.New("start batch is after end batch")
	}

	metadata, err := a.inboxStreamer.inboxReader.Tracker().GetBatchMetadata(startBatch)
	if err != nil {
		return nil, err
	}
	startCount := metadata.DelayedMessageCount

	var startDelayedAcc common.Hash
	if startCount > 0 {
		startDelayedAcc, err = a.inboxStreamer.inboxReader.Tracker().GetDelayedAcc(startCount - 1)
		if err != nil {
			return nil, err
		}
	}

	batchStartBlockNum := metadata.ParentChainBlock

	if startBatch == endBatch {
		return &ValidationData{
			BatchStartBlockNum: batchStartBlockNum + 1,
			BatchEndBlockNum:   batchStartBlockNum,
			BatchStartIndex:    startBatch + 1,
			BatchEndIndex:      endBatch,
			DelayedMessages:    nil,
			StartDelayedAcc:    startDelayedAcc,
			PreimageData:       nil,
		}, nil
	}

	metadata, err = a.inboxStreamer.inboxReader.Tracker().GetBatchMetadata(startBatch + 1)
	if err != nil {
		return nil, err
	}
	batchStartBlockNum = metadata.ParentChainBlock

	endBlock, err := a.inboxStreamer.inboxReader.Tracker().GetBatchMessageCount(endBatch)
	if err != nil {
		return nil, err
	}
	endBlock -= 1

	batchStartBlock, err := a.inboxStreamer.inboxReader.Tracker().GetBatchMessageCount(startBatch)
	if err != nil {
		return nil, err
	}
	if startBlock+1 != uint64(batchStartBlock) {
		return nil, fmt.Errorf("batch starts at block %d, not %d", uint64(batchStartBlock), startBlock+1)
	}

	metadata, err = a.inboxStreamer.inboxReader.Tracker().GetBatchMetadata(endBatch)
	if err != nil {
		return nil, err
	}
	endCount := metadata.DelayedMessageCount

	if startCount > endCount {
		return nil, errors.New("unexpected error: startCount > endCount")
	}

	var messages [][]byte
	for i := startCount; i < endCount; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		msg, err := a.getDelayedMessage(i)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	preimageData, err := a.preimageData(ctx, arbutil.MessageIndex(startBlock), endBlock, startBatch, endBatch)
	if err != nil {
		return nil, err
	}

	return &ValidationData{
		BatchStartBlockNum: batchStartBlockNum,
		BatchEndBlockNum:   metadata.ParentChainBlock,
		BatchStartIndex:    startBatch + 1,
		BatchEndIndex:      endBatch,
		DelayedMessages:    messages,
		StartDelayedAcc:    startDelayedAcc,
		PreimageData:       preimageData,
	}, nil
}

func (a *SyndAPI) PreimageData(ctx context.Context, startBlock uint64, endBatch uint64, isRelative bool) ([][]byte, error) {
	startBatch, found, err := a.inboxStreamer.inboxReader.Tracker().FindInboxBatchContainingMessage(arbutil.MessageIndex(startBlock))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("start batch for block %d not found", startBlock)
	}

	if isRelative {
		endBatch += startBatch
	}

	if startBatch > endBatch {
		return nil, errors.New("start batch is after end batch")
	}

	if startBatch == endBatch {
		return nil, nil
	}

	endBlock, err := a.inboxStreamer.inboxReader.Tracker().GetBatchMessageCount(endBatch)
	if err != nil {
		return nil, err
	}
	endBlock -= 1

	batchStartBlock, err := a.inboxStreamer.inboxReader.Tracker().GetBatchMessageCount(startBatch)
	if err != nil {
		return nil, err
	}
	if startBlock+1 != uint64(batchStartBlock) {
		return nil, fmt.Errorf("batch starts at block %d, not %d", uint64(batchStartBlock), startBlock+1)
	}

	return a.preimageData(ctx, arbutil.MessageIndex(startBlock), endBlock, startBatch, endBatch)
}

func (a *SyndAPI) preimageData(ctx context.Context, startBlock arbutil.MessageIndex, endBlock arbutil.MessageIndex, startBatch uint64, endBatch uint64) ([][]byte, error) {
	m := make(map[common.Hash][]byte)
	var msgs []*arbostypes.MessageWithMetadata
	for i := startBlock + 1; i <= endBlock; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		msg, err := a.inboxStreamer.GetMessage(i)
		if err != nil {
			return nil, err
		}
		prevBatchNums, err := msg.Message.PastBatchesRequired()
		if err != nil {
			return nil, err
		}
		for _, batchNum := range prevBatchNums {
			if batchNum > endBatch {
				return nil, errors.New("future batch is required")
			}
			if batchNum <= startBatch {
				// makes an eth_getLogs request to fetch the batch
				batch, _, err := a.inboxStreamer.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
				if err != nil {
					return nil, err
				}
				m[crypto.Keccak256Hash(batch)] = batch
			}
		}
		msgs = append(msgs, msg)
	}

	// does not make any network requests
	recording, err := a.recorder.RecordBlocks(ctx, startBlock, msgs)
	if err != nil {
		return nil, err
	}
	if recording.Pos != endBlock {
		return nil, errors.New("unexpected recording position")
	}
	maps.Copy(recording.Preimages, m)
	return slices.Collect(maps.Values(recording.Preimages)), nil
}
