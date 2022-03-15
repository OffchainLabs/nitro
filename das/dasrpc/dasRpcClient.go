package dasrpc

import (
	"context"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"google.golang.org/grpc"
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt DASServiceImplClient
}

func NewDASRPCClient(target string) (*DASRPCClient, error) {
	conn, err := grpc.Dial(target)
	if err != nil {
		return nil, err
	}
	clnt := NewDASServiceImplClient(conn)
	return &DASRPCClient{clnt: clnt}, nil
}

func (clnt *DASRPCClient) Retrieve(ctx context.Context, cert []byte) ([]byte, error) {
	response, err := clnt.clnt.Retrieve(ctx, &RetrieveRequest{Cert: cert})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (clnt *DASRPCClient) Store(ctx context.Context, message []byte) (*arbstate.DataAvailabilityCertificate, error) {
	response, err := clnt.clnt.Store(ctx, &StoreRequest{Message: message})
	if err != nil {
		return nil, err
	}
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
	}, nil
}
