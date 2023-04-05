package execserver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type ExecAPI struct {
	exec execution.FullExecutionClient
}

func (c *ExecAPI) DigestMessage(ctx context.Context, num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	return c.exec.DigestMessage(num, msg).Await(ctx)
}

func (c *ExecAPI) Reorg(ctx context.Context, count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) error {
	_, err := c.exec.Reorg(count, newMessages, oldMessages).Await(ctx)
	return err
}

func (c *ExecAPI) HeadMessageNumber(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.exec.HeadMessageNumber().Await(ctx)
}

func (c *ExecAPI) ResultAtPos(ctx context.Context, pos arbutil.MessageIndex) (*execution.MessageResult, error) {
	return c.exec.ResultAtPos(pos).Await(ctx)
}

func (c *ExecAPI) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.RecordResult, error) {
	return c.exec.RecordBlockCreation(pos, msg).Await(ctx)
}

func (c *ExecAPI) MarkValid(ctx context.Context, pos arbutil.MessageIndex, resultHash common.Hash) {
	go func() {
		c.exec.MarkValid(pos, resultHash)
	}()
}

func (c *ExecAPI) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	_, err := c.exec.PrepareForRecord(start, end).Await(ctx)
	return err
}

func (c *ExecAPI) SeqPause(ctx context.Context) error {
	_, err := c.exec.Pause().Await(ctx)
	return err
}

func (c *ExecAPI) SeqActivate(ctx context.Context) error {
	_, err := c.exec.Activate().Await(ctx)
	return err
}

func (c *ExecAPI) ForwardTo(ctx context.Context, url string) error {
	_, err := c.exec.ForwardTo(url).Await(ctx)
	return err
}

func (c *ExecAPI) SequenceDelayedMessage(ctx context.Context, message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	_, err := c.exec.SequenceDelayedMessage(message, delayedSeqNum).Await(ctx)
	return err
}

func (c *ExecAPI) Maintenance(ctx context.Context) error {
	_, err := c.exec.Maintenance().Await(ctx)
	return err
}

func (c *ExecAPI) NextDelayedMessageNumber(ctx context.Context) (uint64, error) {
	return c.exec.NextDelayedMessageNumber().Await(ctx)
}
