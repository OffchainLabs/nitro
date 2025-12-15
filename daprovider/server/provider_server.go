// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dapserver

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/daprovider/server_api"
)

// lint:require-exhaustive-initialization
type ReaderServer struct {
	reader      daprovider.Reader
	headerBytes []byte // Supported header bytes for this provider
}

// lint:require-exhaustive-initialization
type WriterServer struct {
	writer       daprovider.Writer
	dataReceiver *data_streaming.DataStreamReceiver
}

// lint:require-exhaustive-initialization
type ValidatorServer struct {
	validator daprovider.Validator
}

// lint:require-exhaustive-initialization
type ServerConfig struct {
	Addr               string                              `koanf:"addr"`
	Port               uint64                              `koanf:"port"`
	JWTSecret          string                              `koanf:"jwtsecret"`
	EnableDAWriter     bool                                `koanf:"enable-da-writer"`
	ServerTimeouts     genericconf.HTTPServerTimeoutConfig `koanf:"server-timeouts"`
	RPCServerBodyLimit int                                 `koanf:"rpc-server-body-limit"`
}

var DefaultServerConfig = ServerConfig{
	Addr:               "localhost",
	Port:               9880,
	JWTSecret:          "",
	EnableDAWriter:     false,
	ServerTimeouts:     genericconf.HTTPServerTimeoutConfigDefault,
	RPCServerBodyLimit: genericconf.HTTPServerBodyLimitDefault,
}

func ServerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", DefaultServerConfig.Addr, "JSON rpc server listening interface")
	f.Uint64(prefix+".port", DefaultServerConfig.Port, "JSON rpc server listening port")
	f.String(prefix+".jwtsecret", DefaultServerConfig.JWTSecret, "path to file with jwtsecret for validation")
	f.Bool(prefix+".enable-da-writer", DefaultServerConfig.EnableDAWriter, "implies if the das server supports daprovider's writer interface")
	f.Int(prefix+".rpc-server-body-limit", DefaultServerConfig.RPCServerBodyLimit, "HTTP-RPC server maximum request body size in bytes; the default (0) uses geth's 5MB limit")
	genericconf.HTTPServerTimeoutConfigAddOptions(prefix+".server-timeouts", f)
}

func fetchJWTSecret(fileName string) ([]byte, error) {
	if data, err := os.ReadFile(fileName); err == nil {
		jwtSecret := common.FromHex(strings.TrimSpace(string(data)))
		if len(jwtSecret) == 32 {
			log.Info("Loaded JWT secret file", "path", fileName, "crc32", fmt.Sprintf("%#x", crc32.ChecksumIEEE(jwtSecret)))
			return jwtSecret, nil
		}
		log.Error("Invalid JWT secret", "path", fileName, "length", len(jwtSecret))
		return nil, errors.New("invalid JWT secret")
	}
	return nil, errors.New("JWT secret file not found")
}

// NewServerWithDAPProvider creates a new server with pre-created reader/writer/validator components.
// The server supports the Data Stream protocol (see `data_streaming` package). The `verifier` parameter is used for
// authenticating the sender (`daclient`).
func NewServerWithDAPProvider(ctx context.Context, config *ServerConfig, reader daprovider.Reader, writer daprovider.Writer, validator daprovider.Validator, headerBytes []byte, verifier *data_streaming.PayloadVerifier) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Addr, config.Port))
	if err != nil {
		return nil, err
	}

	rpcServer := rpc.NewServer()
	if config.RPCServerBodyLimit > 0 {
		rpcServer.SetHTTPBodyLimit(config.RPCServerBodyLimit)
	}

	if reader != nil {
		readerServer := &ReaderServer{
			reader:      reader,
			headerBytes: headerBytes,
		}
		if err = rpcServer.RegisterName("daprovider", readerServer); err != nil {
			return nil, err
		}
	}

	var dataStreamReceiver *data_streaming.DataStreamReceiver
	if writer != nil {
		dataStreamReceiver = data_streaming.NewDefaultDataStreamReceiver(verifier)
		dataStreamReceiver.Start(ctx)

		writerServer := &WriterServer{
			writer:       writer,
			dataReceiver: dataStreamReceiver,
		}
		if err = rpcServer.RegisterName("daprovider", writerServer); err != nil {
			return nil, err
		}
	}

	if validator != nil {
		validatorServer := &ValidatorServer{
			validator: validator,
		}
		if err = rpcServer.RegisterName("daprovider", validatorServer); err != nil {
			return nil, err
		}
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("failed getting provider server address from listener")
	}

	var handler http.Handler
	if config.JWTSecret != "" {
		jwt, err := fetchJWTSecret(config.JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("failed creating new provider server: %w", err)
		}
		handler = node.NewHTTPHandlerStack(rpcServer, nil, nil, jwt)
	} else {
		handler = rpcServer
	}

	srv := &http.Server{
		Addr:              "http://" + addr.String(),
		Handler:           handler,
		ReadTimeout:       config.ServerTimeouts.ReadTimeout,
		ReadHeaderTimeout: config.ServerTimeouts.ReadHeaderTimeout,
		WriteTimeout:      config.ServerTimeouts.WriteTimeout,
		IdleTimeout:       config.ServerTimeouts.IdleTimeout,
	}
	go func() {
		if err := srv.Serve(listener); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Error("provider server's Serve method returned a non http.ErrServerClosed error", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		if dataStreamReceiver != nil {
			dataStreamReceiver.StopAndWait()
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	return srv, nil
}

// ReaderServer methods

func (s *ReaderServer) GetSupportedHeaderBytes(ctx context.Context) (*server_api.SupportedHeaderBytesResult, error) {
	return &server_api.SupportedHeaderBytesResult{
		HeaderBytes: hexutil.Bytes(s.headerBytes),
	}, nil
}

func (s *ReaderServer) RecoverPayload(
	ctx context.Context,
	batchNum hexutil.Uint64,
	batchBlockHash common.Hash,
	sequencerMsg hexutil.Bytes,
) (*daprovider.PayloadResult, error) {
	promise := s.reader.RecoverPayload(uint64(batchNum), batchBlockHash, sequencerMsg)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ReaderServer) CollectPreimages(
	ctx context.Context,
	batchNum hexutil.Uint64,
	batchBlockHash common.Hash,
	sequencerMsg hexutil.Bytes,
) (*daprovider.PreimagesResult, error) {
	promise := s.reader.CollectPreimages(uint64(batchNum), batchBlockHash, sequencerMsg)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ValidatorServer methods

func (s *ValidatorServer) GenerateReadPreimageProof(ctx context.Context, offset hexutil.Uint64, certificate hexutil.Bytes) (*server_api.GenerateReadPreimageProofResult, error) {
	// #nosec G115
	promise := s.validator.GenerateReadPreimageProof(uint64(offset), certificate)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, err
	}
	return &server_api.GenerateReadPreimageProofResult{Proof: hexutil.Bytes(result.Proof)}, nil
}

func (s *ValidatorServer) GenerateCertificateValidityProof(ctx context.Context, certificate hexutil.Bytes) (*server_api.GenerateCertificateValidityProofResult, error) {
	// #nosec G115
	promise := s.validator.GenerateCertificateValidityProof(certificate)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, err
	}
	return &server_api.GenerateCertificateValidityProofResult{Proof: hexutil.Bytes(result.Proof)}, nil
}

// WriterServer methods (Storing API)

// Storing API: Data Stream methods

func (s *WriterServer) StartChunkedStore(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout hexutil.Uint64, sig hexutil.Bytes) (*data_streaming.StartStreamingResult, error) {
	return s.dataReceiver.StartReceiving(ctx, uint64(timestamp), uint64(nChunks), uint64(chunkSize), uint64(totalSize), uint64(timeout), sig)
}

func (s *WriterServer) SendChunk(ctx context.Context, messageId, chunkId hexutil.Uint64, chunk hexutil.Bytes, sig hexutil.Bytes) error {
	return s.dataReceiver.ReceiveChunk(ctx, data_streaming.MessageId(messageId), uint64(chunkId), chunk, sig)
}

func (s *WriterServer) CommitChunkedStore(ctx context.Context, messageId hexutil.Uint64, sig hexutil.Bytes) (*server_api.StoreResult, error) {
	message, timeout, _, err := s.dataReceiver.FinalizeReceiving(ctx, data_streaming.MessageId(messageId), sig)
	if err != nil {
		return nil, err
	}

	return s.Store(ctx, message, hexutil.Uint64(timeout))
}

// Storing API: Single-call Store method

func (s *WriterServer) Store(ctx context.Context, message hexutil.Bytes, timeout hexutil.Uint64) (*server_api.StoreResult, error) {
	serializedDACert, err := s.writer.Store(message, uint64(timeout)).Await(ctx)
	return &server_api.StoreResult{SerializedDACert: serializedDACert}, err
}

func (s *WriterServer) GetMaxMessageSize(ctx context.Context) (*server_api.MaxMessageSizeResult, error) {
	maxSize, err := s.writer.GetMaxMessageSize().Await(ctx)
	if err != nil {
		return nil, err
	}
	return &server_api.MaxMessageSizeResult{MaxSize: maxSize}, nil
}
