// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/das"

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

func (c *DASRPCClient) Retrieve(ctx context.Context, cert *arbstate.DataAvailabilityCertificate) ([]byte, error) {
	certBytes := das.Serialize(cert)
	var 
	c.clnt.Call("", "")
	response, err := c.clnt.Retrieve(ctx, &RetrieveRequest{CertBytes: certBytes})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64, reqSig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	response, err := c.clnt.Store(ctx, &StoreRequest{Message: message, Timeout: timeout, Sig: reqSig})
	if err != nil {
		return nil, err
	}
	var keysetHash [32]byte
	copy(keysetHash[:], response.KeysetHash)
	var dataHash [32]byte
	copy(dataHash[:], response.DataHash)
	respSig, err := blsSignatures.SignatureFromBytes(response.Sig)
	if err != nil {
		return nil, err
	}
	return &arbstate.DataAvailabilityCertificate{
		DataHash:    dataHash,
		Timeout:     response.Timeout,
		SignersMask: response.SignersMask,
		Sig:         respSig,
		KeysetHash:  keysetHash,
	}, nil
}

func (c *DASRPCClient) KeysetFromHash(ctx context.Context, ksHash []byte) ([]byte, error) {
	response, err := c.clnt.KeysetFromHash(ctx, &KeysetFromHashRequest{KsHash: ksHash})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (c *DASRPCClient) CurrentKeysetBytes(ctx context.Context) ([]byte, error) {
	response, err := c.clnt.CurrentKeysetBytes(ctx, &CurrentKeysetBytesRequest{})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (c *DASRPCClient) String() string {
	return fmt.Sprintf("DASRPCClient{c:%v}", c.clnt)
}
