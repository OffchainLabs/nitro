// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package data_streaming

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/util/rpcclient"
)

const (
	DefaultHttpBodyLimit = 5 * 1024 * 1024 // Taken from go-ethereum http.defaultBodyLimit
	TestHttpBodyLimit    = 1024
)

// lint:require-exhaustive-initialization
type DataStreamerConfig struct {
	MaxStoreChunkBodySize int                     `koanf:"max-store-chunk-body-size"`
	RpcMethods            DataStreamingRPCMethods `koanf:"rpc-methods"`
}

func DefaultDataStreamerConfig(rpcMethods DataStreamingRPCMethods) DataStreamerConfig {
	return DataStreamerConfig{
		MaxStoreChunkBodySize: DefaultHttpBodyLimit,
		RpcMethods:            rpcMethods,
	}
}

func TestDataStreamerConfig(rpcMethods DataStreamingRPCMethods) DataStreamerConfig {
	return DataStreamerConfig{
		MaxStoreChunkBodySize: TestHttpBodyLimit,
		RpcMethods:            rpcMethods,
	}
}

func DataStreamerConfigAddOptions(prefix string, f *pflag.FlagSet, defaultRpcMethods DataStreamingRPCMethods) {
	f.Int(prefix+".max-store-chunk-body-size", DefaultHttpBodyLimit, "maximum HTTP body size for chunked store requests")
	DataStreamingRPCMethodsAddOptions(prefix+".rpc-methods", f, defaultRpcMethods)
}

// DataStreamer allows sending arbitrarily big payloads with JSON RPC. It follows a simple chunk-based protocol.
// lint:require-exhaustive-initialization
type DataStreamer[Result any] struct {
	rpcClient  *rpcclient.RpcClient
	chunkSize  uint64
	dataSigner *PayloadSigner
	rpcMethods DataStreamingRPCMethods
}

// DataStreamingRPCMethods configuration specifies names of the protocol's RPC methods on the server side.
// lint:require-exhaustive-initialization
type DataStreamingRPCMethods struct {
	StartStream    string `koanf:"start-stream"`
	StreamChunk    string `koanf:"stream-chunk"`
	FinalizeStream string `koanf:"finalize-stream"`
}

func DataStreamingRPCMethodsAddOptions(prefix string, f *pflag.FlagSet, defaultRpcMethods DataStreamingRPCMethods) {
	f.String(prefix+".start-stream", defaultRpcMethods.StartStream, "name of the RPC method to start a chunked data stream")
	f.String(prefix+".stream-chunk", defaultRpcMethods.StreamChunk, "name of the RPC method to send a chunk of data")
	f.String(prefix+".finalize-stream", defaultRpcMethods.FinalizeStream, "name of the RPC method to finalize a chunked data stream")
}

func NewDataStreamer[T any](config DataStreamerConfig, dataSigner *PayloadSigner, rpcClient *rpcclient.RpcClient) (*DataStreamer[T], error) {
	chunkSize, err := calculateEffectiveChunkSize(config.MaxStoreChunkBodySize, config.RpcMethods)
	if err != nil {
		return nil, err
	}

	if dataSigner == nil {
		return nil, errors.New("dataSigner must not be nil")
	}

	return &DataStreamer[T]{
		rpcClient:  rpcClient,
		chunkSize:  chunkSize,
		dataSigner: dataSigner,
		rpcMethods: config.RpcMethods,
	}, nil
}

func calculateEffectiveChunkSize(maxStoreChunkBodySize int, rpcMethods DataStreamingRPCMethods) (uint64, error) {
	jsonOverhead := len("{\"jsonrpc\":\"2.0\",\"id\":4294967295,\"method\":\"\",\"params\":[\"\"]}") + len(rpcMethods.StreamChunk)
	chunkSize := (maxStoreChunkBodySize - jsonOverhead - 512 /* headers */) / 2
	if chunkSize <= 0 {
		return 0, fmt.Errorf("max-store-chunk-body-size %d doesn't leave enough room for chunk payload", maxStoreChunkBodySize)
	}
	return uint64(chunkSize), nil
}

// StreamData sends arbitrarily long byte sequence to the receiver using a simple chunking-based protocol.
func (ds *DataStreamer[Result]) StreamData(ctx context.Context, data []byte, timeout uint64) (*Result, error) {
	params := newStreamParams(uint64(len(data)), ds.chunkSize, timeout)

	messageId, err := ds.startStream(ctx, params)
	if err != nil {
		return nil, err
	}

	if err := ds.doStream(ctx, data, messageId, params); err != nil {
		return nil, err
	}

	return ds.finalizeStream(ctx, messageId)
}

func (ds *DataStreamer[Result]) startStream(ctx context.Context, params streamParams) (MessageId, error) {
	payloadSignature, err := ds.sign(nil, params.timestamp, params.nChunks, ds.chunkSize, params.dataLen, params.timeout)
	if err != nil {
		return 0, err
	}

	var result StartStreamingResult
	err = ds.rpcClient.CallContext(
		ctx,
		&result,
		ds.rpcMethods.StartStream,
		hexutil.Uint64(params.timestamp),
		hexutil.Uint64(params.nChunks),
		hexutil.Uint64(ds.chunkSize),
		hexutil.Uint64(params.dataLen),
		hexutil.Uint64(params.timeout),
		hexutil.Bytes(payloadSignature))
	return MessageId(result.MessageId), err
}

func (ds *DataStreamer[Result]) doStream(ctx context.Context, data []byte, messageId MessageId, params streamParams) error {
	chunkRoutines := new(errgroup.Group)
	for i, chunkData := range slices.Collect(slices.Chunk(data, int(ds.chunkSize))) { //nolint:gosec
		chunkRoutines.Go(func() error {
			return ds.sendChunk(ctx, messageId, uint64(i), chunkData) //nolint:gosec
		})
	}
	return chunkRoutines.Wait()
}

func (ds *DataStreamer[Result]) sendChunk(ctx context.Context, messageId MessageId, chunkId uint64, chunkData []byte) error {
	payloadSignature, err := ds.sign(chunkData, uint64(messageId), chunkId)
	if err != nil {
		return err
	}
	return ds.rpcClient.CallContext(ctx, nil, ds.rpcMethods.StreamChunk, hexutil.Uint64(messageId), hexutil.Uint64(chunkId), hexutil.Bytes(chunkData), hexutil.Bytes(payloadSignature))
}

func (ds *DataStreamer[Result]) finalizeStream(ctx context.Context, messageId MessageId) (result *Result, err error) {
	payloadSignature, err := ds.sign(nil, uint64(messageId))
	if err != nil {
		return nil, err
	}
	err = ds.rpcClient.CallContext(ctx, &result, ds.rpcMethods.FinalizeStream, hexutil.Uint64(messageId), hexutil.Bytes(payloadSignature))
	return
}

func (ds *DataStreamer[Result]) sign(bytes []byte, extras ...uint64) ([]byte, error) {
	return ds.dataSigner.signPayload(bytes, extras...)
}

// lint:require-exhaustive-initialization
type streamParams struct {
	timestamp, nChunks, lastChunkSize, dataLen, timeout uint64
}

func newStreamParams(dataLen, chunkSize, timeout uint64) streamParams {
	nChunks := (dataLen + chunkSize - 1) / chunkSize
	lastChunkSize := (dataLen-1)%chunkSize + 1

	return streamParams{
		// #nosec G115
		timestamp:     uint64(time.Now().Unix()),
		nChunks:       nChunks,
		lastChunkSize: lastChunkSize,
		dataLen:       dataLen,
		timeout:       timeout,
	}
}
