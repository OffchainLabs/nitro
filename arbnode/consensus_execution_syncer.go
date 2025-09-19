// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ConsensusExecutionSyncerConfig struct {
	SyncInterval time.Duration `koanf:"sync-interval"`
}

var DefaultConsensusExecutionSyncerConfig = ConsensusExecutionSyncerConfig{
	SyncInterval: 1 * time.Second,
}

func ConsensusExecutionSyncerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".sync-interval", DefaultConsensusExecutionSyncerConfig.SyncInterval, "Interval in which finality data is pushed from consensus to execution")
}

type ConsensusExecutionSyncer struct {
	stopwaiter.StopWaiter

	config func() *ConsensusExecutionSyncerConfig

	inboxReader    *InboxReader
	execClient     execution.ExecutionClient
	blockValidator *staker.BlockValidator
	txStreamer     *TransactionStreamer
}

func NewConsensusExecutionSyncer(
	config func() *ConsensusExecutionSyncerConfig,
	inboxReader *InboxReader,
	execClient execution.ExecutionClient,
	blockValidator *staker.BlockValidator,
	txStreamer *TransactionStreamer,
) *ConsensusExecutionSyncer {
	return &ConsensusExecutionSyncer{
		config:         config,
		inboxReader:    inboxReader,
		execClient:     execClient,
		blockValidator: blockValidator,
		txStreamer:     txStreamer,
	}
}

func (c *ConsensusExecutionSyncer) Start(ctx_in context.Context) {
	c.StopWaiter.Start(ctx_in, c)
	c.CallIteratively(c.pushFinalityDataFromConsensusToExecution)
}

func (c *ConsensusExecutionSyncer) getFinalityData(
	ctx context.Context,
	msgCount arbutil.MessageIndex,
	errMsgCount error,
	scenario string,
) (*arbutil.FinalityData, error) {
	if errors.Is(errMsgCount, headerreader.ErrBlockNumberNotSupported) {
		log.Debug("Finality not supported, not pushing finality data to execution")
		return nil, errMsgCount
	} else if errMsgCount != nil {
		log.Error("Error getting finality msg count", "scenario", scenario, "err", errMsgCount)
		return nil, errMsgCount
	}

	if msgCount == 0 {
		return nil, nil
	}
	msgIdx := msgCount - 1
	msgResult, err := c.txStreamer.ResultAtMessageIndex(msgIdx)
	if errors.Is(err, gethexec.ResultNotFound) {
		log.Debug("Message result not found, node out of sync", "msgIdx", msgIdx, "err", err)
		return nil, nil
	} else if err != nil {
		log.Error("Error getting message result", "msgIdx", msgIdx, "err", err)
		return nil, err
	}

	finalityData := &arbutil.FinalityData{
		MsgIdx:    msgIdx,
		BlockHash: msgResult.BlockHash,
	}
	return finalityData, nil
}

func (c *ConsensusExecutionSyncer) pushFinalityDataFromConsensusToExecution(ctx context.Context) time.Duration {
	safeMsgCount, err := c.inboxReader.GetSafeMsgCount(ctx)
	safeFinalityData, err := c.getFinalityData(ctx, safeMsgCount, err, "safe")
	if err != nil {
		return c.config().SyncInterval
	}

	finalizedMsgCount, err := c.inboxReader.GetFinalizedMsgCount(ctx)
	finalizedFinalityData, err := c.getFinalityData(ctx, finalizedMsgCount, err, "finalized")
	if err != nil {
		return c.config().SyncInterval
	}

	var validatedFinalityData *arbutil.FinalityData
	var validatedMsgCount arbutil.MessageIndex
	if c.blockValidator != nil {
		validatedMsgCount = c.blockValidator.GetValidated()
		validatedFinalityData, err = c.getFinalityData(ctx, validatedMsgCount, nil, "validated")
		if err != nil {
			return c.config().SyncInterval
		}
	}

	_, err = c.execClient.SetFinalityData(ctx, safeFinalityData, finalizedFinalityData, validatedFinalityData).Await(ctx)
	if err != nil {
		log.Error("Error pushing finality data from consensus to execution", "err", err)
	} else {
		finalityMsgCount := func(fd *arbutil.FinalityData) arbutil.MessageIndex {
			if fd != nil {
				return fd.MsgIdx + 1
			}
			return 0
		}
		log.Debug("Pushed finality data from consensus to execution",
			"safeMsgCount", finalityMsgCount(safeFinalityData),
			"finalizedMsgCount", finalityMsgCount(finalizedFinalityData),
			"validatedMsgCount", finalityMsgCount(validatedFinalityData),
		)
	}

	return c.config().SyncInterval
}
