// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt *rpc.Client
}

func NewDASRPCClient(target string) (*DASRPCClient, error) {
	clnt, err := rpc.Dial(target)
	if err != nil {
		return nil, err
	}
	return &DASRPCClient{clnt: clnt}, nil
}

func (c *DASRPCClient) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	var ret hexutil.Bytes
	if err := c.clnt.CallContext(ctx, &ret, "das_getByHash", hexutil.Bytes(hash)); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64, reqSig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	var ret StoreResult
	if err := c.clnt.CallContext(ctx, &ret, "das_store", hexutil.Bytes(message), hexutil.Uint64(timeout), hexutil.Bytes(reqSig)); err != nil {
		return nil, err
	}
	var keysetHash [32]byte
	copy(keysetHash[:], ret.KeysetHash)
	var dataHash [32]byte
	copy(dataHash[:], ret.DataHash)
	respSig, err := blsSignatures.SignatureFromBytes(ret.Sig)
	if err != nil {
		return nil, err
	}
	return &arbstate.DataAvailabilityCertificate{
		DataHash:    dataHash,
		Timeout:     uint64(ret.Timeout),
		SignersMask: uint64(ret.SignersMask),
		Sig:         respSig,
		KeysetHash:  keysetHash,
	}, nil
}

func (c *DASRPCClient) KeysetFromHash(ctx context.Context, ksHash []byte) ([]byte, error) {
	var ret hexutil.Bytes
	if err := c.clnt.CallContext(ctx, &ret, "das_keysetFromHash", hexutil.Bytes(ksHash)); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *DASRPCClient) CurrentKeysetBytes(ctx context.Context) ([]byte, error) {
	var ret hexutil.Bytes
	if err := c.clnt.CallContext(ctx, &ret, "das_currentKeysetBytes"); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *DASRPCClient) String() string {
	return fmt.Sprintf("DASRPCClient{c:%v}", c.clnt)
}
