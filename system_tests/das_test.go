// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/base64"
	"errors"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func startLocalDASServer(
	t *testing.T,
	ctx context.Context,
	dataDir string,
	l1client *ethclient.Client,
	seqInboxAddress common.Address,
) (*http.Server, *blsSignatures.PublicKey, das.BackendConfig, *das.RestfulDasServer, string) {
	keyDir := t.TempDir()
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	config := das.DefaultDataAvailabilityConfig
	config.Enable = true
	config.Key = das.KeyConfig{KeyDir: keyDir}
	config.ParentChainNodeURL = "none"
	config.LocalFileStorage = das.DefaultLocalFileStorageConfig
	config.LocalFileStorage.Enable = true
	config.LocalFileStorage.DataDir = dataDir

	storageService, lifecycleManager, err := das.CreatePersistentStorageService(ctx, &config)
	_ = lifecycleManager // Caller should manage lifecycle if needed
	Require(t, err)
	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(seqInboxAddress, l1client)
	Require(t, err)
	daWriter, err := das.NewSignAfterStoreDASWriter(ctx, config, storageService)
	Require(t, err)
	signatureVerifier, err := das.NewSignatureVerifierWithSeqInboxCaller(seqInboxCaller, "")
	Require(t, err)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	rpcAddr := rpcLis.Addr().String()
	t.Logf("DAS RPC listener created at: %s", rpcAddr)

	rpcServer, err := das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, storageService, daWriter, storageService, signatureVerifier)
	Require(t, err)
	t.Logf("DAS RPC server started and listening on: %s", rpcAddr)

	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restAddr := restLis.Addr().String()
	t.Logf("DAS REST listener created at: %s", restAddr)

	restServer, err := das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, storageService, storageService)
	Require(t, err)
	t.Logf("DAS REST server started and listening on: %s", restAddr)

	beConfig := das.BackendConfig{
		URL:    "http://" + rpcAddr,
		Pubkey: blsPubToBase64(pubkey),
	}
	t.Logf("DAS backend config created with URL: %s", beConfig.URL)
	return rpcServer, pubkey, beConfig, restServer, "http://" + restAddr
}

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

func aggConfigForBackend(backendConfig das.BackendConfig) das.AggregatorConfig {
	rpcConfig := rpcclient.DefaultClientConfig
	rpcConfig.Timeout = 2 * time.Second // Short timeout for tests to fail fast
	return das.AggregatorConfig{
		Enable:        true,
		AssumedHonest: 1,
		Backends:      das.BackendConfigList{backendConfig},
		DASRPCClient: das.DASRPCClientConfig{
			ServerUrl:          backendConfig.URL,
			EnableChunkedStore: true,
			DataStream:         data_streaming.TestDataStreamerConfig(das.DefaultDataStreamRpcMethods),
			RPC:                rpcConfig,
		},
	}
}

func TestDASRekey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.BuildL1(t)

	// Setup DAS servers
	dasDataDir := t.TempDir()
	dasRpcServerA, pubkeyA, backendConfigA, _, restServerUrlA := startLocalDASServer(t, ctx, dasDataDir, builder.L1.Client, builder.addresses.SequencerInbox)
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	{
		authorizeDASKeyset(t, ctx, pubkeyA, builder.L1Info, builder.L1.Client)

		// Setup DAS config
		builder.nodeConfig.DataAvailability.Enable = true
		builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfigA)
		builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
		builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
		builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{restServerUrlA}
		builder.nodeConfig.DataAvailability.ParentChainNodeURL = "none"

		// Setup L2 chain
		builder.L2Info.GenerateAccount("User2")
		builder.BuildL2OnL1(t)

		// Setup second node
		l1NodeConfigB.BlockValidator.Enable = false
		l1NodeConfigB.DataAvailability.Enable = true
		l1NodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
		l1NodeConfigB.DataAvailability.RestAggregator.Enable = true
		l1NodeConfigB.DataAvailability.RestAggregator.Urls = []string{restServerUrlA}
		l1NodeConfigB.DataAvailability.ParentChainNodeURL = "none"
		nodeBParams := SecondNodeParams{
			nodeConfig: l1NodeConfigB,
			initData:   &builder.L2Info.ArbInitData,
		}
		l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
		checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

		builder.L2.cleanup()
		cleanupB()
	}

	err := dasRpcServerA.Shutdown(ctx)
	Require(t, err)
	dasRpcServerB, pubkeyB, backendConfigB, _, _ := startLocalDASServer(t, ctx, dasDataDir, builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() {
		err = dasRpcServerB.Shutdown(ctx)
		Require(t, err)
	}()
	authorizeDASKeyset(t, ctx, pubkeyB, builder.L1Info, builder.L1.Client)

	// Restart the node on the new keyset against the new DAS server running on the same disk as the first with new keys
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfigB)
	builder.l2StackConfig = testhelpers.CreateStackConfigForTest(builder.dataDir)
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	nodeBParams := SecondNodeParams{
		nodeConfig: l1NodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanup := builder.Build2ndNode(t, &nodeBParams)
	defer cleanup()
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(2e12), l2B.Client)
}

func TestDASComplexConfigAndRestMirror(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.BuildL1(t)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	keyDir, fileDataDir := t.TempDir(), t.TempDir()
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	serverConfig := das.DataAvailabilityConfig{
		Enable: true,

		LocalCache: das.TestCacheConfig,

		LocalFileStorage: das.LocalFileStorageConfig{
			Enable:  true,
			DataDir: fileDataDir,
		},

		Key: das.KeyConfig{
			KeyDir: keyDir,
		},

		RequestTimeout: 5 * time.Second,
		// L1NodeURL: normally we would have to set this but we are passing in the already constructed client and addresses to the factory
	}

	daReader, daWriter, signatureVerifier, daHealthChecker, lifecycleManager, err := das.CreateDAComponentsForDaserver(ctx, &serverConfig, l1Reader, &builder.addresses.SequencerInbox)
	Require(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	_, err = das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, daReader, daWriter, daHealthChecker, signatureVerifier)
	Require(t, err)
	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restServer, err := das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, daReader, daHealthChecker)
	Require(t, err)

	pubkeyA := pubkey
	authorizeDASKeyset(t, ctx, pubkeyA, builder.L1Info, builder.L1.Client)

	//
	builder.nodeConfig.DataAvailability = das.DataAvailabilityConfig{
		Enable: true,

		// AggregatorConfig set up below
		RequestTimeout: 5 * time.Second,
	}
	beConfigA := das.BackendConfig{
		URL:    "http://" + rpcLis.Addr().String(),
		Pubkey: blsPubToBase64(pubkey),
	}
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(beConfigA)
	builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
	builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{"http://" + restLis.Addr().String()}
	builder.nodeConfig.DataAvailability.ParentChainNodeURL = "none"

	// Setup L2 chain
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// Create node to sync from chain
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	l1NodeConfigB.DataAvailability = das.DataAvailabilityConfig{
		Enable: true,

		// AggregatorConfig set up below

		ParentChainNodeURL: "none",
		RequestTimeout:     5 * time.Second,
	}

	l1NodeConfigB.BlockValidator.Enable = false
	l1NodeConfigB.DataAvailability.Enable = true
	l1NodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	l1NodeConfigB.DataAvailability.RestAggregator.Enable = true
	l1NodeConfigB.DataAvailability.RestAggregator.Urls = []string{"http://" + restLis.Addr().String()}
	l1NodeConfigB.DataAvailability.ParentChainNodeURL = "none"
	nodeBParams := SecondNodeParams{
		nodeConfig: l1NodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

	err = restServer.Shutdown()
	Require(t, err)
}

func TestDASBatchPosterFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.BuildL1(t)
	l1client := builder.L1.Client
	l1info := builder.L1Info

	// Setup DAS server
	dasDataDir := t.TempDir()
	dasRpcServer, pubkey, backendConfig, _, restServerUrl := startLocalDASServer(
		t, ctx, dasDataDir, l1client, builder.addresses.SequencerInbox)
	authorizeDASKeyset(t, ctx, pubkey, l1info, l1client)

	// Setup sequence/batch-poster L2 node
	builder.nodeConfig.DataAvailability.Enable = true
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
	builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{restServerUrl}
	builder.nodeConfig.DataAvailability.ParentChainNodeURL = "none"
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = true // Disable DAS fallback
	builder.nodeConfig.BatchPoster.ErrorDelay = time.Millisecond * 250       // Increase error delay because we expect errors
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()
	l2client := builder.L2.Client
	l2info := builder.L2Info

	// Setup secondary L2 node
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false
	nodeConfigB.DataAvailability.Enable = true
	nodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	nodeConfigB.DataAvailability.RestAggregator.Enable = true
	nodeConfigB.DataAvailability.RestAggregator.Urls = []string{restServerUrl}
	nodeConfigB.DataAvailability.ParentChainNodeURL = "none"
	nodeBParams := SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &l2info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	// Check batch posting using the DAS
	checkBatchPosting(t, ctx, l1client, l2client, l1info, l2info, big.NewInt(1e12), l2B.Client)

	// Shutdown the DAS
	err := dasRpcServer.Shutdown(ctx)
	Require(t, err)

	// Send 2nd transaction and check it doesn't arrive on second node
	tx, _ := TransferBalanceTo(t, "Owner", l2info.GetAddress("User2"), big.NewInt(1e12), l2info, l2client, ctx)
	_, err = WaitForTx(ctx, l2B.Client, tx.Hash(), time.Second*3)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		Fatal(t, "expected context-deadline exceeded error, but got:", err)
	}

	// Enable the DAP fallback and check the transaction on the second node.
	// (We don't need to restart the node because of the hot-reload.)
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err = WaitForTx(ctx, l2B.Client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := l2B.Client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(2e12)) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}

	// Send another transaction with fallback on
	checkBatchPosting(t, ctx, l1client, l2client, l1info, l2info, big.NewInt(3e12), l2B.Client)
}
