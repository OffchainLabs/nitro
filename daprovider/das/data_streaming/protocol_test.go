// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package data_streaming

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const (
	maxPendingMessages      = 10
	messageCollectionExpiry = time.Duration(2 * time.Second)
	maxStoreChunkBodySize   = 1024
	timeout                 = 10
	serverRPCRoot           = "datastreaming"
)

var rpcMethods = DataStreamingRPCMethods{
	StartStream:    serverRPCRoot + "_start",
	StreamChunk:    serverRPCRoot + "_chunk",
	FinalizeStream: serverRPCRoot + "_finish",
}

func TestDataStreamingProtocol(t *testing.T) {
	t.Run("Single sender, short message", func(t *testing.T) {
		test(t, maxStoreChunkBodySize/2, 10, 1)
	})
	t.Run("Single sender, long message", func(t *testing.T) {
		test(t, 2*maxStoreChunkBodySize, 50, 1)
	})
	t.Run("Many senders, long messages", func(t *testing.T) {
		test(t, 10*maxStoreChunkBodySize, maxStoreChunkBodySize, maxPendingMessages)
	})
	t.Run("Single sender, exact multiple-of-chunk size", func(t *testing.T) {
    	chunkSize, err := calculateEffectiveChunkSize(maxStoreChunkBodySize, rpcMethods)
    	testhelpers.RequireImpl(t, err)
    	test(t, int(chunkSize*3), 0, 1)
    })
}

func test(t *testing.T, messageSizeMean, messageSizeStdDev, concurrency int) {
	ctx := context.Background()
	signer, verifier := prepareCrypto(t)
	serverUrl := launchServer(t, ctx, verifier)

	streamer, err := NewDataStreamer[ProtocolResult]("http://"+serverUrl, maxStoreChunkBodySize, DefaultPayloadSigner(signer), rpcMethods)
	testhelpers.RequireImpl(t, err)

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

func prepareCrypto(t *testing.T) (signature.DataSignerFunc, *signature.Verifier) {
	privateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)

	signatureVerifierConfig := signature.VerifierConfig{
		AllowedAddresses: []string{crypto.PubkeyToAddress(privateKey.PublicKey).Hex()},
		AcceptSequencer:  false,
		Dangerous:        signature.DangerousVerifierConfig{AcceptMissing: false},
	}
	verifier, err := signature.NewVerifier(&signatureVerifierConfig, nil)
	testhelpers.RequireImpl(t, err)

	signer := signature.DataSignerFromPrivateKey(privateKey)
	return signer, verifier
}

func launchServer(t *testing.T, ctx context.Context, signatureVerifier *signature.Verifier) string {
	rpcServer := rpc.NewServer()
	err := rpcServer.RegisterName(serverRPCRoot, &TestServer{
		dataStreamReceiver: NewDataStreamReceiver(DefaultPayloadVerifier(signatureVerifier), maxPendingMessages, messageCollectionExpiry, nil),
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

// Ensure that attempting to start stream with totalSize == 0 is rejected via protocol entrypoint
func TestStartStreamRejectsZeroTotalSize(t *testing.T) {
    ctx := context.Background()
    signer, verifier := prepareCrypto(t)
    serverUrl := launchServer(t, ctx, verifier)

    rpcClient, err := rpc.Dial("http://" + serverUrl)
    testhelpers.RequireImpl(t, err)

    chunkSize, err := calculateEffectiveChunkSize(maxStoreChunkBodySize, rpcMethods)
    testhelpers.RequireImpl(t, err)

    // Prepare signed payload with totalSize == 0
    // #nosec G115
    timestamp := uint64(time.Now().Unix())
    nChunks := uint64(1)
    totalSize := uint64(0)
    payloadSigner := DefaultPayloadSigner(signer)
    sig, err := payloadSigner.signPayload(nil, timestamp, nChunks, chunkSize, totalSize, timeout)
    testhelpers.RequireImpl(t, err)

    var result StartStreamingResult
    err = rpcClient.CallContext(
        ctx,
        &result,
        rpcMethods.StartStream,
        hexutil.Uint64(timestamp),
        hexutil.Uint64(nChunks),
        hexutil.Uint64(chunkSize),
        hexutil.Uint64(totalSize),
        hexutil.Uint64(timeout),
        hexutil.Bytes(sig),
    )
    if err == nil {
        t.Fatalf("expected error when totalSize == 0, got nil")
    }
}

// ======================================= Test server (wrapping the receiver part) ========================== //

// lint:require-exhaustive-initialization
type TestServer struct {
	dataStreamReceiver *DataStreamReceiver
}

func (server *TestServer) Start(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout hexutil.Uint64, sig hexutil.Bytes) (*StartStreamingResult, error) {
	return server.dataStreamReceiver.StartReceiving(ctx, uint64(timestamp), uint64(nChunks), uint64(chunkSize), uint64(totalSize), uint64(timeout), sig)
}

func (server *TestServer) Chunk(ctx context.Context, messageId, chunkId hexutil.Uint64, chunk hexutil.Bytes, sig hexutil.Bytes) error {
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
