// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
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
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/arbutil"
	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/execution/gethexec"
	_ "github.com/offchainlabs/nitro/execution/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	legacystaker "github.com/offchainlabs/nitro/staker/legacy"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/iostat"
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
		return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
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
	nodeConfig, l2DevWallet, err := ParseNode(ctx, args)
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

	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := pathResolver(nodeConfig.Persistent.GlobalConfig)("jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
			log.Error("Failed to prepare jwt secret file", "err", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}
	err = genericconf.InitLog(nodeConfig.LogType, nodeConfig.LogLevel, &nodeConfig.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
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

	if nodeConfig.Execution.Sequencer.Enable != nodeConfig.Node.Sequencer {
		log.Error("consensus and execution must agree if sequencing is enabled or not", "Execution.Sequencer.Enable", nodeConfig.Execution.Sequencer.Enable, "Node.Sequencer", nodeConfig.Node.Sequencer)
	}
	if nodeConfig.Node.SeqCoordinator.Enable && !nodeConfig.Node.ParentChainReader.Enable {
		log.Error("Sequencer coordinator must be enabled with parent chain reader, try starting node with --parent-chain.connection.url")
		return 1
	}
	if nodeConfig.Execution.Sequencer.Enable && !nodeConfig.Execution.Sequencer.Dangerous.Timeboost.Enable && nodeConfig.Node.TransactionStreamer.TrackBlockMetadataFrom != 0 {
		log.Warn("Sequencer node's track-block-metadata-from is set but timeboost is not enabled")
	}

	var dataSigner signature.DataSignerFunc
	var l1TransactionOptsValidator *bind.TransactOpts
	var l1TransactionOptsBatchPoster *bind.TransactOpts
	// If sequencer and signing is enabled or batchposter is enabled without
	// external signing sequencer will need a key.
	sequencerNeedsKey := (nodeConfig.Node.Sequencer && !nodeConfig.Node.Feed.Output.DisableSigning) ||
		(nodeConfig.Node.BatchPoster.Enable && (nodeConfig.Node.BatchPoster.DataPoster.ExternalSigner.URL == "" || nodeConfig.Node.DataAvailability.Enable))
	validatorNeedsKey := nodeConfig.Node.Staker.OnlyCreateWalletContract ||
		(nodeConfig.Node.Staker.Enable && !strings.EqualFold(nodeConfig.Node.Staker.Strategy, "watchtower") && nodeConfig.Node.Staker.DataPoster.ExternalSigner.URL == "")

	defaultL1WalletConfig := conf.DefaultL1WalletConfig
	defaultL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	nodeConfig.Node.Staker.ParentChainWallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultValidatorL1WalletConfig := legacystaker.DefaultValidatorL1WalletConfig
	defaultValidatorL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

	nodeConfig.Node.BatchPoster.ParentChainWallet.ResolveDirectoryNames(nodeConfig.Persistent.Chain)
	defaultBatchPosterL1WalletConfig := arbnode.DefaultBatchPosterL1WalletConfig
	defaultBatchPosterL1WalletConfig.ResolveDirectoryNames(nodeConfig.Persistent.Chain)

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

	if nodeConfig.Node.Staker.Enable {
		if !nodeConfig.Node.ParentChainReader.Enable {
			flag.Usage()
			log.Crit("validator must have the parent chain reader enabled")
		}
		strategy, err := legacystaker.ParseStrategy(nodeConfig.Node.Staker.Strategy)
		if err != nil {
			log.Crit("couldn't parse staker strategy", "err", err)
		}
		if strategy != legacystaker.WatchtowerStrategy && !nodeConfig.Node.Staker.Dangerous.WithoutBlockValidator {
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
		nodeConfig, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})

	var rollupAddrs chaininfo.RollupAddresses
	var l1Client *ethclient.Client
	var l1Reader *headerreader.HeaderReader
	var blobReader daprovider.BlobReader
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

		rollupAddrs, err = chaininfo.GetRollupAddressesConfig(nodeConfig.Chain.ID, nodeConfig.Chain.Name, nodeConfig.Chain.InfoFiles, nodeConfig.Chain.InfoJson)
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
		deployInfo, err := chaininfo.GetRollupAddressesConfig(nodeConfig.Chain.ID, nodeConfig.Chain.Name, nodeConfig.Chain.InfoFiles, nodeConfig.Chain.InfoJson)
		if err != nil {
			log.Crit("error getting rollup addresses config", "err", err)
		}

		dataPoster, err := arbnode.DataposterOnlyUsedToCreateValidatorWalletContract(
			ctx,
			l1Reader,
			l1TransactionOptsValidator,
			&nodeConfig.Node.Staker.DataPoster,
			new(big.Int).SetUint64(nodeConfig.ParentChain.ID),
		)
		if err != nil {
			log.Crit("error creating data poster to create validator wallet contract", "err", err)
		}
		getExtraGas := func() uint64 { return nodeConfig.Node.Staker.ExtraGas }

		// #nosec G115
		addr, err := validatorwallet.GetValidatorWalletContract(ctx, deployInfo.ValidatorWalletCreator, int64(deployInfo.DeployedAt), l1Reader, true, dataPoster, getExtraGas)
		if err != nil {
			log.Crit("error creating validator wallet contract", "error", err, "address", l1TransactionOptsValidator.From.Hex())
		}
		fmt.Printf("Created validator smart contract wallet at %s, remove --node.validator.only-create-wallet-contract and restart\n", addr.String())
		return 0
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

	if nodeConfig.Metrics {
		go iostat.RegisterAndPopulateMetrics(ctx, 1, 5)
	}

	var deferFuncs []func()
	defer func() {
		for i := range deferFuncs {
			deferFuncs[i]()
		}
	}()

	// Check that node is compatible with on-chain WASM module root on startup and before any ArbOS upgrades take effect to prevent divergences
	if nodeConfig.Node.ParentChainReader.Enable && nodeConfig.Validation.Wasm.EnableWasmrootsCheck {
		err := checkWasmModuleRootCompatibility(ctx, nodeConfig.Validation.Wasm, l1Client, rollupAddrs)
		if err != nil {
			log.Warn("failed to check if node is compatible with on-chain WASM module root", "err", err)
		}
	}

	chainDb, l2BlockChain, err := openInitializeChainDb(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.Chain.ID), gethexec.DefaultCacheConfigFor(stack, &nodeConfig.Execution.Caching), &nodeConfig.Execution.StylusTarget, &nodeConfig.Persistent, l1Client, rollupAddrs)
	if l2BlockChain != nil {
		deferFuncs = append(deferFuncs, func() { l2BlockChain.Stop() })
	}
	deferFuncs = append(deferFuncs, func() { closeDb(chainDb, "chainDb") })
	if err != nil {
		flag.Usage()
		log.Error("error initializing database", "err", err)
		return 1
	}

	arbDb, err := stack.OpenDatabaseWithExtraOptions("arbitrumdata", 0, 0, "arbitrumdata/", false, nodeConfig.Persistent.Pebble.ExtraOptions("arbitrumdata"))
	deferFuncs = append(deferFuncs, func() { closeDb(arbDb, "arbDb") })
	if err != nil {
		log.Error("failed to open database", "err", err)
		log.Error("database is corrupt; delete it and try again", "database-directory", stack.InstanceDir())
		return 1
	}
	if err := dbutil.UnfinishedConversionCheck(arbDb); err != nil {
		log.Error("arbitrumdata unfinished conversion check error", "err", err)
		return 1
	}

	fatalErrChan := make(chan error, 10)

	if nodeConfig.BlocksReExecutor.Enable && l2BlockChain != nil {
		if !nodeConfig.Init.ThenQuit {
			log.Error("blocks-reexecutor cannot be enabled without --init.then-quit")
			return 1
		}
		blocksReExecutor, err := blocksreexecutor.New(&nodeConfig.BlocksReExecutor, l2BlockChain, chainDb, fatalErrChan)
		if err != nil {
			log.Error("error initializing blocksReExecutor", "err", err)
			return 1
		}
		if err := gethexec.PopulateStylusTargetCache(&nodeConfig.Execution.StylusTarget); err != nil {
			log.Error("error populating stylus target cache", "err", err)
			return 1
		}
		success := make(chan struct{})
		blocksReExecutor.Start(ctx, success)
		deferFuncs = append(deferFuncs, func() { blocksReExecutor.StopAndWait() })
		select {
		case err := <-fatalErrChan:
			log.Error("shutting down due to fatal error", "err", err)
			defer log.Error("shut down due to fatal error", "err", err)
			return 1
		case <-success:
		}
	}

	if nodeConfig.Init.ThenQuit && !nodeConfig.Init.IsReorgRequested() {
		return 0
	}

	chainInfo, err := chaininfo.ProcessChainInfo(nodeConfig.Chain.ID, nodeConfig.Chain.Name, nodeConfig.Chain.InfoFiles, nodeConfig.Chain.InfoJson)
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
		new(big.Int).SetUint64(nodeConfig.ParentChain.ID),
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
		if err := genericconf.InitLog(newCfg.LogType, newCfg.LogLevel, &newCfg.FileLogging, pathResolver(nodeConfig.Persistent.LogDir)); err != nil {
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

	// Before starting the node, wait until the transaction that deployed rollup is finalized
	if nodeConfig.EnsureRollupDeployment &&
		nodeConfig.Node.ParentChainReader.Enable &&
		rollupAddrs.DeployedAt > 0 {
		currentFinalized, err := l1Reader.LatestFinalizedBlockNr(ctx)
		if err != nil && errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
			log.Info("Finality not supported by parent chain, disabling the check to verify if rollup deployment tx was finalized", "err", err)
		} else {
			newHeaders, unsubscribe := l1Reader.Subscribe(false)
			retriesOnError := 10
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
			for currentFinalized < rollupAddrs.DeployedAt && retriesOnError > 0 {
				select {
				case <-newHeaders:
					if finalized, err := l1Reader.LatestFinalizedBlockNr(ctx); err != nil {
						if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
							log.Error("Finality support was removed from parent chain mid way, disabling the check to verify if the rollup deployment tx was finalized", "err", err)
							retriesOnError = 0 // Break out of for loop as well
							break
						}
						log.Error("Error getting latestFinalizedBlockNr from l1Reader", "err", err)
						retriesOnError--
					} else {
						currentFinalized = finalized
						log.Debug("Finalized block number updated", "finalized", finalized)
					}
				case <-ctx.Done():
					log.Error("Context done while checking if the rollup deployment tx was finalized")
					return 1
				case <-sigint:
					log.Info("shutting down because of sigint")
					return 0
				}
			}
			unsubscribe()
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

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	if err == nil && nodeConfig.Init.IsReorgRequested() {
		err = initReorg(nodeConfig.Init, chainInfo.ChainConfig, currentNode.InboxTracker)
		if err != nil {
			fatalErrChan <- fmt.Errorf("error reorging per init config: %w", err)
		} else if nodeConfig.Init.ThenQuit {
			return 0
		}
	}

	execNodeConfig := execNode.ConfigFetcher()
	if execNodeConfig.Sequencer.Enable && execNodeConfig.Sequencer.Dangerous.Timeboost.Enable {
		err := execNode.Sequencer.InitializeExpressLaneService(
			execNode.Backend.APIBackend(),
			execNode.FilterSystem,
			common.HexToAddress(execNodeConfig.Sequencer.Dangerous.Timeboost.AuctionContractAddress),
			common.HexToAddress(execNodeConfig.Sequencer.Dangerous.Timeboost.AuctioneerAddress),
			execNodeConfig.Sequencer.Dangerous.Timeboost.EarlySubmissionGrace,
		)
		if err != nil {
			log.Error("failed to create express lane service", "err", err)
		}
		execNode.Sequencer.StartExpressLaneService(ctx)
	}

	err = nil
	select {
	case err = <-fatalErrChan:
	case <-sigint:
		// If there was both a sigint and a fatal error, we want to log the fatal error
		select {
		case err = <-fatalErrChan:
		default:
			log.Info("shutting down because of sigint")
		}
	}

	if err != nil {
		log.Error("shutting down due to fatal error", "err", err)
		defer log.Error("shut down due to fatal error", "err", err)
		return 1
	}

	return 0
}

type NodeConfig struct {
	Conf                   genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Node                   arbnode.Config                  `koanf:"node" reload:"hot"`
	Execution              gethexec.Config                 `koanf:"execution" reload:"hot"`
	Validation             valnode.Config                  `koanf:"validation" reload:"hot"`
	ParentChain            conf.ParentChainConfig          `koanf:"parent-chain" reload:"hot"`
	Chain                  conf.L2Config                   `koanf:"chain"`
	LogLevel               string                          `koanf:"log-level" reload:"hot"`
	LogType                string                          `koanf:"log-type" reload:"hot"`
	FileLogging            genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	Persistent             conf.PersistentConfig           `koanf:"persistent"`
	HTTP                   genericconf.HTTPConfig          `koanf:"http"`
	WS                     genericconf.WSConfig            `koanf:"ws"`
	IPC                    genericconf.IPCConfig           `koanf:"ipc"`
	Auth                   genericconf.AuthRPCConfig       `koanf:"auth"`
	GraphQL                genericconf.GraphQLConfig       `koanf:"graphql"`
	Metrics                bool                            `koanf:"metrics"`
	MetricsServer          genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf                  bool                            `koanf:"pprof"`
	PprofCfg               genericconf.PProf               `koanf:"pprof-cfg"`
	Init                   conf.InitConfig                 `koanf:"init"`
	Rpc                    genericconf.RpcConfig           `koanf:"rpc"`
	BlocksReExecutor       blocksreexecutor.Config         `koanf:"blocks-reexecutor"`
	EnsureRollupDeployment bool                            `koanf:"ensure-rollup-deployment" reload:"hot"`
}

var NodeConfigDefault = NodeConfig{
	Conf:                   genericconf.ConfConfigDefault,
	Node:                   arbnode.ConfigDefault,
	Execution:              gethexec.ConfigDefault,
	Validation:             valnode.DefaultValidationConfig,
	ParentChain:            conf.L1ConfigDefault,
	Chain:                  conf.L2ConfigDefault,
	LogLevel:               "INFO",
	LogType:                "plaintext",
	FileLogging:            genericconf.DefaultFileLoggingConfig,
	Persistent:             conf.PersistentConfigDefault,
	HTTP:                   genericconf.HTTPConfigDefault,
	WS:                     genericconf.WSConfigDefault,
	IPC:                    genericconf.IPCConfigDefault,
	Auth:                   genericconf.AuthRPCConfigDefault,
	GraphQL:                genericconf.GraphQLConfigDefault,
	Metrics:                false,
	MetricsServer:          genericconf.MetricsServerConfigDefault,
	Init:                   conf.InitConfigDefault,
	Rpc:                    genericconf.DefaultRpcConfig,
	PProf:                  false,
	PprofCfg:               genericconf.PProfDefault,
	BlocksReExecutor:       blocksreexecutor.DefaultConfig,
	EnsureRollupDeployment: true,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	gethexec.ConfigAddOptions("execution", f)
	valnode.ValidationConfigAddOptions("validation", f)
	conf.L1ConfigAddOptions("parent-chain", f)
	conf.L2ConfigAddOptions("chain", f)
	f.String("log-level", NodeConfigDefault.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
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
	f.Bool("pprof", NodeConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	conf.InitConfigAddOptions("init", f)
	genericconf.RpcConfigAddOptions("rpc", f)
	blocksreexecutor.ConfigAddOptions("blocks-reexecutor", f)
	f.Bool("ensure-rollup-deployment", NodeConfigDefault.EnsureRollupDeployment, "before starting the node, wait until the transaction that deployed rollup is finalized")
}

func (c *NodeConfig) ResolveDirectoryNames() error {
	err := c.Persistent.ResolveDirectoryNames()
	if err != nil {
		return err
	}
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
	if c.Node.ValidatorRequired() && (c.Execution.Caching.StateScheme == rawdb.PathScheme) {
		return errors.New("path cannot be used as execution.caching.state-scheme when validator is required")
	}
	return c.Persistent.Validate()
}

func (c *NodeConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func ParseNode(ctx context.Context, args []string) (*NodeConfig, *genericconf.WalletConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, nil, err
	}

	l2ChainId := k.Int64("chain.id")
	l2ChainName := k.String("chain.name")
	l2ChainInfoFiles := k.Strings("chain.info-files")
	l2ChainInfoJson := k.String("chain.info-json")
	// #nosec G115
	err = applyChainParameters(k, uint64(l2ChainId), l2ChainName, l2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return nil, nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, nil, err
	}

	if err = das.FixKeysetCLIParsing("node.data-availability.rpc-aggregator.backends", k); err != nil {
		return nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := confighelpers.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, err
	}

	// Don't print wallet passwords
	if nodeConfig.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"node.batch-poster.parent-chain-wallet.password":    "",
			"node.batch-poster.parent-chain-wallet.private-key": "",
			"node.staker.parent-chain-wallet.password":          "",
			"node.staker.parent-chain-wallet.private-key":       "",
			"chain.dev-wallet.password":                         "",
			"chain.dev-wallet.private-key":                      "",
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if nodeConfig.Persistent.Chain == "" {
		return nil, nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, nil, err
	}

	// Don't pass around wallet contents with normal configuration
	l2DevWallet := nodeConfig.Chain.DevWallet
	nodeConfig.Chain.DevWallet = genericconf.WalletConfigDefault

	if nodeConfig.Execution.Caching.Archive {
		nodeConfig.Node.MessagePruner.Enable = false
	}

	if nodeConfig.Execution.Caching.Archive && nodeConfig.Execution.TxLookupLimit != 0 {
		log.Info("retaining ability to lookup full transaction history as archive mode is enabled")
		nodeConfig.Execution.TxLookupLimit = 0
	}

	err = nodeConfig.Validate()
	if err != nil {
		return nil, nil, err
	}
	return &nodeConfig, &l2DevWallet, nil
}

func applyChainParameters(k *koanf.Koanf, chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoJson string) error {
	chainInfo, err := chaininfo.ProcessChainInfo(chainId, chainName, l2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return err
	}
	var parentChainIsArbitrum bool
	if chainInfo.ParentChainIsArbitrum != nil {
		parentChainIsArbitrum = *chainInfo.ParentChainIsArbitrum
	} else {
		log.Warn("Chain info field parent-chain-is-arbitrum is missing, in the future this will be required", "chainId", chainInfo.ChainConfig.ChainID, "parentChainId", chainInfo.ParentChainId)
		_, err := chaininfo.ProcessChainInfo(chainInfo.ParentChainId, "", l2ChainInfoFiles, "")
		if err == nil {
			parentChainIsArbitrum = true
		}
	}
	chainDefaults := map[string]interface{}{
		"persistent.chain": chainInfo.ChainName,
		"chain.name":       chainInfo.ChainName,
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
	// 0 is default for any chain unless specified in the chain_defaults
	chainDefaults["node.transaction-streamer.track-block-metadata-from"] = chainInfo.TrackBlockMetadataFrom
	err = k.Load(confmap.Provider(chainDefaults, "."), nil)
	if err != nil {
		return err
	}
	return nil
}

func initReorg(initConfig conf.InitConfig, chainConfig *params.ChainConfig, inboxTracker *arbnode.InboxTracker) error {
	var batchCount uint64
	if initConfig.ReorgToBatch >= 0 {
		// #nosec G115
		batchCount = uint64(initConfig.ReorgToBatch) + 1
	} else {
		var messageIndex arbutil.MessageIndex
		if initConfig.ReorgToMessageBatch >= 0 {
			// #nosec G115
			messageIndex = arbutil.MessageIndex(initConfig.ReorgToMessageBatch)
		} else if initConfig.ReorgToBlockBatch > 0 {
			genesis := chainConfig.ArbitrumChainParams.GenesisBlockNum
			// #nosec G115
			blockNum := uint64(initConfig.ReorgToBlockBatch)
			if blockNum < genesis {
				return fmt.Errorf("ReorgToBlockBatch %d before genesis %d", blockNum, genesis)
			}
			messageIndex = arbutil.MessageIndex(blockNum - genesis)
		} else {
			log.Warn("Tried to do init reorg, but no init reorg options specified")
			return nil
		}
		// Reorg out the batch containing the next message
		var found bool
		var err error
		batchCount, found, err = inboxTracker.FindInboxBatchContainingMessage(messageIndex + 1)
		if err != nil {
			return err
		}
		if !found {
			log.Warn("init-reorg: no need to reorg, because message ahead of chain", "messageIndex", messageIndex)
			return nil
		}
	}
	return inboxTracker.ReorgBatchesTo(batchCount)
}

type NodeConfigFetcher struct {
	*genericconf.LiveConfig[*NodeConfig]
}

func (f *NodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveConfig.Get().Node
}

func checkWasmModuleRootCompatibility(ctx context.Context, wasmConfig valnode.WasmConfig, l1Client *ethclient.Client, rollupAddrs chaininfo.RollupAddresses) error {
	// Fetch current on-chain WASM module root
	rollupUserLogic, err := rollupgen.NewRollupUserLogic(rollupAddrs.Rollup, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create RollupUserLogic: %w", err)
	}
	moduleRoot, err := rollupUserLogic.WasmModuleRoot(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("failed to get on-chain WASM module root: %w", err)
	}
	if (moduleRoot == common.Hash{}) {
		return errors.New("on-chain WASM module root is zero")
	}
	// Check if the on-chain WASM module root belongs to the set of allowed module roots
	allowedWasmModuleRoots := wasmConfig.AllowedWasmModuleRoots
	if len(allowedWasmModuleRoots) > 0 {
		moduleRootMatched := false
		for _, root := range allowedWasmModuleRoots {
			bytes, err := hex.DecodeString(strings.TrimPrefix(root, "0x"))
			if err == nil {
				if common.HexToHash(root) == common.BytesToHash(bytes) {
					moduleRootMatched = true
					break
				}
				continue
			}
			locator, locatorErr := server_common.NewMachineLocator(root)
			if locatorErr != nil {
				log.Warn("allowed-wasm-module-roots: value not a hex nor valid path:", "value", root, "locatorErr", locatorErr, "decodeErr", err)
				continue
			}
			path := locator.GetMachinePath(moduleRoot)
			if _, err := os.Stat(path); err == nil {
				moduleRootMatched = true
				break
			}
		}
		if !moduleRootMatched {
			return errors.New("on-chain WASM module root did not match with any of the allowed WASM module roots")
		}
	} else {
		// If no allowed module roots were provided in config, check if we have a validator machine directory for the on-chain WASM module root
		locator, err := server_common.NewMachineLocator(wasmConfig.RootPath)
		if err != nil {
			return fmt.Errorf("failed to create machine locator: %w", err)
		}
		path := locator.GetMachinePath(moduleRoot)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("unable to find validator machine directory for the on-chain WASM module root: %w", err)
		}
	}
	return nil
}
