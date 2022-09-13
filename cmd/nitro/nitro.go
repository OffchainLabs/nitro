// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

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
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	_ "github.com/offchainlabs/nitro/nodeInterface"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

func printSampleUsage(name string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", name)
}

func initLog(logType string, logLevel log.Lvl) error {
	logFormat, err := genericconf.ParseLogType(logType)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("error parsing log type: %w", err)
	}
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	glogger.Verbosity(logLevel)
	log.Root().SetHandler(glogger)
	return nil
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

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	nodeConfig, l1Wallet, l2DevWallet, l1Client, l1ChainId, err := ParseNode(ctx, args)
	if err != nil {
		util.HandleError(err, printSampleUsage)

		return
	}
	err = initLog(nodeConfig.LogType, log.Lvl(nodeConfig.LogLevel))
	if err != nil {
		panic(err)
	}

	vcsRevision, vcsTime := util.GetVersion()
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
			panic("forwarding-target set when sequencer enabled")
		}
		if nodeConfig.Node.L1Reader.Enable && nodeConfig.Node.InboxReader.HardReorg {
			panic("hard reorgs cannot safely be enabled with sequencer mode enabled")
		}
	} else if nodeConfig.Node.ForwardingTargetImpl == "" {
		flag.Usage()
		panic("forwarding-target unset, and not sequencer (can set to \"null\" to disable forwarding)")
	}

	var l1TransactionOpts *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	sequencerNeedsKey := nodeConfig.Node.Sequencer.Enable && !nodeConfig.Node.Feed.Output.DisableSigning
	setupNeedsKey := l1Wallet.OnlyCreateKey || nodeConfig.Node.Validator.OnlyCreateWalletContract
	validatorNeedsKey := nodeConfig.Node.Validator.Enable && !strings.EqualFold(nodeConfig.Node.Validator.Strategy, "watchtower")
	if sequencerNeedsKey || nodeConfig.Node.BatchPoster.Enable || setupNeedsKey || validatorNeedsKey {
		l1TransactionOpts, dataSigner, err = util.OpenWallet("l1", l1Wallet, new(big.Int).SetUint64(nodeConfig.L1.ChainID))
		if err != nil {
			fmt.Printf("%v\n", err.Error())
			return
		}
	}

	var rollupAddrs arbnode.RollupAddresses
	if nodeConfig.Node.L1Reader.Enable {
		log.Info("connected to l1 chain", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)

		rollupAddrs, err = nodeConfig.L1.Rollup.ParseAddresses()
		if err != nil {
			fmt.Printf("error getting rollup addresses: %v\n", err.Error())
			return
		}
	} else if l1Client != nil {
		// Don't need l1Client anymore
		log.Info("used chain id to get rollup parameters", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)
		l1Client = nil
	}

	if nodeConfig.Node.Validator.Enable {
		if !nodeConfig.Node.L1Reader.Enable {
			flag.Usage()
			panic("validator must read from L1")
		}
		if !nodeConfig.Node.Validator.Dangerous.WithoutBlockValidator {
			nodeConfig.Node.BlockValidator.Enable = true
		}
	}

	liveNodeConfig := NewLiveNodeConfig(args, nodeConfig)
	if nodeConfig.Node.Validator.OnlyCreateWalletContract {
		l1Reader := headerreader.New(l1Client, func() *headerreader.Config { return &liveNodeConfig.get().Node.L1Reader })

		// Just create validator smart wallet if needed then exit
		deployInfo, err := nodeConfig.L1.Rollup.ParseAddresses()
		if err != nil {
			log.Error("error getting deployment info for creating validator wallet contract", "error", err)
			return
		}
		addr, err := validator.GetValidatorWallet(ctx, deployInfo.ValidatorWalletCreator, int64(deployInfo.DeployedAt), l1TransactionOpts, l1Reader, true)
		if err != nil {
			log.Error("error creating validator wallet contract", "error", err, "address", l1TransactionOpts.From.Hex())
			return
		}
		fmt.Printf("created validator smart contract wallet at %s, remove --node.validator.only-create-wallet-contract and restart\n", addr.String())

		return
	}

	if nodeConfig.Node.Archive {
		log.Warn("node.archive has been deprecated. Please use node.caching.archive instead.")
		nodeConfig.Node.Caching.Archive = true
	}
	if nodeConfig.Node.Caching.Archive && nodeConfig.Node.TxLookupLimit != 0 {
		log.Info("retaining ability to lookup full transaction history as archive mode is enabled")
		nodeConfig.Node.TxLookupLimit = 0
	}

	stackConf := node.DefaultConfig
	stackConf.DataDir = nodeConfig.Persistent.Chain
	nodeConfig.HTTP.Apply(&stackConf)
	nodeConfig.WS.Apply(&stackConf)
	nodeConfig.GraphQL.Apply(&stackConf)
	if nodeConfig.WS.ExposeAll {
		stackConf.WSModules = append(stackConf.WSModules, "personal")
	}
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	stackConf.Version = vcsRevision
	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		panic(err)
	}
	{
		devAddr, err := addUnlockWallet(stack.AccountManager(), l2DevWallet)
		if err != nil {
			flag.Usage()
			panic(err)
		}
		if devAddr != (common.Address{}) {
			nodeConfig.Init.DevInitAddr = devAddr.String()
		}
	}

	chainDb, l2BlockChain, err := openInitializeChainDb(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.L2.ChainID), arbnode.DefaultCacheConfigFor(stack, &nodeConfig.Node.Caching))
	if err != nil {
		util.HandleError(err, printSampleUsage)
		return
	}

	arbDb, err := stack.OpenDatabase("arbitrumdata", 0, 0, "", false)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %v", err))
	}

	if nodeConfig.Init.ThenQuit {
		return
	}

	if l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee && !nodeConfig.Node.DataAvailability.Enable {
		flag.Usage()
		panic("a data availability service must be configured for this chain (see the --node.data-availability family of options)")
	}

	if nodeConfig.Metrics {
		go metrics.CollectProcessMetrics(nodeConfig.MetricsServer.UpdateInterval)

		if nodeConfig.MetricsServer.Addr != "" {
			address := fmt.Sprintf("%v:%v", nodeConfig.MetricsServer.Addr, nodeConfig.MetricsServer.Port)
			exp.Setup(address)
		}
	}

	fatalErrChan := make(chan error, 10)
	currentNode, err := arbnode.CreateNode(
		ctx,
		stack,
		chainDb,
		arbDb,
		&NodeConfigFetcher{liveNodeConfig},
		l2BlockChain,
		l1Client,
		&rollupAddrs,
		l1TransactionOpts,
		dataSigner,
		fatalErrChan,
	)
	if err != nil {
		panic(err)
	}
	liveNodeConfig.setOnReloadHook(func(old *NodeConfig, new *NodeConfig) error {
		return currentNode.OnConfigReload(&old.Node, &new.Node)
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
		if err := graphql.New(stack, currentNode.Backend.APIBackend(), gqlConf.CORSDomain, gqlConf.VHosts); err != nil {
			panic(fmt.Sprintf("Failed to register the GraphQL service: %v", err))
		}
	}

	if err := stack.Start(); err != nil {
		panic(fmt.Sprintf("Error starting protocol stack: %v\n", err))
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-fatalErrChan:
		log.Error("shutting down due to fatal error", "err", err)
	case <-sigint:
		log.Info("shutting down because of sigint")
	}

	// cause future ctrl+c's to panic
	close(sigint)

	if err := stack.Close(); err != nil {
		panic(fmt.Sprintf("Error closing stack: %v\n", err))
	}
}

type NodeConfig struct {
	Conf          genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Node          arbnode.Config                  `koanf:"node" reload:"hot"`
	L1            conf.L1Config                   `koanf:"l1"`
	L2            conf.L2Config                   `koanf:"l2"`
	LogLevel      int                             `koanf:"log-level" reload:"hot"`
	LogType       string                          `koanf:"log-type" reload:"hot"`
	Persistent    conf.PersistentConfig           `koanf:"persistent"`
	HTTP          genericconf.HTTPConfig          `koanf:"http"`
	WS            genericconf.WSConfig            `koanf:"ws"`
	GraphQL       genericconf.GraphQLConfig       `koanf:"graphql"`
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	Init          InitConfig                      `koanf:"init"`
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
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	conf.L1ConfigAddOptions("l1", f)
	conf.L2ConfigAddOptions("l2", f)
	f.Int("log-level", NodeConfigDefault.LogLevel, "log level")
	f.String("log-type", NodeConfigDefault.LogType, "log type (plaintext or json)")
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.GraphQLConfigAddOptions("graphql", f)
	f.Bool("metrics", NodeConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	InitConfigAddOptions("init", f)
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
			hot := node.Type().Field(i).Tag.Get("reload") == "hot"
			dot := path + "." + node.Type().Field(i).Name

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
	return c.Node.Validate()
}

func ParseNode(ctx context.Context, args []string) (*NodeConfig, *genericconf.WalletConfig, *genericconf.WalletConfig, *ethclient.Client, *big.Int, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var l1ChainId *big.Int
	var l1Client *ethclient.Client
	l1URL := k.String("l1.url")
	configChainId := uint64(k.Int64("l1.chain-id"))
	if l1URL != "" {
		maxConnectionAttempts := k.Int("l1.connection-attempts")
		if maxConnectionAttempts <= 0 {
			maxConnectionAttempts = math.MaxInt
		}
		for i := 1; i <= maxConnectionAttempts; i++ {
			l1Client, err = ethclient.DialContext(ctx, l1URL)
			if err == nil {
				l1ChainId, err = l1Client.ChainID(ctx)
				if err == nil {
					// Successfully got chain ID
					break
				}
			}
			if i < maxConnectionAttempts {
				log.Warn("error connecting to L1", "err", err)
			} else {
				panic(err)
			}

			timer := time.NewTimer(time.Second * 1)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, nil, nil, nil, nil, errors.New("aborting startup")
			case <-timer.C:
			}
		}
	} else if configChainId == 0 && !k.Bool("conf.dump") {
		return nil, nil, nil, nil, nil, errors.New("l1 chain id not provided")
	} else if k.Bool("node.l1-reader.enable") {
		return nil, nil, nil, nil, nil, errors.New("l1 reader enabled but --l1.url not provided")
	}

	if l1ChainId == nil {
		l1ChainId = big.NewInt(int64(configChainId))
	}

	if configChainId != l1ChainId.Uint64() {
		if configChainId != 0 {
			log.Error("chain id from L1 does not match command line chain id", "l1", l1ChainId.String(), "cli", configChainId)
			return nil, nil, nil, nil, nil, errors.New("chain id from L1 does not match command line chain id")
		}

		err := k.Load(confmap.Provider(map[string]interface{}{
			"l1.chain-id": l1ChainId.Uint64(),
		}, "."), nil)
		if err != nil {
			return nil, nil, nil, nil, nil, errors.Wrap(err, "error setting ")
		}
	}

	chainFound := false
	l2ChainId := k.Int64("l2.chain-id")
	if l1ChainId.Uint64() == 1 { // mainnet
		switch l2ChainId {
		case 0:
			return nil, nil, nil, nil, nil, errors.New("must specify --l2.chain-id to choose rollup")
		case 42161:
			if err := applyArbitrumOneParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
			chainFound = true
		case 42170:
			if err := applyArbitrumNovaParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
			chainFound = true
		}
	} else if l1ChainId.Uint64() == 4 {
		switch l2ChainId {
		case 0:
			return nil, nil, nil, nil, nil, errors.New("must specify --l2.chain-id to choose rollup")
		case 421611:
			if err := applyArbitrumRollupRinkebyTestnetParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
			chainFound = true
		}
	} else if l1ChainId.Uint64() == 5 {
		switch l2ChainId {
		case 0:
			return nil, nil, nil, nil, nil, errors.New("must specify --l2.chain-id to choose rollup")
		case 421613:
			if err := applyArbitrumRollupGoerliTestnetParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
			chainFound = true
		case 421703:
			if err := applyArbitrumAnytrustGoerliTestnetParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
			chainFound = true
		}
	}

	err = util.ApplyOverrides(f, k)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := util.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Don't print wallet passwords
	if nodeConfig.Conf.Dump {
		err = util.DumpConfig(k, map[string]interface{}{
			"l1.wallet.password":        "",
			"l1.wallet.private-key":     "",
			"l2.dev-wallet.password":    "",
			"l2.dev-wallet.private-key": "",
		})
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	if nodeConfig.Persistent.Chain == "" {
		if !chainFound {
			// If persistent-chain not defined, user not creating custom chain
			return nil, nil, nil, nil, nil, fmt.Errorf("Unknown chain with L1: %d, L2: %d.  Change L1, update L2 chain id, or provide --persistent.chain\n", l1ChainId.Uint64(), l2ChainId)
		}
		return nil, nil, nil, nil, nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Don't pass around wallet contents with normal configuration
	l1Wallet := nodeConfig.L1.Wallet
	l2DevWallet := nodeConfig.L2.DevWallet
	nodeConfig.L1.Wallet = genericconf.WalletConfigDefault
	nodeConfig.L2.DevWallet = genericconf.WalletConfigDefault

	err = nodeConfig.Validate()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return &nodeConfig, &l1Wallet, &l2DevWallet, l1Client, l1ChainId, nil
}

func applyArbitrumOneParameters(k *koanf.Koanf) error {
	return k.Load(confmap.Provider(map[string]interface{}{
		"persistent.chain":                   "arb1",
		"node.forwarding-target":             "https://arb1.arbitrum.io/rpc",
		"node.feed.input.url":                "wss://arb1.arbitrum.io/feed",
		"l1.rollup.bridge":                   "0x8315177ab297ba92a06054ce80a67ed4dbd7ed3a",
		"l1.rollup.inbox":                    "0x4dbd4fc535ac27206064b68ffcf827b0a60bab3f",
		"l1.rollup.rollup":                   "0x5ef0d09d1e6204141b4d37530808ed19f60fba35",
		"l1.rollup.sequencer-inbox":          "0x1c479675ad559dc151f6ec7ed3fbf8cee79582b6",
		"l1.rollup.validator-utils":          "0x9e40625f52829cf04bc4839f186d621ee33b0e67",
		"l1.rollup.validator-wallet-creator": "0x960953f7c69cd2bc2322db9223a815c680ccc7ea",
		"l1.rollup.deployed-at":              15411056,
		"l2.chain-id":                        42161,
	}, "."), nil)
}

func applyArbitrumNovaParameters(k *koanf.Koanf) error {
	return k.Load(confmap.Provider(map[string]interface{}{
		"persistent.chain":                                       "nova",
		"node.forwarding-target":                                 "https://nova.arbitrum.io/rpc",
		"node.feed.input.url":                                    "wss://nova.arbitrum.io/feed",
		"node.data-availability.enable":                          true,
		"node.data-availability.rest-aggregator.enable":          true,
		"node.data-availability.rest-aggregator.online-url-list": "https://nova.arbitrum.io/das-servers",
		"l1.rollup.bridge":                                       "0xc1ebd02f738644983b6c4b2d440b8e77dde276bd",
		"l1.rollup.inbox":                                        "0xc4448b71118c9071bcb9734a0eac55d18a153949",
		"l1.rollup.rollup":                                       "0xfb209827c58283535b744575e11953dcc4bead88",
		"l1.rollup.sequencer-inbox":                              "0x211e1c4c7f1bf5351ac850ed10fd68cffcf6c21b",
		"l1.rollup.validator-utils":                              "0x2B081fbaB646D9013f2699BebEf62B7e7d7F0976",
		"l1.rollup.validator-wallet-creator":                     "0xe05465Aab36ba1277dAE36aa27a7B74830e74DE4",
		"l1.rollup.deployed-at":                                  15016829,
		"l2.chain-id":                                            42170,
		"init.empty":                                             true,
	}, "."), nil)
}

func applyArbitrumRollupGoerliTestnetParameters(k *koanf.Koanf) error {
	return k.Load(confmap.Provider(map[string]interface{}{
		"persistent.chain":                   "goerli-rollup",
		"node.forwarding-target":             "https://goerli-rollup.arbitrum.io/rpc",
		"node.feed.input.url":                "wss://goerli-rollup.arbitrum.io/feed",
		"l1.rollup.bridge":                   "0xaf4159a80b6cc41ed517db1c453d1ef5c2e4db72",
		"l1.rollup.inbox":                    "0x6bebc4925716945d46f0ec336d5c2564f419682c",
		"l1.rollup.rollup":                   "0x45e5caea8768f42b385a366d3551ad1e0cbfab17",
		"l1.rollup.sequencer-inbox":          "0x0484a87b144745a2e5b7c359552119b6ea2917a9",
		"l1.rollup.validator-utils":          "0x344f651fe566a02db939c8657427deb5524ea78e",
		"l1.rollup.validator-wallet-creator": "0x53eb4f4524b3b9646d41743054230d3f425397b3",
		"l1.rollup.deployed-at":              7217526,
		"l2.chain-id":                        421613,
		"init.empty":                         true,
	}, "."), nil)
}

func applyArbitrumRollupRinkebyTestnetParameters(k *koanf.Koanf) error {
	return k.Load(confmap.Provider(map[string]interface{}{
		"persistent.chain":                   "rinkeby-nitro",
		"node.forwarding-target":             "https://rinkeby.arbitrum.io/rpc",
		"node.feed.input.url":                "wss://rinkeby.arbitrum.io/feed",
		"l1.rollup.bridge":                   "0x85c720444e436e1f9407e0c3895d3fe149f41168",
		"l1.rollup.inbox":                    "0x578BAde599406A8fE3d24Fd7f7211c0911F5B29e",
		"l1.rollup.rollup":                   "0x71c6093c564eddcfaf03481c3f59f88849f1e644",
		"l1.rollup.sequencer-inbox":          "0x957c9c64f7c2ce091e56af3f33ab20259096355f",
		"l1.rollup.validator-utils":          "0x0ea7372338a589e7f0b00e463a53aa464ef04e17",
		"l1.rollup.validator-wallet-creator": "0x237b8965cebe27108bc1d6b71575c3b070050f7a",
		"l1.rollup.deployed-at":              11088567,
		"l2.chain-id":                        421611,
	}, "."), nil)
}

func applyArbitrumAnytrustGoerliTestnetParameters(k *koanf.Koanf) error {
	return k.Load(confmap.Provider(map[string]interface{}{
		"persistent.chain": "goerli-anytrust",
	}, "."), nil)
}

type OnReloadHook func(old *NodeConfig, new *NodeConfig) error

func noopOnReloadHook(old *NodeConfig, new *NodeConfig) error {
	return nil
}

type LiveNodeConfig struct {
	stopwaiter.StopWaiter

	mutex        sync.RWMutex
	args         []string
	config       *NodeConfig
	onReloadHook OnReloadHook
}

func (c *LiveNodeConfig) get() *NodeConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.config
}

func (c *LiveNodeConfig) set(config *NodeConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.config.CanReload(config); err != nil {
		return err
	}
	if err := initLog(config.LogType, log.Lvl(config.LogLevel)); err != nil {
		return err
	}
	if err := c.onReloadHook(c.config, config); err != nil {
		// TODO(magic) panic? return err? only log the error?
		log.Error("Failed to execute onReloadHook", "err", err)
	}
	c.config = config
	return nil
}

func (c *LiveNodeConfig) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn)

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	c.LaunchThread(func(ctx context.Context) {
		for {
			reloadInterval := c.config.Conf.ReloadInterval
			if reloadInterval == 0 {
				select {
				case <-ctx.Done():
					return
				case <-sigusr1:
					log.Info("Configuration reload triggered by SIGUSR1.")
				}
			} else {
				timer := time.NewTimer(reloadInterval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-sigusr1:
					timer.Stop()
					log.Info("Configuration reload triggered by SIGUSR1.")
				case <-timer.C:
				}
			}
			nodeConfig, _, _, _, _, err := ParseNode(ctx, c.args)
			if err != nil {
				log.Error("error parsing live config", "error", err.Error())
				continue
			}
			err = c.set(nodeConfig)
			if err != nil {
				log.Error("error updating live config", "error", err.Error())
				continue
			}
		}
	})
}

// setOnReloadHook is NOT thread-safe and supports setting only one hook
func (c *LiveNodeConfig) setOnReloadHook(hook OnReloadHook) {
	c.onReloadHook = hook
}

func NewLiveNodeConfig(args []string, config *NodeConfig) *LiveNodeConfig {
	return &LiveNodeConfig{
		args:         args,
		config:       config,
		onReloadHook: noopOnReloadHook,
	}
}

type NodeConfigFetcher struct {
	*LiveNodeConfig
}

func (f *NodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveNodeConfig.get().Node
}

func (f *NodeConfigFetcher) Start(ctx context.Context) {
	f.LiveNodeConfig.Start(ctx)
}

func (f *NodeConfigFetcher) StopAndWait() {
	f.LiveNodeConfig.StopAndWait()
}
