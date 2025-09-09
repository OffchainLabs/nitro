// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dasutil

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/consensys/gnark-crypto/signature"
	"github.com/ethereum/go-ethereum/rpc"
)

type DataStreamer struct {
	rpcClient  *rpc.Client
	chunkSize  uint64
	dataSigner signature.Signer
}

func NewDataStreamer(url string, maxStoreChunkBodySize int, dataSigner signature.Signer) (*DataStreamer, error) {
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
	params := newStreamParams(uint64(len(data)), ds.chunkSize)
}

type streamParams struct {
	timestamp, nChunks, lastChunkSize, dataLen uint64
}

// todo chunksize and datalen must be > 0
func newStreamParams(dataLen, chunkSize uint64) streamParams {
	nChunks := (dataLen + chunkSize - 1) / chunkSize
	lastChunkSize := (dataLen-1)%chunkSize + 1

	return streamParams{
		timestamp:     uint64(time.Now().Unix()),
		nChunks:       nChunks,
		lastChunkSize: lastChunkSize,
		dataLen:       dataLen,
	}
}
