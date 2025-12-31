// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/util/pretty"
)

var (
	rpcStoreRequestCounter     = metrics.NewRegisteredCounter("arb/das/rpc/store/requests", nil)
	rpcStoreSuccessCounter     = metrics.NewRegisteredCounter("arb/das/rpc/store/success", nil)
	rpcStoreFailureCounter     = metrics.NewRegisteredCounter("arb/das/rpc/store/failure", nil)
	rpcStoreStoredBytesCounter = metrics.NewRegisteredCounter("arb/das/rpc/store/bytes", nil)
	rpcStoreDurationHistogram  = metrics.NewRegisteredHistogram("arb/das/rpc/store/duration", nil, metrics.NewBoundedHistogramSample())

	rpcSendChunkSuccessCounter = metrics.NewRegisteredCounter("arb/das/rpc/sendchunk/success", nil)
	rpcSendChunkFailureCounter = metrics.NewRegisteredCounter("arb/das/rpc/sendchunk/failure", nil)
)

// lint:require-exhaustive-initialization
type DASRPCServer struct {
	daReader        dasutil.DASReader
	daWriter        dasutil.DASWriter
	daHealthChecker DataAvailabilityServiceHealthChecker

	signatureVerifier *SignatureVerifier

	dataStreamReceiver *data_streaming.DataStreamReceiver
}

func StartDASRPCServer(ctx context.Context, addr string, portNum uint64, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, rpcServerBodyLimit int, daReader dasutil.DASReader, daWriter dasutil.DASWriter, daHealthChecker DataAvailabilityServiceHealthChecker, signatureVerifier *SignatureVerifier) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, portNum))
	if err != nil {
		return nil, err
	}
	return StartDASRPCServerOnListener(ctx, listener, rpcServerTimeouts, rpcServerBodyLimit, daReader, daWriter, daHealthChecker, signatureVerifier)
}

func StartDASRPCServerOnListener(ctx context.Context, listener net.Listener, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, rpcServerBodyLimit int, daReader dasutil.DASReader, daWriter dasutil.DASWriter, daHealthChecker DataAvailabilityServiceHealthChecker, signatureVerifier *SignatureVerifier) (*http.Server, error) {
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

	var dataStreamPayloadVerifier *data_streaming.PayloadVerifier
	if signatureVerifier == nil {
		// When signature checking is disabled, accept any signature without verification
		dataStreamPayloadVerifier = data_streaming.NoopPayloadVerifier()
	} else {
		dataStreamPayloadVerifier = data_streaming.CustomPayloadVerifier(func(ctx context.Context, signature []byte, bytes []byte, extras ...uint64) error {
			return signatureVerifier.verify(ctx, bytes, signature, extras...)
		})
	}

	dataStreamReceiver := data_streaming.NewDataStreamReceiver(dataStreamPayloadVerifier, data_streaming.DefaultMaxPendingMessages, data_streaming.DefaultMessageCollectionExpiry, data_streaming.DefaultRequestValidity, func(id data_streaming.MessageId) {
		rpcStoreFailureCounter.Inc(1)
	})
	dataStreamReceiver.Start(ctx)

	err := rpcServer.RegisterName("das", &DASRPCServer{
		daReader:           daReader,
		daWriter:           daWriter,
		daHealthChecker:    daHealthChecker,
		signatureVerifier:  signatureVerifier,
		dataStreamReceiver: dataStreamReceiver,
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
		dataStreamReceiver.StopAndWait()
		_ = srv.Shutdown(context.Background())
	}()
	return srv, nil
}

// lint:require-exhaustive-initialization
type StoreResult struct {
	DataHash    hexutil.Bytes  `json:"dataHash,omitempty"`
	Timeout     hexutil.Uint64 `json:"timeout,omitempty"`
	SignersMask hexutil.Uint64 `json:"signersMask,omitempty"`
	KeysetHash  hexutil.Bytes  `json:"keysetHash,omitempty"`
	Sig         hexutil.Bytes  `json:"sig,omitempty"`
	Version     hexutil.Uint64 `json:"version,omitempty"`
}

// The legacy storing API.
func (s *DASRPCServer) Store(ctx context.Context, message hexutil.Bytes, timeout hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	// #nosec G115
	log.Trace("dasRpc.DASRPCServer.Store", "message", pretty.FirstFewBytes(message), "message length", len(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", s)
	rpcStoreRequestCounter.Inc(1)
	start := time.Now()
	success := false
	defer func() {
		if success {
			rpcStoreSuccessCounter.Inc(1)
		} else {
			rpcStoreFailureCounter.Inc(1)
		}
		rpcStoreDurationHistogram.Update(time.Since(start).Nanoseconds())
	}()

	if s.signatureVerifier != nil {
		if err := s.signatureVerifier.verify(ctx, message, sig, uint64(timeout)); err != nil {
			return nil, err
		}
	}

	cert, err := s.daWriter.Store(ctx, message, uint64(timeout))
	if err != nil {
		return nil, err
	}
	rpcStoreStoredBytesCounter.Inc(int64(len(message)))
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

// exposed global for test control
var (
	legacyDASStoreAPIOnly = false
)

func (s *DASRPCServer) StartChunkedStore(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout hexutil.Uint64, sig hexutil.Bytes) (*data_streaming.StartStreamingResult, error) {
	rpcStoreRequestCounter.Inc(1)
	failed := true
	defer func() {
		if failed {
			rpcStoreFailureCounter.Inc(1)
		}
	}()

	result, err := s.dataStreamReceiver.StartReceiving(ctx, uint64(timestamp), uint64(nChunks), uint64(chunkSize), uint64(totalSize), uint64(timeout), sig)
	if err != nil {
		return nil, err
	}

	failed = false
	return result, nil
}

func (s *DASRPCServer) SendChunk(ctx context.Context, messageId, chunkId hexutil.Uint64, chunk hexutil.Bytes, sig hexutil.Bytes) error {
	success := false
	defer func() {
		if success {
			rpcSendChunkSuccessCounter.Inc(1)
		} else {
			rpcSendChunkFailureCounter.Inc(1)
		}
	}()

	if err := s.dataStreamReceiver.ReceiveChunk(ctx, data_streaming.MessageId(messageId), uint64(chunkId), chunk, sig); err != nil {
		return err
	}

	success = true
	return nil
}

func (s *DASRPCServer) CommitChunkedStore(ctx context.Context, messageId hexutil.Uint64, sig hexutil.Bytes) (*StoreResult, error) {
	message, timeout, startTime, err := s.dataStreamReceiver.FinalizeReceiving(ctx, data_streaming.MessageId(messageId), sig)
	if err != nil {
		return nil, err
	}

	cert, err := s.daWriter.Store(ctx, message, timeout)
	success := false
	defer func() {
		if success {
			rpcStoreSuccessCounter.Inc(1)
		} else {
			rpcStoreFailureCounter.Inc(1)
		}
		rpcStoreDurationHistogram.Update(time.Since(startTime).Nanoseconds())
	}()
	if err != nil {
		return nil, err
	}
	rpcStoreStoredBytesCounter.Inc(int64(len(message)))
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

func (s *DASRPCServer) HealthCheck(ctx context.Context) error {
	return s.daHealthChecker.HealthCheck(ctx)
}

func (s *DASRPCServer) ExpirationPolicy(ctx context.Context) (string, error) {
	expirationPolicy, err := s.daReader.ExpirationPolicy(ctx)
	if err != nil {
		return "", err
	}
	return expirationPolicy.String()
}
