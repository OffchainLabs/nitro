// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/eigenda"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
)

const (
	proxyURL = "http://127.0.0.1:4242"
)

func TestEigenDAProxyBatchPosting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.BuildL1(t)
	// Setup DAS servers
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()

	{

		// Setup DAS config
		builder.nodeConfig.EigenDA.Enable = true
		builder.nodeConfig.EigenDA.Rpc = proxyURL

		// Setup L2 chain
		builder.L2Info.GenerateAccount("User2")
		builder.BuildL2OnL1(t)

		// Setup second node
		l1NodeConfigB.BlockValidator.Enable = false
		l1NodeConfigB.EigenDA.Enable = true
		l1NodeConfigB.EigenDA.Rpc = proxyURL

		nodeBParams := SecondNodeParams{
			nodeConfig: l1NodeConfigB,
			initData:   &builder.L2Info.ArbInitData,
		}
		l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
		checkEigenDABatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

		builder.L2.cleanup()
		cleanupB()
	}
}

func TestEigenDAProxyFailOverToETHDA(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.BuildL1(t)
	// Setup DAS servers
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()

	{

		// Setup DAS config
		builder.nodeConfig.EigenDA.Enable = true
		builder.nodeConfig.EigenDA.Rpc = proxyURL
		builder.nodeConfig.BatchPoster.EnableEigenDAFailover = true

		// Setup L2 chain
		builder.L2Info.GenerateAccount("User2")
		builder.BuildL2OnL1(t)

		// Setup second node
		l1NodeConfigB.BlockValidator.Enable = false
		l1NodeConfigB.EigenDA.Enable = true
		l1NodeConfigB.EigenDA.Rpc = proxyURL
		l1NodeConfigB.BatchPoster.EnableEigenDAFailover = true

		nodeBParams := SecondNodeParams{
			nodeConfig: l1NodeConfigB,
			initData:   &builder.L2Info.ArbInitData,
		}
		l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)

		// 1 - Ensure that batches can be submitted and read via EigenDA batch posting
		checkEigenDABatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

		// 2 - Cause EigenDA to fail and ensure that the system falls back to anytrust in the presence of 503 eigenda-proxy errors
		builder.L2.ConsensusNode.BatchPoster.SetEigenDAClientMock()
		checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(2000000000000), l2B.Client)
		// 3 - Emulate EigenDA becoming healthy again and ensure that the system starts using it for DA
		eigenWriter, _ := eigenda.NewEigenDA(&eigenda.EigenDAConfig{
			Enable: true,
			Rpc:    proxyURL,
		})

		builder.L2.ConsensusNode.BatchPoster.SetEigenDAWriter(eigenWriter)

		checkEigenDABatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(3000000000000), l2B.Client)
		builder.L2.cleanup()
		cleanupB()
	}
}

func TestEigenDAProxyFailOverToAnyTrust(t *testing.T) {
	initTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = params.ArbitrumDevTestDASChainConfig()
	builder.BuildL1(t)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	keyDir, fileDataDir, dbDataDir := t.TempDir(), t.TempDir(), t.TempDir()
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	dbConfig := das.DefaultLocalDBStorageConfig
	dbConfig.Enable = true
	dbConfig.DataDir = dbDataDir

	serverConfig := das.DataAvailabilityConfig{
		Enable: true,

		LocalCache: das.TestCacheConfig,

		LocalFileStorage: das.LocalFileStorageConfig{
			Enable:  true,
			DataDir: fileDataDir,
		},
		LocalDBStorage: dbConfig,

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

	// Set AnyTrust params into L2 node config
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

	// set EigenDA params into L2 sequencer config
	builder.nodeConfig.EigenDA.Enable = true
	builder.nodeConfig.EigenDA.Rpc = proxyURL
	builder.nodeConfig.BatchPoster.EnableEigenDAFailover = true

	// Setup L2 chain
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)

	defer cleanup()

	// Create node to sync from chain
	childNodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest().WithEigenDATestConfigParams()
	childNodeConfigB.DataAvailability = das.DataAvailabilityConfig{
		Enable: true,

		// AggregatorConfig set up below

		ParentChainNodeURL: "none",
		RequestTimeout:     5 * time.Second,
	}

	childNodeConfigB.BlockValidator.Enable = false
	childNodeConfigB.DataAvailability.Enable = true
	childNodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	childNodeConfigB.DataAvailability.RestAggregator.Enable = true
	childNodeConfigB.DataAvailability.RestAggregator.Urls = []string{"http://" + restLis.Addr().String()}
	childNodeConfigB.DataAvailability.ParentChainNodeURL = "none"
	childNodeConfigB.EigenDA.Enable = true
	childNodeConfigB.EigenDA.Rpc = proxyURL
	childNodeConfigB.BatchPoster.EnableEigenDAFailover = true
	childNodeConfigB.BatchPoster.CheckBatchCorrectness = true

	nodeBParams := SecondNodeParams{
		nodeConfig: childNodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	// 1 - Ensure that batches can be submitted and read via EigenDA batch posting
	checkEigenDABatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)
	// 2 - Cause EigenDA to fail and ensure that the system falls back to anytrust in the presence of 503 eigenda-proxy errors
	builder.L2.ConsensusNode.BatchPoster.SetEigenDAClientMock()
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12*2), l2B.Client)
	// 3 - Emulate EigenDA becoming healthy again and ensure that the system starts using it for DA
	eigenWriter, err := eigenda.NewEigenDA(&eigenda.EigenDAConfig{
		Enable: true,
		Rpc:    proxyURL,
	})
	Require(t, err)

	builder.L2.ConsensusNode.BatchPoster.SetEigenDAWriter(eigenWriter)
	checkEigenDABatchPosting(t, ctx, builder.L1.Client, builder.L2.Client, builder.L1Info, builder.L2Info, big.NewInt(1e12*3), l2B.Client)

	err = restServer.Shutdown()
	Require(t, err)
}

func checkEigenDABatchPosting(t *testing.T, ctx context.Context, l1client, l2clientA *ethclient.Client, l1info, l2info info, expectedBalance *big.Int, l2ClientsToCheck ...*ethclient.Client) {
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := l2clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 100; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	for _, client := range l2ClientsToCheck {
		_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*100)
		Require(t, err)

		l2balance, err := client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
		Require(t, err)

		if l2balance.Cmp(expectedBalance) != 0 {
			Fatal(t, "Unexpected balance:", l2balance)
		}

	}
}
