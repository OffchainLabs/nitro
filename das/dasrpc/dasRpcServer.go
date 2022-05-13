// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das"
	"google.golang.org/grpc"
)

type DASRPCServer struct {
	UnimplementedDASServiceImplServer // this allows grpc to verify its version invariant
	grpcServer                        *grpc.Server
	localDAS                          das.DataAvailabilityService
}

func StartDASRPCServer(ctx context.Context, addr string, portNum uint64, localDAS das.DataAvailabilityService) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, portNum))
	if err != nil {
		return nil, err
	}
	return StartDASRPCServerOnListener(ctx, listener, localDAS)
}

func StartDASRPCServerOnListener(ctx context.Context, listener net.Listener, localDAS das.DataAvailabilityService) (*http.Server, error) {
	rpcServer := rpc.NewServer()
	err := rpcServer.RegisterName("das", localDAS)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Handler: rpcServer,
	}

	go func() {
		err := srv.Serve(listener)
		if err != nil {
			return
		}
	}()
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	return srv, nil
}

func (serv *DASRPCServer) Stop() {
	serv.grpcServer.GracefulStop()
}

func (serv *DASRPCServer) Store(ctx context.Context, req *StoreRequest) (*StoreResponse, error) {
	cert, err := serv.localDAS.Store(ctx, req.Message, req.Timeout, req.Sig)
	if err != nil {
		return nil, err
	}
	return &StoreResponse{
		KeysetHash:  cert.KeysetHash[:],
		DataHash:    cert.DataHash[:],
		Timeout:     cert.Timeout,
		SignersMask: cert.SignersMask,
		Sig:         blsSignatures.SignatureToBytes(cert.Sig),
	}, nil
}

func (serv *DASRPCServer) Retrieve(ctx context.Context, req *RetrieveRequest) (*RetrieveResponse, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(req.CertBytes))
	if err != nil {
		return nil, err
	}
	result, err := serv.localDAS.Retrieve(ctx, cert)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Result: result}, nil
}

func (serv *DASRPCServer) KeysetFromHash(ctx context.Context, req *KeysetFromHashRequest) (*KeysetFromHashResponse, error) {
	resp, err := serv.localDAS.KeysetFromHash(ctx, req.KsHash)
	if err != nil {
		return nil, err
	}
	return &KeysetFromHashResponse{Result: resp}, nil
}

func (serv *DASRPCServer) CurrentKeysetBytes(ctx context.Context, req *CurrentKeysetBytesRequest) (*CurrentKeysetBytesResponse, error) {
	resp, err := serv.localDAS.CurrentKeysetBytes(ctx)
	if err != nil {
		return nil, err
	}
	return &CurrentKeysetBytesResponse{Result: resp}, nil
}
