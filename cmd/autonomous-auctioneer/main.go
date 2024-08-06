package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/timeboost"
)

func main() {
	os.Exit(mainImpl())
}

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics(cfg *AutonomousAuctioneerConfig) error {
	mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	if cfg.Metrics && !metrics.Enabled {
		return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
	}
	if cfg.Metrics && cfg.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if cfg.Metrics {
		go metrics.CollectProcessMetrics(time.Second)
		exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	}
	if cfg.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
}

func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	nodeConfig, err := parseAuctioneerArgs(ctx, args)
	if err != nil {
		// confighelpers.PrintErrorAndExit(err, printSampleUsage)
		panic(err)
	}
	stackConf := DefaultAuctioneerStackConfig
	stackConf.DataDir = "" // ephemeral
	nodeConfig.HTTP.Apply(&stackConf)
	nodeConfig.WS.Apply(&stackConf)
	nodeConfig.IPC.Apply(&stackConf)
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	vcsRevision, strippedRevision, vcsTime := confighelpers.GetVersion()
	stackConf.Version = strippedRevision

	pathResolver := func(workdir string) func(string) string {
		if workdir == "" {
			workdir, err = os.Getwd()
			if err != nil {
				log.Warn("Failed to get workdir", "err", err)
			}
		}
		return func(path string) string {
			if filepath.IsAbs(path) {
				return path
			}
			return filepath.Join(workdir, path)
		}
	}

	err = genericconf.InitLog(nodeConfig.LogType, nodeConfig.LogLevel, &nodeConfig.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		return 1
	}
	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := pathResolver(nodeConfig.Persistent.GlobalConfig)("jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
			log.Error("Failed to prepare jwt secret file", "err", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}

	liveNodeConfig := genericconf.NewLiveConfig[*AutonomousAuctioneerConfig](args, nodeConfig, parseAuctioneerArgs)
	liveNodeConfig.SetOnReloadHook(func(oldCfg *AutonomousAuctioneerConfig, newCfg *AutonomousAuctioneerConfig) error {

		return genericconf.InitLog(newCfg.LogType, newCfg.LogLevel, &newCfg.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
	})

	timeboost.EnsureBidValidatorExposedViaRPC(&stackConf)

	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}

	if err := startMetrics(nodeConfig); err != nil {
		log.Error("Error starting metrics", "error", err)
		return 1
	}

	fatalErrChan := make(chan error, 10)
	log.Info("Running Arbitrum ", "revision", vcsRevision, "vcs.time", vcsTime)
	// valNode, err := valnode.CreateValidationNode(
	// 	func() *valnode.Config { return &liveNodeConfig.Get().Validation },
	// 	stack,
	// 	fatalErrChan,
	// )
	// if err != nil {
	// 	log.Error("couldn't init validation node", "err", err)
	// 	return 1
	// }

	// err = valNode.Start(ctx)
	// if err != nil {
	// 	log.Error("error starting validator node", "err", err)
	// 	return 1
	// }
	err = stack.Start()
	if err != nil {
		fatalErrChan <- fmt.Errorf("error starting stack: %w", err)
	}
	defer stack.Close()

	if nodeConfig.Mode == autonomousAuctioneerMode {
		auctioneer, err := timeboost.NewAuctioneerServer(
			nil,
			nil,
			nil,
			common.Address{},
			"",
			nil,
		)
		if err != nil {
			log.Error("Error creating new auctioneer", "error", err)
			return 1
		}
		auctioneer.Start(ctx)
	} else if nodeConfig.Mode == bidValidatorMode {
		bidValidator, err := timeboost.NewBidValidator(
			nil,
			nil,
			nil,
			common.Address{},
			"",
			nil,
		)
		if err != nil {
			log.Error("Error creating new auctioneer", "error", err)
			return 1
		}
		if err = bidValidator.Initialize(ctx); err != nil {
			log.Error("error initializing bid validator", "err", err)
			return 1
		}
		bidValidator.Start(ctx)
	} else {
		log.Crit("Unknown mode, should be either autonomous-auctioneer or bid-validator", "mode", nodeConfig.Mode)

	}

	liveNodeConfig.Start(ctx)
	defer liveNodeConfig.StopAndWait()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	select {
	case err := <-fatalErrChan:
		log.Error("shutting down due to fatal error", "err", err)
		defer log.Error("shut down due to fatal error", "err", err)
		exitCode = 1
	case <-sigint:
		log.Info("shutting down because of sigint")
	}
	// cause future ctrl+c's to panic
	close(sigint)
	return exitCode
}

func parseAuctioneerArgs(ctx context.Context, args []string) (*AutonomousAuctioneerConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	// ValidationNodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, err
	}

	var cfg AutonomousAuctioneerConfig
	if err := confighelpers.EndCommonParse(k, &cfg); err != nil {
		return nil, err
	}

	// Don't print wallet passwords
	if cfg.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"l1.wallet.password":        "",
			"l1.wallet.private-key":     "",
			"l2.dev-wallet.password":    "",
			"l2.dev-wallet.private-key": "",
		})
		if err != nil {
			return nil, err
		}
	}

	// Don't pass around wallet contents with normal configuration
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}