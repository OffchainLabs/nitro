// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/signature"
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt   *rpc.Client
	url    string
	signer signature.DataSignerFunc
}

func nilSigner(_ []byte) ([]byte, error) {
	return []byte{}, nil
}

func NewDASRPCClient(target string, signer signature.DataSignerFunc) (*DASRPCClient, error) {
	clnt, err := rpc.Dial(target)
	if err != nil {
		return nil, err
	}
	if signer == nil {
		signer = nilSigner
	}
	return &DASRPCClient{
		clnt:   clnt,
		url:    target,
		signer: signer,
	}, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64, reqSig []byte) (*daprovider.DataAvailabilityCertificate, error) {
	/*
		var ret StartChunkedStoreResult
		if err := c.clnt.CallContext(ctx, &ret, "das_startChunkedStore", hexutil.Bytes(message), hexutil.Uint64(timeout), hexutil.Bytes(reqSig)); err != nil {
		}
	*/

	return c.legacyStore(ctx, message, timeout)

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
