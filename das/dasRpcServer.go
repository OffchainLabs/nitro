// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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

	signatureVerifier *SignatureVerifier

	batches batchBuilder
}

func StartDASRPCServer(ctx context.Context, addr string, portNum uint64, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, daReader DataAvailabilityServiceReader, daWriter DataAvailabilityServiceWriter, daHealthChecker DataAvailabilityServiceHealthChecker, signatureVerifier *SignatureVerifier) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, portNum))
	if err != nil {
		return nil, err
	}
	return StartDASRPCServerOnListener(ctx, listener, rpcServerTimeouts, daReader, daWriter, daHealthChecker, signatureVerifier)
}

func StartDASRPCServerOnListener(ctx context.Context, listener net.Listener, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, daReader DataAvailabilityServiceReader, daWriter DataAvailabilityServiceWriter, daHealthChecker DataAvailabilityServiceHealthChecker, signatureVerifier *SignatureVerifier) (*http.Server, error) {
	if daWriter == nil {
		return nil, errors.New("No writer backend was configured for DAS RPC server. Has the BLS signing key been set up (--data-availability.key.key-dir or --data-availability.key.priv-key options)?")
	}
	rpcServer := rpc.NewServer()
	err := rpcServer.RegisterName("das", &DASRPCServer{
		daReader:          daReader,
		daWriter:          daWriter,
		daHealthChecker:   daHealthChecker,
		signatureVerifier: signatureVerifier,
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

func (s *DASRPCServer) Store(ctx context.Context, message hexutil.Bytes, timeout hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	log.Trace("dasRpc.DASRPCServer.Store", "message", pretty.FirstFewBytes(message), "message length", len(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", s)
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

	if err := s.signatureVerifier.verify(ctx, message, sig, uint64(timeout)); err != nil {
		return nil, err
	}

	cert, err := s.daWriter.Store(ctx, message, uint64(timeout), nil)
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
	BatchId hexutil.Uint64 `json:"batchId,omitempty"`
}

type SendChunkResult struct {
	Ok hexutil.Uint64 `json:"sendChunkResult,omitempty"`
}

type batch struct {
	chunks                     [][]byte
	expectedChunks, seenChunks uint64
	timeout                    uint64
}

const (
	maxPendingBatches = 10
)

type batchBuilder struct {
	batches map[uint64]batch
}

func (b *batchBuilder) assign(nChunks, timeout uint64) (uint64, error) {
	if len(b.batches) >= maxPendingBatches {
		return 0, fmt.Errorf("can't start new batch, already %d pending", b.batches)
	}

	id := rand.Uint64()
	_, ok := b.batches[id]
	if ok {
		return 0, fmt.Errorf("can't start new batch, try again")
	}

	b.batches[id] = batch{
		chunks:         make([][]byte, nChunks),
		expectedChunks: nChunks,
		timeout:        timeout,
	}
	return id, nil
}

func (b *batchBuilder) add(id, idx uint64, data []byte) error {
	batch, ok := b.batches[id]
	if !ok {
		return fmt.Errorf("unknown batch(%d)", id)
	}

	if idx >= uint64(len(batch.chunks)) {
		return fmt.Errorf("batch(%d): chunk(%d) out of range", id, idx)
	}

	if batch.chunks[idx] != nil {
		return fmt.Errorf("batch(%d): chunk(%d) already added", id, idx)
	}

	// todo check chunk size

	batch.chunks[idx] = data
	batch.seenChunks++
	return nil
}

func (b *batchBuilder) close(id uint64) ([]byte, uint64, error) {
	batch, ok := b.batches[id]
	if !ok {
		return nil, 0, fmt.Errorf("unknown batch(%d)", id)
	}

	if batch.expectedChunks != batch.seenChunks {
		return nil, 0, fmt.Errorf("incomplete batch(%d): got %d/%d chunks", id, batch.seenChunks, batch.expectedChunks)
	}

	// todo check total size

	var flattened []byte
	for _, chunk := range batch.chunks {
		flattened = append(flattened, chunk...)
	}
	return flattened, batch.timeout, nil
}

func (s *DASRPCServer) StartChunkedStore(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout hexutil.Uint64, sig hexutil.Bytes) (*StartChunkedStoreResult, error) {
	if err := s.signatureVerifier.verify(ctx, []byte{}, sig, uint64(timestamp), uint64(nChunks), uint64(totalSize), uint64(timeout)); err != nil {
		return nil, err
	}

	// Prevent replay of old messages
	if time.Since(time.Unix(int64(timestamp), 0)).Abs() > time.Minute {
		return nil, errors.New("too much time has elapsed since request was signed")
	}

	id, err := s.batches.assign(uint64(nChunks), uint64(timeout))
	if err != nil {
		return nil, err
	}

	return &StartChunkedStoreResult{
		BatchId: hexutil.Uint64(id),
	}, nil

}

func (s *DASRPCServer) SendChunk(ctx context.Context, batchId, chunkId hexutil.Uint64, message hexutil.Bytes, sig hexutil.Bytes) (*SendChunkResult, error) {
	if err := s.signatureVerifier.verify(ctx, message, sig, uint64(batchId), uint64(chunkId)); err != nil {
		return nil, err
	}

	if err := s.batches.add(uint64(batchId), uint64(chunkId), message); err != nil {
		return nil, err
	}

	return &SendChunkResult{
		Ok: hexutil.Uint64(1), // TODO probably not needed
	}, nil
}

func (s *DASRPCServer) CommitChunkedStore(ctx context.Context, batchId hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	if err := s.signatureVerifier.verify(ctx, []byte{}, sig, uint64(batchId)); err != nil {
		return nil, err
	}

	message, timeout, err := s.batches.close(uint64(batchId))
	if err != nil {
		return nil, err
	}

	cert, err := s.daWriter.Store(ctx, message, timeout, nil)
	if err != nil {
		return nil, err
	}
	return &StoreResult{
		KeysetHash:  cert.KeysetHash[:],
		DataHash:    cert.DataHash[:],
		Timeout:     hexutil.Uint64(cert.Timeout),
		SignersMask: hexutil.Uint64(cert.SignersMask),
		Sig:         blsSignatures.SignatureToBytes(cert.Sig),
		Version:     hexutil.Uint64(cert.Version),
	}, nil

	// TODO tracing, metrics, and timers

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
