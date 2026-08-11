// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package melrpcrunner_test exercises Nitro's MEL consumer-side JSON-RPC code end to end over
// an in-process HTTP RPC server: the "meldataprovider" query client (melrpcclient) and the
// "nitromelconsumer" node-side server (melrpcserver). The provider side (the "meldataprovider"
// query server and the provider->node RPC client) lives outside this repo, so it is stubbed
// locally here (queryServer, pushClient, reorgClient).
package melrpcrunner_test

import (
	"context"
	"errors"
	"math/big"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/rpc_runner/rpc_client"
	"github.com/offchainlabs/nitro/arbnode/mel/rpc_runner/rpc_server"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

// fakeProvider is a minimal mel.MELNative for the server side. It embeds the interface
// (nil) so only the methods exercised here need implementing; any unexpected call panics.
type fakeProvider struct {
	mel.MELNative

	batchCount       uint64
	headState        *mel.State
	stateByBlock     map[uint64]*mel.State
	delayed          map[uint64]*mel.DelayedInboxMessage
	notYetFinalized  uint64
	notFinalizedPCBN uint64
}

func (f *fakeProvider) GetBatchCount() (uint64, error) { return f.batchCount, nil }

func (f *fakeProvider) GetSyncProgress(ctx context.Context) (mel.MessageSyncProgress, error) {
	return mel.MessageSyncProgress{BatchSeen: f.batchCount, BatchProcessed: f.batchCount, MsgCount: 100}, nil
}

// CaughtUp returns a closed channel so the server's caughtUp RPC reports caught-up.
func (f *fakeProvider) CaughtUp() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (f *fakeProvider) GetHeadState() (*mel.State, error) { return f.headState, nil }

func (f *fakeProvider) GetState(parentChainBlockNumber uint64) (*mel.State, error) {
	s, ok := f.stateByBlock[parentChainBlockNumber]
	if !ok {
		return nil, errors.New("state not found")
	}
	return s, nil
}

func (f *fakeProvider) GetDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	m, ok := f.delayed[index]
	if !ok {
		return nil, errors.New("delayed message not found")
	}
	return m, nil
}

func (f *fakeProvider) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	return []byte{0xde, 0xad, 0xbe, 0xef}, common.HexToHash("0xfeed"), nil
}

func (f *fakeProvider) FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error) {
	if pos >= 100 {
		return 0, false, nil
	}
	return 3, true, nil
}

func (f *fakeProvider) FinalizedDelayedMessageAtPosition(ctx context.Context, finalizedBlock uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error) {
	if requestedPosition == f.notYetFinalized {
		return nil, common.Hash{}, f.notFinalizedPCBN, mel.ErrDelayedMessageNotYetFinalized
	}
	m := f.delayed[requestedPosition]
	return m.Message, m.AfterInboxAcc(), m.ParentChainBlockNumber, nil
}

func makeL1Msg(tag byte) *arbostypes.L1IncomingMessage {
	reqID := common.BytesToHash([]byte{tag})
	return &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:      arbostypes.L1MessageType_EndOfBlock,
			RequestId: &reqID,
			L1BaseFee: big.NewInt(int64(tag)),
		},
		L2msg: []byte{tag, tag, tag},
	}
}

// queryServer is a local stand-in for the external MEL provider's "meldataprovider" query server
// (which lives outside this repo): it adapts a mel.MELNative to the RPC method shape
// (ctx-first, result structs for multi-returns) that melrpcclient calls. Only the methods
// exercised by these tests are implemented.
type queryServer struct{ svc mel.MELNative }

func (s *queryServer) GetBatchCount(ctx context.Context) (uint64, error) {
	return s.svc.GetBatchCount()
}

func (s *queryServer) GetHeadState(ctx context.Context) (*mel.State, error) {
	return s.svc.GetHeadState()
}

func (s *queryServer) GetState(ctx context.Context, parentChainBlockNumber uint64) (*mel.State, error) {
	return s.svc.GetState(parentChainBlockNumber)
}

func (s *queryServer) GetDelayedMessage(ctx context.Context, index uint64) (*mel.DelayedInboxMessage, error) {
	return s.svc.GetDelayedMessage(index)
}

func (s *queryServer) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) (mel.SequencerMessageResult, error) {
	data, blockHash, err := s.svc.GetSequencerMessageBytes(ctx, seqNum)
	if err != nil {
		return mel.SequencerMessageResult{}, err
	}
	return mel.SequencerMessageResult{Data: data, BlockHash: blockHash}, nil
}

func (s *queryServer) FindInboxBatchContainingMessage(ctx context.Context, pos arbutil.MessageIndex) (mel.FindInboxBatchResult, error) {
	seqNum, found, err := s.svc.FindInboxBatchContainingMessage(pos)
	if err != nil {
		return mel.FindInboxBatchResult{}, err
	}
	return mel.FindInboxBatchResult{SeqNum: seqNum, Found: found}, nil
}

func (s *queryServer) FinalizedDelayedMessageAtPosition(ctx context.Context, finalizedBlock uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64) (mel.FinalizedDelayedResult, error) {
	msg, acc, pcbn, err := s.svc.FinalizedDelayedMessageAtPosition(ctx, finalizedBlock, lastDelayedAccumulator, requestedPosition)
	if errors.Is(err, mel.ErrDelayedMessageNotYetFinalized) {
		return mel.FinalizedDelayedResult{ParentChainBlockNumber: pcbn, NotYetFinalized: true}, nil
	}
	if err != nil {
		return mel.FinalizedDelayedResult{}, err
	}
	return mel.FinalizedDelayedResult{Message: msg, AfterInboxAcc: acc, ParentChainBlockNumber: pcbn}, nil
}

func (s *queryServer) CaughtUp(ctx context.Context) (bool, error) {
	select {
	case <-s.svc.CaughtUp():
		return true, nil
	default:
		return false, nil
	}
}

// startServer registers the "meldataprovider" query server and "nitromelconsumer" node-side
// server (with no reorg notifier) on an in-process HTTP RPC server and returns its URL.
func startServer(t *testing.T, provider mel.MELNative, consumer mel.MessageConsumer) string {
	return startServerWithReorg(t, provider, consumer, nil)
}

// startServerWithReorg is like startServer but wires the node-side server's reorg notifier
// channel (the node's melReorgDetector), which ReorgedToParentChainBlock feeds.
func startServerWithReorg(t *testing.T, provider mel.MELNative, consumer mel.MessageConsumer, reorgNotifier chan<- uint64) string {
	t.Helper()
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName(mel.RPCNamespace, &queryServer{svc: provider}))
	require.NoError(t, srv.RegisterName(mel.ConsumerRPCNamespace, melrpcserver.NewServer(consumer, reorgNotifier)))
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts.URL
}

func clientConfigFetcher(url string) rpcclient.ClientConfigFetcher {
	cfg := &rpcclient.ClientConfig{URL: url, Timeout: 5 * time.Second}
	return func() *rpcclient.ClientConfig { return cfg }
}

// pushClient is a local stand-in for the external MEL provider's message-push RPC client
// (which lives outside this repo): it implements mel.MessageConsumer by calling the
// nitromelconsumer sink over JSON-RPC.
type pushClient struct{ rc *rpc.Client }

var _ mel.MessageConsumer = (*pushClient)(nil)

func (p *pushClient) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	return p.rc.CallContext(ctx, nil, mel.ConsumerRPCNamespace+"_pushMessages", firstMsgIdx, messages)
}

// reorgClient is a local stand-in for the external MEL provider notifying the node that it
// reorged, by calling the nitromelconsumer server over JSON-RPC.
type reorgClient struct{ rc *rpc.Client }

func (r *reorgClient) ReorgedToParentChainBlock(ctx context.Context, parentChainBlockNumber uint64) error {
	return r.rc.CallContext(ctx, nil, mel.ConsumerRPCNamespace+"_reorgedToParentChainBlock", parentChainBlockNumber)
}

func TestMELQueryClientRoundTrips(t *testing.T) {
	headState := &mel.State{
		Version:                3,
		ParentChainId:          42161,
		ParentChainBlockNumber: 9000,
		BatchCount:             7,
		MsgCount:               100,
		DelayedMessagesSeen:    4,
		DelayedMessageInboxAcc: common.HexToHash("0xaa"),
	}
	provider := &fakeProvider{
		batchCount:       7,
		headState:        headState,
		stateByBlock:     map[uint64]*mel.State{9000: headState},
		delayed:          map[uint64]*mel.DelayedInboxMessage{0: {Message: makeL1Msg(1), ParentChainBlockNumber: 11}},
		notYetFinalized:  5,
		notFinalizedPCBN: 8888,
	}
	url := startServer(t, provider, &fakeConsumer{})

	client := melrpcclient.NewClient(clientConfigFetcher(url), nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, client.Start(ctx))
	defer client.StopAndWait()

	// Scalar.
	bc, err := client.GetBatchCount()
	require.NoError(t, err)
	require.Equal(t, uint64(7), bc)

	// State: hashes must match across the wire.
	gotHead, err := client.GetHeadState()
	require.NoError(t, err)
	require.Equal(t, headState.Hash(), gotHead.Hash())
	require.Equal(t, headState.Hash(), gotHead.Hash())

	gotState, err := client.GetState(9000)
	require.NoError(t, err)
	require.Equal(t, headState.Hash(), gotState.Hash())

	// Delayed message.
	dm, err := client.GetDelayedMessage(0)
	require.NoError(t, err)
	require.Equal(t, provider.delayed[0].AfterInboxAcc(), dm.AfterInboxAcc())

	// Tuple return.
	data, blockHash, err := client.GetSequencerMessageBytes(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, data)
	require.Equal(t, common.HexToHash("0xfeed"), blockHash)

	// Tuple (found / not-found).
	seqNum, found, err := client.FindInboxBatchContainingMessage(10)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(3), seqNum)
	_, found, err = client.FindInboxBatchContainingMessage(1000)
	require.NoError(t, err)
	require.False(t, found)

	// Finalized delayed message: the happy path returns the message.
	msg, _, pcbn, err := client.FinalizedDelayedMessageAtPosition(ctx, 9000, common.Hash{}, 0)
	require.NoError(t, err)
	require.NotNil(t, msg)
	require.Equal(t, uint64(11), pcbn)

	// Not-yet-finalized: the sentinel must reconstruct AND ParentChainBlockNumber must survive,
	// since the delayed sequencer relies on it even on the error path.
	_, _, pcbn, err = client.FinalizedDelayedMessageAtPosition(ctx, 9000, common.Hash{}, 5)
	require.ErrorIs(t, err, mel.ErrDelayedMessageNotYetFinalized)
	require.Equal(t, uint64(8888), pcbn)

	// CaughtUp polls the dedicated provider RPC and closes the channel once the provider is caught up.
	select {
	case <-client.CaughtUp():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for CaughtUp")
	}
}

// fakeConsumer records pushed messages for the sink-service test.
type fakeConsumer struct {
	mu     sync.Mutex
	first  uint64
	pushed []*arbostypes.MessageWithMetadata
}

func (c *fakeConsumer) PushMessages(_ context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.first = firstMsgIdx
	c.pushed = append(c.pushed, messages...)
	return nil
}

func TestMELConsumerSinkReceivesPushedMessages(t *testing.T) {
	consumer := &fakeConsumer{}
	url := startServer(t, &fakeProvider{}, consumer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rc, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer rc.Close()
	pusher := &pushClient{rc: rc}

	msgs := []*arbostypes.MessageWithMetadata{
		{Message: makeL1Msg(1)},
		{Message: makeL1Msg(2)},
	}
	require.NoError(t, pusher.PushMessages(ctx, 42, msgs))

	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	require.Equal(t, uint64(42), consumer.first)
	require.Len(t, consumer.pushed, 2)
	require.Equal(t, msgs[0].Message.L2msg, consumer.pushed[0].Message.L2msg)
	require.Equal(t, msgs[1].Message.L2msg, consumer.pushed[1].Message.L2msg)
}

func TestMELSinkReorgNotification(t *testing.T) {
	// melReorgDetector stand-in; buffered like the real node-side channel.
	reorgCh := make(chan uint64, 8)
	url := startServerWithReorg(t, &fakeProvider{}, &fakeConsumer{}, reorgCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rc, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer rc.Close()
	notifier := &reorgClient{rc: rc}

	require.NoError(t, notifier.ReorgedToParentChainBlock(ctx, 4242))
	select {
	case got := <-reorgCh:
		require.Equal(t, uint64(4242), got)
	case <-time.After(5 * time.Second):
		t.Fatal("reorg notification did not reach the melReorgDetector channel")
	}
}

func TestMELSinkReorgNotificationNilNotifier(t *testing.T) {
	// A consumer node without a MEL validator has a nil reorg channel; the call must be a
	// no-op that returns nil without blocking.
	url := startServerWithReorg(t, &fakeProvider{}, &fakeConsumer{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rc, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer rc.Close()
	notifier := &reorgClient{rc: rc}

	require.NoError(t, notifier.ReorgedToParentChainBlock(ctx, 1))
}
