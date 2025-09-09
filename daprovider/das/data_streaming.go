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
	"golang.org/x/sync/errgroup"
)

// DataStreamer allows sending arbitrarily big payloads with JSON RPC. It follows a simple chunk-based protocol.
type DataStreamer struct {
	// rpcClient is the underlying client for making RPC calls to the receiver.
	rpcClient *rpc.Client
	// chunkSize is the preconfigured size limit on a single data chunk to be sent.
	chunkSize uint64
	// dataSigner is used for sender authentication during the protocol.
	dataSigner signature.DataSignerFunc
}

// NewDataStreamer creates a new DataStreamer instance.
//
// Requirements:
//   - connecting to `url` must succeed;
//   - `maxStoreChunkBodySize` must be big enough (it should cover `sendChunkJSONBoilerplate` and leave some space for the data);
//   - `dataSigner` must not be nil;
//
// otherwise an `error` is returned.
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

// JSON request template for the `"das_sendChunked"` RPC method.
const sendChunkJSONBoilerplate = "{\"jsonrpc\":\"2.0\",\"id\":4294967295,\"method\":\"das_sendChunked\",\"params\":[\"\"]}"

func calculateEffectiveChunkSize(maxStoreChunkBodySize int) (uint64, error) {
	chunkSize := (maxStoreChunkBodySize - len(sendChunkJSONBoilerplate) - 512 /* headers */) / 2
	if chunkSize <= 0 {
		return 0, fmt.Errorf("max-store-chunk-body-size %d doesn't leave enough room for chunk payload", maxStoreChunkBodySize)
	}
	return uint64(chunkSize), nil
}

// StreamData sends arbitrarily long byte sequence to the receiver using a simple chunking-based protocol.
func (ds *DataStreamer) StreamData(ctx context.Context, data []byte, timeout uint64) (storeResult *StoreResult, err error) {
	params := newStreamParams(uint64(len(data)), ds.chunkSize, timeout)

	startReqSig, err := ds.generateStartReqSignature(params)
	if err != nil {
		return nil, err
	}

	batchId, err := ds.startStream(ctx, startReqSig, params)
	if err != nil {
		return nil, err
	}

	if err := ds.doStream(ctx, data, batchId, params); err != nil {
		return nil, err
	}

	finalReqSig, err := ds.generateFinalReqSignature(batchId)
	if err != nil {
		return nil, err
	}

	return ds.finalizeStream(ctx, finalReqSig, batchId)
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

func (ds *DataStreamer) doStream(ctx context.Context, data []byte, batchId hexutil.Uint64, params streamParams) error {
	chunkRoutines := new(errgroup.Group)
	for i := uint64(0); i < params.nChunks; i++ {
		startIndex := i * ds.chunkSize
		endIndex := (i + 1) * ds.chunkSize
		if endIndex > params.dataLen {
			endIndex = params.dataLen
		}
		chunkData := data[startIndex:endIndex]

		chunkRoutines.Go(func() error {
			return ds.sendChunk(ctx, batchId, i, chunkData)
		})
	}
	return chunkRoutines.Wait()
}

func (ds *DataStreamer) sendChunk(ctx context.Context, batchId hexutil.Uint64, chunkId uint64, chunkData []byte) error {
	chunkReqSig, err := ds.generateChunkReqSignature(chunkData, uint64(batchId), chunkId)
	if err != nil {
		return err
	}

	err = ds.rpcClient.CallContext(ctx, nil, "das_sendChunk", batchId, hexutil.Uint64(chunkId), hexutil.Bytes(chunkData), hexutil.Bytes(chunkReqSig))
	if err != nil {
		rpcClientSendChunkFailureGauge.Inc(1)
		return err
	}

	rpcClientSendChunkSuccessGauge.Inc(1)
	return nil
}

func (ds *DataStreamer) finalizeStream(ctx context.Context, finalReqSig []byte, batchId hexutil.Uint64) (storeResult *StoreResult, err error) {
	err = ds.rpcClient.CallContext(ctx, &storeResult, "das_commitChunkedStore", batchId, hexutil.Bytes(finalReqSig))
	return
}

func (ds *DataStreamer) generateStartReqSignature(params streamParams) ([]byte, error) {
	return applyDasSigner(ds.dataSigner, []byte{}, params.timestamp, params.nChunks, ds.chunkSize, params.dataLen, params.timeout)
}

func (ds *DataStreamer) generateChunkReqSignature(chunkData []byte, batchId, chunkId uint64) ([]byte, error) {
	return applyDasSigner(ds.dataSigner, chunkData, batchId, chunkId)
}

func (ds *DataStreamer) generateFinalReqSignature(batchId hexutil.Uint64) ([]byte, error) {
	return applyDasSigner(ds.dataSigner, []byte{}, uint64(batchId))
}

type streamParams struct {
	timestamp, nChunks, lastChunkSize, dataLen, timeout uint64
}

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
