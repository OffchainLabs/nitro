// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package data_streaming

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const (
	maxPendingMessages      = 10
	messageCollectionExpiry = 1 * time.Second
	maxStoreChunkBodySize   = 1024
	timeout                 = 10
	serverRPCRoot           = "datastreaming"
)

var rpcMethods = DataStreamingRPCMethods{
	StartStream:    serverRPCRoot + "_start",
	StreamChunk:    serverRPCRoot + "_chunk",
	FinalizeStream: serverRPCRoot + "_finish",
}

func TestDataStreaming_PositiveScenario(t *testing.T) {
	t.Run("Single sender, short message", func(t *testing.T) {
		testBasic(t, maxStoreChunkBodySize/2, 10, 1)
	})
	t.Run("Single sender, long message", func(t *testing.T) {
		testBasic(t, 2*maxStoreChunkBodySize, 50, 1)
	})
	t.Run("Many senders, long messages", func(t *testing.T) {
		testBasic(t, 10*maxStoreChunkBodySize, maxStoreChunkBodySize, maxPendingMessages)
	})
}

func TestDataStreaming_ServerIdempotency(t *testing.T) {
	ctx, streamer := prepareTestEnv(t, nil)
	message, chunks := getLongRandomMessage(streamer.chunkSize)
	redundancy := 3

	// ========== Implementation of streamer.StreamData that sends every chunk multiple times. ==========

	// 1. Start the protocol as usual
	params := newStreamParams(uint64(len(message)), streamer.chunkSize, timeout)
	messageId, err := streamer.startStream(ctx, params)
	testhelpers.RequireImpl(t, err)

	// 2. Send chunks with redundancy
	for i, chunkData := range chunks {
		for try := 0; try < redundancy; try++ {
			err = streamer.sendChunk(ctx, messageId, uint64(i), chunkData) //nolint:gosec
			testhelpers.RequireImpl(t, err)
		}
	}

	// 3. Ensure we can still finalize the protocol.
	result, err := streamer.finalizeStream(ctx, messageId)
	testhelpers.RequireImpl(t, err)
	require.Equal(t, message, ([]byte)(result.Message), "protocol resulted in an incorrect message")
}

func TestDataStreaming_ServerHaltsProtocolWhenObservesInconsistency(t *testing.T) {
	ctx, streamer := prepareTestEnv(t, nil)
	message, chunks := getLongRandomMessage(streamer.chunkSize)

	// ========== Implementation of streamer.StreamData that will repeat a chunk with different data. ==========

	// 1. Start the protocol as usual
	params := newStreamParams(uint64(len(message)), streamer.chunkSize, timeout)
	messageId, err := streamer.startStream(ctx, params)
	testhelpers.RequireImpl(t, err)

	// 2. Send chunks in a malicious way
	// 2.1 Send first chunk
	err = streamer.sendChunk(ctx, messageId, 0, chunks[0])
	testhelpers.RequireImpl(t, err)
	// 2.2 Send again the first chunk, but with different data
	err = streamer.sendChunk(ctx, messageId, 0, chunks[1])
	require.Error(t, err)
	// 2.3 Ensure that we cannot send next chunk
	err = streamer.sendChunk(ctx, messageId, 1, chunks[1])
	require.Error(t, err)
}

func TestDataStreaming_ServerAbortsProtocolAfterExpiry(t *testing.T) {
	ctx, streamer := prepareTestEnv(t, nil)
	message, chunks := getLongRandomMessage(streamer.chunkSize)

	// ========== Implementation of streamer.StreamData that wait too long before sending next message ==========

	// 1. Start the protocol as usual
	params := newStreamParams(uint64(len(message)), streamer.chunkSize, timeout)
	messageId, err := streamer.startStream(ctx, params)
	testhelpers.RequireImpl(t, err)

	// 2. Send first chunk
	err = streamer.sendChunk(ctx, messageId, 0, chunks[0])
	testhelpers.RequireImpl(t, err)

	// 3. Wait for long enough
	time.Sleep(messageCollectionExpiry * 2)

	// 4. Ensure that we cannot proceed with the protocol
	err = streamer.sendChunk(ctx, messageId, 1, chunks[1])
	require.Error(t, err)
}

func TestDataStreaming_ProtocolSucceedsEvenWithDelays(t *testing.T) {
	ctx, streamer := prepareTestEnv(t, nil)
	message, chunks := getLongRandomMessage(streamer.chunkSize)

	// ========== Implementation of streamer.StreamData that sends every message just before expiry ==========

	// 1. Start the protocol as usual
	params := newStreamParams(uint64(len(message)), streamer.chunkSize, timeout)
	messageId, err := streamer.startStream(ctx, params)
	testhelpers.RequireImpl(t, err)

	// 2. Send chunks with delay
	for i, chunkData := range chunks {
		time.Sleep(messageCollectionExpiry * 9 / 10)
		err = streamer.sendChunk(ctx, messageId, uint64(i), chunkData) //nolint:gosec
		testhelpers.RequireImpl(t, err)
	}

	// 3. Ensure we can still finalize the protocol.
	time.Sleep(messageCollectionExpiry * 9 / 10)
	result, err := streamer.finalizeStream(ctx, messageId)
	testhelpers.RequireImpl(t, err)
	require.Equal(t, message, ([]byte)(result.Message), "protocol resulted in an incorrect message")
}

var alreadyWentOffline = false

func TestDataStreaming_ClientRetriesWhenThereAreConnectionProblems(t *testing.T) {
	// Server 'goes offline' for a moment just before reading the second chunk
	ctx, streamer := prepareTestEnv(t, func(i uint64) error {
		if i == 1 && !alreadyWentOffline {
			alreadyWentOffline = true
			return errors.New("service unavailable")
		}
		return nil
	})
	message, _ := getLongRandomMessage(streamer.chunkSize)
	result, err := streamer.StreamData(ctx, message, timeout)
	testhelpers.RequireImpl(t, err)
	require.Equal(t, message, ([]byte)(result.Message), "protocol resulted in an incorrect message")
}

func testBasic(t *testing.T, messageSizeMean, messageSizeStdDev, concurrency int) {
	ctx, streamer := prepareTestEnv(t, nil)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			messageSize := int(rand.NormFloat64()*float64(messageSizeStdDev) + float64(messageSizeMean))

			message := testhelpers.RandomizeSlice(make([]byte, messageSize))
			result, err := streamer.StreamData(ctx, message, timeout)
			testhelpers.RequireImpl(t, err)
			require.Equal(t, message, ([]byte)(result.Message), "protocol resulted in an incorrect message")
		}()
	}
	wg.Wait()
}

func prepareTestEnv(t *testing.T, onChunkInjection func(uint64) error) (context.Context, *DataStreamer[ProtocolResult]) {
	ctx := context.Background()
	signer, verifier := prepareCrypto(t)
	serverUrl := launchServer(t, ctx, verifier, onChunkInjection)

	clientConfig := func() *rpcclient.ClientConfig { return &rpcclient.ClientConfig{URL: "http://" + serverUrl} }
	rpcClient := rpcclient.NewRpcClient(clientConfig, nil)
	err := rpcClient.Start(ctx)
	testhelpers.RequireImpl(t, err)

	streamer, err := NewDataStreamer[ProtocolResult](maxStoreChunkBodySize, DefaultPayloadSigner(signer), rpcClient, rpcMethods)
	testhelpers.RequireImpl(t, err)

	return ctx, streamer
}

func launchServer(t *testing.T, ctx context.Context, signatureVerifier *signature.Verifier, onChunkInjection func(uint64) error) string {
	rpcServer := rpc.NewServer()
	err := rpcServer.RegisterName(serverRPCRoot, &TestServer{
		dataStreamReceiver: NewDataStreamReceiver(DefaultPayloadVerifier(signatureVerifier), maxPendingMessages, messageCollectionExpiry, nil),
		onChunkInject:      onChunkInjection,
	})
	testhelpers.RequireImpl(t, err)

	listener, err := net.Listen("tcp", "localhost:0")
	testhelpers.RequireImpl(t, err)

	httpServer := &http.Server{Handler: rpcServer, ReadTimeout: genericconf.HTTPServerTimeoutConfigDefault.ReadTimeout}
	go func() {
		err = httpServer.Serve(listener)
		testhelpers.RequireImpl(t, err)
	}()
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()

	return listener.Addr().String()
}

func getLongRandomMessage(chunkSize uint64) ([]byte, [][]byte) {
	message := testhelpers.RandomizeSlice(make([]byte, maxStoreChunkBodySize))
	chunks := slices.Collect(slices.Chunk(message, int(chunkSize))) //nolint:gosec
	return message, chunks
}

// ======================================= Test server (wrapping the receiver part) ========================== //

// lint:require-exhaustive-initialization
type TestServer struct {
	dataStreamReceiver *DataStreamReceiver
	onChunkInject      func(uint64) error
}

func (server *TestServer) Start(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout hexutil.Uint64, sig hexutil.Bytes) (*StartStreamingResult, error) {
	return server.dataStreamReceiver.StartReceiving(ctx, uint64(timestamp), uint64(nChunks), uint64(chunkSize), uint64(totalSize), uint64(timeout), sig)
}

func (server *TestServer) Chunk(ctx context.Context, messageId, chunkId hexutil.Uint64, chunk hexutil.Bytes, sig hexutil.Bytes) error {
	if server.onChunkInject != nil {
		maybeInjection := server.onChunkInject(uint64(chunkId))
		if maybeInjection != nil {
			return maybeInjection
		}
	}
	return server.dataStreamReceiver.ReceiveChunk(ctx, MessageId(messageId), uint64(chunkId), chunk, sig)
}

func (server *TestServer) Finish(ctx context.Context, messageId hexutil.Uint64, sig hexutil.Bytes) (*ProtocolResult, error) {
	message, _, _, err := server.dataStreamReceiver.FinalizeReceiving(ctx, MessageId(messageId), sig)
	return &ProtocolResult{Message: message}, err
}

// lint:require-exhaustive-initialization
type ProtocolResult struct {
	Message hexutil.Bytes `json:"message"`
}
