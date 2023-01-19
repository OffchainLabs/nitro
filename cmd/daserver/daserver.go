// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	koanfjson "github.com/knadh/koanf/parsers/json"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type DAServerConfig struct {
	EnableRPC         bool                                `koanf:"enable-rpc"`
	RPCAddr           string                              `koanf:"rpc-addr"`
	RPCPort           uint64                              `koanf:"rpc-port"`
	RPCServerTimeouts genericconf.HTTPServerTimeoutConfig `koanf:"rpc-server-timeouts"`

	EnableREST         bool                                `koanf:"enable-rest"`
	RESTAddr           string                              `koanf:"rest-addr"`
	RESTPort           uint64                              `koanf:"rest-port"`
	RESTServerTimeouts genericconf.HTTPServerTimeoutConfig `koanf:"rest-server-timeouts"`

	DAConf das.DataAvailabilityConfig `koanf:"data-availability"`

	ConfConfig genericconf.ConfConfig `koanf:"conf"`
	LogLevel   int                    `koanf:"log-level"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
}

var DefaultDAServerConfig = DAServerConfig{
	EnableRPC:          false,
	RPCAddr:            "localhost",
	RPCPort:            9876,
	RPCServerTimeouts:  genericconf.HTTPServerTimeoutConfigDefault,
	EnableREST:         false,
	RESTAddr:           "localhost",
	RESTPort:           9877,
	RESTServerTimeouts: genericconf.HTTPServerTimeoutConfigDefault,
	DAConf:             das.DefaultDataAvailabilityConfig,
	ConfConfig:         genericconf.ConfConfigDefault,
	Metrics:            false,
	MetricsServer:      genericconf.MetricsServerConfigDefault,
	LogLevel:           3,
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running DAServer", "err", err)
	}
}

func printSampleUsage(progname string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func parseDAServer(args []string) (*DAServerConfig, error) {
	f := flag.NewFlagSet("daserver", flag.ContinueOnError)
	f.Bool("enable-rpc", DefaultDAServerConfig.EnableRPC, "enable the HTTP-RPC server listening on rpc-addr and rpc-port")
	f.String("rpc-addr", DefaultDAServerConfig.RPCAddr, "HTTP-RPC server listening interface")
	f.Uint64("rpc-port", DefaultDAServerConfig.RPCPort, "HTTP-RPC server listening port")
	genericconf.HTTPServerTimeoutConfigAddOptions("rpc-server-timeouts", f)

	f.Bool("enable-rest", DefaultDAServerConfig.EnableREST, "enable the REST server listening on rest-addr and rest-port")
	f.String("rest-addr", DefaultDAServerConfig.RESTAddr, "REST server listening interface")
	f.Uint64("rest-port", DefaultDAServerConfig.RESTPort, "REST server listening port")
	genericconf.HTTPServerTimeoutConfigAddOptions("rest-server-timeouts", f)

	f.Bool("metrics", DefaultDAServerConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Int("log-level", int(log.LvlInfo), "log level; 1: ERROR, 2: WARN, 3: INFO, 4: DEBUG, 5: TRACE")
	das.DataAvailabilityConfigAddDaserverOptions("data-availability", f)
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var serverConfig DAServerConfig
	if err := confighelpers.EndCommonParse(k, &serverConfig); err != nil {
		return nil, err
	}
	if serverConfig.ConfConfig.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"data-availability.key.priv-key": "",
		})
		if err != nil {
			return nil, fmt.Errorf("error removing extra parameters before dump: %w", err)
		}

		c, err := k.Marshal(koanfjson.Parser())
		if err != nil {
			return nil, fmt.Errorf("unable to marshal config file to JSON: %w", err)
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	return &serverConfig, nil
}

type L1ReaderCloser struct {
	l1Reader *headerreader.HeaderReader
}

func (c *L1ReaderCloser) Close(_ context.Context) error {
	c.l1Reader.StopOnly()
	return nil
}

func (c *L1ReaderCloser) String() string {
	return "l1 reader closer"
}

func startup() error {
	// Some different defaults to DAS config in a node.
	das.DefaultDataAvailabilityConfig.Enable = true

	serverConfig, err := parseDAServer(os.Args[1:])
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}
	if !(serverConfig.EnableRPC || serverConfig.EnableREST) {
		confighelpers.PrintErrorAndExit(errors.New("please specify at least one of --enable-rest or --enable-rpc"), printSampleUsage)
	}

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(serverConfig.LogLevel))
	log.Root().SetHandler(glogger)

	if serverConfig.Metrics {
		if len(serverConfig.MetricsServer.Addr) == 0 {
			fmt.Printf("Metrics is enabled, but missing --metrics-server.addr")
			return nil
		}

		go metrics.CollectProcessMetrics(serverConfig.MetricsServer.UpdateInterval)

		address := fmt.Sprintf("%v:%v", serverConfig.MetricsServer.Addr, serverConfig.MetricsServer.Port)
		exp.Setup(address)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var l1Reader *headerreader.HeaderReader
	if serverConfig.DAConf.L1NodeURL != "" && serverConfig.DAConf.L1NodeURL != "none" {
		l1Client, err := das.GetL1Client(ctx, serverConfig.DAConf.L1ConnectionAttempts, serverConfig.DAConf.L1NodeURL)
		if err != nil {
			return err
		}
		l1Reader = headerreader.New(l1Client, func() *headerreader.Config { return &headerreader.DefaultConfig }) // TODO: config
	}

	var seqInboxAddress *common.Address
	if serverConfig.DAConf.SequencerInboxAddress == "none" {
		seqInboxAddress = nil
	} else if len(serverConfig.DAConf.SequencerInboxAddress) > 0 {
		seqInboxAddress, err = das.OptionalAddressFromString(serverConfig.DAConf.SequencerInboxAddress)
		if err != nil {
			return err
		}
		if seqInboxAddress == nil {
			return errors.New("must provide data-availability.sequencer-inbox-address set to a valid contract address or 'none'")
		}
	} else {
		return errors.New("sequencer-inbox-address must be set to a valid L1 URL and contract address, or 'none'")
	}

	daReader, daWriter, daHealthChecker, dasLifecycleManager, err := das.CreateDAComponentsForDaserver(ctx, &serverConfig.DAConf, l1Reader, seqInboxAddress)
	if err != nil {
		return err
	}

	if l1Reader != nil {
		l1Reader.Start(ctx)
		dasLifecycleManager.Register(&L1ReaderCloser{l1Reader})
	}

	vcsRevision, vcsTime := confighelpers.GetVersion()
	var rpcServer *http.Server
	if serverConfig.EnableRPC {
		log.Info("Starting HTTP-RPC server", "addr", serverConfig.RPCAddr, "port", serverConfig.RPCPort, "revision", vcsRevision, "vcs.time", vcsTime)

		rpcServer, err = das.StartDASRPCServer(ctx, serverConfig.RPCAddr, serverConfig.RPCPort, serverConfig.RPCServerTimeouts, daReader, daWriter, daHealthChecker)
		if err != nil {
			return err
		}
	}

	var restServer *das.RestfulDasServer
	if serverConfig.EnableREST {
		log.Info("Starting REST server", "addr", serverConfig.RESTAddr, "port", serverConfig.RESTPort, "revision", vcsRevision, "vcs.time", vcsTime)

		restServer, err = das.NewRestfulDasServer(serverConfig.RESTAddr, serverConfig.RESTPort, serverConfig.RESTServerTimeouts, daReader, daHealthChecker)
		if err != nil {
			return err
		}
	}

	<-sigint
	dasLifecycleManager.StopAndWaitUntil(2 * time.Second)

	var err1, err2 error
	if rpcServer != nil {
		err1 = rpcServer.Shutdown(ctx)
	}

	if restServer != nil {
		err2 = restServer.Shutdown()
	}

	if err1 != nil {
		return err1
	}
	return err2
}
