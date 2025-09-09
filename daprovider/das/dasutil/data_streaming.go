package dasutil

import (
	"errors"
	"fmt"

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
