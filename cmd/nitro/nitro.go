// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/pflag"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/live"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/graphql"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/nitro/config"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/execution/gethexec"
	_ "github.com/offchainlabs/nitro/execution/nodeinterface"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	legacystaker "github.com/offchainlabs/nitro/staker/legacy"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	nitroutil "github.com/offchainlabs/nitro/util"
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

// Returns the exit code
func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	nodeConfig, l2DevWallet, err := config.ParseNode(ctx, args)
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
	log.Info("Resources detected", "GOMAXPROCS", nitroutil.GoMaxProcs())

	if nodeConfig.Node.Dangerous.NoL1Listener {
		nodeConfig.Node.ParentChainReader.Enable = false
		nodeConfig.Node.BatchPoster.Enable = false
		nodeConfig.Node.DelayedSequencer.Enable = false
		nodeConfig.Init.ValidateGenesisAssertion = false
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
	if nodeConfig.Execution.Sequencer.Enable && !nodeConfig.Execution.Sequencer.Timeboost.Enable && nodeConfig.Node.TransactionStreamer.TrackBlockMetadataFrom != 0 {
		log.Warn("Sequencer node's track-block-metadata-from is set but timeboost is not enabled")
	}

	var dataSigner signature.DataSignerFunc
	var l1TransactionOptsValidator *bind.TransactOpts
	var l1TransactionOptsBatchPoster *bind.TransactOpts
	// If sequencer and signing is enabled or batchposter is enabled without
	// external signing sequencer will need a key.
	sequencerNeedsKey := (nodeConfig.Node.Sequencer && nodeConfig.Node.Feed.Output.Signed) ||
		(nodeConfig.Node.BatchPoster.Enable && (nodeConfig.Node.BatchPoster.DataPoster.ExternalSigner.URL == "" || nodeConfig.Node.DA.AnyTrust.Enable))
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
			pflag.Usage()
			log.Crit("error opening Batch poster parent chain wallet", "path", nodeConfig.Node.BatchPoster.ParentChainWallet.Pathname, "account", nodeConfig.Node.BatchPoster.ParentChainWallet.Account, "err", err)
		}
		if nodeConfig.Node.BatchPoster.ParentChainWallet.OnlyCreateKey {
			return 0
		}
	}
	if validatorNeedsKey || nodeConfig.Node.Staker.ParentChainWallet.OnlyCreateKey {
		l1TransactionOptsValidator, _, err = util.OpenWallet("l1-validator", &nodeConfig.Node.Staker.ParentChainWallet, new(big.Int).SetUint64(nodeConfig.ParentChain.ID))
		if err != nil {
			pflag.Usage()
			log.Crit("error opening Validator parent chain wallet", "path", nodeConfig.Node.Staker.ParentChainWallet.Pathname, "account", nodeConfig.Node.Staker.ParentChainWallet.Account, "err", err)
		}
		if nodeConfig.Node.Staker.ParentChainWallet.OnlyCreateKey {
			return 0
		}
	}

	if nodeConfig.Node.Staker.Enable {
		if !nodeConfig.Node.ParentChainReader.Enable {
			pflag.Usage()
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
	if nodeConfig.Execution.Caching.StateHistory == gethexec.UninitializedStateHistory {
		if nodeConfig.Execution.Caching.Archive {
			nodeConfig.Execution.Caching.StateHistory = gethexec.DefaultArchiveNodeStateHistory
		} else {
			nodeConfig.Execution.Caching.StateHistory = gethexec.GetStateHistory(gethexec.DefaultSequencerConfig.MaxBlockSpeed)
		}
	}
	liveNodeConfig := genericconf.NewLiveConfig[*config.NodeConfig](args, nodeConfig, func(ctx context.Context, args []string) (*config.NodeConfig, error) {
		nodeConfig, _, err := config.ParseNode(ctx, args)
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
				pflag.Usage()
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
			pflag.Usage()
			log.Crit("--node.validator.only-create-wallet-contract requires --node.validator.use-smart-contract-wallet")
		}
		if l1Reader == nil {
			pflag.Usage()
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
		pflag.Usage()
		log.Crit("Failed to start resource management module", "err", err)
	}

	var sameProcessValidationNodeEnabled bool
	if nodeConfig.Node.BlockValidator.Enable && (nodeConfig.Node.BlockValidator.ValidationServerConfigs[0].URL == "self" || nodeConfig.Node.BlockValidator.ValidationServerConfigs[0].URL == "self-auth") {
		sameProcessValidationNodeEnabled = true
		valnode.EnsureValidationExposedViaAuthRPC(&stackConf)
	}
	stack, err := node.New(&stackConf)
	if err != nil {
		pflag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}
	{
		devAddr, err := addUnlockWallet(stack.AccountManager(), l2DevWallet)
		if err != nil {
			pflag.Usage()
			log.Crit("error opening L2 dev wallet", "err", err)
		}
		if devAddr != (common.Address{}) {
			nodeConfig.Init.DevInitAddress = devAddr.String()
		}
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

	traceConfig := nodeConfig.Execution.VmTrace
	var tracer *tracing.Hooks
	if traceConfig.TracerName != "" {
		tracer, err = tracers.LiveDirectory.New(traceConfig.TracerName, json.RawMessage(traceConfig.JSONConfig))
		if err != nil {
			log.Error("custom tracer error:", "name", traceConfig.TracerName, "err", err)
			return 1
		}
		log.Info("enabling custom tracer", "name", traceConfig.TracerName)
	}

	if err := gethexec.PopulateStylusTargetCache(&nodeConfig.Execution.StylusTarget); err != nil {
		log.Error("error populating stylus target cache", "err", err)
		return 1
	}

	executionDB, l2BlockChain, err := config.OpenInitializeExecutionDB(ctx, stack, nodeConfig, new(big.Int).SetUint64(nodeConfig.Chain.ID), gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching), tracer, &nodeConfig.Persistent, l1Client, rollupAddrs)
	if l2BlockChain != nil {
		deferFuncs = append(deferFuncs, func() { l2BlockChain.Stop() })
	}
	deferFuncs = append(deferFuncs, func() { closeDb(executionDB, "executionDB") })
	if err != nil {
		pflag.Usage()
		log.Error("error initializing database", "err", err)
		return 1
	}

	initDataReader, _, _, err := config.GetInit(nodeConfig, executionDB)
	if err != nil {
		log.Error("error fetching initDataReader a second time", "err", err)
		return 1
	}
	err = config.GetAndValidateGenesisAssertion(ctx, nodeConfig, l2BlockChain, initDataReader, &rollupAddrs, l1Client)
	if err != nil {
		log.Error("error trying to validate genesis assertion", "err", err)
		return 1
	}

	consensusDB, err := config.OpenConsensusDB(stack, nodeConfig)
	if consensusDB != nil {
		deferFuncs = append(deferFuncs, func() { closeDb(consensusDB, "consensusDB") })
	}
	if err != nil {
		return 1
	}

	if nodeConfig.BlocksReExecutor.Enable && l2BlockChain != nil {
		if !nodeConfig.Init.ThenQuit {
			log.Error("blocks-reexecutor cannot be enabled without --init.then-quit")
			return 1
		}
		blocksReExecutor, err := blocksreexecutor.New(&nodeConfig.BlocksReExecutor, l2BlockChain, executionDB)
		if err != nil {
			log.Error("error initializing blocksReExecutor", "err", err)
			return 1
		}

		blocksReExecutor.Start(ctx)
		deferFuncs = append(deferFuncs, func() { blocksReExecutor.StopAndWait() })
		err = blocksReExecutor.WaitForReExecution(ctx)
		if err != nil {
			defer log.Error("shut down due to fatal error", "err", err)
			return 1
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
	if err := config.ValidateBlockChain(l2BlockChain, chainInfo.ChainConfig); err != nil {
		log.Error("user provided chain config is not compatible with onchain chain config", "err", err)
		return 1
	}

	if l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee != nodeConfig.Node.DA.AnyTrust.Enable {
		pflag.Usage()
		log.Error(fmt.Sprintf("AnyTrust DA usage for this chain is set to %v but --node.da.anytrust.enable is set to %v", l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee, nodeConfig.Node.DA.AnyTrust.Enable))
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

	var wasmModuleRoot common.Hash
	if liveNodeConfig.Get().Node.ValidatorRequired() {
		locator, err := server_common.NewMachineLocator(liveNodeConfig.Get().Validation.Wasm.RootPath)
		if err != nil {
			log.Error("failed to create machine locator: %w", err)
		}
		wasmModuleRoot = locator.LatestWasmModuleRoot()
	}

	var execNode *gethexec.ExecutionNode
	var consensusNode *arbnode.Node
	if nodeConfig.Node.ExecutionRPCClient.URL == "" || nodeConfig.Node.ExecutionRPCClient.URL == "self" || nodeConfig.Node.ExecutionRPCClient.URL == "self-auth" {
		execNode, err = gethexec.CreateExecutionNode(
			ctx,
			stack,
			executionDB,
			l2BlockChain,
			l1Client,
			&config.ExecutionNodeConfigFetcher{LiveConfig: liveNodeConfig},
			new(big.Int).SetUint64(nodeConfig.ParentChain.ID),
			liveNodeConfig.Get().Node.TransactionStreamer.SyncTillBlock,
		)
		if err != nil {
			log.Error("failed to create execution node", "err", err)
			return 1
		}
		consensusNode, err = arbnode.CreateConsensusNodeConnectedWithFullExecutionClient(
			ctx,
			stack,
			execNode,
			consensusDB,
			&config.ConsensusNodeConfigFetcher{LiveConfig: liveNodeConfig},
			l2BlockChain.Config(),
			l1Client,
			&rollupAddrs,
			l1TransactionOptsValidator,
			l1TransactionOptsBatchPoster,
			dataSigner,
			fatalErrChan,
			new(big.Int).SetUint64(nodeConfig.ParentChain.ID),
			blobReader,
			wasmModuleRoot,
		)
		if err != nil {
			log.Error("failed to create consensus node", "err", err)
			return 1
		}
	} else {
		consensusNode, err = arbnode.CreateConsensusNodeConnectedWithSimpleExecutionClient(
			ctx,
			stack,
			nil,
			consensusDB,
			&config.ConsensusNodeConfigFetcher{LiveConfig: liveNodeConfig},
			l2BlockChain.Config(),
			l1Client,
			&rollupAddrs,
			l1TransactionOptsValidator,
			l1TransactionOptsBatchPoster,
			dataSigner,
			fatalErrChan,
			new(big.Int).SetUint64(nodeConfig.ParentChain.ID),
			blobReader,
			wasmModuleRoot,
		)
		if err != nil {
			log.Error("failed to create consensus node", "err", err)
			return 1
		}
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
		} else if !headerreader.IsExecutionReverted(err) {
			log.Error("error fetching MaxDataSize from sequencer inbox", "err", err)
			return 1
		}
	}
	// If batchPoster is enabled, validate MaxCalldataBatchSize to be at least 10kB below the sequencer inbox's maxDataSize if AnyTrust DA is not enabled.
	// The 10kB gap is because its possible for the batch poster to exceed its MaxCalldataBatchSize limit and produce batches of slightly larger size.
	if nodeConfig.Node.BatchPoster.Enable && !nodeConfig.Node.DA.AnyTrust.Enable {
		if nodeConfig.Node.BatchPoster.MaxCalldataBatchSize > seqInboxMaxDataSize-10000 {
			log.Error("batchPoster's MaxCalldataBatchSize is too large")
			return 1
		}
	}

	if nodeConfig.Execution.Sequencer.Enable {
		// Validate MaxTxDataSize to be at least 5kB below the batch poster's MaxCalldataBatchSize to allow space for headers and such.
		if nodeConfig.Execution.Sequencer.MaxTxDataSize > nodeConfig.Node.BatchPoster.MaxCalldataBatchSize-5000 {
			log.Error("sequencer's MaxTxDataSize too large compared to the batchPoster's MaxCalldataBatchSize")
			return 1
		}
		// Since the batchposter's MaxCalldataBatchSize must be at least 10kB below the sequencer inbox's maxDataSize, then MaxTxDataSize must also be 15kB below the sequencer inbox's maxDataSize.
		if nodeConfig.Execution.Sequencer.MaxTxDataSize > seqInboxMaxDataSize-15000 && !nodeConfig.Execution.Sequencer.Dangerous.DisableSeqInboxMaxDataSizeCheck {
			log.Error("sequencer's MaxTxDataSize too large compared to the sequencer inbox's MaxDataSize")
			return 1
		}
	}

	liveNodeConfig.SetOnReloadHook(func(oldCfg *config.NodeConfig, newCfg *config.NodeConfig) error {
		if err := genericconf.InitLog(newCfg.LogType, newCfg.LogLevel, &newCfg.FileLogging, pathResolver(nodeConfig.Persistent.LogDir)); err != nil {
			return fmt.Errorf("failed to re-init logging: %w", err)
		}
		return consensusNode.OnConfigReload(&oldCfg.Node, &newCfg.Node)
	})

	if nodeConfig.Node.Dangerous.NoL1Listener && nodeConfig.Init.DevInit {
		// If we don't have any messages, we're not connected to the L1, and we're using a dev init,
		// we should create our own fake init message.
		count, err := consensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			log.Warn("Getmessagecount failed. Assuming new database", "err", err)
			count = 0
		}
		if count == 0 {
			err = consensusNode.TxStreamer.AddFakeInitMessage()
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
			if !l1Reader.Started() {
				l1Reader.Start(ctx)
			}
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
			l1Reader.StopAndWait()
		}
	}

	gqlConf := nodeConfig.GraphQL
	if execNode != nil && gqlConf.Enable {
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
			defer valNode.Stop()
		}
	}
	if err == nil {
		cleanup, err := execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, stack, execNode, consensusNode)
		if err != nil {
			log.Error("Error initializing and starting execution and consensus", "err", err)
			return 1
		}
		// remove previous deferFuncs, StopAndWait closes database and blockchain.
		deferFuncs = []func(){cleanup}
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	if err == nil && nodeConfig.Init.IsReorgRequested() {
		err = config.InitReorg(nodeConfig.Init, chainInfo.ChainConfig, consensusNode.InboxTracker)
		if err != nil {
			fatalErrChan <- fmt.Errorf("error reorging per init config: %w", err)
		} else if nodeConfig.Init.ThenQuit {
			return 0
		}
	}

	if execNode != nil {
		err = execNode.InitializeTimeboost(ctx, chainInfo.ChainConfig)
		if err != nil {
			fatalErrChan <- fmt.Errorf("error initializing timeboost: %w", err)
		}
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
				if common.BytesToHash(bytes) == moduleRoot {
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
