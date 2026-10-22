// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package melrpcclient implements mel.MELNative by forwarding reads to a remote MEL provider over
// the "meldataprovider" namespace. Extracted messages are not pulled here — the provider pushes
// them into the nitromelconsumer sink — so this client has no message loop.
package melrpcclient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var _ mel.MELNative = (*Client)(nil)

type Client struct {
	stopwaiter.StopWaiter
	client       *rpcclient.RpcClient
	l1Reader     *headerreader.HeaderReader
	caughtUpChan chan struct{}
}

// NewClient takes the local node's parent chain reader, since it cannot be fetched over RPC.
func NewClient(config rpcclient.ClientConfigFetcher, stack *node.Node, l1Reader *headerreader.HeaderReader) *Client {
	return &Client{
		client:       rpcclient.NewRpcClient(config, stack),
		l1Reader:     l1Reader,
		caughtUpChan: make(chan struct{}),
	}
}

func (c *Client) Start(ctxIn context.Context) error {
	c.StopWaiter.Start(ctxIn, c)
	ctx := c.GetContext()
	if err := c.client.Start(ctx); err != nil {
		return err
	}
	// Mirrors the native extractor's caughtUpChan, driven by the provider's caughtUp RPC.
	c.LaunchThread(c.waitForCaughtUp)
	return nil
}

func (c *Client) StopAndWait() {
	c.StopWaiter.StopAndWait()
	c.client.Close()
}

func (c *Client) waitForCaughtUp(ctx context.Context) {
	for {
		caughtUp, err := call[bool](c, ctx, "_caughtUp")
		if err != nil {
			log.Error("meldataprovider_caughtUp failed", "err", err)
		} else if caughtUp {
			close(c.caughtUpChan)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// callCtx serves the ctx-less interface methods, falling back to Background before Start.
func (c *Client) callCtx() context.Context {
	if ctx, err := c.GetContextSafe(); err == nil {
		return ctx
	}
	return context.Background()
}

func call[T any](c *Client, ctx context.Context, method string, args ...any) (T, error) {
	var res T
	err := c.client.CallContext(ctx, &res, mel.RPCNamespace+method, args...)
	return res, convertError(err)
}

// convertError rebuilds the sentinels consumers branch on, since JSON-RPC flattens errors to
// strings. Mirrors execution/rpcclient.convertError.
func convertError(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	switch {
	case strings.Contains(s, mel.ErrDelayedMessageNotYetFinalized.Error()):
		return mel.ErrDelayedMessageNotYetFinalized
	case strings.Contains(s, mel.ErrDelayedAccumulatorMismatch.Error()):
		return mel.ErrDelayedAccumulatorMismatch
	case strings.Contains(s, mel.ErrDelayedMessagePreimageNotFound.Error()):
		return mel.ErrDelayedMessagePreimageNotFound
	case strings.Contains(s, mel.ErrFindDelayedNotImplementedByMEL.Error()):
		return mel.ErrFindDelayedNotImplementedByMEL
	}
	return err
}

// --- Remote (forwarded) MELNative methods ---

func (c *Client) GetBatchCount() (uint64, error) {
	return call[uint64](c, c.callCtx(), "_getBatchCount")
}

func (c *Client) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	return call[arbutil.MessageIndex](c, c.callCtx(), "_getBatchMessageCount", seqNum)
}

func (c *Client) GetBatchMetadata(seqNum uint64) (mel.BatchMetadata, error) {
	return call[mel.BatchMetadata](c, c.callCtx(), "_getBatchMetadata", seqNum)
}

func (c *Client) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	return call[common.Hash](c, c.callCtx(), "_getBatchAcc", seqNum)
}

func (c *Client) GetBatchParentChainBlock(seqNum uint64) (uint64, error) {
	return call[uint64](c, c.callCtx(), "_getBatchParentChainBlock", seqNum)
}

func (c *Client) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	return call[common.Hash](c, c.callCtx(), "_getDelayedAcc", seqNum)
}

func (c *Client) GetDelayedCount() (uint64, error) {
	return call[uint64](c, c.callCtx(), "_getDelayedCount")
}

func (c *Client) GetDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	return call[*mel.DelayedInboxMessage](c, c.callCtx(), "_getDelayedMessage", index)
}

func (c *Client) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	return call[[]byte](c, ctx, "_getDelayedMessageBytes", seqNum)
}

func (c *Client) FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error) {
	r, err := call[mel.FindInboxBatchResult](c, c.callCtx(), "_findInboxBatchContainingMessage", pos)
	return r.SeqNum, r.Found, err
}

func (c *Client) FindParentChainBlockContainingDelayed(ctx context.Context, index uint64) (uint64, error) {
	return call[uint64](c, ctx, "_findParentChainBlockContainingDelayed", index)
}

func (c *Client) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	r, err := call[mel.SequencerMessageResult](c, ctx, "_getSequencerMessageBytes", seqNum)
	return r.Data, r.BlockHash, err
}

func (c *Client) GetSequencerMessageBytesForParentBlock(ctx context.Context, seqNum uint64, parentChainBlock uint64) ([]byte, common.Hash, error) {
	r, err := call[mel.SequencerMessageResult](c, ctx, "_getSequencerMessageBytesForParentBlock", seqNum, parentChainBlock)
	return r.Data, r.BlockHash, err
}

func (c *Client) FinalizedDelayedMessageAtPosition(ctx context.Context, finalizedBlock uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error) {
	r, err := call[mel.FinalizedDelayedResult](c, ctx, "_finalizedDelayedMessageAtPosition", finalizedBlock, lastDelayedAccumulator, requestedPosition)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	if r.NotYetFinalized {
		return nil, common.Hash{}, r.ParentChainBlockNumber, mel.ErrDelayedMessageNotYetFinalized
	}
	return r.Message, r.AfterInboxAcc, r.ParentChainBlockNumber, nil
}

func (c *Client) GetMsgCount() (arbutil.MessageIndex, error) {
	return call[arbutil.MessageIndex](c, c.callCtx(), "_getMsgCount")
}

func (c *Client) GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return call[arbutil.MessageIndex](c, ctx, "_getSafeMsgCount")
}

func (c *Client) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return call[arbutil.MessageIndex](c, ctx, "_getFinalizedMsgCount")
}

func (c *Client) GetSyncProgress(ctx context.Context) (mel.MessageSyncProgress, error) {
	return call[mel.MessageSyncProgress](c, ctx, "_getSyncProgress")
}

func (c *Client) SupportsPushingFinalityData() bool {
	// Always true for MEL; best-effort over RPC.
	res, err := call[bool](c, c.callCtx(), "_supportsPushingFinalityData")
	if err == nil && !res {
		return false
	}
	return true
}

func (c *Client) GetState(parentChainBlockNumber uint64) (*mel.State, error) {
	return call[*mel.State](c, c.callCtx(), "_getState", parentChainBlockNumber)
}

func (c *Client) GetHeadState() (*mel.State, error) {
	return call[*mel.State](c, c.callCtx(), "_getHeadState")
}

// ReorgTo does not feed melReorgDetector: the provider calls back ReorgedToParentChainBlock for
// that. Doing it here would add a second writer, and the node has no MEL DB to pick a target.
func (c *Client) ReorgTo(parentChainBlockNumber uint64) error {
	_, err := call[struct{}](c, c.callCtx(), "_reorgTo", parentChainBlockNumber)
	return err
}

// --- Local (non-forwarded) MELNative methods ---

// GetL1Reader returns the local node's parent chain reader.
func (c *Client) GetL1Reader() *headerreader.HeaderReader {
	return c.l1Reader
}

// SetMessageConsumer is a no-op: the provider pushes to the nitromelconsumer sink instead.
func (c *Client) SetMessageConsumer(consumer mel.MessageConsumer) error {
	if consumer == nil {
		return errors.New("nil message consumer")
	}
	return nil
}

func (c *Client) CaughtUp() chan struct{} {
	return c.caughtUpChan
}
