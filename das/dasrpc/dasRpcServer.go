// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"bytes"
	"context"
	"fmt"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das"
	"google.golang.org/grpc"
	"net"
)

type DASRPCServer struct {
	UnimplementedDASServiceImplServer // this allows grpc to verify its version invariant
	grpcServer                        *grpc.Server
	localDAS                          das.DataAvailabilityService
}

func StartDASRPCServer(ctx context.Context, portNum uint64, localDAS das.DataAvailabilityService) (*DASRPCServer, error) {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", portNum))
	if err != nil {
		return nil, err
	}
	dasServer := &DASRPCServer{grpcServer: grpcServer, localDAS: localDAS}
	RegisterDASServiceImplServer(grpcServer, dasServer)
	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			return
		}
	}()
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()
	return dasServer, nil
}

func (serv *DASRPCServer) Stop() {
	serv.grpcServer.GracefulStop()
}

func (serv *DASRPCServer) Store(ctx context.Context, req *StoreRequest) (*StoreResponse, error) {
	cert, err := serv.localDAS.Store(ctx, req.Message, req.Timeout)
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
