package dasserver

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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/daprovider/das"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	flag "github.com/spf13/pflag"
)

type Server struct {
	reader daprovider.Reader
	writer daprovider.Writer
}

type ServerConfig struct {
	Addr             string                              `koanf:"addr"`
	Port             uint64                              `koanf:"port"`
	JWTSecret        string                              `koanf:"jwtsecret"`
	EnableDAWriter   bool                                `koanf:"enable-da-writer"`
	DataAvailability das.DataAvailabilityConfig          `koanf:"data-availability"`
	ServerTimeouts   genericconf.HTTPServerTimeoutConfig `koanf:"server-timeouts"`
}

var DefaultServerConfig = ServerConfig{
	Addr:             "localhost",
	Port:             9880,
	JWTSecret:        "",
	EnableDAWriter:   false,
	DataAvailability: das.DefaultDataAvailabilityConfig,
	ServerTimeouts:   genericconf.HTTPServerTimeoutConfigDefault,
}

func ServerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", DefaultServerConfig.Addr, "JSON rpc server listening interface")
	f.Uint64(prefix+".port", DefaultServerConfig.Port, "JSON rpc server listening port")
	f.String(prefix+".jwtsecret", DefaultServerConfig.JWTSecret, "path to file with jwtsecret for validation")
	f.Bool(prefix+".enable-da-writer", DefaultServerConfig.EnableDAWriter, "implies if the das server supports daprovider's writer interface")
	das.DataAvailabilityConfigAddNodeOptions(prefix+".data-availability", f)
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

func NewServer(ctx context.Context, config *ServerConfig, dataSigner signature.DataSignerFunc, l1Client arbutil.L1Interface, l1Reader *headerreader.HeaderReader, sequencerInboxAddr common.Address) (*http.Server, func(), error) {
	var err error
	var daWriter das.DataAvailabilityServiceWriter
	var daReader das.DataAvailabilityServiceReader
	var dasKeysetFetcher *das.KeysetFetcher
	var dasLifecycleManager *das.LifecycleManager
	if config.EnableDAWriter {
		daWriter, daReader, dasKeysetFetcher, dasLifecycleManager, err = das.CreateDAReaderAndWriter(ctx, &config.DataAvailability, dataSigner, l1Client, sequencerInboxAddr)
		if err != nil {
			return nil, nil, err
		}
	} else {
		daReader, dasKeysetFetcher, dasLifecycleManager, err = das.CreateDAReader(ctx, &config.DataAvailability, l1Reader, &sequencerInboxAddr)
		if err != nil {
			return nil, nil, err
		}
	}

	daReader = das.NewReaderTimeoutWrapper(daReader, config.DataAvailability.RequestTimeout)
	if config.DataAvailability.PanicOnError {
		if daWriter != nil {
			daWriter = das.NewWriterPanicWrapper(daWriter)
		}
		daReader = das.NewReaderPanicWrapper(daReader)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Addr, config.Port))
	if err != nil {
		return nil, nil, err
	}

	rpcServer := rpc.NewServer()
	server := &Server{
		reader: dasutil.NewReaderForDAS(daReader, dasKeysetFetcher),
		writer: dasutil.NewWriterForDAS(daWriter),
	}
	if err = rpcServer.RegisterName("daprovider", server); err != nil {
		return nil, nil, err
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, nil, errors.New("failed getting dasserver address from listener")
	}

	var handler http.Handler
	if config.JWTSecret != "" {
		jwt, err := fetchJWTSecret(config.JWTSecret)
		if err != nil {
			return nil, nil, fmt.Errorf("failed creating new dasserver: %w", err)
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
		err := srv.Serve(listener)
		if err != nil {
			return
		}
	}()

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	return srv, func() {
		if dasLifecycleManager != nil {
			dasLifecycleManager.StopAndWaitUntil(2 * time.Second)
		}
	}, nil
}

func (s *Server) IsValidHeaderByte(ctx context.Context, headerByte byte) (*daclient.IsValidHeaderByteResult, error) {
	return &daclient.IsValidHeaderByteResult{IsValid: s.reader.IsValidHeaderByte(ctx, headerByte)}, nil
}

func (s *Server) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum hexutil.Uint64,
	batchBlockHash common.Hash,
	sequencerMsg hexutil.Bytes,
	preimages map[arbutil.PreimageType]map[common.Hash][]byte,
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
