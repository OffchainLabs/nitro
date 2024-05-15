// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/pretty"
)

var (
	rpcStoreRequestGauge      = metrics.NewRegisteredGauge("arb/das/rpc/store/requests", nil)
	rpcStoreSuccessGauge      = metrics.NewRegisteredGauge("arb/das/rpc/store/success", nil)
	rpcStoreFailureGauge      = metrics.NewRegisteredGauge("arb/das/rpc/store/failure", nil)
	rpcStoreStoredBytesGauge  = metrics.NewRegisteredGauge("arb/das/rpc/store/bytes", nil)
	rpcStoreDurationHistogram = metrics.NewRegisteredHistogram("arb/das/rpc/store/duration", nil, metrics.NewBoundedHistogramSample())

	// TODO chunk store metrics
)

type DASRPCServer struct {
	daReader        DataAvailabilityServiceReader
	daWriter        DataAvailabilityServiceWriter
	daHealthChecker DataAvailabilityServiceHealthChecker
}

func StartDASRPCServer(ctx context.Context, addr string, portNum uint64, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, daReader DataAvailabilityServiceReader, daWriter DataAvailabilityServiceWriter, daHealthChecker DataAvailabilityServiceHealthChecker) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, portNum))
	if err != nil {
		return nil, err
	}
	return StartDASRPCServerOnListener(ctx, listener, rpcServerTimeouts, daReader, daWriter, daHealthChecker)
}

func StartDASRPCServerOnListener(ctx context.Context, listener net.Listener, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, daReader DataAvailabilityServiceReader, daWriter DataAvailabilityServiceWriter, daHealthChecker DataAvailabilityServiceHealthChecker) (*http.Server, error) {
	if daWriter == nil {
		return nil, errors.New("No writer backend was configured for DAS RPC server. Has the BLS signing key been set up (--data-availability.key.key-dir or --data-availability.key.priv-key options)?")
	}
	rpcServer := rpc.NewServer()
	err := rpcServer.RegisterName("das", &DASRPCServer{
		daReader:        daReader,
		daWriter:        daWriter,
		daHealthChecker: daHealthChecker,
	})
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Handler:           rpcServer,
		ReadTimeout:       rpcServerTimeouts.ReadTimeout,
		ReadHeaderTimeout: rpcServerTimeouts.ReadHeaderTimeout,
		WriteTimeout:      rpcServerTimeouts.WriteTimeout,
		IdleTimeout:       rpcServerTimeouts.IdleTimeout,
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
	Version     hexutil.Uint64 `json:"version,omitempty"`
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

	cert, err := serv.daWriter.Store(ctx, message, uint64(timeout), sig)
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
		Version:     hexutil.Uint64(cert.Version),
	}, nil
}

type StartChunkedStoreResult struct {
	ChunkedStoreId hexutil.Uint64 `json:"chunkedStoreId,omitempty"`
}

type SendChunkResult struct {
	Ok hexutil.Uint64 `json:"sendChunkResult,omitempty"`
}

func (serv *DASRPCServer) StartChunkedStore(ctx context.Context, timestamp hexutil.Uint64, nChunks hexutil.Uint64, totalSize hexutil.Uint64, timeout hexutil.Uint64, sig hexutil.Bytes) (*StartChunkedStoreResult, error) {
	return &StartChunkedStoreResult{}, nil

}

func (serv *DASRPCServer) SendChunk(ctx context.Context, message hexutil.Bytes, timeout hexutil.Uint64, sig hexutil.Bytes) (*SendChunkResult, error) {
	return &SendChunkResult{}, nil
}

func (serv *DASRPCServer) CommitChunkedStore(ctx context.Context, message hexutil.Bytes, timeout hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	// TODO tracing, metrics, and timers
	return &StoreResult{}, nil
}

func (serv *DASRPCServer) HealthCheck(ctx context.Context) error {
	return serv.daHealthChecker.HealthCheck(ctx)
}

func (serv *DASRPCServer) ExpirationPolicy(ctx context.Context) (string, error) {
	expirationPolicy, err := serv.daReader.ExpirationPolicy(ctx)
	if err != nil {
		return "", err
	}
	return expirationPolicy.String()
}
