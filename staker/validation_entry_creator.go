package staker

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
)

var _ ValidationEntryCreator = (*BlockValidatorInstance)(nil)

type BlockValidatorInstance struct {
	stateless *StatelessBlockValidator
	// can only be accessed from creation thread or if holding reorg-write
	nextCreateBatch       *FullBatchInfo
	nextCreateBatchReread bool
	prevBatchCache        map[uint64][]byte

	nextCreateStartGS     validator.GoGlobalState
	nextCreatePrevDelayed uint64

	config BlockValidatorConfigFetcher
}

func (bv *BlockValidatorInstance) CreateValidationEntry(
	ctx context.Context,
	position uint64,
) (*validationEntry, bool, error) {
	streamerMsgCount, err := bv.stateless.streamer.GetProcessedMessageCount()
	if err != nil {
		return nil, false, err
	}
	msgPos := arbutil.MessageIndex(position)
	if msgPos >= streamerMsgCount {
		log.Trace("create validation entry: nothing to do", "pos", msgPos, "streamerMsgCount", streamerMsgCount)
		return nil, false, nil
	}
	msg, err := bv.stateless.streamer.GetMessage(msgPos)
	if err != nil {
		return nil, false, err
	}
	endRes, err := bv.stateless.streamer.ResultAtMessageIndex(msgPos)
	if err != nil {
		return nil, false, err
	}
	if bv.nextCreateStartGS.PosInBatch == 0 || bv.nextCreateBatchReread {
		// new batch
		found, fullBatchInfo, err := bv.stateless.readFullBatch(ctx, bv.nextCreateStartGS.Batch)
		if !found {
			return nil, false, err
		}
		if bv.nextCreateBatch != nil {
			bv.prevBatchCache[bv.nextCreateBatch.Number] = bv.nextCreateBatch.PostedData
		}
		bv.nextCreateBatch = fullBatchInfo
		// #nosec G115
		validatorMsgCountCurrentBatch.Update(int64(fullBatchInfo.MsgCount))
		batchCacheLimit := bv.config().BatchCacheLimit
		if len(bv.prevBatchCache) > int(batchCacheLimit) {
			for num := range bv.prevBatchCache {
				if num+uint64(batchCacheLimit) < bv.nextCreateStartGS.Batch {
					delete(bv.prevBatchCache, num)
				}
			}
		}
		bv.nextCreateBatchReread = false
	}
	endGS := validator.GoGlobalState{
		BlockHash: endRes.BlockHash,
		SendRoot:  endRes.SendRoot,
	}
	if msgPos+1 < bv.nextCreateBatch.MsgCount {
		endGS.Batch = bv.nextCreateStartGS.Batch
		endGS.PosInBatch = bv.nextCreateStartGS.PosInBatch + 1
	} else if msgPos+1 == bv.nextCreateBatch.MsgCount {
		endGS.Batch = bv.nextCreateStartGS.Batch + 1
		endGS.PosInBatch = 0
	} else {
		return nil, false, fmt.Errorf("illegal batch msg count %d pos %d batch %d", bv.nextCreateBatch.MsgCount, msgPos, endGS.Batch)
	}
	chainConfig := bv.stateless.streamer.ChainConfig()
	prevBatchNums, err := msg.Message.PastBatchesRequired()
	if err != nil {
		return nil, false, err
	}
	prevBatches := make([]validator.BatchInfo, 0, len(prevBatchNums))
	// prevBatchNums are only used for batch reports, each is only used once
	for _, batchNum := range prevBatchNums {
		data, found := bv.prevBatchCache[batchNum]
		if found {
			delete(bv.prevBatchCache, batchNum)
		} else {
			data, err = bv.stateless.readPostedBatch(ctx, batchNum)
			if err != nil {
				return nil, false, err
			}
		}
		prevBatches = append(prevBatches, validator.BatchInfo{
			Number: batchNum,
			Data:   data,
		})
	}
	entry, err := newValidationEntry(
		msgPos, bv.nextCreateStartGS, endGS, msg, bv.nextCreateBatch, prevBatches, bv.nextCreatePrevDelayed, chainConfig,
	)
	if err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

func (bv *BlockValidatorInstance) OnReset() {}

func (bv *BlockValidatorInstance) UpdateNextCreationState(
	nextCreateStartGS validator.GoGlobalState,
	nextCreatePrevDelayed uint64,
	nextCreateBatchReread bool,
) {
	bv.nextCreateStartGS = nextCreateStartGS
	bv.nextCreatePrevDelayed = nextCreatePrevDelayed
	bv.nextCreateBatchReread = nextCreateBatchReread
	if bv.nextCreateBatch != nil {
		bv.prevBatchCache[bv.nextCreateBatch.Number] = bv.nextCreateBatch.PostedData
	}
}

func (bv *BlockValidatorInstance) ResetCaches() {
	bv.nextCreateBatchReread = true
	bv.prevBatchCache = make(map[uint64][]byte)
}
