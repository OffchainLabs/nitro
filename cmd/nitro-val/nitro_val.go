package main

import (
	"context"
	"fmt"
	"math"
	_ "net/http/pprof" // #nosec G108
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	flag "github.com/spf13/pflag"

	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	_ "github.com/offchainlabs/nitro/execution/nodeInterface"
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
	stackConf.HTTPBodyLimit = math.MaxInt
	stackConf.WSReadLimit = math.MaxInt64
	nodeConfig.HTTP.Apply(&stackConf)
	nodeConfig.WS.Apply(&stackConf)
	nodeConfig.Auth.Apply(&stackConf)
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

	log.Info("Running Arbitrum nitro validation node", "revision", vcsRevision, "vcs.time", vcsTime)

	liveNodeConfig := genericconf.NewLiveConfig[*ValidationNodeConfig](args, nodeConfig, ParseNode)
	liveNodeConfig.SetOnReloadHook(func(oldCfg *ValidationNodeConfig, newCfg *ValidationNodeConfig) error {

		return genericconf.InitLog(newCfg.LogType, newCfg.LogLevel, &newCfg.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
	})

	valnode.EnsureValidationExposedViaAuthRPC(&stackConf)

	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}

	err = util.StartMetricsAndPProf(&util.MetricsPProfOpts{
		Metrics:       nodeConfig.Metrics,
		MetricsServer: nodeConfig.MetricsServer,
		PProf:         nodeConfig.PProf,
		PprofCfg:      nodeConfig.PprofCfg,
	})
	if err != nil {
		log.Error("Error starting metrics", "error", err)
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
	defer valNode.Stop()
	err = stack.Start()
	if err != nil {
		fatalErrChan <- fmt.Errorf("error starting stack: %w", err)
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
