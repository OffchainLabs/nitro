// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package melrpcrunner_test exercises Nitro's MEL consumer-side JSON-RPC code end to end over
// an in-process HTTP RPC server: the "meldataprovider" query client (melrpcclient) and the
// "nitromelconsumer" node-side server (melrpcserver). The provider side (the "meldataprovider"
// query server and the provider->node RPC client) lives outside this repo, so it is stubbed
// locally here (queryServer, pushClient, reorgClient) — which also makes queryServer the
// executable spec of the wire shape the external provider has to implement.
package melrpcrunner_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
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
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

// Fixture values the fake provider returns; distinct per method so a wrong method name or a
// swapped argument cannot coincidentally produce the expected result.
const (
	fixtureBatchCount      = 7
	fixtureDelayedCount    = 4
	fixtureMsgCount        = 100
	fixtureSafeMsgCount    = 90
	fixtureFinalMsgCount   = 80
	fixtureHeadPCBN        = 9000
	fixtureDelayedPCBN     = 11
	fixtureNotFinalizedPos = 5
	fixtureNotFinalizedPC  = 8888
)

// argEcho encodes a method's arguments into a hash so assertions catch swapped or dropped args.
func argEcho(args ...uint64) common.Hash {
	n := new(big.Int)
	for _, a := range args {
		n.Mul(n, big.NewInt(1<<32))
		n.Add(n, new(big.Int).SetUint64(a))
	}
	return common.BigToHash(n)
}

// fakeProvider is a minimal mel.MELNative for the server side. It embeds the interface
// (nil) so only the methods exercised here need implementing; any unexpected call panics.
type fakeProvider struct {
	mel.MELNative

	headState        *mel.State
	stateByBlock     map[uint64]*mel.State
	delayed          map[uint64]*mel.DelayedInboxMessage
	notYetFinalized  uint64
	notFinalizedPCBN uint64
	reorgedTo        atomic.Uint64
}

func newFakeProvider() *fakeProvider {
	headState := &mel.State{
		Version:                3,
		ParentChainId:          42161,
		ParentChainBlockNumber: fixtureHeadPCBN,
		BatchCount:             fixtureBatchCount,
		MsgCount:               fixtureMsgCount,
		DelayedMessagesSeen:    fixtureDelayedCount,
		DelayedMessageInboxAcc: common.HexToHash("0xaa"),
	}
	return &fakeProvider{
		headState:        headState,
		stateByBlock:     map[uint64]*mel.State{fixtureHeadPCBN: headState},
		delayed:          map[uint64]*mel.DelayedInboxMessage{0: {Message: makeL1Msg(1), ParentChainBlockNumber: fixtureDelayedPCBN}},
		notYetFinalized:  fixtureNotFinalizedPos,
		notFinalizedPCBN: fixtureNotFinalizedPC,
	}
}

func (f *fakeProvider) GetBatchCount() (uint64, error) { return fixtureBatchCount, nil }

func (f *fakeProvider) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	return arbutil.MessageIndex(seqNum*10 + 1), nil
}

func (f *fakeProvider) GetBatchMetadata(seqNum uint64) (mel.BatchMetadata, error) {
	return mel.BatchMetadata{
		Accumulator:         argEcho(0xb0, seqNum),
		MessageCount:        arbutil.MessageIndex(seqNum*10 + 1),
		DelayedMessageCount: seqNum + 2,
		ParentChainBlock:    seqNum + 3,
	}, nil
}

func (f *fakeProvider) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	return argEcho(0xacc, seqNum), nil
}

func (f *fakeProvider) GetBatchParentChainBlock(seqNum uint64) (uint64, error) {
	return seqNum + 500, nil
}

func (f *fakeProvider) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	return argEcho(0xda, seqNum), nil
}

func (f *fakeProvider) GetDelayedCount() (uint64, error) { return fixtureDelayedCount, nil }

func (f *fakeProvider) GetMsgCount() (arbutil.MessageIndex, error) { return fixtureMsgCount, nil }

func (f *fakeProvider) GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return fixtureSafeMsgCount, nil
}

func (f *fakeProvider) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return fixtureFinalMsgCount, nil
}

func (f *fakeProvider) GetSyncProgress(ctx context.Context) (mel.MessageSyncProgress, error) {
	return mel.MessageSyncProgress{BatchSeen: fixtureBatchCount, BatchProcessed: fixtureBatchCount - 1, MsgCount: fixtureMsgCount}, nil
}

func (f *fakeProvider) SupportsPushingFinalityData() bool { return true }

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

func (f *fakeProvider) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	return []byte{0x0d, 0xe1, 0xa4, 0xed}, nil
}

func (f *fakeProvider) FindParentChainBlockContainingDelayed(ctx context.Context, index uint64) (uint64, error) {
	return index + 1000, nil
}

func (f *fakeProvider) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	return []byte{0xde, 0xad, 0xbe, 0xef}, common.HexToHash("0xfeed"), nil
}

func (f *fakeProvider) GetSequencerMessageBytesForParentBlock(ctx context.Context, seqNum uint64, parentChainBlock uint64) ([]byte, common.Hash, error) {
	return []byte{0x5e, 0x71}, argEcho(seqNum, parentChainBlock), nil
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

func (f *fakeProvider) ReorgTo(parentChainBlockNumber uint64) error {
	f.reorgedTo.Store(parentChainBlockNumber)
	return nil
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
// (ctx-first, result structs for multi-returns) that melrpcclient calls.
type queryServer struct{ svc mel.MELNative }

func (s *queryServer) GetBatchCount(ctx context.Context) (uint64, error) {
	return s.svc.GetBatchCount()
}

func (s *queryServer) GetBatchMessageCount(ctx context.Context, seqNum uint64) (arbutil.MessageIndex, error) {
	return s.svc.GetBatchMessageCount(seqNum)
}

func (s *queryServer) GetBatchMetadata(ctx context.Context, seqNum uint64) (mel.BatchMetadata, error) {
	return s.svc.GetBatchMetadata(seqNum)
}

func (s *queryServer) GetBatchAcc(ctx context.Context, seqNum uint64) (common.Hash, error) {
	return s.svc.GetBatchAcc(seqNum)
}

func (s *queryServer) GetBatchParentChainBlock(ctx context.Context, seqNum uint64) (uint64, error) {
	return s.svc.GetBatchParentChainBlock(seqNum)
}

func (s *queryServer) GetDelayedAcc(ctx context.Context, seqNum uint64) (common.Hash, error) {
	return s.svc.GetDelayedAcc(seqNum)
}

func (s *queryServer) GetDelayedCount(ctx context.Context) (uint64, error) {
	return s.svc.GetDelayedCount()
}

func (s *queryServer) GetMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return s.svc.GetMsgCount()
}

func (s *queryServer) GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return s.svc.GetSafeMsgCount(ctx)
}

func (s *queryServer) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return s.svc.GetFinalizedMsgCount(ctx)
}

func (s *queryServer) GetSyncProgress(ctx context.Context) (mel.MessageSyncProgress, error) {
	return s.svc.GetSyncProgress(ctx)
}

func (s *queryServer) SupportsPushingFinalityData(ctx context.Context) (bool, error) {
	return s.svc.SupportsPushingFinalityData(), nil
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

func (s *queryServer) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	return s.svc.GetDelayedMessageBytes(ctx, seqNum)
}

func (s *queryServer) FindParentChainBlockContainingDelayed(ctx context.Context, index uint64) (uint64, error) {
	return s.svc.FindParentChainBlockContainingDelayed(ctx, index)
}

func (s *queryServer) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) (mel.SequencerMessageResult, error) {
	data, blockHash, err := s.svc.GetSequencerMessageBytes(ctx, seqNum)
	if err != nil {
		return mel.SequencerMessageResult{}, err
	}
	return mel.SequencerMessageResult{Data: data, BlockHash: blockHash}, nil
}

func (s *queryServer) GetSequencerMessageBytesForParentBlock(ctx context.Context, seqNum uint64, parentChainBlock uint64) (mel.SequencerMessageResult, error) {
	data, blockHash, err := s.svc.GetSequencerMessageBytesForParentBlock(ctx, seqNum, parentChainBlock)
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

func (s *queryServer) ReorgTo(ctx context.Context, parentChainBlockNumber uint64) error {
	return s.svc.ReorgTo(parentChainBlockNumber)
}

func (s *queryServer) CaughtUp(ctx context.Context) (bool, error) {
	select {
	case <-s.svc.CaughtUp():
		return true, nil
	default:
		return false, nil
	}
}

// startRPC registers namespace->service pairs on an in-process HTTP RPC server and returns its URL.
func startRPC(t *testing.T, services map[string]interface{}) string {
	t.Helper()
	srv := rpc.NewServer()
	for namespace, service := range services {
		require.NoError(t, srv.RegisterName(namespace, service))
	}
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts.URL
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
	return startRPC(t, map[string]interface{}{
		mel.RPCNamespace:         &queryServer{svc: provider},
		mel.ConsumerRPCNamespace: melrpcserver.NewServer(consumer, reorgNotifier),
	})
}

func clientConfigFetcher(url string) rpcclient.ClientConfigFetcher {
	cfg := &rpcclient.ClientConfig{URL: url, Timeout: 5 * time.Second}
	return func() *rpcclient.ClientConfig { return cfg }
}

// startQueryClient starts a query client against url and tears it down with the test.
func startQueryClient(t *testing.T, url string) (*melrpcclient.Client, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	client := melrpcclient.NewClient(clientConfigFetcher(url), nil, nil)
	require.NoError(t, client.Start(ctx))
	t.Cleanup(client.StopAndWait)
	return client, ctx
}

// roundTripCase covers one MELNative method end to end. name must be the exact interface method
// name; melNativeCoverage below fails if a method has no case and no exemption.
type roundTripCase struct {
	name string
	run  func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider)
}

var roundTripCases = []roundTripCase{
	{"GetBatchCount", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetBatchCount()
		require.NoError(t, err)
		require.Equal(t, uint64(fixtureBatchCount), got)
	}},
	{"GetBatchMessageCount", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetBatchMessageCount(3)
		require.NoError(t, err)
		require.Equal(t, arbutil.MessageIndex(31), got)
	}},
	{"GetBatchMetadata", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetBatchMetadata(3)
		require.NoError(t, err)
		want, err := p.GetBatchMetadata(3)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}},
	{"GetBatchAcc", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetBatchAcc(3)
		require.NoError(t, err)
		require.Equal(t, argEcho(0xacc, 3), got)
	}},
	{"GetBatchParentChainBlock", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetBatchParentChainBlock(3)
		require.NoError(t, err)
		require.Equal(t, uint64(503), got)
	}},
	{"GetDelayedAcc", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetDelayedAcc(2)
		require.NoError(t, err)
		require.Equal(t, argEcho(0xda, 2), got)
	}},
	{"GetDelayedCount", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetDelayedCount()
		require.NoError(t, err)
		require.Equal(t, uint64(fixtureDelayedCount), got)
	}},
	{"GetDelayedMessage", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetDelayedMessage(0)
		require.NoError(t, err)
		// The accumulator is recomputed from the decoded message, so it pins the whole round trip.
		require.Equal(t, p.delayed[0].AfterInboxAcc(), got.AfterInboxAcc())
		require.Equal(t, uint64(fixtureDelayedPCBN), got.ParentChainBlockNumber)
	}},
	{"GetDelayedMessageBytes", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetDelayedMessageBytes(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, []byte{0x0d, 0xe1, 0xa4, 0xed}, got)
	}},
	{"FindInboxBatchContainingMessage", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		seqNum, found, err := c.FindInboxBatchContainingMessage(10)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(3), seqNum)
		// Not-found must survive as found=false rather than an error.
		_, found, err = c.FindInboxBatchContainingMessage(1000)
		require.NoError(t, err)
		require.False(t, found)
	}},
	{"FindParentChainBlockContainingDelayed", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.FindParentChainBlockContainingDelayed(ctx, 7)
		require.NoError(t, err)
		require.Equal(t, uint64(1007), got)
	}},
	{"GetSequencerMessageBytes", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		data, blockHash, err := c.GetSequencerMessageBytes(ctx, 0)
		require.NoError(t, err)
		require.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, data)
		require.Equal(t, common.HexToHash("0xfeed"), blockHash)
	}},
	{"GetSequencerMessageBytesForParentBlock", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		data, blockHash, err := c.GetSequencerMessageBytesForParentBlock(ctx, 4, 9001)
		require.NoError(t, err)
		require.Equal(t, []byte{0x5e, 0x71}, data)
		// Echoed args: catches the two uint64 params being swapped on the wire.
		require.Equal(t, argEcho(4, 9001), blockHash)
	}},
	{"FinalizedDelayedMessageAtPosition", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		msg, acc, pcbn, err := c.FinalizedDelayedMessageAtPosition(ctx, fixtureHeadPCBN, common.Hash{}, 0)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, p.delayed[0].AfterInboxAcc(), acc)
		require.Equal(t, uint64(fixtureDelayedPCBN), pcbn)

		// Not-yet-finalized: the sentinel must reconstruct AND ParentChainBlockNumber must survive,
		// since the delayed sequencer relies on it even on the error path.
		_, _, pcbn, err = c.FinalizedDelayedMessageAtPosition(ctx, fixtureHeadPCBN, common.Hash{}, fixtureNotFinalizedPos)
		require.ErrorIs(t, err, mel.ErrDelayedMessageNotYetFinalized)
		require.Equal(t, uint64(fixtureNotFinalizedPC), pcbn)
	}},
	{"GetMsgCount", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetMsgCount()
		require.NoError(t, err)
		require.Equal(t, arbutil.MessageIndex(fixtureMsgCount), got)
	}},
	{"GetSafeMsgCount", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetSafeMsgCount(ctx)
		require.NoError(t, err)
		require.Equal(t, arbutil.MessageIndex(fixtureSafeMsgCount), got)
	}},
	{"GetFinalizedMsgCount", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetFinalizedMsgCount(ctx)
		require.NoError(t, err)
		require.Equal(t, arbutil.MessageIndex(fixtureFinalMsgCount), got)
	}},
	{"GetSyncProgress", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetSyncProgress(ctx)
		require.NoError(t, err)
		require.Equal(t, mel.MessageSyncProgress{
			BatchSeen:      fixtureBatchCount,
			BatchProcessed: fixtureBatchCount - 1,
			MsgCount:       fixtureMsgCount,
		}, got)
	}},
	{"SupportsPushingFinalityData", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		require.True(t, c.SupportsPushingFinalityData())
	}},
	{"GetState", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetState(fixtureHeadPCBN)
		require.NoError(t, err)
		require.Equal(t, p.headState.Hash(), got.Hash())
	}},
	{"GetHeadState", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		got, err := c.GetHeadState()
		require.NoError(t, err)
		require.Equal(t, p.headState.Hash(), got.Hash())
	}},
	{"ReorgTo", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		// Returns only an error over the wire; the client decodes the null result into struct{}.
		require.NoError(t, c.ReorgTo(1234))
		require.Equal(t, uint64(1234), p.reorgedTo.Load())
	}},
	{"CaughtUp", func(t *testing.T, ctx context.Context, c *melrpcclient.Client, p *fakeProvider) {
		// Polls the dedicated provider RPC and closes the channel once the provider is caught up.
		select {
		case <-c.CaughtUp():
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for CaughtUp")
		}
	}},
}

func TestMELQueryClientRoundTrips(t *testing.T) {
	provider := newFakeProvider()
	url := startServer(t, provider, &fakeConsumer{})
	client, ctx := startQueryClient(t, url)

	for _, tc := range roundTripCases {
		t.Run(tc.name, func(t *testing.T) { tc.run(t, ctx, client, provider) })
	}
}

// TestMELQueryClientLocalMethods covers the two MELNative methods the client answers locally
// instead of forwarding.
func TestMELQueryClientLocalMethods(t *testing.T) {
	l1Reader := &headerreader.HeaderReader{}
	client := melrpcclient.NewClient(clientConfigFetcher("http://invalid.invalid"), nil, l1Reader)

	// GetL1Reader hands back the local node's reader; it cannot go over RPC.
	require.Same(t, l1Reader, client.GetL1Reader())

	// SetMessageConsumer is a no-op because the provider pushes into the sink, but a nil
	// consumer still has to be rejected rather than silently accepted.
	require.NoError(t, client.SetMessageConsumer(&fakeConsumer{}))
	require.Error(t, client.SetMessageConsumer(nil))
}

// TestMELNativeCoverage fails when a method is added to mel.MELNative without a round-trip case,
// so the wire shape cannot drift ahead of these tests.
func TestMELNativeCoverage(t *testing.T) {
	covered := map[string]bool{
		// Answered locally, covered by TestMELQueryClientLocalMethods.
		"GetL1Reader":        true,
		"SetMessageConsumer": true,
		// StopWaiter lifecycle, exercised by every test's client setup.
		"Start":       true,
		"StopAndWait": true,
		"Started":     true,
	}
	for _, tc := range roundTripCases {
		covered[tc.name] = true
	}

	melNative := reflect.TypeOf((*mel.MELNative)(nil)).Elem()
	for i := 0; i < melNative.NumMethod(); i++ {
		name := melNative.Method(i).Name
		require.True(t, covered[name], "mel.MELNative.%s has no round-trip case in roundTripCases", name)
	}
}

// sentinelServer returns a preset error from every method it serves, so the client's
// convertError rebuild of the sentinels is exercised over a real JSON-RPC hop.
type sentinelServer struct{ err error }

func (s *sentinelServer) CaughtUp(ctx context.Context) (bool, error) { return true, nil }

func (s *sentinelServer) FindParentChainBlockContainingDelayed(ctx context.Context, index uint64) (uint64, error) {
	return 0, s.err
}

func (s *sentinelServer) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	return nil, s.err
}

func (s *sentinelServer) FinalizedDelayedMessageAtPosition(ctx context.Context, finalizedBlock uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64) (mel.FinalizedDelayedResult, error) {
	return mel.FinalizedDelayedResult{}, s.err
}

func (s *sentinelServer) SupportsPushingFinalityData(ctx context.Context) (bool, error) {
	return true, s.err
}

// TestMELQueryClientRebuildsSentinelErrors covers convertError: JSON-RPC flattens errors to
// strings, so every sentinel callers branch on has to survive the hop, including when the
// provider wraps it with context.
func TestMELQueryClientRebuildsSentinelErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		serverErr error
		call      func(ctx context.Context, c *melrpcclient.Client) error
		want      error
	}{
		{
			name:      "FindDelayedNotImplemented",
			serverErr: mel.ErrFindDelayedNotImplementedByMEL,
			call: func(ctx context.Context, c *melrpcclient.Client) error {
				_, err := c.FindParentChainBlockContainingDelayed(ctx, 1)
				return err
			},
			want: mel.ErrFindDelayedNotImplementedByMEL,
		},
		{
			name: "PreimageNotFoundWrapped",
			// Wrapped as mel.State.delayedMsgPreimage does.
			serverErr: fmt.Errorf("%w: for hash: %s", mel.ErrDelayedMessagePreimageNotFound, common.HexToHash("0x1").Hex()),
			call: func(ctx context.Context, c *melrpcclient.Client) error {
				_, err := c.GetDelayedMessageBytes(ctx, 1)
				return err
			},
			want: mel.ErrDelayedMessagePreimageNotFound,
		},
		{
			name: "AccumulatorMismatchWrapped",
			// Wrapped as runner.FinalizedDelayedMessageAtPosition does.
			serverErr: fmt.Errorf("position %d: BeforeInboxAcc mismatch: %w", 3, mel.ErrDelayedAccumulatorMismatch),
			call: func(ctx context.Context, c *melrpcclient.Client) error {
				_, _, _, err := c.FinalizedDelayedMessageAtPosition(ctx, 1, common.Hash{}, 3)
				return err
			},
			want: mel.ErrDelayedAccumulatorMismatch,
		},
		{
			name: "NotYetFinalizedAsError",
			// A provider may return the sentinel as an error instead of the in-band flag.
			serverErr: mel.ErrDelayedMessageNotYetFinalized,
			call: func(ctx context.Context, c *melrpcclient.Client) error {
				_, _, _, err := c.FinalizedDelayedMessageAtPosition(ctx, 1, common.Hash{}, 3)
				return err
			},
			want: mel.ErrDelayedMessageNotYetFinalized,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url := startRPC(t, map[string]interface{}{mel.RPCNamespace: &sentinelServer{err: tc.serverErr}})
			client, ctx := startQueryClient(t, url)
			require.ErrorIs(t, tc.call(ctx, client), tc.want)
		})
	}
}

// TestMELQueryClientPassesThroughUnknownErrors pins that convertError only rewrites the sentinels
// and leaves everything else legible.
func TestMELQueryClientPassesThroughUnknownErrors(t *testing.T) {
	url := startRPC(t, map[string]interface{}{mel.RPCNamespace: &sentinelServer{err: errors.New("provider database is corrupt")}})
	client, ctx := startQueryClient(t, url)

	_, err := client.GetDelayedMessageBytes(ctx, 1)
	require.ErrorContains(t, err, "provider database is corrupt")
	require.NotErrorIs(t, err, mel.ErrDelayedMessagePreimageNotFound)
}

// TestMELQueryClientSupportsPushingFinalityDataFalseOnError pins the swallowed-error behavior:
// the method has no error return, so an unreachable provider degrades to false.
func TestMELQueryClientSupportsPushingFinalityDataFalseOnError(t *testing.T) {
	url := startRPC(t, map[string]interface{}{mel.RPCNamespace: &sentinelServer{err: errors.New("provider unavailable")}})
	client, _ := startQueryClient(t, url)

	require.False(t, client.SupportsPushingFinalityData())
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

func TestMELConsumerSinkReceivesPushedMessages(t *testing.T) {
	consumer := &fakeConsumer{}
	url := startServer(t, newFakeProvider(), consumer)

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
	url := startServerWithReorg(t, newFakeProvider(), &fakeConsumer{}, reorgCh)

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

// TestMELSinkReorgNotificationUnblocksOnCancel covers the ctx guard: when melReorgDetector
// cannot accept the send, a cancelled request must return instead of wedging the handler, and
// the abandoned notification must not surface later. Called directly rather than over RPC so
// the send is provably still parked when the ctx is cancelled.
func TestMELSinkReorgNotificationUnblocksOnCancel(t *testing.T) {
	const queuedPCBN = 7777
	for _, tc := range []struct {
		name        string
		newNotifier func() chan uint64
		queued      []uint64 // already in the channel, so it stays full
	}{
		{
			name:        "unbuffered",
			newNotifier: func() chan uint64 { return make(chan uint64) },
		},
		{
			name: "buffer full",
			newNotifier: func() chan uint64 {
				ch := make(chan uint64, 1)
				ch <- queuedPCBN
				return ch
			},
			queued: []uint64{queuedPCBN},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			notifier := tc.newNotifier()
			srv := melrpcserver.NewServer(&fakeConsumer{}, notifier)

			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)
			go func() { errCh <- srv.ReorgedToParentChainBlock(ctx, 4242) }()

			// Nothing drains the channel, so the handler must still be parked on the send.
			select {
			case err := <-errCh:
				t.Fatalf("returned before cancellation: %v", err)
			case <-time.After(50 * time.Millisecond):
			}

			cancel()
			select {
			case err := <-errCh:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(5 * time.Second):
				t.Fatal("ReorgedToParentChainBlock stayed blocked after its ctx was cancelled")
			}

			// The handler has returned, so no one can send: closing drains what is really there.
			close(notifier)
			var got []uint64
			for pcbn := range notifier {
				got = append(got, pcbn)
			}
			require.Equal(t, tc.queued, got, "cancelled notification must not be delivered late")
		})
	}
}

func TestMELSinkReorgNotificationNilNotifier(t *testing.T) {
	// A consumer node without a MEL validator has a nil reorg channel; the call must be a
	// no-op that returns nil without blocking.
	url := startServerWithReorg(t, newFakeProvider(), &fakeConsumer{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rc, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer rc.Close()
	notifier := &reorgClient{rc: rc}

	require.NoError(t, notifier.ReorgedToParentChainBlock(ctx, 1))
}
