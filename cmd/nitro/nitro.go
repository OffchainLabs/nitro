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
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	flag "github.com/spf13/pflag"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/execution/gethexec"
	_ "github.com/offchainlabs/nitro/execution/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s [OPTIONS] \n\n", name)
	fmt.Printf("Options:\n")
	fmt.Printf("  --help\n")
	fmt.Printf("  --dev: Start a default L2-only dev chain\n")
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
			if walletConf.Pwd() == nil {
				return common.Address{}, errors.New("l2 password not set")
			}
			if devPrivKey == nil {
				return common.Address{}, errors.New("l2 private key not set")
			}
			account, err = myKeystore.ImportECDSA(devPrivKey, *walletConf.Pwd())
		}
		if err != nil {
			return common.Address{}, err
		}
		if walletConf.Pwd() == nil {
			return common.Address{}, errors.New("l2 password not set")
		}
		err = myKeystore.Unlock(account, *walletConf.Pwd())
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
		if err != nil && !strings.Contains(err.Error(), leveldb.ErrClosed.Error()) && !strings.Contains(err.Error(), pebble.ErrClosed.Error()) {
			log.Warn("failed to close database on shutdown", "db", name, "err", err)
		}
	}
}

func main() {
	os.Exit(mainImpl())
}

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics(cfg *NodeConfig) error {
	mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	if cfg.Metrics && !metrics.Enabled {
		return errors.New("metrics must be enabled via command line by adding --metrics, json config has no effect")
	}
	if cfg.Metrics && cfg.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if cfg.Metrics {
		go metrics.CollectProcessMetrics(cfg.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	}
	if cfg.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
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
	stackConf.DBEngine = nodeConfig.Persistent.DBEngine
	nodeConfig.Rpc.Apply(&stackConf)
	nodeConfig.HTTP.Apply(&stackConf)
	nodeConfig.WS.Apply(&stackConf)
	nodeConfig.Auth.Apply(&stackConf)
	nodeConfig.IPC.Apply(&stackConf)
	nodeConfig.GraphQL.Apply(&stackConf)
	if nodeConfig.WS.ExposeAll {
		stackConf.WSModules = append(stackConf.WSModules, "personal")
	}
	nodeConfig.P2P.Apply(&stackConf)
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

	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := pathResolver(nodeConfig.Persistent.GlobalConfig)("jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
			log.Error("Failed to prepare jwt secret file", "err", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}
	err = genericconf.InitLog(nodeConfig.LogType, log.Lvl(nodeConfig.LogLevel), &nodeConfig.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		return 1
	}

	log.Info("Running Arbitrum nitro node", "revision", vcsRevision, "vcs.time", vcsTime)

	if nodeConfig.Node.Dangerous.NoL1Listener {
		nodeConfig.Node.ParentChainReader.Enable = false
		nodeConfig.Node.BatchPoster.Enable = false
		nodeConfig.Node.DelayedSequencer.Enable = false
	} else {
		nodeConfig.Node.ParentChainReader.Enable = true
	}

	if nodeConfig.Execution.Sequencer.Enable && nodeConfig.Node.ParentChainReader.Enable && nodeConfig.Node.InboxReader.HardReorg {
		flag.Usage()
		log.Crit("hard reorgs cannot safely be enabled with sequencer mode enabled")
	}
	if nodeConfig.Execution.Sequencer.Enable != nodeConfig.Node.Sequencer {
		log.Error("consensus and execution must agree if sequencing is enabled or not", "Execution.Sequencer.Enable", nodeConfig.Execution.Sequencer.Enable, "Node.Sequencer", nodeConfig.Node.Sequencer)
	}

	var l1TransactionOpts *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	var l1TransactionOptsValidator *bind.TransactOpts
	var l1TransactionOptsBatchPoster *bind.TransactOpts
	// If sequencer and signing is enabled or batchposter is enabled without
	// external signing sequencer will need a key.
	sequencerNeedsKey := (nodeConfig.Node.Sequencer && !nodeConfig.Node.Feed.Output.DisableSigning) ||
		(nodeConfig.Node.BatchPoster.Enable && nodeConfig.Node.BatchPoster.DataPoster.ExternalSigner.URL == "")
	validatorNeedsKey := nodeConfig.Node.Staker.OnlyCreateWalletContract ||
		(nodeConfig.Node.Staker.Enable && !strings.EqualFold(nodeConfig.Node.Staker.Strategy, "watchtower") && nodeConfig.Node.Staker.DataPoster.ExternalSigner.URL == "")

	l1Wallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultL1WalletConfig := conf.DefaultL1WalletConfig
	defaultL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	nodeConfig.Node.Staker.ParentChainWallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultValidatorL1WalletConfig := staker.DefaultValidatorL1WalletConfig
	defaultValidatorL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	nodeConfig.Node.BatchPoster.ParentChainWallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultBatchPosterL1WalletConfig := arbnode.DefaultBatchPosterL1WalletConfig
	defaultBatchPosterL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	if nodeConfig.Node.Staker.ParentChainWallet == defaultValidatorL1WalletConfig && nodeConfig.Node.BatchPoster.ParentChainWallet == defaultBatchPosterL1WalletConfig {
		if sequencerNeedsKey || validatorNeedsKey || l1Wallet.OnlyCreateKey {
			l1TransactionOpts, dataSigner, err = util.OpenWallet("l1", l1Wallet, new(big.Int).SetUint64(nodeConfig.ParentChain.ID))
			if err != nil {
				flag.Usage()
				log.Crit("error opening parent chain wallet", "path", l1Wallet.Pathname, "account", l1Wallet.Account, "err", err)
			}
			if l1Wallet.OnlyCreateKey {
				return 0
			}
			l1TransactionOptsBatchPoster = l1TransactionOpts
			l1TransactionOptsValidator = l1TransactionOpts
		}
	} else {
		if *l1Wallet != defaultL1WalletConfig {
			log.Crit("--parent-chain.wallet cannot be set if either --node.staker.l1-wallet or --node.batch-poster.l1-wallet are set")
		}
		if sequencerNeedsKey || nodeConfig.Node.BatchPoster.ParentChainWallet.OnlyCreateKey {
			l1TransactionOptsBatchPoster, dataSigner, err = util.OpenWallet("l1-batch-poster", &nodeConfig.Node.BatchPoster.ParentChainWallet, new(big.Int).SetUint64(nodeConfig.ParentChain.ID))
			if err != nil {
				flag.Usage()
				log.Crit("error opening Batch poster parent chain wallet", "path", nodeConfig.Node.BatchPoster.ParentChainWallet.Pathname, "account", nodeConfig.Node.BatchPoster.ParentChainWallet.Account, "err", err)
			}
			if nodeConfig.Node.BatchPoster.ParentChainWallet.OnlyCreateKey {
				return 0
			}
		}
		if validatorNeedsKey || nodeConfig.Node.Staker.ParentChainWallet.OnlyCreateKey {
			l1TransactionOptsValidator, _, err = util.OpenWallet("l1-validator", &nodeConfig.Node.Staker.ParentChainWallet, new(big.Int).SetUint64(nodeConfig.ParentChain.ID))
			if err != nil {
				flag.Usage()
				log.Crit("error opening Validator parent chain wallet", "path", nodeConfig.Node.Staker.ParentChainWallet.Pathname, "account", nodeConfig.Node.Staker.ParentChainWallet.Account, "err", err)
			}
			if nodeConfig.Node.Staker.ParentChainWallet.OnlyCreateKey {
				return 0
			}
		}
	}

	combinedL2ChainInfoFile := aggregateL2ChainInfoFiles(ctx, nodeConfig.Chain.InfoFiles, nodeConfig.Chain.InfoIpfsUrl, nodeConfig.Chain.InfoIpfsDownloadPath)

	if nodeConfig.Node.Staker.Enable {
		if !nodeConfig.Node.ParentChainReader.Enable {
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

	if nodeConfig.Execution.RPC.MaxRecreateStateDepth == arbitrum.UninitializedMaxRecreateStateDepth {
		if nodeConfig.Execution.Caching.Archive {
			nodeConfig.Execution.RPC.MaxRecreateStateDepth = arbitrum.DefaultArchiveNodeMaxRecreateStateDepth
		} else {
			nodeConfig.Execution.RPC.MaxRecreateStateDepth = arbitrum.DefaultNonArchiveNodeMaxRecreateStateDepth
		}
	}
	liveNodeConfig := genericconf.NewLiveConfig[*NodeConfig](args, nodeConfig, func(ctx context.Context, args []string) (*NodeConfig, error) {
		nodeConfig, _, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})

	var rollupAddrs chaininfo.RollupAddresses
	var l1Client *ethclient.Client
	var l1Reader *headerreader.HeaderReader
	var blobReader arbstate.BlobReader
	if nodeConfig.Node.ParentChainReader.Enable {
		confFetcher := func() *rpcclient.ClientConfig { return &liveNodeConfig.Get().ParentChain.Connection }
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
		if l1ChainId.Uint64() != nodeConfig.ParentChain.ID {
			log.Crit("L1 chainID doesn't fit config", "found", l1ChainId.Uint64(), "expected", nodeConfig.ParentChain.ID)
		}

		log.Info("connected to l1 chain", "l1url", nodeConfig.ParentChain.Connection.URL, "l1chainid", nodeConfig.ParentChain.ID)

		rollupAddrs, err = chaininfo.GetRollupAddressesConfig(nodeConfig.Chain.ID, nodeConfig.Chain.Name, combinedL2ChainInfoFile, nodeConfig.Chain.InfoJson)
		if err != nil {
			log.Crit("error getting rollup addresses", "err", err)
		}
		arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
		l1Reader, err = headerreader.New(ctx, l1Client, func() *headerreader.Config { return &liveNodeConfig.Get().Node.ParentChainReader }, arbSys)
		if err != nil {
			log.Crit("failed to get L1 headerreader", "err", err)
		}
		if !l1Reader.IsParentChainArbitrum() && !nodeConfig.Node.Dangerous.DisableBlobReader {
			if nodeConfig.ParentChain.BlobClient.BeaconUrl == "" {
				flag.Usage()
				log.Crit("a beacon chain RPC URL is required to read batches, but it was not configured (CLI argument: --parent-chain.blob-client.beacon-url [URL])")
			}
			blobClient, err := headerreader.NewBlobClient(nodeConfig.ParentChain.BlobClient, l1Client)
			if err != nil {
				log.Crit("failed to initialize blob client", "err", err)
			}
			blobReader = blobClient
		}
	}

	if nodeConfig.Node.Staker.OnlyCreateWalletContract {
		if !nodeConfig.Node.Staker.UseSmartContractWallet {
			flag.Usage()
			log.Crit("--node.validator.only-create-wallet-contract requires --node.validator.use-smart-contract-wallet")
		}
		if l1Reader == nil {
			flag.Usage()
			log.Crit("--node.validator.only-create-wallet-contract conflicts with --node.dangerous.no-l1-listener")
		}
		// Just create validator smart wallet if needed then exit
		deployInfo, err := chaininfo.GetRollupAddressesConfig(nodeConfig.Chain.ID, nodeConfig.Chain.Name, combinedL2ChainInfoFile, nodeConfig.Chain.InfoJson)
		if err != nil {
			log.Crit("error getting rollup addresses config", "err", err)
		}
		addr, err := validatorwallet.GetValidatorWalletContract(ctx, deployInfo.ValidatorWalletCreator, int64(deployInfo.DeployedAt), l1TransactionOptsValidator, l1Reader, true)
		if err != nil {
			log.Crit("error creating validator wallet contract", "error", err, "address", l1TransactionOptsValidator.From.Hex())
		}
		fmt.Printf("Created validator smart contract wallet at %s, remove --node.validator.only-create-wallet-contract and restart\n", addr.String())
		return 0
	}

	if nodeConfig.Execution.Caching.Archive && nodeConfig.Execution.TxLookupLimit != 0 {
		log.Info("retaining ability to lookup full transaction history as archive mode is enabled")
		nodeConfig.Execution.TxLookupLimit = 0
	}

	if err := resourcemanager.Init(&nodeConfig.Node.ResourceMgmt); err != nil {
		flag.Usage()
		log.Crit("Failed to start resource management module", "err", err)
	}

	var sameProcessValidationNodeEnabled bool
	if nodeConfig.Node.BlockValidator.Enable && (nodeConfig.Node.BlockValidator.ValidationServerConfigs[0].URL == "self" || nodeConfig.Node.BlockValidator.ValidationServerConfigs[0].URL == "self-auth") {
		sameProcessValidationNodeEnabled = true
		valnode.EnsureValidationExposedViaAuthRPC(&stackConf)
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
			nodeConfig.Init.DevInitAddress = devAddr.String()
		}
	}

	if err := startMetrics(nodeConfig); err != nil {
		log.Error("Error starting metrics", "error", err)
		return 1
	}

	var deferFuncs []func()
	defer func() {
		for i := range deferFuncs {
			deferFuncs[i]()
		}
	}()

	// Check that node is compatible with on-chain WASM module root on startup and before any ArbOS upgrades take effect to prevent divergences
	if nodeConfig.Node.ParentChainReader.Enable && nodeConfig.Validation.Wasm.EnableWasmrootsCheck {
		// Fetch current on-chain WASM module root
		rollupUserLogic, err := rollupgen.NewRollupUserLogic(rollupAddrs.Rollup, l1Client)
		if err != nil {
			log.Error("failed to create rollupUserLogic", "err", err)
			return 1
		}
		moduleRoot, err := rollupUserLogic.WasmModuleRoot(&bind.CallOpts{Context: ctx})
		if err != nil {
			log.Error("failed to get on-chain WASM module root", "err", err)
			return 1
		}
		if (moduleRoot == common.Hash{}) {
			log.Error("on-chain WASM module root is zero")
			return 1
		}
		// Check if the on-chain WASM module root belongs to the set of allowed module roots
		allowedWasmModuleRoots := nodeConfig.Validation.Wasm.AllowedWasmModuleRoots
		if len(allowedWasmModuleRoots) > 0 {
			moduleRootMatched := false
			for _, root := range allowedWasmModuleRoots {
				if common.HexToHash(root) == moduleRoot {
					moduleRootMatched = true
					break
				}
			}
			if !moduleRootMatched {
				log.Error("on-chain WASM module root did not match with any of the allowed WASM module roots")
				return 1
			}
		} else {
			// If no allowed module roots were provided in config, check if we have a validator machine directory for the on-chain WASM module root
			locator, err := server_common.NewMachineLocator(nodeConfig.Validation.Wasm.RootPath)
			if err != nil {
				log.Warn("failed to create machine locator. Skipping the check for compatibility with on-chain WASM module root", "err", err)
			} else {
				path := locator.GetMachinePath(moduleRoot)
				if _, err := os.Stat(path); err != nil {
					log.Error("unable to find validator machine directory for the on-chain WASM module root", "err", err)
					return 1
				}
			}
		}
	}

	chainDb, l2BlockChain, err := openInitializeChainDb(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.Chain.ID), gethexec.DefaultCacheConfigFor(stack, &nodeConfig.Execution.Caching), l1Client, rollupAddrs)
	if l2BlockChain != nil {
		deferFuncs = append(deferFuncs, func() { l2BlockChain.Stop() })
	}
	deferFuncs = append(deferFuncs, func() { closeDb(chainDb, "chainDb") })
	if err != nil {
		flag.Usage()
		log.Error("error initializing database", "err", err)
		return 1
	}

	arbDb, err := stack.OpenDatabase("arbitrumdata", 0, 0, "arbitrumdata/", false)
	deferFuncs = append(deferFuncs, func() { closeDb(arbDb, "arbDb") })
	if err != nil {
		log.Error("failed to open database", "err", err)
		return 1
	}

	if nodeConfig.Init.ThenQuit && nodeConfig.Init.ResetToMessage < 0 {
		return 0
	}

	chainInfo, err := chaininfo.ProcessChainInfo(nodeConfig.Chain.ID, nodeConfig.Chain.Name, combinedL2ChainInfoFile, nodeConfig.Chain.InfoJson)
	if err != nil {
		log.Error("error processing l2 chain info", "err", err)
		return 1
	}
	if err := validateBlockChain(l2BlockChain, chainInfo.ChainConfig); err != nil {
		log.Error("user provided chain config is not compatible with onchain chain config", "err", err)
		return 1
	}

	if l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee != nodeConfig.Node.DataAvailability.Enable {
		flag.Usage()
		log.Error(fmt.Sprintf("data availability service usage for this chain is set to %v but --node.data-availability.enable is set to %v", l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee, nodeConfig.Node.DataAvailability.Enable))
		return 1
	}

	fatalErrChan := make(chan error, 10)

	var valNode *valnode.ValidationNode
	if sameProcessValidationNodeEnabled {
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

	execNode, err := gethexec.CreateExecutionNode(
		ctx,
		stack,
		chainDb,
		l2BlockChain,
		l1Client,
		func() *gethexec.Config { return &liveNodeConfig.Get().Execution },
	)
	if err != nil {
		log.Error("failed to create execution node", "err", err)
		return 1
	}

	currentNode, err := arbnode.CreateNode(
		ctx,
		stack,
		execNode,
		arbDb,
		&NodeConfigFetcher{liveNodeConfig},
		l2BlockChain.Config(),
		l1Client,
		&rollupAddrs,
		l1TransactionOptsValidator,
		l1TransactionOptsBatchPoster,
		dataSigner,
		fatalErrChan,
		big.NewInt(int64(nodeConfig.ParentChain.ID)),
		blobReader,
	)
	if err != nil {
		log.Error("failed to create node", "err", err)
		return 1
	}

	// Validate sequencer's MaxTxDataSize and batchPoster's MaxSize params.
	// SequencerInbox's maxDataSize is defaulted to 117964 which is 90% of Geth's 128KB tx size limit, leaving ~13KB for proving.
	seqInboxMaxDataSize := 117964
	if nodeConfig.Node.ParentChainReader.Enable {
		seqInbox, err := bridgegen.NewSequencerInbox(rollupAddrs.SequencerInbox, l1Client)
		if err != nil {
			log.Error("failed to create sequencer inbox for validating sequencer's MaxTxDataSize and batchposter's MaxSize", "err", err)
			return 1
		}
		res, err := seqInbox.MaxDataSize(&bind.CallOpts{Context: ctx})
		if err == nil {
			seqInboxMaxDataSize = int(res.Int64())
		} else if !headerreader.ExecutionRevertedRegexp.MatchString(err.Error()) {
			log.Error("error fetching MaxDataSize from sequencer inbox", "err", err)
			return 1
		}
	}
	// If batchPoster is enabled, validate MaxSize to be at least 10kB below the sequencer inbox’s maxDataSize if the data availability service is not enabled.
	// The 10kB gap is because its possible for the batch poster to exceed its MaxSize limit and produce batches of slightly larger size.
	if nodeConfig.Node.BatchPoster.Enable && !nodeConfig.Node.DataAvailability.Enable {
		if nodeConfig.Node.BatchPoster.MaxSize > seqInboxMaxDataSize-10000 {
			log.Error("batchPoster's MaxSize is too large")
			return 1
		}
	}
	// If sequencer is enabled, validate MaxTxDataSize to be at least 5kB below the batch poster's MaxSize to allow space for headers and such.
	// And since batchposter's MaxSize is to be at least 10kB below the sequencer inbox’s maxDataSize, this leads to another condition of atlest 15kB below the sequencer inbox’s maxDataSize.
	if nodeConfig.Execution.Sequencer.Enable {
		if nodeConfig.Execution.Sequencer.MaxTxDataSize > nodeConfig.Node.BatchPoster.MaxSize-5000 ||
			nodeConfig.Execution.Sequencer.MaxTxDataSize > seqInboxMaxDataSize-15000 {
			log.Error("sequencer's MaxTxDataSize too large")
			return 1
		}
	}

	liveNodeConfig.SetOnReloadHook(func(oldCfg *NodeConfig, newCfg *NodeConfig) error {
		if err := genericconf.InitLog(newCfg.LogType, log.Lvl(newCfg.LogLevel), &newCfg.FileLogging, pathResolver(nodeConfig.Persistent.LogDir)); err != nil {
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
		if err := graphql.New(stack, execNode.Backend.APIBackend(), execNode.FilterSystem, gqlConf.CORSDomain, gqlConf.VHosts); err != nil {
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
		// remove previous deferFuncs, StopAndWait closes database and blockchain.
		deferFuncs = []func(){func() { currentNode.StopAndWait() }}
	}
	if nodeConfig.BlocksReExecutor.Enable && l2BlockChain != nil {
		blocksReExecutor := blocksreexecutor.New(&nodeConfig.BlocksReExecutor, l2BlockChain, fatalErrChan)
		blocksReExecutor.Start(ctx)
		deferFuncs = append(deferFuncs, func() { blocksReExecutor.StopAndWait() })
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	exitCode := 0

	if err == nil && nodeConfig.Init.ResetToMessage > 0 {
		err = currentNode.TxStreamer.ReorgTo(arbutil.MessageIndex(nodeConfig.Init.ResetToMessage))
		if err != nil {
			fatalErrChan <- fmt.Errorf("error reseting message: %w", err)
			exitCode = 1
		}
		if nodeConfig.Init.ThenQuit {
			close(sigint)

			return exitCode
		}
	}

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

type NodeConfig struct {
	Conf             genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Node             arbnode.Config                  `koanf:"node" reload:"hot"`
	Execution        gethexec.Config                 `koanf:"execution" reload:"hot"`
	Validation       valnode.Config                  `koanf:"validation" reload:"hot"`
	ParentChain      conf.ParentChainConfig          `koanf:"parent-chain" reload:"hot"`
	Chain            conf.L2Config                   `koanf:"chain"`
	LogLevel         int                             `koanf:"log-level" reload:"hot"`
	LogType          string                          `koanf:"log-type" reload:"hot"`
	FileLogging      genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	Persistent       conf.PersistentConfig           `koanf:"persistent"`
	HTTP             genericconf.HTTPConfig          `koanf:"http"`
	WS               genericconf.WSConfig            `koanf:"ws"`
	IPC              genericconf.IPCConfig           `koanf:"ipc"`
	Auth             genericconf.AuthRPCConfig       `koanf:"auth"`
	GraphQL          genericconf.GraphQLConfig       `koanf:"graphql"`
	P2P              genericconf.P2PConfig           `koanf:"p2p"`
	Metrics          bool                            `koanf:"metrics"`
	MetricsServer    genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf            bool                            `koanf:"pprof"`
	PprofCfg         genericconf.PProf               `koanf:"pprof-cfg"`
	Init             conf.InitConfig                 `koanf:"init"`
	Rpc              genericconf.RpcConfig           `koanf:"rpc"`
	BlocksReExecutor blocksreexecutor.Config         `koanf:"blocks-reexecutor"`
}

var NodeConfigDefault = NodeConfig{
	Conf:             genericconf.ConfConfigDefault,
	Node:             arbnode.ConfigDefault,
	Execution:        gethexec.ConfigDefault,
	Validation:       valnode.DefaultValidationConfig,
	ParentChain:      conf.L1ConfigDefault,
	Chain:            conf.L2ConfigDefault,
	LogLevel:         int(log.LvlInfo),
	LogType:          "plaintext",
	FileLogging:      genericconf.DefaultFileLoggingConfig,
	Persistent:       conf.PersistentConfigDefault,
	HTTP:             genericconf.HTTPConfigDefault,
	WS:               genericconf.WSConfigDefault,
	IPC:              genericconf.IPCConfigDefault,
	Auth:             genericconf.AuthRPCConfigDefault,
	GraphQL:          genericconf.GraphQLConfigDefault,
	P2P:              genericconf.P2PConfigDefault,
	Metrics:          false,
	MetricsServer:    genericconf.MetricsServerConfigDefault,
	Init:             conf.InitConfigDefault,
	Rpc:              genericconf.DefaultRpcConfig,
	PProf:            false,
	PprofCfg:         genericconf.PProfDefault,
	BlocksReExecutor: blocksreexecutor.DefaultConfig,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	gethexec.ConfigAddOptions("execution", f)
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
	genericconf.P2PConfigAddOptions("p2p", f)
	genericconf.GraphQLConfigAddOptions("graphql", f)
	f.Bool("metrics", NodeConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", NodeConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	conf.InitConfigAddOptions("init", f)
	genericconf.RpcConfigAddOptions("rpc", f)
	blocksreexecutor.ConfigAddOptions("blocks-reexecutor", f)
}

func (c *NodeConfig) ResolveDirectoryNames() error {
	err := c.Persistent.ResolveDirectoryNames()
	if err != nil {
		return err
	}
	c.ParentChain.ResolveDirectoryNames(c.Persistent.Chain)
	c.Chain.ResolveDirectoryNames(c.Persistent.Chain)

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
	if c.Init.RecreateMissingStateFrom > 0 && !c.Execution.Caching.Archive {
		return errors.New("recreate-missing-state-from enabled for a non-archive node")
	}
	if err := c.Init.Validate(); err != nil {
		return err
	}
	if err := c.ParentChain.Validate(); err != nil {
		return err
	}
	if err := c.Node.Validate(); err != nil {
		return err
	}
	if err := c.Execution.Validate(); err != nil {
		return err
	}
	if err := c.BlocksReExecutor.Validate(); err != nil {
		return err
	}
	return c.Persistent.Validate()
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
	l2ChainInfoFiles := k.Strings("chain.info-files")
	l2ChainInfoJson := k.String("chain.info-json")
	err = applyChainParameters(ctx, k, uint64(l2ChainId), l2ChainName, l2ChainInfoFiles, l2ChainInfoJson, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
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
		return nil, nil, nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, nil, nil, err
	}

	// Don't pass around wallet contents with normal configuration
	l1Wallet := nodeConfig.ParentChain.Wallet
	l2DevWallet := nodeConfig.Chain.DevWallet
	nodeConfig.ParentChain.Wallet = genericconf.WalletConfigDefault
	nodeConfig.Chain.DevWallet = genericconf.WalletConfigDefault

	if nodeConfig.Execution.Caching.Archive {
		nodeConfig.Node.MessagePruner.Enable = false
	}
	err = nodeConfig.Validate()
	if err != nil {
		return nil, nil, nil, err
	}
	return &nodeConfig, &l1Wallet, &l2DevWallet, nil
}

func aggregateL2ChainInfoFiles(ctx context.Context, l2ChainInfoFiles []string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) []string {
	if l2ChainInfoIpfsUrl != "" {
		l2ChainInfoIpfsFile, err := util.GetL2ChainInfoIpfsFile(ctx, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
		if err != nil {
			log.Error("error getting l2 chain info file from ipfs", "err", err)
		}
		l2ChainInfoFiles = append(l2ChainInfoFiles, l2ChainInfoIpfsFile)
	}
	return l2ChainInfoFiles
}

func applyChainParameters(ctx context.Context, k *koanf.Koanf, chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoJson string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) error {
	combinedL2ChainInfoFiles := aggregateL2ChainInfoFiles(ctx, l2ChainInfoFiles, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
	chainInfo, err := chaininfo.ProcessChainInfo(chainId, chainName, combinedL2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return err
	}
	var parentChainIsArbitrum bool
	if chainInfo.ParentChainIsArbitrum != nil {
		parentChainIsArbitrum = *chainInfo.ParentChainIsArbitrum
	} else {
		log.Warn("Chain info field parent-chain-is-arbitrum is missing, in the future this will be required", "chainId", chainInfo.ChainConfig.ChainID, "parentChainId", chainInfo.ParentChainId)
		_, err := chaininfo.ProcessChainInfo(chainInfo.ParentChainId, "", combinedL2ChainInfoFiles, "")
		if err == nil {
			parentChainIsArbitrum = true
		}
	}
	chainDefaults := map[string]interface{}{
		"persistent.chain": chainInfo.ChainName,
		"chain.id":         chainInfo.ChainConfig.ChainID.Uint64(),
		"parent-chain.id":  chainInfo.ParentChainId,
	}
	// Only use chainInfo.SequencerUrl as default forwarding-target if sequencer is not enabled
	if !k.Bool("execution.sequencer.enable") && chainInfo.SequencerUrl != "" {
		chainDefaults["execution.forwarding-target"] = chainInfo.SequencerUrl
	}
	if chainInfo.SecondaryForwardingTarget != "" {
		chainDefaults["execution.secondary-forwarding-target"] = strings.Split(chainInfo.SecondaryForwardingTarget, ",")
	}
	if chainInfo.FeedUrl != "" {
		chainDefaults["node.feed.input.url"] = strings.Split(chainInfo.FeedUrl, ",")
	}
	if chainInfo.SecondaryFeedUrl != "" {
		chainDefaults["node.feed.input.secondary-url"] = strings.Split(chainInfo.SecondaryFeedUrl, ",")
	}
	if chainInfo.DasIndexUrl != "" {
		chainDefaults["node.data-availability.enable"] = true
		chainDefaults["node.data-availability.rest-aggregator.enable"] = true
		chainDefaults["node.data-availability.rest-aggregator.online-url-list"] = chainInfo.DasIndexUrl
	} else if chainInfo.ChainConfig.ArbitrumChainParams.DataAvailabilityCommittee {
		chainDefaults["node.data-availability.enable"] = true
	}
	if !chainInfo.HasGenesisState {
		chainDefaults["init.empty"] = true
	}
	if parentChainIsArbitrum {
		l2MaxTxSize := gethexec.DefaultSequencerConfig.MaxTxDataSize
		bufferSpace := 5000
		if l2MaxTxSize < bufferSpace*2 {
			return fmt.Errorf("not enough room in parent chain max tx size %v for bufferSpace %v * 2", l2MaxTxSize, bufferSpace)
		}
		safeBatchSize := l2MaxTxSize - bufferSpace
		chainDefaults["node.batch-poster.max-size"] = safeBatchSize
		chainDefaults["execution.sequencer.max-tx-data-size"] = safeBatchSize - bufferSpace
		// Arbitrum chains produce blocks more quickly, so the inbox reader should read more blocks at once.
		// Even if this is too large, on error the inbox reader will reset its query size down to the default.
		chainDefaults["node.inbox-reader.max-blocks-to-read"] = 10_000
	}
	if chainInfo.DasIndexUrl != "" {
		chainDefaults["node.batch-poster.max-size"] = 1_000_000
	}
	err = k.Load(confmap.Provider(chainDefaults, "."), nil)
	if err != nil {
		return err
	}
	return nil
}

type NodeConfigFetcher struct {
	*genericconf.LiveConfig[*NodeConfig]
}

func (f *NodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveConfig.Get().Node
}
