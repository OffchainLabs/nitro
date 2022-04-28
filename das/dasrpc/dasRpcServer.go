// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"context"
	"fmt"
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
		DataHash:    cert.DataHash[:],
		Timeout:     cert.Timeout,
		SignersMask: cert.SignersMask,
		Sig:         blsSignatures.SignatureToBytes(cert.Sig),
	}, nil
}

func (serv *DASRPCServer) Retrieve(ctx context.Context, req *RetrieveRequest) (*RetrieveResponse, error) {
	result, err := serv.localDAS.Retrieve(ctx, req.Cert)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Result: result}, nil
}
