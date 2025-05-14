package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	koanfjson "github.com/knadh/koanf/parsers/json"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider/das"
	dapserver "github.com/offchainlabs/nitro/daprovider/server"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

type Config struct {
	ProviderServer   dapserver.ServerConfig      `koanf:"provider-server"`
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
	ProviderServer:   dapserver.DefaultServerConfig,
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
	f := flag.NewFlagSet("daprovider", flag.ContinueOnError)
	f.Bool("with-data-signer", DefaultConfig.WithDataSigner, "set to enable data signing when processing store requests. If enabled requires data-signer-wallet config")
	genericconf.WalletConfigAddOptions("data-signer-wallet", f, DefaultConfig.DataSignerWallet.Pathname)

	f.Bool("metrics", DefaultConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Bool("pprof", DefaultConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	f.String("log-level", DefaultConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultConfig.LogType, "log type (plaintext or json)")

	dapserver.ServerConfigAddOptions("provider-server", f)
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	if err = das.FixKeysetCLIParsing("provider-server.data-availability.rpc-aggregator.backends", k); err != nil {
		return nil, err
	}

	var config Config
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}

	if config.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"provider-server.data-availability.key.priv-key": "",
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
		flag.Usage()
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

	if !config.ProviderServer.DataAvailability.Enable {
		return errors.New("--provider-server.data-availability.enable is required to start a provider server")
	}

	if config.ProviderServer.DataAvailability.ParentChainNodeURL == "" || config.ProviderServer.DataAvailability.ParentChainNodeURL == "none" {
		return errors.New("--provider-server.data-availability.parent-chain-node-url is required to start a provider server")
	}

	if config.ProviderServer.DataAvailability.SequencerInboxAddress == "" || config.ProviderServer.DataAvailability.SequencerInboxAddress == "none" {
		return errors.New("sequencer-inbox-address must be set to a valid L1 URL and contract address")
	}

	l1Client, err := das.GetL1Client(ctx, config.ProviderServer.DataAvailability.ParentChainConnectionAttempts, config.ProviderServer.DataAvailability.ParentChainNodeURL)
	if err != nil {
		return err
	}

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
	l1Reader, err := headerreader.New(ctx, l1Client, func() *headerreader.Config { return &headerreader.DefaultConfig }, arbSys)
	if err != nil {
		return err
	}

	seqInboxAddr, err := das.OptionalAddressFromString(config.ProviderServer.DataAvailability.SequencerInboxAddress)
	if err != nil {
		return err
	}
	if seqInboxAddr == nil {
		return errors.New("must provide --provider-server.data-availability.sequencer-inbox-address set to a valid contract address or 'none'")
	}

	var dataSigner signature.DataSignerFunc
	if config.WithDataSigner && config.ProviderServer.EnableDAWriter {
		l1ChainId, err := l1Client.ChainID(ctx)
		if err != nil {
			return fmt.Errorf("couldn't read L1 chainid: %w", err)
		}
		if _, dataSigner, err = util.OpenWallet("data-signer", &config.DataSignerWallet, l1ChainId); err != nil {
			return err
		}
	}

	log.Info("Starting json rpc server", "addr", config.ProviderServer.Addr, "port", config.ProviderServer.Port)
	providerServer, closeFn, err := dapserver.NewServer(ctx, &config.ProviderServer, dataSigner, l1Client, l1Reader, *seqInboxAddr)
	if err != nil {
		return err
	}

	if l1Reader != nil {
		l1Reader.Start(ctx)
	}

	<-sigint

	if err = providerServer.Shutdown(ctx); err != nil {
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
