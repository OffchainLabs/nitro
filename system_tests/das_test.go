// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func startLocalDASServer(
	t *testing.T,
	ctx context.Context,
	dataDir string,
	l1client arbutil.L1Interface,
	seqInboxAddress common.Address,
) (*http.Server, *blsSignatures.PublicKey, das.BackendConfig, *das.RestfulDasServer, string) {
	keyDir := t.TempDir()
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	config := das.DataAvailabilityConfig{
		Enable: true,
		Key: das.KeyConfig{
			KeyDir: keyDir,
		},
		LocalFileStorage: das.LocalFileStorageConfig{
			Enable:  true,
			DataDir: dataDir,
		},
		ParentChainNodeURL: "none",
		RequestTimeout:     5 * time.Second,
	}

	storageService, lifecycleManager, err := das.CreatePersistentStorageService(ctx, &config)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	Require(t, err)
	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(seqInboxAddress, l1client)
	Require(t, err)
	daWriter, err := das.NewSignAfterStoreDASWriter(ctx, config, storageService)
	Require(t, err)
	signatureVerifier, err := das.NewSignatureVerifierWithSeqInboxCaller(seqInboxCaller, "")
	Require(t, err)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	rpcServer, err := das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, storageService, daWriter, storageService, signatureVerifier)
	Require(t, err)
	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restServer, err := das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, storageService, storageService)
	Require(t, err)
	beConfig := das.BackendConfig{
		URL:    "http://" + rpcLis.Addr().String(),
		Pubkey: blsPubToBase64(pubkey),
	}
	return rpcServer, pubkey, beConfig, restServer, "http://" + restLis.Addr().String()
}

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

func aggConfigForBackend(t *testing.T, backendConfig das.BackendConfig) das.AggregatorConfig {
	return das.AggregatorConfig{
		Enable:                true,
		AssumedHonest:         1,
		Backends:              das.BackendConfigList{backendConfig},
		MaxStoreChunkBodySize: 512 * 1024,
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
		builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(t, backendConfigA)
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
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(t, backendConfigB)
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

func checkBatchPosting(t *testing.T, ctx context.Context, l1client, l2clientA *ethclient.Client, l1info, l2info info, expectedBalance *big.Int, l2ClientsToCheck ...*ethclient.Client) {
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := l2clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	for _, client := range l2ClientsToCheck {
		_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*30)
		Require(t, err)

		l2balance, err := client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
		Require(t, err)

		if l2balance.Cmp(expectedBalance) != 0 {
			Fatal(t, "Unexpected balance:", l2balance)
		}

	}
}

func TestDASComplexConfigAndRestMirror(t *testing.T) {
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
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(t, beConfigA)
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

func enableLogging(logLvl int) {
	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(slog.Level(logLvl))
	log.SetDefault(log.NewLogger(glogger))
}

func initTest(t *testing.T) {
	t.Parallel()
	loggingStr := os.Getenv("LOGGING")
	if len(loggingStr) > 0 {
		var err error
		logLvl, err := strconv.Atoi(loggingStr)
		Require(t, err, "Failed to parse string")
		enableLogging(logLvl)
	}
}
