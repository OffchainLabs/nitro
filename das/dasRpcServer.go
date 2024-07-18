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
	"sync"
	"sync/atomic"
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

	rpcSendChunkSuccessGauge = metrics.NewRegisteredGauge("arb/das/rpc/sendchunk/success", nil)
	rpcSendChunkFailureGauge = metrics.NewRegisteredGauge("arb/das/rpc/sendchunk/failure", nil)
)

type DASRPCServer struct {
	daReader        DataAvailabilityServiceReader
	daWriter        DataAvailabilityServiceWriter
	daHealthChecker DataAvailabilityServiceHealthChecker

	signatureVerifier *SignatureVerifier

	batches *batchBuilder
}

func StartDASRPCServer(ctx context.Context, addr string, portNum uint64, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, rpcServerBodyLimit int, daReader DataAvailabilityServiceReader, daWriter DataAvailabilityServiceWriter, daHealthChecker DataAvailabilityServiceHealthChecker, signatureVerifier *SignatureVerifier) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, portNum))
	if err != nil {
		return nil, err
	}
	return StartDASRPCServerOnListener(ctx, listener, rpcServerTimeouts, rpcServerBodyLimit, daReader, daWriter, daHealthChecker, signatureVerifier)
}

func StartDASRPCServerOnListener(ctx context.Context, listener net.Listener, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, rpcServerBodyLimit int, daReader DataAvailabilityServiceReader, daWriter DataAvailabilityServiceWriter, daHealthChecker DataAvailabilityServiceHealthChecker, signatureVerifier *SignatureVerifier) (*http.Server, error) {
	if daWriter == nil {
		return nil, errors.New("No writer backend was configured for DAS RPC server. Has the BLS signing key been set up (--data-availability.key.key-dir or --data-availability.key.priv-key options)?")
	}
	rpcServer := rpc.NewServer()
	if legacyDASStoreAPIOnly {
		rpcServer.ApplyAPIFilter(map[string]bool{"das_store": true})
	}
	if rpcServerBodyLimit > 0 {
		rpcServer.SetHTTPBodyLimit(rpcServerBodyLimit)
	}

	err := rpcServer.RegisterName("das", &DASRPCServer{
		daReader:          daReader,
		daWriter:          daWriter,
		daHealthChecker:   daHealthChecker,
		signatureVerifier: signatureVerifier,
		batches:           newBatchBuilder(),
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

	cert, err := s.daWriter.Store(ctx, message, uint64(timeout))
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
	chunks                          [][]byte
	expectedChunks                  uint64
	seenChunks                      atomic.Int64
	expectedChunkSize, expectedSize uint64
	timeout                         uint64
	startTime                       time.Time
}

const (
	maxPendingBatches   = 10
	batchBuildingExpiry = 1 * time.Minute
)

// exposed global for test control
var (
	legacyDASStoreAPIOnly = false
)

type batchBuilder struct {
	mutex   sync.Mutex
	batches map[uint64]*batch
}

func newBatchBuilder() *batchBuilder {
	return &batchBuilder{
		batches: make(map[uint64]*batch),
	}
}

func (b *batchBuilder) assign(nChunks, timeout, chunkSize, totalSize uint64) (uint64, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.batches) >= maxPendingBatches {
		return 0, fmt.Errorf("can't start new batch, already %d pending", len(b.batches))
	}

	id := rand.Uint64()
	_, ok := b.batches[id]
	if ok {
		return 0, errors.New("can't start new batch, try again")
	}

	b.batches[id] = &batch{
		chunks:            make([][]byte, nChunks),
		expectedChunks:    nChunks,
		expectedChunkSize: chunkSize,
		expectedSize:      totalSize,
		timeout:           timeout,
		startTime:         time.Now(),
	}
	go func(id uint64) {
		<-time.After(batchBuildingExpiry)
		b.mutex.Lock()
		// Batch will only exist if expiry was reached without it being complete.
		if _, exists := b.batches[id]; exists {
			rpcStoreFailureGauge.Inc(1)
			delete(b.batches, id)
		}
		b.mutex.Unlock()
	}(id)
	return id, nil
}

func (b *batchBuilder) add(id, idx uint64, data []byte) error {
	b.mutex.Lock()
	batch, ok := b.batches[id]
	b.mutex.Unlock()
	if !ok {
		return fmt.Errorf("unknown batch(%d)", id)
	}

	if idx >= uint64(len(batch.chunks)) {
		return fmt.Errorf("batch(%d): chunk(%d) out of range", id, idx)
	}

	if batch.chunks[idx] != nil {
		return fmt.Errorf("batch(%d): chunk(%d) already added", id, idx)
	}

	if batch.expectedChunkSize < uint64(len(data)) {
		return fmt.Errorf("batch(%d): chunk(%d) greater than expected size %d, was %d", id, idx, batch.expectedChunkSize, len(data))
	}

	batch.chunks[idx] = data
	batch.seenChunks.Add(1)
	return nil
}

func (b *batchBuilder) close(id uint64) ([]byte, uint64, time.Time, error) {
	b.mutex.Lock()
	batch, ok := b.batches[id]
	delete(b.batches, id)
	b.mutex.Unlock()
	if !ok {
		return nil, 0, time.Time{}, fmt.Errorf("unknown batch(%d)", id)
	}

	if batch.expectedChunks != uint64(batch.seenChunks.Load()) {
		return nil, 0, time.Time{}, fmt.Errorf("incomplete batch(%d): got %d/%d chunks", id, batch.seenChunks.Load(), batch.expectedChunks)
	}

	var flattened []byte
	for _, chunk := range batch.chunks {
		flattened = append(flattened, chunk...)
	}

	if batch.expectedSize != uint64(len(flattened)) {
		return nil, 0, time.Time{}, fmt.Errorf("batch(%d) was not expected size %d, was %d", id, batch.expectedSize, len(flattened))
	}

	return flattened, batch.timeout, batch.startTime, nil
}

func (s *DASRPCServer) StartChunkedStore(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout hexutil.Uint64, sig hexutil.Bytes) (*StartChunkedStoreResult, error) {
	rpcStoreRequestGauge.Inc(1)
	failed := true
	defer func() {
		if failed {
			rpcStoreFailureGauge.Inc(1)
		} // success gague will be incremented on successful commit
	}()

	if err := s.signatureVerifier.verify(ctx, []byte{}, sig, uint64(timestamp), uint64(nChunks), uint64(chunkSize), uint64(totalSize), uint64(timeout)); err != nil {
		return nil, err
	}

	// Prevent replay of old messages
	if time.Since(time.Unix(int64(timestamp), 0)).Abs() > time.Minute {
		return nil, errors.New("too much time has elapsed since request was signed")
	}

	id, err := s.batches.assign(uint64(nChunks), uint64(timeout), uint64(chunkSize), uint64(totalSize))
	if err != nil {
		return nil, err
	}

	failed = false
	return &StartChunkedStoreResult{
		BatchId: hexutil.Uint64(id),
	}, nil

}

func (s *DASRPCServer) SendChunk(ctx context.Context, batchId, chunkId hexutil.Uint64, message hexutil.Bytes, sig hexutil.Bytes) error {
	success := false
	defer func() {
		if success {
			rpcSendChunkSuccessGauge.Inc(1)
		} else {
			rpcSendChunkFailureGauge.Inc(1)
		}
	}()

	if err := s.signatureVerifier.verify(ctx, message, sig, uint64(batchId), uint64(chunkId)); err != nil {
		return err
	}

	if err := s.batches.add(uint64(batchId), uint64(chunkId), message); err != nil {
		return err
	}

	success = true
	return nil
}

func (s *DASRPCServer) CommitChunkedStore(ctx context.Context, batchId hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	if err := s.signatureVerifier.verify(ctx, []byte{}, sig, uint64(batchId)); err != nil {
		return nil, err
	}

	message, timeout, startTime, err := s.batches.close(uint64(batchId))
	if err != nil {
		return nil, err
	}

	cert, err := s.daWriter.Store(ctx, message, timeout)
	success := false
	defer func() {
		if success {
			rpcStoreSuccessGauge.Inc(1)
		} else {
			rpcStoreFailureGauge.Inc(1)
		}
		rpcStoreDurationHistogram.Update(time.Since(startTime).Nanoseconds())
	}()
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
