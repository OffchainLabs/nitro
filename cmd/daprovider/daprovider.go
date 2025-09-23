package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/knadh/koanf/parsers/json"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/das/dasserver"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

type Config struct {
	DASServer        dasserver.ServerConfig   `koanf:"das-server"`
	WithDataSigner   bool                     `koanf:"with-data-signer"`
	DataSignerWallet genericconf.WalletConfig `koanf:"data-signer-wallet"`

	Conf     genericconf.ConfConfig `koanf:"conf"`
	LogLevel string                 `koanf:"log-level"`
	LogType  string                 `koanf:"log-type"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
}

var DefaultConfig = Config{
	DASServer:        dasserver.DefaultServerConfig,
	WithDataSigner:   false,
	DataSignerWallet: arbnode.DefaultBatchPosterL1WalletConfig,
	Conf:             genericconf.ConfConfigDefault,
	LogLevel:         "INFO",
	LogType:          "plaintext",
	Metrics:          false,
	MetricsServer:    genericconf.MetricsServerConfigDefault,
	PProf:            false,
	PprofCfg:         genericconf.PProfDefault,
}

func printSampleUsage(progname string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func parseDAProvider(args []string) (*Config, error) {
	f := pflag.NewFlagSet("daprovider", pflag.ContinueOnError)
	f.Bool("with-data-signer", DefaultConfig.WithDataSigner, "set to enable data signing when processing store requests. If enabled requires data-signer-wallet config")
	genericconf.WalletConfigAddOptions("data-signer-wallet", f, DefaultConfig.DataSignerWallet.Pathname)

	f.Bool("metrics", DefaultConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Bool("pprof", DefaultConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	f.String("log-level", DefaultConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultConfig.LogType, "log type (plaintext or json)")

	dasserver.ServerConfigAddOptions("das-server", f)
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	if err = das.FixKeysetCLIParsing("das-server.data-availability.rpc-aggregator.backends", k); err != nil {
		return nil, err
	}

	var config Config
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}

	if config.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"das-server.data-availability.key.priv-key": "",
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

func main() {
	if err := startup(); err != nil {
		log.Error("Error running daprovider server", "err", err)
	}
}

func startup() error {
	// Some different defaults to DAS config in a node.
	das.DefaultDataAvailabilityConfig.Enable = true

	config, err := parseDAProvider(os.Args[1:])
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

	err = util.StartMetricsAndPProf(&util.MetricsPProfOpts{
		Metrics:       config.Metrics,
		MetricsServer: config.MetricsServer,
		PProf:         config.PProf,
		PprofCfg:      config.PprofCfg,
	})
	if err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !config.DASServer.DataAvailability.Enable {
		return errors.New("--das-server.data-availability.enable is a required to start a das-server")
	}

	if config.DASServer.DataAvailability.ParentChainNodeURL == "" || config.DASServer.DataAvailability.ParentChainNodeURL == "none" {
		return errors.New("--das-server.data-availability.parent-chain-node-url is a required to start a das-server")
	}

	if config.DASServer.DataAvailability.SequencerInboxAddress == "" || config.DASServer.DataAvailability.SequencerInboxAddress == "none" {
		return errors.New("sequencer-inbox-address must be set to a valid L1 URL and contract address")
	}

	l1Client, err := das.GetL1Client(ctx, config.DASServer.DataAvailability.ParentChainConnectionAttempts, config.DASServer.DataAvailability.ParentChainNodeURL)
	if err != nil {
		return err
	}

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
	l1Reader, err := headerreader.New(ctx, l1Client, func() *headerreader.Config { return &headerreader.DefaultConfig }, arbSys)
	if err != nil {
		return err
	}

	seqInboxAddr, err := das.OptionalAddressFromString(config.DASServer.DataAvailability.SequencerInboxAddress)
	if err != nil {
		return err
	}
	if seqInboxAddr == nil {
		return errors.New("must provide --das-server.data-availability.sequencer-inbox-address set to a valid contract address or 'none'")
	}

	var dataSigner signature.DataSignerFunc
	if config.WithDataSigner && config.DASServer.EnableDAWriter {
		l1ChainId, err := l1Client.ChainID(ctx)
		if err != nil {
			return fmt.Errorf("couldn't read L1 chainid: %w", err)
		}
		if _, dataSigner, err = util.OpenWallet("data-signer", &config.DataSignerWallet, l1ChainId); err != nil {
			return err
		}
	}

	log.Info("Starting json rpc server", "addr", config.DASServer.Addr, "port", config.DASServer.Port)
	dasServer, closeFn, err := dasserver.NewServer(ctx, &config.DASServer, dataSigner, l1Client, l1Reader, *seqInboxAddr)
	if err != nil {
		return err
	}

	if l1Reader != nil {
		l1Reader.Start(ctx)
	}

	<-sigint

	if err = dasServer.Shutdown(ctx); err != nil {
		return err
	}
	if closeFn != nil {
		closeFn()
	}
	if l1Reader != nil && l1Reader.Started() {
		l1Reader.StopAndWait()
	}
	return nil
}
