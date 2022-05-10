// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"context"
	"fmt"
	"github.com/offchainlabs/nitro/das"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt DASServiceImplClient
}

func NewDASRPCClient(target string) (*DASRPCClient, error) {
	// TODO revisit insecure setting
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	clnt := NewDASServiceImplClient(conn)
	return &DASRPCClient{clnt: clnt}, nil
}

func (clnt *DASRPCClient) Retrieve(ctx context.Context, cert *arbstate.DataAvailabilityCertificate) ([]byte, error) {
	certBytes := das.Serialize(cert)
	response, err := clnt.clnt.Retrieve(ctx, &RetrieveRequest{CertBytes: certBytes})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (clnt *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64) (*arbstate.DataAvailabilityCertificate, error) {
	response, err := clnt.clnt.Store(ctx, &StoreRequest{Message: message, Timeout: timeout})
	if err != nil {
		return nil, err
	}
	var keysetHash [32]byte
	copy(keysetHash[:], response.KeysetHash)
	var dataHash [32]byte
	copy(dataHash[:], response.DataHash)
	sig, err := blsSignatures.SignatureFromBytes(response.Sig)
	if err != nil {
		return nil, err
	}
	return &arbstate.DataAvailabilityCertificate{
		DataHash:    dataHash,
		Timeout:     response.Timeout,
		SignersMask: response.SignersMask,
		Sig:         sig,
		KeysetHash:  keysetHash,
	}, nil
}

func (clnt *DASRPCClient) KeysetFromHash(ctx context.Context, ksHash []byte) ([]byte, error) {
	response, err := clnt.clnt.KeysetFromHash(ctx, &KeysetFromHashRequest{KsHash: ksHash})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (clnt *DASRPCClient) CurrentKeysetBytes(ctx context.Context) ([]byte, error) {
	response, err := clnt.clnt.CurrentKeysetBytes(ctx, &CurrentKeysetBytesRequest{})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (clnt *DASRPCClient) String() string {
	return fmt.Sprintf("DASRPCClient{clnt:%v}", clnt.clnt)
}
