// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
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

func ConsensusExecutionSyncerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".sync-interval", DefaultConsensusExecutionSyncerConfig.SyncInterval, "Interval in which finality data is pushed from consensus to execution")
}

type ConsensusExecutionSyncer struct {
	stopwaiter.StopWaiter

	config func() *ConsensusExecutionSyncerConfig

	inboxReader    *InboxReader
	execClient     execution.ExecutionClient
	blockValidator *staker.BlockValidator
}

func NewConsensusExecutionSyncer(
	config func() *ConsensusExecutionSyncerConfig,
	inboxReader *InboxReader,
	execClient execution.ExecutionClient,
	blockValidator *staker.BlockValidator,
) *ConsensusExecutionSyncer {
	return &ConsensusExecutionSyncer{
		config:         config,
		inboxReader:    inboxReader,
		execClient:     execClient,
		blockValidator: blockValidator,
	}
}

func (c *ConsensusExecutionSyncer) Start(ctx_in context.Context) {
	c.StopWaiter.Start(ctx_in, c)
	c.CallIteratively(c.pushFinalityDataFromConsensusToExecution)
}

func (c *ConsensusExecutionSyncer) pushFinalityDataFromConsensusToExecution(ctx context.Context) time.Duration {
	safeMsgCount, err := c.inboxReader.GetSafeMsgCount(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		log.Info("Finality not supported, not pushing finality data to execution")
		return c.config().SyncInterval
	} else if err != nil {
		log.Error("Error getting safe message count", "err", err)
		return c.config().SyncInterval
	}

	finalizedMsgCount, err := c.inboxReader.GetFinalizedMsgCount(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		log.Info("Finality not supported, not pushing finality data to execution")
		return c.config().SyncInterval
	} else if err != nil {
		log.Error("Error getting finalized message count", "err", err)
		return c.config().SyncInterval
	}

	var validatedMsgCount arbutil.MessageIndex
	if c.blockValidator != nil {
		validatedMsgCount = c.blockValidator.GetValidated()
	}

	finalityData := &arbutil.FinalityData{
		SafeMsgCount:      safeMsgCount,
		FinalizedMsgCount: finalizedMsgCount,
		ValidatedMsgCount: &validatedMsgCount,
	}

	_, err = c.execClient.SetFinalityData(ctx, finalityData).Await(ctx)
	if err != nil {
		log.Error("Error pushing finality data from consensus to execution", "err", err)
	} else {
		log.Info("Pushed finality data from consensus to execution", "SafeMsgCount", safeMsgCount, "FinalizedMsgCount", finalizedMsgCount, "ValidatedMsgCount", validatedMsgCount)
	}

	return c.config().SyncInterval
}
