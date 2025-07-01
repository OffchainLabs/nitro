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

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/daclient"
)

type Server struct {
	reader    daprovider.Reader
	writer    daprovider.Writer
	validator daprovider.Validator
}

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
	f.Int("rpc-server-body-limit", DefaultServerConfig.RPCServerBodyLimit, "HTTP-RPC server maximum request body size in bytes; the default (0) uses geth's 5MB limit")
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

// NewServerWithDAPProvider creates a new server with pre-created reader/writer/validator components
func NewServerWithDAPProvider(ctx context.Context, config *ServerConfig, reader daprovider.Reader, writer daprovider.Writer, validator daprovider.Validator) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Addr, config.Port))
	if err != nil {
		return nil, err
	}

	rpcServer := rpc.NewServer()
	if config.RPCServerBodyLimit > 0 {
		rpcServer.SetHTTPBodyLimit(config.RPCServerBodyLimit)
	}

	server := &Server{
		reader:    reader,
		writer:    writer,
		validator: validator,
	}
	if err = rpcServer.RegisterName("daprovider", server); err != nil {
		return nil, err
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
		_ = srv.Shutdown(context.Background())
	}()

	return srv, nil
}

func (s *Server) IsValidHeaderByte(ctx context.Context, headerByte byte) (*daclient.IsValidHeaderByteResult, error) {
	return &daclient.IsValidHeaderByteResult{IsValid: s.reader.IsValidHeaderByte(ctx, headerByte)}, nil
}

func (s *Server) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum hexutil.Uint64,
	batchBlockHash common.Hash,
	sequencerMsg hexutil.Bytes,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) (*daclient.RecoverPayloadFromBatchResult, error) {
	payload, preimages, err := s.reader.RecoverPayloadFromBatch(ctx, uint64(batchNum), batchBlockHash, sequencerMsg, preimages, validateSeqMsg)
	if err != nil {
		return nil, err
	}
	return &daclient.RecoverPayloadFromBatchResult{
		Payload:   payload,
		Preimages: preimages,
	}, nil
}

func (s *Server) Store(
	ctx context.Context,
	message hexutil.Bytes,
	timeout hexutil.Uint64,
	disableFallbackStoreDataOnChain bool,
) (*daclient.StoreResult, error) {
	serializedDACert, err := s.writer.Store(ctx, message, uint64(timeout), disableFallbackStoreDataOnChain)
	if err != nil {
		return nil, err
	}
	return &daclient.StoreResult{SerializedDACert: serializedDACert}, nil
}

func (s *Server) GenerateProof(ctx context.Context, preimageType uint8, certHash common.Hash, offset hexutil.Uint64, certificate hexutil.Bytes) (hexutil.Bytes, error) {
	if s.validator == nil {
		return nil, errors.New("validator not available")
	}
	proof, err := s.validator.GenerateProof(ctx, arbutil.PreimageType(preimageType), certHash, uint64(offset), certificate)
	if err != nil {
		return nil, err
	}
	return hexutil.Bytes(proof), nil
}
