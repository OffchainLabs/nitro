package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/execution/chess"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func main() {
	os.Exit(mainImpl())
}

var DefaultChessNodeStackConfig = node.Config{
	DataDir:             "chess",
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{""},
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{""},
	GraphQLVirtualHosts: []string{""},
	P2P: p2p.Config{
		ListenAddr: ":30303",
		MaxPeers:   50,
		NAT:        nat.Any(),
	},
}

// Returns the exit code
func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	nodeConfig, err := ParseNode(ctx, args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, func(input string) {
			fmt.Print(input)
		})
		return 1
	}
	stackConf := DefaultChessNodeStackConfig
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

	log.Info("Running Arbitrum bitro chess node", "revision", vcsRevision, "vcs.time", vcsTime)

	liveNodeConfig := genericconf.NewLiveConfig[*NodeConfig](args, nodeConfig, ParseNode)
	liveNodeConfig.SetOnReloadHook(func(oldCfg *NodeConfig, newCfg *NodeConfig) error {

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

	confFetcher := func() *rpcclient.ClientConfig { return &liveNodeConfig.Get().ParentChain.Connection }
	rpcClient := rpcclient.NewRpcClient(confFetcher, nil)
	err = rpcClient.Start(ctx)
	if err != nil {
		log.Crit("couldn't connect to L1", "err", err)
	}
	l1Client := ethclient.NewClient(rpcClient)
	parentChainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return 3
	}
	fatalErrChan := make(chan error, 10)

	execNode := chess.NewChessNode(chess.NewChessEngine())
	arbDb, err := stack.OpenDatabaseWithExtraOptions("arbitrumdata", 0, 0, "arbitrumdata/", false, nodeConfig.Persistent.Pebble.ExtraOptions("arbitrumdata"))
	if err != nil {
		log.Error("opening DB", "err", err)
		return 7
	}
	defer func() {
		err := arbDb.Close()
		if err != nil {
			log.Error("closing DB", "err", err)
		}
	}()
	log.Info("DefAddress", "address", DefAddresses)
	consensus, err := arbnode.CreateNodeExecutionClient(
		ctx,
		stack,
		execNode,
		arbDb,
		&NodeConfigFetcher{liveNodeConfig},
		nil,
		l1Client,
		&DefAddresses,
		nil, // TODO
		nil,
		nil,
		fatalErrChan,
		parentChainID,
		nil,
		"",
	)
	if err != nil {
		log.Error("opening DB", "err", err)
		return 11
	}

	if err := consensus.Start(ctx); err != nil {
		log.Error("starting consensus", "err", err)
	}
	defer consensus.StopAndWait()

	// err = stack.Start()
	// if err != nil {
	// 	log.Error("starting stack", "err", err)
	// 	fatalErrChan <- fmt.Errorf("error starting stack: %w", err)
	// }
	// defer stack.Close()

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
