// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/base64"
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
	"github.com/offchainlabs/nitro/daprovider/anytrust"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func startLocalAnyTrustServer(
	t *testing.T,
	ctx context.Context,
	dataDir string,
	l1client *ethclient.Client,
	seqInboxAddress common.Address,
) (*http.Server, *blsSignatures.PublicKey, anytrust.BackendConfig, *anytrust.RestfulServer, string) {
	keyDir := t.TempDir()
	pubkey, _, err := anytrust.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	config := anytrust.DefaultConfig
	config.Enable = true
	config.Key = anytrust.KeyConfig{KeyDir: keyDir}
	config.LocalFileStorage = anytrust.DefaultLocalFileStorageConfig
	config.LocalFileStorage.Enable = true
	config.LocalFileStorage.DataDir = dataDir

	storageService, lifecycleManager, err := anytrust.CreatePersistentStorageService(ctx, &config)
	_ = lifecycleManager // Caller should manage lifecycle if needed
	Require(t, err)
	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(seqInboxAddress, l1client)
	Require(t, err)
	daWriter, err := anytrust.NewSignAfterStoreWriter(ctx, config, storageService)
	Require(t, err)
	signatureVerifier, err := anytrust.NewSignatureVerifierWithSeqInboxCaller(seqInboxCaller, "")
	Require(t, err)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	rpcAddr := rpcLis.Addr().String()
	t.Logf("AnyTrust RPC listener created at: %s", rpcAddr)

	rpcServer, err := anytrust.StartRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, storageService, daWriter, storageService, signatureVerifier)
	Require(t, err)
	t.Logf("AnyTrust RPC server started and listening on: %s", rpcAddr)

	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restAddr := restLis.Addr().String()
	t.Logf("AnyTrust REST listener created at: %s", restAddr)

	restServer, err := anytrust.NewRestfulServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, storageService, storageService)
	Require(t, err)
	t.Logf("AnyTrust REST server started and listening on: %s", restAddr)

	beConfig := anytrust.BackendConfig{
		URL:    "http://" + rpcAddr,
		Pubkey: blsPubToBase64(pubkey),
	}
	t.Logf("AnyTrust backend config created with URL: %s", beConfig.URL)
	return rpcServer, pubkey, beConfig, restServer, "http://" + restAddr
}

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

func aggConfigForBackend(backendConfig anytrust.BackendConfig) anytrust.AggregatorConfig {
	rpcConfig := rpcclient.DefaultClientConfig
	rpcConfig.Timeout = 2 * time.Second // Short timeout for tests to fail fast
	rpcConfig.URL = backendConfig.URL
	return anytrust.AggregatorConfig{
		Enable:        true,
		AssumedHonest: 1,
		Backends:      anytrust.BackendConfigList{backendConfig},
		RPCClient: anytrust.RPCClientConfig{
			EnableChunkedStore: true,
			DataStream:         data_streaming.TestDataStreamerConfig(anytrust.DefaultDataStreamRpcMethods),
			RPC:                rpcConfig,
		},
	}
}

func TestAnyTrustRekey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.BuildL1(t)

	// Setup AnyTrust servers
	anyTrustDataDir := t.TempDir()
	anyTrustRpcServerA, pubkeyA, backendConfigA, _, restServerUrlA := startLocalAnyTrustServer(t, ctx, anyTrustDataDir, builder.L1.Client, builder.addresses.SequencerInbox)
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	{
		authorizeAnyTrustKeyset(t, ctx, pubkeyA, builder.L1Info, builder.L1.Client)

		// Setup AnyTrust config
		builder.nodeConfig.DA.AnyTrust.Enable = true
		builder.nodeConfig.DA.AnyTrust.RPCAggregator = aggConfigForBackend(backendConfigA)
		builder.nodeConfig.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
		builder.nodeConfig.DA.AnyTrust.RestAggregator.Enable = true
		builder.nodeConfig.DA.AnyTrust.RestAggregator.Urls = []string{restServerUrlA}

		// Setup L2 chain
		builder.L2Info.GenerateAccount("User2")
		builder.BuildL2OnL1(t)

		// Setup second node
		l1NodeConfigB.BlockValidator.Enable = false
		l1NodeConfigB.DA.AnyTrust.Enable = true
		l1NodeConfigB.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
		l1NodeConfigB.DA.AnyTrust.RestAggregator.Enable = true
		l1NodeConfigB.DA.AnyTrust.RestAggregator.Urls = []string{restServerUrlA}
		nodeBParams := SecondNodeParams{
			nodeConfig: l1NodeConfigB,
			initData:   &builder.L2Info.ArbInitData,
		}
		l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
		checkBatchPosting(t, ctx, builder, l2B.Client)

		builder.L2.cleanup()
		cleanupB()
	}

	err := anyTrustRpcServerA.Shutdown(ctx)
	Require(t, err)
	anyTrustRpcServerB, pubkeyB, backendConfigB, _, _ := startLocalAnyTrustServer(t, ctx, anyTrustDataDir, builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() {
		err = anyTrustRpcServerB.Shutdown(ctx)
		Require(t, err)
	}()
	authorizeAnyTrustKeyset(t, ctx, pubkeyB, builder.L1Info, builder.L1.Client)

	// Restart the node on the new keyset against the new AnyTrust server running on the same disk as the first with new keys
	builder.nodeConfig.DA.AnyTrust.RPCAggregator = aggConfigForBackend(backendConfigB)
	builder.l2StackConfig = testhelpers.CreateStackConfigForTest(builder.dataDir)
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	nodeBParams := SecondNodeParams{
		nodeConfig: l1NodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanup := builder.Build2ndNode(t, &nodeBParams)
	defer cleanup()
	checkBatchPosting(t, ctx, builder, l2B.Client)
}

func TestAnyTrustComplexConfigAndRestMirror(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestAnyTrustChainConfig()
	builder.BuildL1(t)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	keyDir, fileDataDir := t.TempDir(), t.TempDir()
	pubkey, _, err := anytrust.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	serverConfig := anytrust.DefaultConfig
	serverConfig.Enable = true
	serverConfig.LocalCache = anytrust.TestCacheConfig
	serverConfig.LocalFileStorage.Enable = true
	serverConfig.LocalFileStorage.DataDir = fileDataDir
	serverConfig.LocalFileStorage.MaxRetention = time.Hour * 24 * 30
	serverConfig.Key.KeyDir = keyDir
	// L1NodeURL: normally we would have to set this but we are passing in the already constructed client and addresses to the factory

	daReader, daWriter, signatureVerifier, daHealthChecker, lifecycleManager, err := anytrust.CreateDAComponentsForAnyTrustServer(ctx, &serverConfig, l1Reader, &builder.addresses.SequencerInbox)
	Require(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	_, err = anytrust.StartRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, daReader, daWriter, daHealthChecker, signatureVerifier)
	Require(t, err)
	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restServer, err := anytrust.NewRestfulServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, daReader, daHealthChecker)
	Require(t, err)

	pubkeyA := pubkey
	authorizeAnyTrustKeyset(t, ctx, pubkeyA, builder.L1Info, builder.L1.Client)

	//
	builder.nodeConfig.DA.AnyTrust = anytrust.DefaultConfig
	builder.nodeConfig.DA.AnyTrust.Enable = true
	// AggregatorConfig set up below
	beConfigA := anytrust.BackendConfig{
		URL:    "http://" + rpcLis.Addr().String(),
		Pubkey: blsPubToBase64(pubkey),
	}
	builder.nodeConfig.DA.AnyTrust.RPCAggregator = aggConfigForBackend(beConfigA)
	builder.nodeConfig.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Enable = true
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Urls = []string{"http://" + restLis.Addr().String()}

	// Setup L2 chain
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// Create node to sync from chain
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	l1NodeConfigB.DA.AnyTrust = anytrust.DefaultConfig
	l1NodeConfigB.DA.AnyTrust.Enable = true
	// AggregatorConfig set up below
	l1NodeConfigB.BlockValidator.Enable = false
	l1NodeConfigB.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	l1NodeConfigB.DA.AnyTrust.RestAggregator.Enable = true
	l1NodeConfigB.DA.AnyTrust.RestAggregator.Urls = []string{"http://" + restLis.Addr().String()}
	nodeBParams := SecondNodeParams{
		nodeConfig: l1NodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	checkBatchPosting(t, ctx, builder, l2B.Client)

	err = restServer.Shutdown()
	Require(t, err)
}
