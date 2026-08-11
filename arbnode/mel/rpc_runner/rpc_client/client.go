// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package melrpcclient implements mel.MELNative by forwarding read/query calls to a
// remote MEL provider over the "meldataprovider" JSON-RPC namespace. It runs on the Nitro
// node. Extracted messages are NOT pulled here; they are pushed to the local node via
// the nitromelconsumer sink (see melrpcserver), so this client has no message loop.
package melrpcclient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

// NewClient builds a MEL query client. l1Reader is the local node's parent chain reader,
// returned by GetL1Reader (a *headerreader.HeaderReader cannot be fetched over RPC).
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
	// Mirror the native extractor's caughtUpChan: close it once the remote provider reports
	// (via the dedicated caughtUp RPC) that it has caught up with the parent chain.
	c.LaunchThread(c.waitForCaughtUp)
	return nil
}

func (c *Client) StopAndWait() {
	c.client.Close()
	c.StopWaiter.StopAndWait()
}

func (c *Client) waitForCaughtUp(ctx context.Context) {
	defer close(c.caughtUpChan)
	for {
		caughtUp, err := call[bool](c, ctx, "_caughtUp")
		if err == nil && caughtUp {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// callCtx returns the stopwaiter context for the synchronous (ctx-less) interface methods,
// falling back to context.Background if the client hasn't been started yet.
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

// convertError reconstructs the typed sentinels that consumers branch on (errors.Is),
// since JSON-RPC flattens errors to strings. Mirrors execution/rpcclient.convertError.
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
	// MEL always supports pushing finality data; best-effort over RPC (default false if unreachable).
	res, err := call[bool](c, c.callCtx(), "_supportsPushingFinalityData")
	if err != nil {
		return false
	}
	return res
}

func (c *Client) GetState(parentChainBlockNumber uint64) (*mel.State, error) {
	return call[*mel.State](c, c.callCtx(), "_getState", parentChainBlockNumber)
}

func (c *Client) GetHeadState() (*mel.State, error) {
	return call[*mel.State](c, c.callCtx(), "_getHeadState")
}

// ReorgTo instructs the remote provider to reorg (node->provider). It does NOT feed the local
// melReorgDetector: the provider is expected to call back the node's nitromelconsumer
// ReorgedToParentChainBlock so local consumers rewind. Feeding the channel here would add a
// second writer, and in RPC mode the node has no MEL DB to know the correct rewind target.
func (c *Client) ReorgTo(parentChainBlockNumber uint64) error {
	_, err := call[struct{}](c, c.callCtx(), "_reorgTo", parentChainBlockNumber)
	return err
}

// --- Local (non-forwarded) MELNative methods ---

// GetL1Reader returns the local node's parent chain reader (see NewClient).
func (c *Client) GetL1Reader() *headerreader.HeaderReader {
	return c.l1Reader
}

// SetMessageConsumer is a no-op for the RPC client. On a MEL-consumer node, extracted
// messages are delivered by the remote provider to the local nitromelconsumer sink
// (wired to the TransactionStreamer in arbnode.registerAPIs), not pulled by this client.
func (c *Client) SetMessageConsumer(consumer mel.MessageConsumer) error {
	if consumer == nil {
		return errors.New("nil message consumer")
	}
	return nil
}

func (c *Client) CaughtUp() chan struct{} {
	return c.caughtUpChan
}
