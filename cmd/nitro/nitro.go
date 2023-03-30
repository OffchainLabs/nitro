// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"math"
	"math/big"
	"net/http"
	_ "net/http/pprof" // #nosec G108
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/natefinch/lumberjack.v2"

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
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	_ "github.com/offchainlabs/nitro/nodeInterface"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s --help \n", name)
}

type fileHandlerFactory struct {
	writer  *lumberjack.Logger
	records chan *log.Record
	cancel  context.CancelFunc
}

// newHandler is not threadsafe
func (l *fileHandlerFactory) newHandler(logFormat log.Format, config *genericconf.FileLoggingConfig, pathResolver func(string) string) log.Handler {
	l.close()
	l.writer = &lumberjack.Logger{
		Filename:   pathResolver(config.File),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
	// capture copy of the pointer
	writer := l.writer
	// lumberjack.Logger already locks on Write, no need for SyncHandler proxy which is used in StreamHandler
	unsafeStreamHandler := log.LazyHandler(log.FuncHandler(func(r *log.Record) error {
		_, err := writer.Write(logFormat.Format(r))
		return err
	}))
	l.records = make(chan *log.Record, config.BufSize)
	// capture copy
	records := l.records
	var consumerCtx context.Context
	consumerCtx, l.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case r := <-records:
				_ = unsafeStreamHandler.Log(r)
			case <-consumerCtx.Done():
				return
			}
		}
	}()
	return log.FuncHandler(func(r *log.Record) error {
		select {
		case records <- r:
			return nil
		default:
			return fmt.Errorf("Buffer overflow, dropping record")
		}
	})
}

// close is not threadsafe
func (l *fileHandlerFactory) close() error {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	if l.writer != nil {
		if err := l.writer.Close(); err != nil {
			return err
		}
		l.writer = nil
	}
	return nil
}

var globalFileHandlerFactory = fileHandlerFactory{}

// initLog is not threadsafe
func initLog(logType string, logLevel log.Lvl, fileLoggingConfig *genericconf.FileLoggingConfig, pathResolver func(string) string) error {
	logFormat, err := genericconf.ParseLogType(logType)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("error parsing log type: %w", err)
	}
	var glogger *log.GlogHandler
	// always close previous instance of file logger
	if err := globalFileHandlerFactory.close(); err != nil {
		return fmt.Errorf("failed to close file writer: %w", err)
	}
	if fileLoggingConfig.Enable {
		glogger = log.NewGlogHandler(
			log.MultiHandler(
				log.StreamHandler(os.Stderr, logFormat),
				// on overflow records are dropped silently as MultiHandler ignores errors
				globalFileHandlerFactory.newHandler(logFormat, fileLoggingConfig, pathResolver),
			))
	} else {
		glogger = log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	}
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
	nodeConfig, l1Wallet, l2DevWallet, l1Client, l1ChainId, err := ParseNode(ctx, args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}
	stackConf := node.DefaultConfig
	stackConf.DataDir = nodeConfig.Persistent.Chain
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
		fileName := stackConf.ResolvePath("jwtsecret")
		secret := common.Hash{}
		_, err := rand.Read(secret[:])
		if err != nil {
			log.Crit("couldn't create jwt secret", "err", err, "fileName", fileName)
		}
		err = os.MkdirAll(filepath.Dir(fileName), 0755)
		if err != nil {
			log.Crit("couldn't create directory for jwt secret", "err", err, "dirName", filepath.Dir(fileName))
		}
		err = os.WriteFile(fileName, []byte(secret.Hex()), fs.FileMode(0600|os.O_CREATE))
		if errors.Is(err, fs.ErrExist) {
			log.Info("using existing jwt file", "fileName", fileName)
		} else {
			if err != nil {
				log.Crit("couldn't create jwt secret", "err", err, "fileName", fileName)
			}
			log.Info("created jwt file", "fileName", fileName)
		}
		stackConf.JWTSecret = fileName
	}

	if nodeConfig.Node.BlockValidator.JWTSecret == "self" {
		nodeConfig.Node.BlockValidator.JWTSecret = stackConf.JWTSecret
	}

	err = initLog(nodeConfig.LogType, log.Lvl(nodeConfig.LogLevel), &nodeConfig.FileLogging, stackConf.ResolvePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		os.Exit(1)
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
	sequencerNeedsKey := nodeConfig.Node.Sequencer.Enable && !nodeConfig.Node.Feed.Output.DisableSigning
	setupNeedsKey := l1Wallet.OnlyCreateKey || nodeConfig.Node.Staker.OnlyCreateWalletContract
	validatorCanAct := nodeConfig.Node.Staker.Enable && !strings.EqualFold(nodeConfig.Node.Staker.Strategy, "watchtower")
	if sequencerNeedsKey || nodeConfig.Node.BatchPoster.Enable || setupNeedsKey || validatorCanAct {
		l1TransactionOpts, dataSigner, err = util.OpenWallet("l1", l1Wallet, new(big.Int).SetUint64(nodeConfig.L1.ChainID))
		if err != nil {
			flag.Usage()
			log.Crit("error opening L1 wallet", "path", l1Wallet.Pathname, "account", l1Wallet.Account, "err", err)
		}
	}

	var rollupAddrs arbnode.RollupAddresses
	if nodeConfig.Node.L1Reader.Enable {
		log.Info("connected to l1 chain", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)

		rollupAddrs, err = nodeConfig.L1.Rollup.ParseAddresses()
		if err != nil {
			log.Crit("error getting rollup addresses", "err", err)
		}
	} else if l1Client != nil {
		// Don't need l1Client anymore
		log.Info("used chain id to get rollup parameters", "l1url", nodeConfig.L1.URL, "l1chainid", l1ChainId)
		l1Client = nil
	}

	if nodeConfig.Node.Staker.Enable {
		if !nodeConfig.Node.L1Reader.Enable {
			flag.Usage()
			log.Crit("validator have the L1 reader enabled")
		}
		if !nodeConfig.Node.Staker.Dangerous.WithoutBlockValidator {
			nodeConfig.Node.BlockValidator.Enable = true
		}
	}

	liveNodeConfig := NewLiveNodeConfig(args, nodeConfig, stackConf.ResolvePath)
	if nodeConfig.Node.Staker.OnlyCreateWalletContract {
		if !nodeConfig.Node.Staker.UseSmartContractWallet {
			flag.Usage()
			log.Crit("--node.validator.only-create-wallet-contract requires --node.validator.use-smart-contract-wallet")
		}
		l1Reader := headerreader.New(l1Client, func() *headerreader.Config { return &liveNodeConfig.get().Node.L1Reader })

		// Just create validator smart wallet if needed then exit
		deployInfo, err := nodeConfig.L1.Rollup.ParseAddresses()
		if err != nil {
			log.Crit("error getting deployment info for creating validator wallet contract", "error", err)
		}
		addr, err := staker.GetValidatorWalletContract(ctx, deployInfo.ValidatorWalletCreator, int64(deployInfo.DeployedAt), l1TransactionOpts, l1Reader, true)
		if err != nil {
			log.Crit("error creating validator wallet contract", "error", err, "address", l1TransactionOpts.From.Hex())
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

	chainDb, l2BlockChain, err := openInitializeChainDb(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.L2.ChainID), arbnode.DefaultCacheConfigFor(stack, &nodeConfig.Node.Caching), l1Client, rollupAddrs)
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
				startPprof(address)
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
		func() *valnode.Config { return &liveNodeConfig.get().Validation },
		stack,
		fatalErrChan,
	)
	if err != nil {
		valNode = nil
		log.Warn("couldn't init validation node", "err", err)
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
		l1TransactionOpts,
		dataSigner,
		fatalErrChan,
	)
	if err != nil {
		log.Error("failed to create node", "err", err)
		return 1
	}
	liveNodeConfig.setOnReloadHook(func(oldCfg *NodeConfig, newCfg *NodeConfig) error {
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
		if err := graphql.New(stack, currentNode.Backend.APIBackend(), currentNode.FilterSystem, gqlConf.CORSDomain, gqlConf.VHosts); err != nil {
			log.Error("failed to register the GraphQL service", "err", err)
			return 1
		}
	}

	if valNode != nil {
		err = valNode.Start(ctx)
		if err != nil {
			fatalErrChan <- fmt.Errorf("error starting validator node: %w", err)
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
	L1            conf.L1Config                   `koanf:"l1"`
	L2            conf.L2Config                   `koanf:"l2"`
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
	conf.L1ConfigAddOptions("l1", f)
	conf.L2ConfigAddOptions("l2", f)
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
	return c.Node.Validate()
}

type RpcLogger struct{}

func (l RpcLogger) OnRequest(request interface{}) rpc.ResultHook {
	log.Trace("sending L1 RPC request", "request", request)
	return RpcResultLogger{request}
}

type RpcResultLogger struct {
	request interface{}
}

func (l RpcResultLogger) OnResult(response interface{}, err error) {
	if err != nil {
		// The request might not've been logged if the log level is debug not trace, so we log it again here
		log.Info("received error response from L1 RPC", "request", l.request, "response", response, "err", err)
	} else {
		// The request was already logged and can be cross-referenced by JSON-RPC id
		log.Trace("received response from L1 RPC", "response", response)
	}
}

func ParseNode(ctx context.Context, args []string) (*NodeConfig, *genericconf.WalletConfig, *genericconf.WalletConfig, *ethclient.Client, *big.Int, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
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
			rawRpc, err := rpc.DialContextWithRequestHook(ctx, l1URL, RpcLogger{})
			if err == nil {
				l1Client = ethclient.NewClient(rawRpc)
				l1ChainId, err = l1Client.ChainID(ctx)
				if err == nil {
					// Successfully got chain ID
					break
				}
			}
			if i < maxConnectionAttempts {
				log.Warn("error connecting to L1", "err", err)
			} else {
				return nil, nil, nil, nil, nil, fmt.Errorf("too many errors trying to connect to L1: %w", err)
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

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := confighelpers.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, nil, nil, nil, err
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
		"node.forwarding-target":             "https://arb1-sequencer.arbitrum.io/rpc",
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

func noopOnReloadHook(_ *NodeConfig, _ *NodeConfig) error {
	return nil
}

type LiveNodeConfig struct {
	stopwaiter.StopWaiter

	mutex        sync.RWMutex
	args         []string
	config       *NodeConfig
	pathResolver func(string) string
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
	if err := initLog(config.LogType, log.Lvl(config.LogLevel), &config.FileLogging, c.pathResolver); err != nil {
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
	c.StopWaiter.Start(ctxIn, c)

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

func NewLiveNodeConfig(args []string, config *NodeConfig, pathResolver func(string) string) *LiveNodeConfig {
	return &LiveNodeConfig{
		args:         args,
		config:       config,
		pathResolver: pathResolver,
		onReloadHook: noopOnReloadHook,
	}
}

type NodeConfigFetcher struct {
	*LiveNodeConfig
}

func (f *NodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveNodeConfig.get().Node
}

func startPprof(address string) {
	exp.Exp(metrics.DefaultRegistry)
	log.Info("Starting metrics server with pprof", "addr", fmt.Sprintf("http://%s/debug/metrics", address))
	log.Info("Pprof endpoint", "addr", fmt.Sprintf("http://%s/debug/pprof", address))
	go func() {
		// #nosec G114
		if err := http.ListenAndServe(address, http.DefaultServeMux); err != nil {
			log.Error("Failure in running pprof server", "err", err)
		}
	}()
}
