package arbnode

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ConsensusExecutionSyncer struct {
	stopwaiter.StopWaiter
	inboxReader    *InboxReader
	execClient     execution.ExecutionClient
	blockValidator *staker.BlockValidator
}

func NewConsensusExecutionSyncer(
	inboxReader *InboxReader,
	execClient execution.ExecutionClient,
	blockValidator *staker.BlockValidator,
) *ConsensusExecutionSyncer {
	return &ConsensusExecutionSyncer{
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
	sleepTime := time.Second
	finalitySupported := true

	safeMsgCount, err := c.inboxReader.GetSafeMsgCount(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		finalitySupported = false
	} else if err != nil {
		log.Warn("Error getting safe message count", "err", err)
		return sleepTime
	}

	finalizedMsgCount, err := c.inboxReader.GetFinalizedMsgCount(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		finalitySupported = false
	} else if err != nil {
		log.Warn("Error getting finalized message count", "err", err)
		return sleepTime
	}

	var validatedMsgCount arbutil.MessageIndex
	blockValidatorSet := false
	if c.blockValidator != nil {
		validatedMsgCount = c.blockValidator.GetValidated()
		blockValidatorSet = true
	}

	finalityData := &arbutil.FinalityData{
		SafeMsgCount:      safeMsgCount,
		FinalizedMsgCount: finalizedMsgCount,
		ValidatedMsgCount: validatedMsgCount,
		FinalitySupported: finalitySupported,
		BlockValidatorSet: blockValidatorSet,
	}
	c.execClient.StoreFinalityData(finalityData)

	log.Info("Pushed finality data from consensus to execution", "finalityData", finalityData)

	return sleepTime
}
