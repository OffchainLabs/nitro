// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package data_streaming

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// DataStreamer allows sending arbitrarily big payloads with JSON RPC. It follows a simple chunk-based protocol.
// lint:require-exhaustive-initialization
type DataStreamer[Result any] struct {
	// rpcClient is the underlying client for making RPC calls to the receiver.
	rpcClient *rpc.Client
	// chunkSize is the preconfigured size limit on a single data chunk to be sent.
	chunkSize uint64
	// dataSigner is used for sender authentication during the protocol.
	dataSigner PayloadSigner
	// rpcMethods define the actual server API
	rpcMethods DataStreamingRPCMethods
}

// DataStreamingRPCMethods configuration specifies names of the protocol's RPC methods on the server side.
// lint:require-exhaustive-initialization
type DataStreamingRPCMethods struct {
	StartStream, StreamChunk, FinalizeStream string
}

// NewDataStreamer creates a new DataStreamer instance.
//
// Requirements:
//   - connecting to `url` must succeed;
//   - `maxStoreChunkBodySize` must be big enough (it should cover `sendChunkJSONBoilerplate` and leave some space for the data);
//   - `dataSigner` must not be nil;
//
// otherwise an `error` is returned.
func NewDataStreamer[T any](url string, maxStoreChunkBodySize int, dataSigner PayloadSigner, rpcMethods DataStreamingRPCMethods) (*DataStreamer[T], error) {
	rpcClient, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}

	chunkSize, err := calculateEffectiveChunkSize(maxStoreChunkBodySize, rpcMethods)
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
		rpcMethods: rpcMethods,
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
	for i := uint64(0); i < params.nChunks; i++ {
		startIndex := i * ds.chunkSize
		endIndex := (i + 1) * ds.chunkSize
		if endIndex > params.dataLen {
			endIndex = params.dataLen
		}
		chunkData := data[startIndex:endIndex]

		chunkRoutines.Go(func() error {
			return ds.sendChunk(ctx, messageId, i, chunkData)
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
	return ds.dataSigner.SignPayload(bytes, extras...)
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
