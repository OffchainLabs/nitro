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
	"strings"
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
	"github.com/offchainlabs/nitro/util/headerreader"
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
		return fmt.Errorf("Error parsing log type: %w", err)
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

	vcsRevision, vcsTime := genericconf.GetVersion()
	nodeConfig, l1Wallet, l2DevWallet, l1Client, l1ChainId, err := ParseNode(ctx, os.Args[1:])
	if err != nil {
		fmt.Printf("\nrevision: %v, vcs.time: %v\n", vcsRevision, vcsTime)
		printSampleUsage(os.Args[0])
		if !strings.Contains(err.Error(), "help requested") {
			fmt.Printf("%s\n", err.Error())
		}

		return
	}
	err = initLog(nodeConfig.LogType, log.Lvl(nodeConfig.LogLevel))
	if err != nil {
		panic(err)
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
			panic("forwarding-target set when sequencer enabled")
		}
		if nodeConfig.Node.L1Reader.Enable && nodeConfig.Node.InboxReader.HardReorg {
			panic("hard reorgs cannot safely be enabled with sequencer mode enabled")
		}
	} else if nodeConfig.Node.ForwardingTargetImpl == "" {
		flag.Usage()
		panic("forwarding-target unset, and not sequencer (can set to \"null\" to disable forwarding)")
	}

	if nodeConfig.Node.SeqCoordinator.Enable {
		if nodeConfig.Node.SeqCoordinator.SigningKey == "" && !nodeConfig.Node.SeqCoordinator.Dangerous.DisableSignatureVerification {
			panic("sequencer coordinator enabled, but signing key unset, and signature verification isn't disabled")
		}
	}

	var rollupAddrs arbnode.RollupAddresses
	var l1TransactionOpts *bind.TransactOpts
	var daSigner func([]byte) ([]byte, error)
	setupNeedsKey := l1Wallet.OnlyCreateKey || nodeConfig.Node.Validator.OnlyCreateWalletContract
	if nodeConfig.Node.L1Reader.Enable || setupNeedsKey {
		log.Info("connected to l1 chain", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)

		rollupAddrs, err = nodeConfig.L1.Rollup.ParseAddresses()
		if err != nil {
			panic(err)
		}

		validatorNeedsKey := nodeConfig.Node.Validator.Enable && !strings.EqualFold(nodeConfig.Node.Validator.Strategy, "watchtower")
		if nodeConfig.Node.BatchPoster.Enable || validatorNeedsKey || setupNeedsKey {
			l1TransactionOpts, daSigner, err = util.OpenWallet("l1", l1Wallet, new(big.Int).SetUint64(nodeConfig.L1.ChainID))
			if err != nil {
				fmt.Printf("%v\n", err.Error())
				return
			}
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

	if nodeConfig.Node.Validator.OnlyCreateWalletContract {
		l1Reader := headerreader.New(l1Client, nodeConfig.Node.L1Reader)

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

	if nodeConfig.Node.Archive && nodeConfig.Node.TxLookupLimit != 0 {
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

	chainDb, l2BlockChain, err := openInitializeChainDb(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.L2.ChainID), arbnode.DefaultCacheConfigFor(stack, nodeConfig.Node.Archive))
	if err != nil {
		printSampleUsage(os.Args[0])
		fmt.Printf("%s\n", err.Error())
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

	feedErrChan := make(chan error, 10)
	currentNode, err := arbnode.CreateNode(
		ctx,
		stack,
		chainDb,
		arbDb,
		&nodeConfig.Node,
		l2BlockChain,
		l1Client,
		&rollupAddrs,
		l1TransactionOpts,
		daSigner,
		feedErrChan,
	)
	if err != nil {
		panic(err)
	}
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
	case err := <-feedErrChan:
		log.Error("shutting down because broadcaster stopped", "err", err)
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
	Conf          genericconf.ConfConfig          `koanf:"conf"`
	Node          arbnode.Config                  `koanf:"node"`
	L1            conf.L1Config                   `koanf:"l1"`
	L2            conf.L2Config                   `koanf:"l2"`
	LogLevel      int                             `koanf:"log-level"`
	LogType       string                          `koanf:"log-type"`
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
			return nil, nil, nil, nil, nil, errors.New("mainnet not supported yet")
		case 42170:
			if err := applyArbitrumNovaRollupParameters(k); err != nil {
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

	return &nodeConfig, &l1Wallet, &l2DevWallet, l1Client, l1ChainId, nil
}

func applyArbitrumNovaRollupParameters(k *koanf.Koanf) error {
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
		"l1.rollup.inbox":                    "0xd394acec33ca1c7fc14212b41892bd82deddda94",
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
