// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/util/pretty"
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt *rpc.Client
	url  string
}

func NewDASRPCClient(target string) (*DASRPCClient, error) {
	clnt, err := rpc.Dial(target)
	if err != nil {
		return nil, err
	}
	return &DASRPCClient{
		clnt: clnt,
		url:  target,
	}, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64, reqSig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	log.Trace("das.DASRPCClient.Store(...)", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(reqSig), "this", *c)
	var ret StoreResult
	if err := c.clnt.CallContext(ctx, &ret, "das_store", hexutil.Bytes(message), hexutil.Uint64(timeout), hexutil.Bytes(reqSig)); err != nil {
		return nil, err
	}
	respSig, err := blsSignatures.SignatureFromBytes(ret.Sig)
	if err != nil {
		return nil, err
	}
	return &arbstate.DataAvailabilityCertificate{
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

func (c *DASRPCClient) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	var res string
	err := c.clnt.CallContext(ctx, &res, "das_expirationPolicy")
	if err != nil {
		return -1, err
	}
	return arbstate.StringToExpirationPolicy(res)
}
