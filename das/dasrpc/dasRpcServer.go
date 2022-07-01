// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/util/pretty"
)

var (
	rpcGetByHashRequestGauge       = metrics.NewRegisteredGauge("arb/das/rpc/getbyhash/requests", nil)
	rpcGetByHashSuccessGauge       = metrics.NewRegisteredGauge("arb/das/rpc/getbyhash/success", nil)
	rpcGetByHashFailureGauge       = metrics.NewRegisteredGauge("arb/das/rpc/getbyhash/failure", nil)
	rpcGetByHashReturnedBytesGauge = metrics.NewRegisteredGauge("arb/das/rpc/getbyhash/bytes", nil)

	rpcStoreRequestGauge     = metrics.NewRegisteredGauge("arb/das/rpc/store/requests", nil)
	rpcStoreSuccessGauge     = metrics.NewRegisteredGauge("arb/das/rpc/store/success", nil)
	rpcStoreFailureGauge     = metrics.NewRegisteredGauge("arb/das/rpc/store/failure", nil)
	rpcStoreStoredBytesGauge = metrics.NewRegisteredGauge("arb/das/rpc/store/bytes", nil)

	// This histogram is set with the default parameters of go-ethereum/metrics/Timer.
	// If requests are infrequent, then the reservoir size parameter can be adjusted
	// downwards to make a smaller window of samples that are included. The alpha parameter
	// can be adjusted to downweight the importance of older samples.
	rpcGetByHashDurationHistogram = metrics.NewRegisteredHistogram("arb/das/rpc/getbyhash/duration", nil, metrics.NewExpDecaySample(1028, 0.015))

	// Lower reservoir size for stores since we guess stores will be ~1 per minute.
	rpcStoreDurationHistogram = metrics.NewRegisteredHistogram("arb/das/rpc/store/duration", nil, metrics.NewExpDecaySample(32, 0.015))
)

type DASRPCServer struct {
	localDAS das.DataAvailabilityService
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
	err := rpcServer.RegisterName("das", &DASRPCServer{localDAS: localDAS})
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
		_ = srv.Shutdown(context.Background())
	}()
	return srv, nil
}

type StoreResult struct {
	DataHash    hexutil.Bytes  `json:"dataHash,omitempty"`
	Timeout     hexutil.Uint64 `json:"timeout,omitempty"`
	SignersMask hexutil.Uint64 `json:"signersMask,omitempty"`
	KeysetHash  hexutil.Bytes  `json:"keysetHash,omitempty"`
	Sig         hexutil.Bytes  `json:"sig,omitempty"`
}

func (serv *DASRPCServer) Store(ctx context.Context, message hexutil.Bytes, timeout hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	log.Trace("dasRpc.DASRPCServer.Store", "message", pretty.FirstFewBytes(message), "message length", len(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", serv)
	rpcStoreRequestGauge.Inc(1)
	start := time.Now()
	success := false
	defer func() {
		if success {
			rpcStoreSuccessGauge.Inc(1)
		} else {
			rpcStoreFailureGauge.Inc(1)
		}
		rpcStoreDurationHistogram.Update(time.Since(start).Nanoseconds())
	}()

	cert, err := serv.localDAS.Store(ctx, message, uint64(timeout), sig)
	if err != nil {
		return nil, err
	}
	rpcStoreStoredBytesGauge.Inc(int64(len(message)))
	success = true
	return &StoreResult{
		KeysetHash:  cert.KeysetHash[:],
		DataHash:    cert.DataHash[:],
		Timeout:     hexutil.Uint64(cert.Timeout),
		SignersMask: hexutil.Uint64(cert.SignersMask),
		Sig:         blsSignatures.SignatureToBytes(cert.Sig),
	}, nil
}

func (serv *DASRPCServer) GetByHash(ctx context.Context, certBytes hexutil.Bytes) (hexutil.Bytes, error) {
	rpcGetByHashRequestGauge.Inc(1)
	start := time.Now()
	success := false
	defer func() {
		if success {
			rpcGetByHashSuccessGauge.Inc(1)
		} else {
			rpcGetByHashFailureGauge.Inc(1)
		}
		rpcGetByHashDurationHistogram.Update(time.Since(start).Nanoseconds())
	}()

	bytes, err := serv.localDAS.GetByHash(ctx, certBytes)
	if err != nil {
		return nil, err
	}
	rpcGetByHashReturnedBytesGauge.Inc(int64(len(bytes)))
	success = true
	return bytes, nil
}

func (serv *DASRPCServer) HealthCheck(ctx context.Context) error {
	return serv.localDAS.HealthCheck(ctx)
}

func (serv *DASRPCServer) ExpirationPolicy(ctx context.Context) (string, error) {
	expirationPolicy, err := serv.localDAS.ExpirationPolicy(ctx)
	if err != nil {
		return "", err
	}
	return expirationPolicy.String()
}
