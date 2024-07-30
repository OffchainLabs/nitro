// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/signature"
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt      *rpc.Client
	url       string
	signer    signature.DataSignerFunc
	chunkSize uint64
}

func nilSigner(_ []byte) ([]byte, error) {
	return []byte{}, nil
}

const sendChunkJSONBoilerplate = "{\"jsonrpc\":\"2.0\",\"id\":4294967295,\"method\":\"das_sendChunked\",\"params\":[\"\"]}"

func NewDASRPCClient(target string, signer signature.DataSignerFunc, maxStoreChunkBodySize int) (*DASRPCClient, error) {
	clnt, err := rpc.Dial(target)
	if err != nil {
		return nil, err
	}
	if signer == nil {
		signer = nilSigner
	}

	// Byte arrays are encoded in base64
	chunkSize := (maxStoreChunkBodySize - len(sendChunkJSONBoilerplate) - 512 /* headers */) / 2
	if chunkSize <= 0 {
		return nil, fmt.Errorf("max-store-chunk-body-size %d doesn't leave enough room for chunk payload", maxStoreChunkBodySize)
	}

	return &DASRPCClient{
		clnt:      clnt,
		url:       target,
		signer:    signer,
		chunkSize: uint64(chunkSize),
	}, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64) (*daprovider.DataAvailabilityCertificate, error) {
	timestamp := uint64(time.Now().Unix())
	nChunks := uint64(len(message)) / c.chunkSize
	lastChunkSize := uint64(len(message)) % c.chunkSize
	if lastChunkSize > 0 {
		nChunks++
	} else {
		lastChunkSize = c.chunkSize
	}
	totalSize := uint64(len(message))

	startReqSig, err := applyDasSigner(c.signer, []byte{}, timestamp, nChunks, c.chunkSize, totalSize, timeout)
	if err != nil {
		return nil, err
	}

	var startChunkedStoreResult StartChunkedStoreResult
	if err := c.clnt.CallContext(ctx, &startChunkedStoreResult, "das_startChunkedStore", hexutil.Uint64(timestamp), hexutil.Uint64(nChunks), hexutil.Uint64(c.chunkSize), hexutil.Uint64(totalSize), hexutil.Uint64(timeout), hexutil.Bytes(startReqSig)); err != nil {
		if strings.Contains(err.Error(), "the method das_startChunkedStore does not exist") {
			return c.legacyStore(ctx, message, timeout)
		}
		return nil, err
	}
	batchId := uint64(startChunkedStoreResult.BatchId)

	g := new(errgroup.Group)
	for i := uint64(0); i < nChunks; i++ {
		var chunk []byte
		if i == nChunks-1 {
			chunk = message[i*c.chunkSize : i*c.chunkSize+lastChunkSize]
		} else {
			chunk = message[i*c.chunkSize : (i+1)*c.chunkSize]
		}

		inner := func(_i uint64, _chunk []byte) func() error {
			return func() error { return c.sendChunk(ctx, batchId, _i, _chunk) }
		}
		g.Go(inner(i, chunk))
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	finalReqSig, err := applyDasSigner(c.signer, []byte{}, uint64(startChunkedStoreResult.BatchId))
	if err != nil {
		return nil, err
	}

	var storeResult StoreResult
	if err := c.clnt.CallContext(ctx, &storeResult, "das_commitChunkedStore", startChunkedStoreResult.BatchId, hexutil.Bytes(finalReqSig)); err != nil {
		return nil, err
	}

	respSig, err := blsSignatures.SignatureFromBytes(storeResult.Sig)
	if err != nil {
		return nil, err
	}

	return &daprovider.DataAvailabilityCertificate{
		DataHash:    common.BytesToHash(storeResult.DataHash),
		Timeout:     uint64(storeResult.Timeout),
		SignersMask: uint64(storeResult.SignersMask),
		Sig:         respSig,
		KeysetHash:  common.BytesToHash(storeResult.KeysetHash),
		Version:     byte(storeResult.Version),
	}, nil
}

func (c *DASRPCClient) sendChunk(ctx context.Context, batchId, i uint64, chunk []byte) error {
	chunkReqSig, err := applyDasSigner(c.signer, chunk, batchId, i)
	if err != nil {
		return err
	}

	if err := c.clnt.CallContext(ctx, nil, "das_sendChunk", hexutil.Uint64(batchId), hexutil.Uint64(i), hexutil.Bytes(chunk), hexutil.Bytes(chunkReqSig)); err != nil {
		return err
	}
	return nil
}

func (c *DASRPCClient) legacyStore(ctx context.Context, message []byte, timeout uint64) (*daprovider.DataAvailabilityCertificate, error) {
	log.Trace("das.DASRPCClient.Store(...)", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "this", *c)

	reqSig, err := applyDasSigner(c.signer, message, timeout)
	if err != nil {
		return nil, err
	}

	var ret StoreResult
	if err := c.clnt.CallContext(ctx, &ret, "das_store", hexutil.Bytes(message), hexutil.Uint64(timeout), hexutil.Bytes(reqSig)); err != nil {
		return nil, err
	}
	respSig, err := blsSignatures.SignatureFromBytes(ret.Sig)
	if err != nil {
		return nil, err
	}
	return &daprovider.DataAvailabilityCertificate{
		DataHash:    common.BytesToHash(ret.DataHash),
		Timeout:     uint64(ret.Timeout),
		SignersMask: uint64(ret.SignersMask),
		Sig:         respSig,
		KeysetHash:  common.BytesToHash(ret.KeysetHash),
		Version:     byte(ret.Version),
	}, nil
}

func (c *DASRPCClient) String() string {
	return fmt.Sprintf("DASRPCClient{url:%s}", c.url)
}

func (c *DASRPCClient) HealthCheck(ctx context.Context) error {
	return c.clnt.CallContext(ctx, nil, "das_healthCheck")
}

func (c *DASRPCClient) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	var res string
	err := c.clnt.CallContext(ctx, &res, "das_expirationPolicy")
	if err != nil {
		return -1, err
	}
	return daprovider.StringToExpirationPolicy(res)
}
