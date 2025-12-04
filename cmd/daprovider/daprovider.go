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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	dapserver "github.com/offchainlabs/nitro/daprovider/server"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

type ParentChainConfig struct {
	NodeURL               string `koanf:"node-url"`
	ConnectionAttempts    int    `koanf:"connection-attempts"`
	SequencerInboxAddress string `koanf:"sequencer-inbox-address"`
}

type Config struct {
	Mode             string                   `koanf:"mode"`
	ProviderServer   dapserver.ServerConfig   `koanf:"provider-server"`
	WithDataSigner   bool                     `koanf:"with-data-signer"`
	DataSignerWallet genericconf.WalletConfig `koanf:"data-signer-wallet"`

	// Shared parent chain connection config
	ParentChain ParentChainConfig `koanf:"parent-chain"`

	// Mode-specific configs
	Anytrust    das.DataAvailabilityConfig `koanf:"anytrust"`
	ReferenceDA referenceda.Config         `koanf:"referenceda"`

	Conf     genericconf.ConfConfig `koanf:"conf"`
	LogLevel string                 `koanf:"log-level"`
	LogType  string                 `koanf:"log-type"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
}

var DefaultParentChainConfig = ParentChainConfig{
	NodeURL:               "",
	ConnectionAttempts:    15,
	SequencerInboxAddress: "",
}

var DefaultConfig = Config{
	Mode:             "", // Must be explicitly set
	ProviderServer:   dapserver.DefaultServerConfig,
	WithDataSigner:   false,
	DataSignerWallet: arbnode.DefaultBatchPosterL1WalletConfig,
	ParentChain:      DefaultParentChainConfig,
	Anytrust:         das.DefaultDataAvailabilityConfig,
	ReferenceDA:      referenceda.DefaultConfig,
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
	f.String("mode", string(DefaultConfig.Mode), "DA provider mode (anytrust or referenceda) - REQUIRED")
	f.Bool("with-data-signer", DefaultConfig.WithDataSigner, "set to enable data signing when processing store requests. If enabled requires data-signer-wallet config")
	genericconf.WalletConfigAddOptions("data-signer-wallet", f, DefaultConfig.DataSignerWallet.Pathname)

	f.Bool("metrics", DefaultConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Bool("pprof", DefaultConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	f.String("log-level", DefaultConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultConfig.LogType, "log type (plaintext or json)")

	dapserver.ServerConfigAddOptions("provider-server", f)

	// Add shared parent chain connection options
	f.String("parent-chain.node-url", DefaultParentChainConfig.NodeURL, "URL for parent chain node")
	f.Int("parent-chain.connection-attempts", DefaultParentChainConfig.ConnectionAttempts, "parent chain RPC connection attempts (spaced out at least 1 second per attempt, 0 to retry infinitely)")
	f.String("parent-chain.sequencer-inbox-address", DefaultParentChainConfig.SequencerInboxAddress, "parent chain address of SequencerInbox contract")

	// Add mode-specific options
	das.DataAvailabilityConfigAddDaserverOptions("anytrust", f)
	referenceda.ConfigAddOptions("referenceda", f)

	genericconf.ConfConfigAddOptions("conf", f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	if err = das.FixKeysetCLIParsing("anytrust.rpc-aggregator.backends", k); err != nil {
		return nil, err
	}

	var config Config
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}

	if config.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"anytrust.key.priv-key":               "",
			"referenceda.signing-key.private-key": "",
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

	// Validate mode
	if config.Mode == "" {
		return errors.New("--mode must be explicitly specified (anytrust or referenceda)")
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

	// Mode-specific validation and setup
	var l1Client *ethclient.Client
	var l1Reader *headerreader.HeaderReader
	var seqInboxAddr common.Address
	var dataSigner signature.DataSignerFunc

	if config.Mode == "anytrust" {
		if !config.Anytrust.Enable {
			return errors.New("--anytrust.enable is required to start an AnyTrust provider server")
		}

		if config.ParentChain.NodeURL == "" {
			return errors.New("--parent-chain.node-url is required to start an AnyTrust provider server")
		}

		if config.ParentChain.SequencerInboxAddress == "" {
			return errors.New("--parent-chain.sequencer-inbox-address must be set to a valid L1 contract address")
		}

		l1Client, err = das.GetL1Client(ctx, config.ParentChain.ConnectionAttempts, config.ParentChain.NodeURL)
		if err != nil {
			return err
		}

		arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
		l1Reader, err = headerreader.New(ctx, l1Client, func() *headerreader.Config { return &headerreader.DefaultConfig }, arbSys)
		if err != nil {
			return err
		}

		seqInboxAddrPtr, err := das.OptionalAddressFromString(config.ParentChain.SequencerInboxAddress)
		if err != nil {
			return err
		}
		if seqInboxAddrPtr == nil {
			return errors.New("must provide --parent-chain.sequencer-inbox-address set to a valid contract address")
		}
		seqInboxAddr = *seqInboxAddrPtr

		if config.WithDataSigner && config.ProviderServer.EnableDAWriter {
			l1ChainId, err := l1Client.ChainID(ctx)
			if err != nil {
				return fmt.Errorf("couldn't read L1 chainid: %w", err)
			}
			if _, dataSigner, err = util.OpenWallet("data-signer", &config.DataSignerWallet, l1ChainId); err != nil {
				return err
			}
		}
	} else if config.Mode == "referenceda" {
		if !config.ReferenceDA.Enable {
			return errors.New("--referenceda.enable is required to start a ReferenceDA provider server")
		}
		l1Client, err = das.GetL1Client(ctx, config.ParentChain.ConnectionAttempts, config.ParentChain.NodeURL)
		if err != nil {
			return err
		}
	}

	// Create reader/writer/validator based on mode
	var reader daprovider.Reader
	var writer daprovider.Writer
	var validator daprovider.Validator
	var headerBytes []byte
	var cleanupFuncs []func()

	switch config.Mode {
	case "anytrust":
		factory := das.NewFactory(
			&config.Anytrust,
			dataSigner,
			l1Client,
			l1Reader,
			seqInboxAddr,
			config.ProviderServer.EnableDAWriter,
		)
		if err := factory.ValidateConfig(); err != nil {
			return err
		}
		var readerCleanup func()
		reader, readerCleanup, err = factory.CreateReader(ctx)
		if err != nil {
			return err
		}
		if readerCleanup != nil {
			cleanupFuncs = append(cleanupFuncs, readerCleanup)
		}
		if config.ProviderServer.EnableDAWriter {
			var writerCleanup func()
			writer, writerCleanup, err = factory.CreateWriter(ctx)
			if err != nil {
				return err
			}
			if writerCleanup != nil {
				cleanupFuncs = append(cleanupFuncs, writerCleanup)
			}
		}
		headerBytes = das.SupportedHeaderBytes

	case "referenceda":
		factory := referenceda.NewFactory(
			&config.ReferenceDA,
			dataSigner,
			l1Client,
			config.ProviderServer.EnableDAWriter,
		)
		if err := factory.ValidateConfig(); err != nil {
			return err
		}
		reader, _, err = factory.CreateReader(ctx)
		if err != nil {
			return err
		}
		if config.ProviderServer.EnableDAWriter {
			writer, _, err = factory.CreateWriter(ctx)
			if err != nil {
				return err
			}
		}
		validator, _, err = factory.CreateValidator(ctx)
		if err != nil {
			return err
		}
		headerBytes = []byte{daprovider.DACertificateMessageHeaderFlag}

	default:
		return fmt.Errorf("unsupported DA provider mode: %s", config.Mode)
	}

	log.Info("Starting json rpc server", "mode", config.Mode, "addr", config.ProviderServer.Addr, "port", config.ProviderServer.Port)
	providerServer, err := dapserver.NewServerWithDAPProvider(ctx, &config.ProviderServer, reader, writer, validator, headerBytes, data_streaming.PayloadCommitmentVerifier())
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

	// Call all cleanup functions
	for _, cleanup := range cleanupFuncs {
		cleanup()
	}

	if l1Reader != nil && l1Reader.Started() {
		l1Reader.StopAndWait()
	}
	return nil
}
