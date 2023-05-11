package main

import (
	"context"
	"fmt"
	_ "net/http/pprof" // #nosec G108
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/cmd/util/nodehelpers"
	_ "github.com/offchainlabs/nitro/nodeInterface"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s --help \n", name)
}

func main() {
	os.Exit(mainImpl())
}

// Returns the exit code
func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	nodeConfig, err := ParseNode(ctx, args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}
	stackConf := DefaultValidationNodeStackConfig
	stackConf.DataDir = "" // ephemeral
	nodeConfig.HTTP.Apply(&stackConf)
	nodeConfig.WS.Apply(&stackConf)
	nodeConfig.AuthRPC.Apply(&stackConf)
	nodeConfig.IPC.Apply(&stackConf)
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	vcsRevision, vcsTime := confighelpers.GetVersion()
	stackConf.Version = vcsRevision

	pathResolver := func(dataDir string) func(string) string {
		return func(path string) string {
			if filepath.IsAbs(path) {
				return path
			}
			return filepath.Join(dataDir, path)
		}
	}

	err = nodehelpers.InitLog(nodeConfig.LogType, log.Lvl(nodeConfig.LogLevel), &nodeConfig.FileLogging, pathResolver(nodeConfig.Persistent.Chain))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		return 1
	}
	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := pathResolver(nodeConfig.Persistent.Chain)("jwtsecret")
		if err := nodehelpers.TryCreatingJWTSecret(filename); err != nil {
			log.Error("Failed to prepare jwt secret file", "err", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}

	log.Info("Running Arbitrum nitro validation node", "revision", vcsRevision, "vcs.time", vcsTime)

	liveNodeConfig := nodehelpers.NewLiveConfig[*ValidationNodeConfig](args, nodeConfig, stackConf.ResolvePath, ParseNode)
	liveNodeConfig.SetOnReloadHook(func(oldCfg *ValidationNodeConfig, newCfg *ValidationNodeConfig) error {
		dataDir := newCfg.Persistent.Chain
		return nodehelpers.InitLog(newCfg.LogType, log.Lvl(newCfg.LogLevel), &newCfg.FileLogging, pathResolver(dataDir))
	})
	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}

	if nodeConfig.Metrics {
		go metrics.CollectProcessMetrics(nodeConfig.MetricsServer.UpdateInterval)

		if nodeConfig.MetricsServer.Addr != "" {
			address := fmt.Sprintf("%v:%v", nodeConfig.MetricsServer.Addr, nodeConfig.MetricsServer.Port)
			if nodeConfig.MetricsServer.Pprof {
				nodehelpers.StartPprof(address)
			} else {
				exp.Setup(address)
			}
		}
	} else if nodeConfig.MetricsServer.Pprof {
		flag.Usage()
		log.Error("--metrics must be enabled in order to use pprof with the metrics server")
		return 1
	}

	fatalErrChan := make(chan error, 10)

	valNode, err := valnode.CreateValidationNode(
		func() *valnode.Config { return &liveNodeConfig.Get().Validation },
		stack,
		fatalErrChan,
	)
	if err != nil {
		log.Error("couldn't init validation node", "err", err)
		return 1
	}

	err = valNode.Start(ctx)
	if err != nil {
		log.Error("error starting validator node", "err", err)
		return 1
	}
	err = stack.Start()
	if err != nil {
		fatalErrChan <- errors.Wrap(err, "error starting stack")
	}
	defer stack.Close()

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

func ParseNode(ctx context.Context, args []string) (*ValidationNodeConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	ValidationNodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, err
	}

	var nodeConfig ValidationNodeConfig
	if err := confighelpers.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, err
	}

	// Don't print wallet passwords
	if nodeConfig.Conf.Dump {
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

	err = nodeConfig.Validate()
	if err != nil {
		return nil, err
	}
	return &nodeConfig, nil
}
