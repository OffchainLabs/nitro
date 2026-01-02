// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/knadh/koanf/parsers/json"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
)

type TxFiltererManagerConfig struct {
	RPCAddr            string                              `koanf:"rpc-addr"`
	RPCPort            uint64                              `koanf:"rpc-port"`
	RPCServerTimeouts  genericconf.HTTPServerTimeoutConfig `koanf:"rpc-server-timeouts"`
	RPCServerBodyLimit int                                 `koanf:"rpc-server-body-limit"`

	Conf     genericconf.ConfConfig `koanf:"conf"`
	LogLevel string                 `koanf:"log-level"`
	LogType  string                 `koanf:"log-type"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
}

var DefaultTxFiltererManagerConfig = TxFiltererManagerConfig{
	RPCAddr:            "localhost",
	RPCPort:            9876,
	RPCServerTimeouts:  genericconf.HTTPServerTimeoutConfigDefault,
	RPCServerBodyLimit: genericconf.HTTPServerBodyLimitDefault,
	Conf:               genericconf.ConfConfigDefault,
	LogLevel:           "INFO",
	LogType:            "plaintext",
	Metrics:            false,
	MetricsServer:      genericconf.MetricsServerConfigDefault,
	PProf:              false,
	PprofCfg:           genericconf.PProfDefault,
}

func parseTxFiltererManagerConfig(args []string) (*TxFiltererManagerConfig, error) {
	f := pflag.NewFlagSet("tx-filterer-signer", pflag.ContinueOnError)
	f.String("rpc-addr", DefaultTxFiltererManagerConfig.RPCAddr, "HTTP-RPC server listening interface")
	f.Uint64("rpc-port", DefaultTxFiltererManagerConfig.RPCPort, "HTTP-RPC server listening port")
	f.Int("rpc-server-body-limit", DefaultTxFiltererManagerConfig.RPCServerBodyLimit, "HTTP-RPC server maximum request body size in bytes; the default (0) uses geth's 5MB limit")
	genericconf.HTTPServerTimeoutConfigAddOptions("rpc-server-timeouts", f)

	f.Bool("metrics", DefaultTxFiltererManagerConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Bool("pprof", DefaultTxFiltererManagerConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	f.String("log-level", DefaultTxFiltererManagerConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultTxFiltererManagerConfig.LogType, "log type (plaintext or json)")

	genericconf.ConfConfigAddOptions("conf", f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config TxFiltererManagerConfig
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	if config.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			// "data-availability.key.priv-key": "",
		})
		if err != nil {
			return nil, fmt.Errorf("error removing extra parameters before dump: %w", err)
		}

		c, err := k.Marshal(json.Parser())
		if err != nil {
			return nil, fmt.Errorf("unable to marshal config file to JSON: %w", err)
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	return &config, nil
}

func printSampleUsage(progname string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

// TODO: move this to genericconf package
// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics(cfg *TxFiltererManagerConfig) error {
	mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	if cfg.Metrics && cfg.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if cfg.Metrics {
		log.Info("Enabling metrics collection")
		metrics.Enable()
		go metrics.CollectProcessMetrics(cfg.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	}
	if cfg.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
}

type RPCServer struct {
}

func (r *RPCServer) FilterTx(ctx context.Context, txHash common.Hash) error {
	log.Info("Received request to filter transaction", "txHash", txHash.Hex())
	return nil
}

func startRPCServer(ctx context.Context, addr string, portNum uint64, rpcServerTimeouts genericconf.HTTPServerTimeoutConfig, rpcServerBodyLimit int) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, portNum))
	if err != nil {
		return nil, err
	}

	rpcServer := rpc.NewServer()
	if rpcServerBodyLimit > 0 {
		rpcServer.SetHTTPBodyLimit(rpcServerBodyLimit)
	}

	err = rpcServer.RegisterName("tx-filterer-manager", &RPCServer{})
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

func startup() error {
	config, err := parseTxFiltererManagerConfig(os.Args[1:])
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	logLevel, err := genericconf.ToSlogLevel(config.LogLevel)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	handler, err := genericconf.HandlerFromLogType(config.LogType, io.Writer(os.Stderr))
	if err != nil {
		pflag.Usage()
		return fmt.Errorf("error parsing log type when creating handler: %w", err)
	}
	glogger := log.NewGlogHandler(handler)
	glogger.Verbosity(logLevel)
	log.SetDefault(log.NewLogger(glogger))

	if err := startMetrics(config); err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vcsRevision, _, vcsTime := confighelpers.GetVersion()
	log.Info("Starting HTTP-RPC server", "addr", config.RPCAddr, "port", config.RPCPort, "revision", vcsRevision, "vcs.time", vcsTime)
	rpcServer, err := startRPCServer(ctx, config.RPCAddr, config.RPCPort, config.RPCServerTimeouts, config.RPCServerBodyLimit)
	if err != nil {
		return err
	}

	<-sigint

	err = rpcServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running TxFiltererManager", "err", err)
	}
}
