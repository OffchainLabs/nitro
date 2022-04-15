// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/knadh/koanf"
	"math"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	koanfjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/cmd/conf"
	cmdutil "github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/statetransfer"
	nitroutil "github.com/offchainlabs/nitro/util"

	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

func printSampleUsage(name string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", name)
}

func main() {
	ctx := context.Background()

	nodeConfig, l1Wallet, l2DevWallet, l1Client, l1ChainId, err := ParseNode(ctx, os.Args[1:])
	if err != nil {
		printSampleUsage(os.Args[0])
		if !strings.Contains(err.Error(), "help requested") {
			fmt.Printf("%s\n", err.Error())
		}

		return
	}
	logFormat, err := conf.ParseLogType(nodeConfig.LogType)
	if err != nil {
		flag.Usage()
		panic(fmt.Sprintf("Error parsing log type: %v", err))
	}
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	glogger.Verbosity(log.Lvl(nodeConfig.LogLevel))
	log.Root().SetHandler(glogger)

	log.Info("Running Arbitrum nitro node")

	if nodeConfig.Node.Dangerous.NoL1Listener {
		nodeConfig.Node.L1Reader.Enable = false
		nodeConfig.Node.Sequencer.Enable = true // we sequence messages, but not to l1
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

	// Perform sanity check on mode
	_, err = nodeConfig.Node.DataAvailability.Mode()
	if err != nil {
		panic(err.Error())
	}

	var rollupAddrs arbnode.RollupAddresses
	var l1TransactionOpts *bind.TransactOpts
	if nodeConfig.Node.L1Reader.Enable {
		log.Info("connected to l1 chain", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)

		rollupAddrs, err = nodeConfig.L1.Rollup.ParseAddresses()
		if err != nil {
			panic(err)
		}

		if nodeConfig.Node.BatchPoster.Enable || nodeConfig.Node.Validator.Enable {
			l1TransactionOpts, err = nitroutil.GetTransactOptsFromKeystore(
				l1Wallet.Pathname,
				l1Wallet.Account,
				*l1Wallet.Password(),
				new(big.Int).SetUint64(nodeConfig.L1.ChainID),
			)
			if err != nil {
				panic(err)
			}
		}
	} else {
		// Don't need l1Client anymore
		log.Info("using chain id to get rollup parameters", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)
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

	stackConf := node.DefaultConfig
	stackConf.DataDir = nodeConfig.Persistent.ChainData
	stackConf.HTTPHost = nodeConfig.HTTP.Addr
	stackConf.HTTPPort = nodeConfig.HTTP.Port
	stackConf.HTTPVirtualHosts = nodeConfig.HTTP.VHosts
	stackConf.HTTPModules = nodeConfig.HTTP.API
	stackConf.HTTPCors = nodeConfig.HTTP.CORSDomain
	stackConf.WSHost = nodeConfig.WS.Addr
	stackConf.WSPort = nodeConfig.WS.Port
	stackConf.WSOrigins = nodeConfig.WS.Origins
	stackConf.WSModules = nodeConfig.WS.API
	stackConf.WSExposeAll = nodeConfig.WS.ExposeAll
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

	// temporary until test-node is updated to pass dev private key in on startup
	var devAddr common.Address
	if nodeConfig.L1.ChainID == 1337 {
		devPrivKeyStr := "e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2"
		devPrivKey, err := crypto.HexToECDSA(devPrivKeyStr)
		if err != nil {
			panic(err)
		}

		devAddr = crypto.PubkeyToAddress(devPrivKey.PublicKey)
		log.Info("Dev node funded private key", "priv", devPrivKeyStr)
		log.Info("Funded public address", "addr", devAddr)

		if l2DevWallet.Pathname != "" {
			mykeystore := keystore.NewPlaintextKeyStore(l2DevWallet.Pathname)
			stack.AccountManager().AddBackend(mykeystore)
			var account accounts.Account
			if mykeystore.HasAddress(devAddr) {
				account.Address = devAddr
				account, err = mykeystore.Find(account)
			} else {
				if l2DevWallet.Password() == nil {
					panic("l2 password not set")
				}
				account, err = mykeystore.ImportECDSA(devPrivKey, *l2DevWallet.Password())
			}
			if err != nil {
				panic(err)
			}
			if l2DevWallet.Password() == nil {
				panic("l2 password not set")
			}
			err = mykeystore.Unlock(account, *l2DevWallet.Password())
			if err != nil {
				panic(err)
			}
		}
	}
	var initDataReader statetransfer.InitDataReader = nil

	chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, "", "", false)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %v", err))
	}

	if nodeConfig.ImportFile != "" {
		initDataReader, err = statetransfer.NewJsonInitDataReader(nodeConfig.ImportFile)
		if err != nil {
			panic(err)
		}
	} else {
		var initData statetransfer.ArbosInitializationInfo
		if nodeConfig.DevInit {
			initData = statetransfer.ArbosInitializationInfo{
				Accounts: []statetransfer.AccountInitializationInfo{
					{
						Addr:       devAddr,
						EthBalance: new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)),
						Nonce:      0,
					},
				},
			}
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
	}

	chainConfig, err := arbos.GetChainConfig(new(big.Int).SetUint64(nodeConfig.L2.ChainID))
	if err != nil {
		panic(err)
	}

	var l2BlockChain *core.BlockChain
	if nodeConfig.NoInit {
		blocksInDb, err := chainDb.Ancients()
		if err != nil {
			panic(err)
		}
		if blocksInDb == 0 {
			panic("No initialization mode supplied, no blocks in Db")
		}
		l2BlockChain, err = arbnode.GetBlockChain(chainDb, arbnode.DefaultCacheConfigFor(stack, nodeConfig.Node.Archive), chainConfig)
		if err != nil {
			panic(err)
		}
	} else {
		blockReader, err := initDataReader.GetStoredBlockReader()
		if err != nil {
			panic(err)
		}
		blockNum, err := arbnode.ImportBlocksToChainDb(chainDb, blockReader)
		if err != nil {
			panic(err)
		}
		l2BlockChain, err = arbnode.WriteOrTestBlockChain(chainDb, arbnode.DefaultCacheConfigFor(stack, nodeConfig.Node.Archive), initDataReader, blockNum, chainConfig)
		if err != nil {
			panic(err)
		}
	}

	// Check that this ArbOS state has the correct chain ID
	{
		statedb, err := l2BlockChain.State()
		if err != nil {
			panic(err)
		}
		currentArbosState, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			panic(err)
		}
		chainId, err := currentArbosState.ChainId()
		if err != nil {
			panic(err)
		}
		if chainId.Cmp(chainConfig.ChainID) != 0 {
			panic(fmt.Sprintf("attempted to launch node with chain ID %v on ArbOS state with chain ID %v", chainConfig.ChainID, chainId))
		}
	}

	if nodeConfig.Metrics {
		go metrics.CollectProcessMetrics(3 * time.Second)

		if nodeConfig.MetricsServer.Addr != "" {
			address := fmt.Sprintf("%v:%v", nodeConfig.MetricsServer.Addr, nodeConfig.MetricsServer.Port)
			exp.Setup(address)
		}
	}

	currentNode, err := arbnode.CreateNode(stack, chainDb, &nodeConfig.Node, l2BlockChain, l1Client, &rollupAddrs, l1TransactionOpts)
	if err != nil {
		panic(err)
	}
	if nodeConfig.Node.Dangerous.NoL1Listener && nodeConfig.DevInit {
		// If we don't have any messages, we're not connected to the L1, and we're using a dev init,
		// we should create our own fake init message.
		count, err := currentNode.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}
		if count == 0 {
			err = currentNode.TxStreamer.AddFakeInitMessage()
			if err != nil {
				panic(err)
			}
		}
	}
	if err := stack.Start(); err != nil {
		panic(fmt.Sprintf("Error starting protocol stack: %v\n", err))
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	<-sigint
	// cause future ctrl+c's to panic
	close(sigint)

	if err := stack.Close(); err != nil {
		panic(fmt.Sprintf("Error closing stack: %v\n", err))
	}
}

type NodeConfig struct {
	Conf          conf.ConfConfig          `koanf:"conf"`
	Node          arbnode.Config           `koanf:"node"`
	L1            conf.L1Config            `koanf:"l1"`
	L2            conf.L2Config            `koanf:"l2"`
	LogLevel      int                      `koanf:"log-level"`
	LogType       string                   `koanf:"log-type"`
	Persistent    conf.PersistentConfig    `koanf:"persistent"`
	HTTP          conf.HTTPConfig          `koanf:"http"`
	WS            conf.WSConfig            `koanf:"ws"`
	DevInit       bool                     `koanf:"dev-init"`
	NoInit        bool                     `koanf:"no-init"`
	ImportFile    string                   `koanf:"import-file"`
	Metrics       bool                     `koanf:"metrics"`
	MetricsServer conf.MetricsServerConfig `koanf:"metrics-server"`
}

var NodeConfigDefault = NodeConfig{
	Conf:          conf.ConfConfigDefault,
	Node:          arbnode.ConfigDefault,
	L1:            conf.L1ConfigDefault,
	L2:            conf.L2ConfigDefault,
	LogLevel:      int(log.LvlInfo),
	LogType:       "plaintext",
	Persistent:    conf.PersistentConfigDefault,
	HTTP:          conf.HTTPConfigDefault,
	WS:            conf.WSConfigDefault,
	DevInit:       false,
	ImportFile:    "",
	Metrics:       false,
	MetricsServer: conf.MetricsServerConfigDefault,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	conf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	conf.L1ConfigAddOptions("l1", f)
	conf.L2ConfigAddOptions("l2", f)
	f.Int("log-level", NodeConfigDefault.LogLevel, "log level")
	f.String("log-type", NodeConfigDefault.LogType, "log type (plaintext or json)")
	conf.PersistentConfigAddOptions("persistent", f)
	conf.HTTPConfigAddOptions("http", f)
	conf.WSConfigAddOptions("ws", f)
	f.Bool("dev-init", NodeConfigDefault.DevInit, "init with dev data (1 account with balance) instead of file import")
	f.Bool("no-init", NodeConfigDefault.DevInit, "Do not init chain. Data must be valid in database.")
	f.String("import-file", NodeConfigDefault.ImportFile, "path for json data to import")
	f.Bool("metrics", NodeConfigDefault.Metrics, "enable metrics")
	conf.MetricsServerAddOptions("metrics-server", f)
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

func ParseNode(ctx context.Context, args []string) (*NodeConfig, *conf.WalletConfig, *conf.WalletConfig, *ethclient.Client, *big.Int, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := cmdutil.BeginCommonParse(f, args)
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

	switch l1ChainId.Uint64() {
	case 5: // goerli
		switch k.String("l2.rollup.rollup") {
		case "":
			if err := applyNitroDevNetRollupParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
		case "0x767cff8d8de386d7cbe91dbd39675132ba2f5967":
			if err := applyNitroDevNetRollupParameters(k); err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}

	err = cmdutil.ApplyOverrides(f, k)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := cmdutil.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if nodeConfig.Conf.Dump {
		// Print out current configuration

		// Don't keep printing configuration file and don't print wallet passwords
		err := k.Load(confmap.Provider(map[string]interface{}{
			"conf.dump":             false,
			"l1.wallet.password":    "",
			"l1.wallet.private-key": "",
			"l2.wallet.password":    "",
			"l2.wallet.private-key": "",
		}, "."), nil)
		if err != nil {
			return nil, nil, nil, nil, nil, errors.Wrap(err, "error removing extra parameters before dump")
		}

		c, err := k.Marshal(koanfjson.Parser())
		if err != nil {
			return nil, nil, nil, nil, nil, errors.Wrap(err, "unable to marshal config file to JSON")
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	if nodeConfig.Persistent.Chain == "" {
		return nil, nil, nil, nil, nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Don't pass around wallet contents with normal configuration
	l1Wallet := nodeConfig.L1.Wallet
	l2DevWallet := nodeConfig.L2.DevWallet
	nodeConfig.L1.Wallet = conf.WalletConfigDefault
	nodeConfig.L2.DevWallet = conf.WalletConfigDefault

	return &nodeConfig, &l1Wallet, &l2DevWallet, l1Client, l1ChainId, nil
}

func applyNitroDevNetRollupParameters(k *koanf.Koanf) error {
	return k.Load(confmap.Provider(map[string]interface{}{
		"persistent.chain":                   "goerli",
		"node.forwarding-target":             "https://nitro-devnet.arbitrum.io/rpc",
		"node.feed.input.url":                "https://nitro-devnet.arbitrum.io/rpc",
		"l1.rollup.bridge":                   "0x9903a892da86c1e04522d63b08e5514a921e81df",
		"l1.rollup.inbox":                    "0x1fdbbcc914e84af593884bf8e8dd6877c29035a2",
		"l1.rollup.rollup":                   "0x767cff8d8de386d7cbe91dbd39675132ba2f5967",
		"l1.rollup.sequencer-inbox":          "0x96f42d78bac19a050595c4ea6f64fe355e0af90a",
		"l1.rollup.validator-utils":          "0x96f42d78bac19a050595c4ea6f64fe355e0af90a",
		"l1.rollup.validator-wallet-creator": "0xd562adc7ff479461d29e3a3c602a017c34196add",
		"l1.rollup.deployed-at":              6664425,
		"l2.chain-id":                        421612,
	}, "."), nil)
}
