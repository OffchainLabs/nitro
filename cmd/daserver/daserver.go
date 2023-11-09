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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
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

	DataAvailability das.DataAvailabilityConfig `koanf:"data-availability"`

	Conf     genericconf.ConfConfig `koanf:"conf"`
	LogLevel int                    `koanf:"log-level"`
	LogType  string                 `koanf:"log-type"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
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
	DataAvailability:   das.DefaultDataAvailabilityConfig,
	Conf:               genericconf.ConfConfigDefault,
	LogLevel:           int(log.LvlInfo),
	LogType:            "plaintext",
	Metrics:            false,
	MetricsServer:      genericconf.MetricsServerConfigDefault,
	PProf:              false,
	PprofCfg:           genericconf.PProfDefault,
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

	f.Bool("pprof", DefaultDAServerConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	f.Int("log-level", int(log.LvlInfo), "log level; 1: ERROR, 2: WARN, 3: INFO, 4: DEBUG, 5: TRACE")
	f.String("log-type", DefaultDAServerConfig.LogType, "log type (plaintext or json)")

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
	if serverConfig.Conf.Dump {
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

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics(cfg *DAServerConfig) error {
	mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	if cfg.Metrics && !metrics.Enabled {
		return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
	}
	if cfg.Metrics && cfg.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if cfg.Metrics {
		go metrics.CollectProcessMetrics(cfg.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	}
	if cfg.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
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

	logFormat, err := genericconf.ParseLogType(serverConfig.LogType)
	if err != nil {
		flag.Usage()
		panic(fmt.Sprintf("Error parsing log type: %v", err))
	}
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	glogger.Verbosity(log.Lvl(serverConfig.LogLevel))
	log.Root().SetHandler(glogger)

	if err := startMetrics(serverConfig); err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var l1Reader *headerreader.HeaderReader
	if serverConfig.DataAvailability.ParentChainNodeURL != "" && serverConfig.DataAvailability.ParentChainNodeURL != "none" {
		l1Client, err := das.GetL1Client(ctx, serverConfig.DataAvailability.ParentChainConnectionAttempts, serverConfig.DataAvailability.ParentChainNodeURL)
		if err != nil {
			return err
		}
		arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
		l1Reader, err = headerreader.New(ctx, l1Client, func() *headerreader.Config { return &headerreader.DefaultConfig }, arbSys) // TODO: config
		if err != nil {
			return err
		}
	}

	var seqInboxAddress *common.Address
	if serverConfig.DataAvailability.SequencerInboxAddress == "none" {
		seqInboxAddress = nil
	} else if len(serverConfig.DataAvailability.SequencerInboxAddress) > 0 {
		seqInboxAddress, err = das.OptionalAddressFromString(serverConfig.DataAvailability.SequencerInboxAddress)
		if err != nil {
			return err
		}
		if seqInboxAddress == nil {
			return errors.New("must provide data-availability.sequencer-inbox-address set to a valid contract address or 'none'")
		}
	} else {
		return errors.New("sequencer-inbox-address must be set to a valid L1 URL and contract address, or 'none'")
	}

	daReader, daWriter, daHealthChecker, dasLifecycleManager, err := das.CreateDAComponentsForDaserver(ctx, &serverConfig.DataAvailability, l1Reader, seqInboxAddress)
	if err != nil {
		return err
	}

	if l1Reader != nil {
		l1Reader.Start(ctx)
		dasLifecycleManager.Register(&L1ReaderCloser{l1Reader})
	}

	vcsRevision, _, vcsTime := confighelpers.GetVersion()
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
