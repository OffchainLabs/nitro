// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/util/signature"
)

type DataStreamer struct {
	rpcClient  *rpc.Client
	chunkSize  uint64
	dataSigner signature.DataSignerFunc
}

func NewDataStreamer(url string, maxStoreChunkBodySize int, dataSigner signature.DataSignerFunc) (*DataStreamer, error) {
	rpcClient, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}

	chunkSize, err := calculateEffectiveChunkSize(maxStoreChunkBodySize)
	if err != nil {
		return nil, err
	}

	if dataSigner == nil {
		return nil, errors.New("dataSigner must not be nil")
	}

	return &DataStreamer{
		rpcClient:  rpcClient,
		chunkSize:  chunkSize,
		dataSigner: dataSigner,
	}, nil
}

const sendChunkJSONOverhead = "{\"jsonrpc\":\"2.0\",\"id\":4294967295,\"method\":\"das_sendChunked\",\"params\":[\"\"]}"

func calculateEffectiveChunkSize(maxStoreChunkBodySize int) (uint64, error) {
	chunkSize := (maxStoreChunkBodySize - len(sendChunkJSONOverhead) - 512 /* headers */) / 2
	if chunkSize <= 0 {
		return -1, fmt.Errorf("max-store-chunk-body-size %d doesn't leave enough room for chunk payload", maxStoreChunkBodySize)
	}
	return uint64(chunkSize), nil
}

func (ds *DataStreamer) StreamData(ctx context.Context, data []byte, timeout uint64) error {
	params := newStreamParams(uint64(len(data)), ds.chunkSize, timeout)

	startReqSig, err := ds.generateStartReqSignature(params)
	if err != nil {
		return err
	}

	batchId, err := ds.startStream(ctx, startReqSig, params)
	if err != nil {
		return err
	}

	if err := ds.doStream(ctx, batchId, params); err != nil {
		return err
	}
}

func (ds *DataStreamer) generateStartReqSignature(params streamParams) ([]byte, error) {
	return applyDasSigner(ds.dataSigner, []byte{}, params.timestamp, params.nChunks, ds.chunkSize, params.dataLen, params.timeout)
}

func (ds *DataStreamer) startStream(ctx context.Context, startReqSig []byte, params streamParams) (hexutil.Uint64, error) {
	var startChunkedStoreResult StartChunkedStoreResult
	err := ds.rpcClient.CallContext(
		ctx,
		&startChunkedStoreResult,
		"das_startChunkedStore",
		hexutil.Uint64(params.timestamp),
		hexutil.Uint64(params.nChunks),
		hexutil.Uint64(ds.chunkSize),
		hexutil.Uint64(params.dataLen),
		hexutil.Uint64(params.timeout),
		hexutil.Bytes(startReqSig))
	return startChunkedStoreResult.BatchId, err
}

type streamParams struct {
	timestamp, nChunks, lastChunkSize, dataLen, timeout uint64
}

// todo chunksize and datalen must be > 0
func newStreamParams(dataLen, chunkSize, timeout uint64) streamParams {
	nChunks := (dataLen + chunkSize - 1) / chunkSize
	lastChunkSize := (dataLen-1)%chunkSize + 1

	return streamParams{
		timestamp:     uint64(time.Now().Unix()),
		nChunks:       nChunks,
		lastChunkSize: lastChunkSize,
		dataLen:       dataLen,
		timeout:       timeout,
	}
}
