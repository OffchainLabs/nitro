// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	flag "github.com/spf13/pflag"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/graphql"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/execution"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	_ "github.com/offchainlabs/nitro/nodeInterface"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s --help \n", name)
}

func addUnlockWallet(accountManager *accounts.Manager, walletConf *genericconf.WalletConfig) (common.Address, error) {
	var devAddr common.Address

	var devPrivKey *ecdsa.PrivateKey
	var err error
	if walletConf.PrivateKey != "" {
		devPrivKey, err = crypto.HexToECDSA(walletConf.PrivateKey)
		if err != nil {
			return common.Address{}, err
		}

		devAddr = crypto.PubkeyToAddress(devPrivKey.PublicKey)

		log.Info("Dev node funded private key", "priv", walletConf.PrivateKey)
		log.Info("Funded public address", "addr", devAddr)
	}

	if walletConf.Pathname != "" {
		myKeystore := keystore.NewKeyStore(walletConf.Pathname, keystore.StandardScryptN, keystore.StandardScryptP)
		accountManager.AddBackend(myKeystore)
		var account accounts.Account
		if myKeystore.HasAddress(devAddr) {
			account.Address = devAddr
			account, err = myKeystore.Find(account)
		} else if walletConf.Account != "" && myKeystore.HasAddress(common.HexToAddress(walletConf.Account)) {
			account.Address = common.HexToAddress(walletConf.Account)
			account, err = myKeystore.Find(account)
		} else {
			if walletConf.Password() == nil {
				return common.Address{}, errors.New("l2 password not set")
			}
			if devPrivKey == nil {
				return common.Address{}, errors.New("l2 private key not set")
			}
			account, err = myKeystore.ImportECDSA(devPrivKey, *walletConf.Password())
		}
		if err != nil {
			return common.Address{}, err
		}
		if walletConf.Password() == nil {
			return common.Address{}, errors.New("l2 password not set")
		}
		err = myKeystore.Unlock(account, *walletConf.Password())
		if err != nil {
			return common.Address{}, err
		}
	}
	return devAddr, nil
}

func closeDb(db io.Closer, name string) {
	if db != nil {
		err := db.Close()
		// unfortunately the freezer db means we can't just use errors.Is
		if err != nil && !strings.Contains(err.Error(), leveldb.ErrClosed.Error()) {
			log.Warn("failed to close database on shutdown", "db", name, "err", err)
		}
	}
}

func main() {
	os.Exit(mainImpl())
}

// Returns the exit code
func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	nodeConfig, l1Wallet, l2DevWallet, err := ParseNode(ctx, args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}
	stackConf := node.DefaultConfig
	stackConf.DataDir = nodeConfig.Persistent.Chain
	stackConf.DBEngine = "leveldb"
	nodeConfig.HTTP.Apply(&stackConf)
	nodeConfig.WS.Apply(&stackConf)
	nodeConfig.AuthRPC.Apply(&stackConf)
	nodeConfig.IPC.Apply(&stackConf)
	nodeConfig.GraphQL.Apply(&stackConf)
	if nodeConfig.WS.ExposeAll {
		stackConf.WSModules = append(stackConf.WSModules, "personal")
	}
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	vcsRevision, vcsTime := confighelpers.GetVersion()
	stackConf.Version = vcsRevision

	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := stackConf.ResolvePath("jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
			log.Error("Failed to prepare jwt secret file", "err", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}

	err = genericconf.InitLog(nodeConfig.LogType, log.Lvl(nodeConfig.LogLevel), &nodeConfig.FileLogging, stackConf.ResolvePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		return 1
	}
	if nodeConfig.Node.Archive {
		log.Warn("--node.archive has been deprecated. Please use --node.caching.archive instead.")
		nodeConfig.Node.Caching.Archive = true
	}

	log.Info("Running Arbitrum nitro node", "revision", vcsRevision, "vcs.time", vcsTime)

	if nodeConfig.Node.Dangerous.NoL1Listener {
		nodeConfig.Node.L1Reader.Enable = false
		nodeConfig.Node.BatchPoster.Enable = false
		nodeConfig.Node.DelayedSequencer.Enable = false
	} else {
		nodeConfig.Node.L1Reader.Enable = true
	}

	if nodeConfig.Node.Sequencer.Enable {
		if nodeConfig.Node.ForwardingTarget() != "" {
			flag.Usage()
			log.Crit("forwarding-target cannot be set when sequencer is enabled")
		}
		if nodeConfig.Node.L1Reader.Enable && nodeConfig.Node.InboxReader.HardReorg {
			flag.Usage()
			log.Crit("hard reorgs cannot safely be enabled with sequencer mode enabled")
		}
	} else if nodeConfig.Node.ForwardingTargetImpl == "" {
		flag.Usage()
		log.Crit("forwarding-target unset, and not sequencer (can set to \"null\" to disable forwarding)")
	}

	var l1TransactionOpts *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	var l1TransactionOptsValidator *bind.TransactOpts
	var l1TransactionOptsBatchPoster *bind.TransactOpts
	sequencerNeedsKey := (nodeConfig.Node.Sequencer.Enable && !nodeConfig.Node.Feed.Output.DisableSigning) || nodeConfig.Node.BatchPoster.Enable
	validatorNeedsKey := nodeConfig.Node.Staker.OnlyCreateWalletContract || nodeConfig.Node.Staker.Enable && !strings.EqualFold(nodeConfig.Node.Staker.Strategy, "watchtower")

	l1Wallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultL1WalletConfig := conf.DefaultL1WalletConfig
	defaultL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	nodeConfig.Node.Staker.L1Wallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultValidatorL1WalletConfig := staker.DefaultValidatorL1WalletConfig
	defaultValidatorL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	nodeConfig.Node.BatchPoster.L1Wallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultBatchPosterL1WalletConfig := arbnode.DefaultBatchPosterL1WalletConfig
	defaultBatchPosterL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	if nodeConfig.Node.Staker.L1Wallet == defaultValidatorL1WalletConfig && nodeConfig.Node.BatchPoster.L1Wallet == defaultBatchPosterL1WalletConfig {
		if sequencerNeedsKey || validatorNeedsKey || l1Wallet.OnlyCreateKey {
			l1TransactionOpts, dataSigner, err = util.OpenWallet("l1", l1Wallet, new(big.Int).SetUint64(nodeConfig.L1.ChainID))
			if err != nil {
				flag.Usage()
				log.Crit("error opening parent chain wallet", "path", l1Wallet.Pathname, "account", l1Wallet.Account, "err", err)
			}
			l1TransactionOptsBatchPoster = l1TransactionOpts
			l1TransactionOptsValidator = l1TransactionOpts
		}
	} else {
		if *l1Wallet != defaultL1WalletConfig {
			log.Crit("--parent-chain.wallet cannot be set if either --node.staker.l1-wallet or --node.batch-poster.l1-wallet are set")
		}
		if sequencerNeedsKey || nodeConfig.Node.BatchPoster.L1Wallet.OnlyCreateKey {
			l1TransactionOptsBatchPoster, dataSigner, err = util.OpenWallet("l1-batch-poster", &nodeConfig.Node.BatchPoster.L1Wallet, new(big.Int).SetUint64(nodeConfig.L1.ChainID))
			if err != nil {
				flag.Usage()
				log.Crit("error opening Batch poster parent chain wallet", "path", nodeConfig.Node.BatchPoster.L1Wallet.Pathname, "account", nodeConfig.Node.BatchPoster.L1Wallet.Account, "err", err)
			}
		}
		if validatorNeedsKey || nodeConfig.Node.Staker.L1Wallet.OnlyCreateKey {
			l1TransactionOptsValidator, _, err = util.OpenWallet("l1-validator", &nodeConfig.Node.Staker.L1Wallet, new(big.Int).SetUint64(nodeConfig.L1.ChainID))
			if err != nil {
				flag.Usage()
				log.Crit("error opening Validator parent chain wallet", "path", nodeConfig.Node.Staker.L1Wallet.Pathname, "account", nodeConfig.Node.Staker.L1Wallet.Account, "err", err)
			}
		}
	}

	combinedL2ChainInfoFile := nodeConfig.L2.ChainInfoFiles
	if nodeConfig.L2.ChainInfoIpfsUrl != "" {
		l2ChainInfoIpfsFile, err := util.GetL2ChainInfoIpfsFile(ctx, nodeConfig.L2.ChainInfoIpfsUrl, nodeConfig.L2.ChainInfoIpfsDownloadPath)
		if err != nil {
			log.Error("error getting chain info file from ipfs", "err", err)
		}
		combinedL2ChainInfoFile = append(combinedL2ChainInfoFile, l2ChainInfoIpfsFile)
	}

	if nodeConfig.Node.Staker.Enable {
		if !nodeConfig.Node.L1Reader.Enable {
			flag.Usage()
			log.Crit("validator must have the parent chain reader enabled")
		}
		strategy, err := nodeConfig.Node.Staker.ParseStrategy()
		if err != nil {
			log.Crit("couldn't parse staker strategy", "err", err)
		}
		if strategy != staker.WatchtowerStrategy && !nodeConfig.Node.Staker.Dangerous.WithoutBlockValidator {
			nodeConfig.Node.BlockValidator.Enable = true
		}
	}

	liveNodeConfig := genericconf.NewLiveConfig[*NodeConfig](args, nodeConfig, func(ctx context.Context, args []string) (*NodeConfig, error) {
		nodeConfig, _, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})

	var rollupAddrs chaininfo.RollupAddresses
	var l1Client *ethclient.Client
	if nodeConfig.Node.L1Reader.Enable {
		confFetcher := func() *rpcclient.ClientConfig { return &liveNodeConfig.Get().L1.Connection }
		rpcClient := rpcclient.NewRpcClient(confFetcher, nil)
		err := rpcClient.Start(ctx)
		if err != nil {
			log.Crit("couldn't connect to L1", "err", err)
		}
		l1Client = ethclient.NewClient(rpcClient)
		l1ChainId, err := l1Client.ChainID(ctx)
		if err != nil {
			log.Crit("couldn't read L1 chainid", "err", err)
		}
		if l1ChainId.Uint64() != nodeConfig.L1.ChainID {
			log.Crit("L1 chainID doesn't fit config", "found", l1ChainId.Uint64(), "expected", nodeConfig.L1.ChainID)
		}

		log.Info("connected to l1 chain", "l1url", nodeConfig.L1.Connection.URL, "l1chainid", nodeConfig.L1.ChainID)

		rollupAddrs, err = chaininfo.GetRollupAddressesConfig(nodeConfig.L2.ChainID, nodeConfig.L2.ChainName, combinedL2ChainInfoFile, nodeConfig.L2.ChainInfoJson)
		if err != nil {
			log.Crit("error getting rollup addresses", "err", err)
		}
	}

	if nodeConfig.Node.Staker.OnlyCreateWalletContract {
		if !nodeConfig.Node.Staker.UseSmartContractWallet {
			flag.Usage()
			log.Crit("--node.validator.only-create-wallet-contract requires --node.validator.use-smart-contract-wallet")
		}
		l1Reader, err := headerreader.New(ctx, l1Client, func() *headerreader.Config { return &liveNodeConfig.Get().Node.L1Reader })
		if err != nil {
			log.Crit("failed to get L1 headerreader", "error", err)

		}

		// Just create validator smart wallet if needed then exit
		deployInfo, err := chaininfo.GetRollupAddressesConfig(nodeConfig.L2.ChainID, nodeConfig.L2.ChainName, combinedL2ChainInfoFile, nodeConfig.L2.ChainInfoJson)
		if err != nil {
			log.Crit("error getting rollup addresses config", "err", err)
		}
		addr, err := staker.GetValidatorWalletContract(ctx, deployInfo.ValidatorWalletCreator, int64(deployInfo.DeployedAt), l1TransactionOptsValidator, l1Reader, true)
		if err != nil {
			log.Crit("error creating validator wallet contract", "error", err, "address", l1TransactionOptsValidator.From.Hex())
		}
		fmt.Printf("Created validator smart contract wallet at %s, remove --node.validator.only-create-wallet-contract and restart\n", addr.String())
		return 0
	}

	if nodeConfig.Node.Caching.Archive && nodeConfig.Node.TxLookupLimit != 0 {
		log.Info("retaining ability to lookup full transaction history as archive mode is enabled")
		nodeConfig.Node.TxLookupLimit = 0
	}

	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}
	{
		devAddr, err := addUnlockWallet(stack.AccountManager(), l2DevWallet)
		if err != nil {
			flag.Usage()
			log.Crit("error opening L2 dev wallet", "err", err)
		}
		if devAddr != (common.Address{}) {
			nodeConfig.Init.DevInitAddr = devAddr.String()
		}
	}

	chainDb, l2BlockChain, err := openInitializeChainDb(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.L2.ChainID), execution.DefaultCacheConfigFor(stack, &nodeConfig.Node.Caching), l1Client, rollupAddrs)
	defer closeDb(chainDb, "chainDb")
	if l2BlockChain != nil {
		// Calling Stop on the blockchain multiple times does nothing
		defer l2BlockChain.Stop()
	}
	if err != nil {
		flag.Usage()
		log.Error("error initializing database", "err", err)
		return 1
	}

	arbDb, err := stack.OpenDatabase("arbitrumdata", 0, 0, "", false)
	defer closeDb(arbDb, "arbDb")
	if err != nil {
		log.Error("failed to open database", "err", err)
		return 1
	}

	if nodeConfig.Init.ThenQuit {
		return 0
	}

	if l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee && !nodeConfig.Node.DataAvailability.Enable {
		flag.Usage()
		log.Error("a data availability service must be configured for this chain (see the --node.data-availability family of options)")
		return 1
	}

	if nodeConfig.Metrics {
		go metrics.CollectProcessMetrics(nodeConfig.MetricsServer.UpdateInterval)

		if nodeConfig.MetricsServer.Addr != "" {
			address := fmt.Sprintf("%v:%v", nodeConfig.MetricsServer.Addr, nodeConfig.MetricsServer.Port)
			if nodeConfig.MetricsServer.Pprof {
				genericconf.StartPprof(address)
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

	var valNode *valnode.ValidationNode
	if nodeConfig.Node.BlockValidator.Enable && (nodeConfig.Node.BlockValidator.ValidationServer.URL == "self" || nodeConfig.Node.BlockValidator.ValidationServer.URL == "self-auth") {
		valNode, err = valnode.CreateValidationNode(
			func() *valnode.Config { return &liveNodeConfig.Get().Validation },
			stack,
			fatalErrChan,
		)
		if err != nil {
			valNode = nil
			log.Warn("couldn't init validation node", "err", err)
		}
	}

	currentNode, err := arbnode.CreateNode(
		ctx,
		stack,
		chainDb,
		arbDb,
		&NodeConfigFetcher{liveNodeConfig},
		l2BlockChain,
		l1Client,
		&rollupAddrs,
		l1TransactionOptsValidator,
		l1TransactionOptsBatchPoster,
		dataSigner,
		fatalErrChan,
	)
	if err != nil {
		log.Error("failed to create node", "err", err)
		return 1
	}
	liveNodeConfig.SetOnReloadHook(func(oldCfg *NodeConfig, newCfg *NodeConfig) error {
		if err := genericconf.InitLog(newCfg.LogType, log.Lvl(newCfg.LogLevel), &newCfg.FileLogging, stackConf.ResolvePath); err != nil {
			return fmt.Errorf("failed to re-init logging: %w", err)
		}
		return currentNode.OnConfigReload(&oldCfg.Node, &newCfg.Node)
	})

	if nodeConfig.Node.Dangerous.NoL1Listener && nodeConfig.Init.DevInit {
		// If we don't have any messages, we're not connected to the L1, and we're using a dev init,
		// we should create our own fake init message.
		count, err := currentNode.TxStreamer.GetMessageCount()
		if err != nil {
			log.Warn("Getmessagecount failed. Assuming new database", "err", err)
			count = 0
		}
		if count == 0 {
			err = currentNode.TxStreamer.AddFakeInitMessage()
			if err != nil {
				panic(err)
			}
		}
	}
	gqlConf := nodeConfig.GraphQL
	if gqlConf.Enable {
		if err := graphql.New(stack, currentNode.Execution.Backend.APIBackend(), currentNode.Execution.FilterSystem, gqlConf.CORSDomain, gqlConf.VHosts); err != nil {
			log.Error("failed to register the GraphQL service", "err", err)
			return 1
		}
	}

	if valNode != nil {
		err = valNode.Start(ctx)
		if err != nil {
			fatalErrChan <- fmt.Errorf("error starting validator node: %w", err)
		} else {
			log.Info("validation node started")
		}
	}
	if err == nil {
		err = currentNode.Start(ctx)
		if err != nil {
			fatalErrChan <- fmt.Errorf("error starting node: %w", err)
		}
	}

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

	currentNode.StopAndWait()

	return exitCode
}

type NodeConfig struct {
	Conf          genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Node          arbnode.Config                  `koanf:"node" reload:"hot"`
	Validation    valnode.Config                  `koanf:"validation" reload:"hot"`
	L1            conf.L1Config                   `koanf:"parent-chain" reload:"hot"`
	L2            conf.L2Config                   `koanf:"chain"`
	LogLevel      int                             `koanf:"log-level" reload:"hot"`
	LogType       string                          `koanf:"log-type" reload:"hot"`
	FileLogging   genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	Persistent    conf.PersistentConfig           `koanf:"persistent"`
	HTTP          genericconf.HTTPConfig          `koanf:"http"`
	WS            genericconf.WSConfig            `koanf:"ws"`
	IPC           genericconf.IPCConfig           `koanf:"ipc"`
	AuthRPC       genericconf.AuthRPCConfig       `koanf:"auth"`
	GraphQL       genericconf.GraphQLConfig       `koanf:"graphql"`
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	Init          InitConfig                      `koanf:"init"`
	Rpc           genericconf.RpcConfig           `koanf:"rpc"`
}

var NodeConfigDefault = NodeConfig{
	Conf:          genericconf.ConfConfigDefault,
	Node:          arbnode.ConfigDefault,
	L1:            conf.L1ConfigDefault,
	L2:            conf.L2ConfigDefault,
	LogLevel:      int(log.LvlInfo),
	LogType:       "plaintext",
	Persistent:    conf.PersistentConfigDefault,
	HTTP:          genericconf.HTTPConfigDefault,
	WS:            genericconf.WSConfigDefault,
	IPC:           genericconf.IPCConfigDefault,
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	valnode.ValidationConfigAddOptions("validation", f)
	conf.L1ConfigAddOptions("parent-chain", f)
	conf.L2ConfigAddOptions("chain", f)
	f.Int("log-level", NodeConfigDefault.LogLevel, "log level")
	f.String("log-type", NodeConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	genericconf.AuthRPCConfigAddOptions("auth", f)
	genericconf.GraphQLConfigAddOptions("graphql", f)
	f.Bool("metrics", NodeConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	InitConfigAddOptions("init", f)
	genericconf.RpcConfigAddOptions("rpc", f)
}

func (c *NodeConfig) ResolveDirectoryNames() error {
	err := c.Persistent.ResolveDirectoryNames()
	if err != nil {
		return err
	}
	c.L1.ResolveDirectoryNames(c.Persistent.Chain)
	c.L2.ResolveDirectoryNames(c.Persistent.Chain)

	return nil
}

func (c *NodeConfig) ShallowClone() *NodeConfig {
	config := &NodeConfig{}
	*config = *c
	return config
}

func (c *NodeConfig) CanReload(new *NodeConfig) error {
	var check func(node, other reflect.Value, path string)
	var err error

	check = func(node, value reflect.Value, path string) {
		if node.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < node.NumField(); i++ {
			fieldTy := node.Type().Field(i)
			if !fieldTy.IsExported() {
				continue
			}
			hot := fieldTy.Tag.Get("reload") == "hot"
			dot := path + "." + fieldTy.Name

			first := node.Field(i).Interface()
			other := value.Field(i).Interface()

			if !hot && !reflect.DeepEqual(first, other) {
				err = fmt.Errorf("illegal change to %v%v%v", colors.Red, dot, colors.Clear)
			} else {
				check(node.Field(i), value.Field(i), dot)
			}
		}
	}

	check(reflect.ValueOf(c).Elem(), reflect.ValueOf(new).Elem(), "config")
	return err
}

func (c *NodeConfig) Validate() error {
	if err := c.L1.Validate(); err != nil {
		return err
	}
	return c.Node.Validate()
}

func (c *NodeConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func ParseNode(ctx context.Context, args []string) (*NodeConfig, *genericconf.WalletConfig, *genericconf.WalletConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, nil, nil, err
	}

	l2ChainId := k.Int64("chain.id")
	l2ChainName := k.String("chain.name")
	l2ChainInfoIpfsUrl := k.String("chain.info-ipfs-url")
	l2ChainInfoIpfsDownloadPath := k.String("chain.info-ipfs-download-path")
	if l2ChainId == 0 && l2ChainName == "" {
		return nil, nil, nil, errors.New("must specify --chain.id or --chain.name to choose rollup")
	}
	l2ChainInfoFiles := k.Strings("chain.info-files")
	l2ChainInfoJson := k.String("chain.info-json")
	chainFound, err := applyChainParameters(ctx, k, uint64(l2ChainId), l2ChainName, l2ChainInfoFiles, l2ChainInfoJson, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
	if err != nil {
		return nil, nil, nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := confighelpers.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, nil, err
	}

	// Don't print wallet passwords
	if nodeConfig.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"parent-chain.wallet.password":    "",
			"parent-chain.wallet.private-key": "",
			"chain.dev-wallet.password":       "",
			"chain.dev-wallet.private-key":    "",
		})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if nodeConfig.Persistent.Chain == "" {
		if !chainFound {
			// If persistent-chain not defined, user not creating custom chain
			if l2ChainId != 0 {
				return nil, nil, nil, fmt.Errorf("Unknown chain id: %d, L2ChainInfoFiles: %v.  update chain id, modify --chain.info-files or provide --persistent.chain\n", l2ChainId, l2ChainInfoFiles)
			}
			return nil, nil, nil, fmt.Errorf("Unknown chain name: %s, L2ChainInfoFiles: %v.  update chain name, modify --chain.info-files or provide --persistent.chain\n", l2ChainName, l2ChainInfoFiles)
		}
		return nil, nil, nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, nil, nil, err
	}

	// Don't pass around wallet contents with normal configuration
	l1Wallet := nodeConfig.L1.Wallet
	l2DevWallet := nodeConfig.L2.DevWallet
	nodeConfig.L1.Wallet = genericconf.WalletConfigDefault
	nodeConfig.L2.DevWallet = genericconf.WalletConfigDefault

	err = nodeConfig.Validate()
	if err != nil {
		return nil, nil, nil, err
	}
	nodeConfig.Rpc.Apply()
	return &nodeConfig, &l1Wallet, &l2DevWallet, nil
}

func applyChainParameters(ctx context.Context, k *koanf.Koanf, chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoJson string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) (bool, error) {
	combinedL2ChainInfoFiles := l2ChainInfoFiles
	if l2ChainInfoIpfsUrl != "" {
		l2ChainInfoIpfsFile, err := util.GetL2ChainInfoIpfsFile(ctx, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
		if err != nil {
			log.Error("error getting l2 chain info file from ipfs", "err", err)
		}
		combinedL2ChainInfoFiles = append(combinedL2ChainInfoFiles, l2ChainInfoIpfsFile)
	}
	chainInfo, err := chaininfo.ProcessChainInfo(chainId, chainName, combinedL2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return false, err
	}
	chainDefaults := map[string]interface{}{
		"persistent.chain": chainInfo.ChainName,
		"chain.id":         chainInfo.ChainId,
		"parent-chain.id":  chainInfo.ParentChainId,
	}
	if chainInfo.SequencerUrl != "" {
		chainDefaults["node.forwarding-target"] = chainInfo.SequencerUrl
	}
	if chainInfo.FeedUrl != "" {
		chainDefaults["node.feed.input.url"] = chainInfo.FeedUrl
	}
	if chainInfo.DasIndexUrl != "" {
		chainDefaults["node.data-availability.enable"] = true
		chainDefaults["node.data-availability.rest-aggregator.enable"] = true
		chainDefaults["node.data-availability.rest-aggregator.online-url-list"] = chainInfo.DasIndexUrl
	}
	if !chainInfo.HasGenesisState {
		chainDefaults["init.empty"] = true
	}
	err = k.Load(confmap.Provider(chainDefaults, "."), nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

type NodeConfigFetcher struct {
	*genericconf.LiveConfig[*NodeConfig]
}

func (f *NodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveConfig.Get().Node
}
