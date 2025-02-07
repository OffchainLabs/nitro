package arbnode

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ConsensusExecutionSync struct {
	stopwaiter.StopWaiter
	inboxReader *InboxReader
	execClient  execution.ExecutionClient
}

func NewConsensusExecutionSync(inboxReader *InboxReader, execClient execution.ExecutionClient) *ConsensusExecutionSync {
	return &ConsensusExecutionSync{
		inboxReader: inboxReader,
		execClient:  execClient,
	}
}

func (c *ConsensusExecutionSync) Start(ctx_in context.Context) {
	c.StopWaiter.Start(ctx_in, c)
	c.CallIteratively(c.pushFinalityDataFromConsensusToExecution)
}

func (c *ConsensusExecutionSync) pushFinalityDataFromConsensusToExecution(ctx context.Context) time.Duration {
	sleepTime := time.Second
	finalityNotSupported := false

	safeMsgCount, err := c.inboxReader.GetSafeMsgCount(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		finalityNotSupported = true
	} else if err != nil {
		log.Warn("Error getting safe message count", "err", err)
		return sleepTime
	}

	finalizedMsgCount, err := c.inboxReader.GetFinalizedMsgCount(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		finalityNotSupported = true
	} else if err != nil {
		log.Warn("Error getting finalized message count", "err", err)
		return sleepTime
	}

	finalityData := &arbutil.FinalityData{
		SafeMsgCount:         safeMsgCount,
		FinalizedMsgCount:    finalizedMsgCount,
		FinalityNotSupported: finalityNotSupported,
	}
	c.execClient.StoreFinalityData(finalityData)

	log.Debug("Pushed finality data from consensus to execution", "finalityData", finalityData)

	return sleepTime
}
