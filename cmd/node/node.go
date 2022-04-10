// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
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
	"github.com/offchainlabs/nitro/validator"

	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

func printSampleUsage() {
	progname := os.Args[0]
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func main() {
	ctx := context.Background()

	nodeConfig, l1wallet, l2wallet, err := ParseNode(ctx, os.Args[1:])
	if err != nil {
		printSampleUsage()
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
		nodeConfig.Node.EnableL1Reader = false
		nodeConfig.Node.Sequencer.Enable = true // we sequence messages, but not to l1
		nodeConfig.Node.BatchPoster.Enable = false
		nodeConfig.Node.DelayedSequencer.Enable = false
	} else {
		nodeConfig.Node.EnableL1Reader = true
	}

	if nodeConfig.Node.Sequencer.Enable {
		if nodeConfig.Node.ForwardingTarget() != "" {
			flag.Usage()
			panic("forwarding-target set when sequencer enabled")
		}
		if nodeConfig.Node.EnableL1Reader && nodeConfig.Node.InboxReader.HardReorg {
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

	if nodeConfig.Node.Wasm.RootPath != "" {
		validator.StaticNitroMachineConfig.RootPath = nodeConfig.Node.Wasm.RootPath
	} else {
		execfile, err := os.Executable()
		if err != nil {
			panic(err)
		}
		targetDir := filepath.Dir(filepath.Dir(execfile))
		validator.StaticNitroMachineConfig.RootPath = filepath.Join(targetDir, "machine")
	}

	var wasmModuleRoot common.Hash
	if nodeConfig.Node.Wasm.ModuleRoot != "" {
		wasmModuleRoot = common.HexToHash(nodeConfig.Node.Wasm.ModuleRoot)
	} else {
		wasmModuleRoot, err = validator.ReadWasmModuleRoot()
		if err != nil {
			if nodeConfig.Node.Validator.Enable && !nodeConfig.Node.Validator.Dangerous.WithoutBlockValidator {
				panic(fmt.Errorf("failed reading wasmModuleRoot from file, err %w", err))
			} else {
				wasmModuleRoot = common.Hash{}
			}
		}
	}

	if nodeConfig.Node.Validator.Enable {
		if !nodeConfig.Node.EnableL1Reader {
			flag.Usage()
			panic("validator must read from L1")
		}
		if !nodeConfig.Node.Validator.Dangerous.WithoutBlockValidator {
			nodeConfig.Node.BlockValidator.Enable = true
			if nodeConfig.Node.Wasm.CachePath != "" {
				validator.StaticNitroMachineConfig.InitialMachineCachePath = nodeConfig.Node.Wasm.CachePath
			}
			go func() {
				expectedRoot := wasmModuleRoot
				foundRoot, err := validator.GetInitialModuleRoot(ctx)
				if err != nil {
					panic(fmt.Errorf("failed reading wasmModuleRoot from machine: %w", err))
				}
				if foundRoot != expectedRoot {
					panic(fmt.Errorf("incompatible wasmModuleRoot expected: %v found %v", expectedRoot, foundRoot))
				} else {
					log.Info("loaded wasm machine", "wasmModuleRoot", foundRoot)
				}
			}()
		}
	}

	var l1client *ethclient.Client
	var deployInfo arbnode.RollupAddresses
	var l1TransactionOpts *bind.TransactOpts
	if nodeConfig.Node.EnableL1Reader {
		var err error

		l1client, err = ethclient.Dial(nodeConfig.L1.URL)
		if err != nil {
			flag.Usage()
			panic(err)
		}
		if nodeConfig.Node.BatchPoster.Enable || nodeConfig.Node.Validator.Enable {
			l1TransactionOpts, err = nitroutil.GetTransactOptsFromKeystore(
				l1wallet.Pathname,
				l1wallet.Account,
				*l1wallet.Password(),
				new(big.Int).SetUint64(nodeConfig.L1.ChainID),
			)
			if err != nil {
				panic(err)
			}
		}

		if nodeConfig.L1.Deployment == "" {
			flag.Usage()
			panic("no deployment specified")
		}
		rawDeployment, err := ioutil.ReadFile(nodeConfig.L1.Deployment)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(rawDeployment, &deployInfo); err != nil {
			panic(err)
		}
	}

	stackConf := node.DefaultConfig
	stackConf.DataDir = nodeConfig.Persistent.Data
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

	devPrivKeyStr := "e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2"
	devPrivKey, err := crypto.HexToECDSA(devPrivKeyStr)
	if err != nil {
		panic(err)
	}
	devAddr := crypto.PubkeyToAddress(devPrivKey.PublicKey)
	log.Info("Dev node funded private key", "priv", devPrivKeyStr)
	log.Info("Funded public address", "addr", devAddr)

	if l2wallet.Pathname != "" {
		mykeystore := keystore.NewPlaintextKeyStore(l2wallet.Pathname)
		stack.AccountManager().AddBackend(mykeystore)
		var account accounts.Account
		if mykeystore.HasAddress(devAddr) {
			account.Address = devAddr
			account, err = mykeystore.Find(account)
		} else {
			if l2wallet.Password() == nil {
				panic("l2 password not set")
			}
			account, err = mykeystore.ImportECDSA(devPrivKey, *l2wallet.Password())
		}
		if err != nil {
			panic(err)
		}
		if l2wallet.Password() == nil {
			panic("l2 password not set")
		}
		err = mykeystore.Unlock(account, *l2wallet.Password())
		if err != nil {
			panic(err)
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
		arbosState, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			panic(err)
		}
		chainId, err := arbosState.ChainId()
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

	node, err := arbnode.CreateNode(stack, chainDb, &nodeConfig.Node, l2BlockChain, l1client, &deployInfo, l1TransactionOpts)
	if err != nil {
		panic(err)
	}
	if nodeConfig.Node.Dangerous.NoL1Listener && nodeConfig.DevInit {
		// If we don't have any messages, we're not connected to the L1, and we're using a dev init,
		// we should create our own fake init message.
		count, err := node.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}
		if count == 0 {
			err = node.TxStreamer.AddFakeInitMessage()
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

func ParseNode(_ context.Context, args []string) (*NodeConfig, *conf.WalletConfig, *conf.WalletConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := cmdutil.BeginCommonParse(f, args)
	if err != nil {
		return nil, nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := cmdutil.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, nil, err
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
			return nil, nil, nil, errors.Wrap(err, "error removing extra parameters before dump")
		}

		c, err := k.Marshal(koanfjson.Parser())
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "unable to marshal config file to JSON")
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	// Don't pass around wallet contents with normal configuration
	l1wallet := nodeConfig.L1.Wallet
	l2wallet := nodeConfig.L2.Wallet
	nodeConfig.L1.Wallet = conf.WalletConfigDefault
	nodeConfig.L2.Wallet = conf.WalletConfigDefault

	return &nodeConfig, &l1wallet, &l2wallet, nil
}
