// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
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
		KeyConfig: das.KeyConfig{
			KeyDir: keyDir,
		},
		LocalFileStorageConfig: das.LocalFileStorageConfig{
			Enable:  true,
			DataDir: dataDir,
		},
		L1NodeURL:      "none",
		RequestTimeout: 5 * time.Second,
	}

	storageService, lifecycleManager, err := das.CreatePersistentStorageService(ctx, &config)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	Require(t, err)
	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(seqInboxAddress, l1client)
	Require(t, err)
	privKey, err := config.KeyConfig.BLSPrivKey()
	Require(t, err)
	currentDas, err := das.NewSignAfterStoreDASWithSeqInboxCaller(privKey, seqInboxCaller, storageService, "")
	Require(t, err)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	rpcServer, err := das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, currentDas)
	Require(t, err)
	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restServer, err := das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, currentDas)
	Require(t, err)
	beConfig := das.BackendConfig{
		URL:                 "http://" + rpcLis.Addr().String(),
		PubKeyBase64Encoded: blsPubToBase64(pubkey),
		SignerMask:          1,
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
	backendsJsonByte, err := json.Marshal([]das.BackendConfig{backendConfig})
	Require(t, err)
	return das.AggregatorConfig{
		Enable:        true,
		AssumedHonest: 1,
		Backends:      string(backendsJsonByte),
	}
}

func TestDASRekey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	chainConfig := params.ArbitrumDevTestDASChainConfig()
	l1info, l1client, _, l1stack := createTestL1BlockChain(t, nil)
	defer requireClose(t, l1stack)
	feedErrChan := make(chan error, 10)
	addresses := DeployOnTestL1(t, ctx, l1info, l1client, chainConfig.ChainID)

	// Setup DAS servers
	dasDataDir := t.TempDir()
	nodeDir := t.TempDir()
	dasRpcServerA, pubkeyA, backendConfigA, _, restServerUrlA := startLocalDASServer(t, ctx, dasDataDir, l1client, addresses.SequencerInbox)
	l2info := NewArbTestInfo(t, chainConfig.ChainID)
	l1NodeConfigA := arbnode.ConfigDefaultL1Test()
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	sequencerTxOptsPtr := &sequencerTxOpts

	{
		authorizeDASKeyset(t, ctx, pubkeyA, l1info, l1client)

		// Setup L2 chain
		_, l2stackA, l2chainDb, l2arbDb, l2blockchain := createL2BlockChain(t, l2info, nodeDir, chainConfig)
		l2info.GenerateAccount("User2")

		// Setup DAS config

		l1NodeConfigA.DataAvailability.Enable = true
		l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, backendConfigA)
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig = das.DefaultRestfulClientAggregatorConfig
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Enable = true
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Urls = []string{restServerUrlA}
		l1NodeConfigA.DataAvailability.L1NodeURL = "none"

		nodeA, err := arbnode.CreateNode(ctx, l2stackA, l2chainDb, l2arbDb, l1NodeConfigA, l2blockchain, l1client, addresses, sequencerTxOptsPtr, nil, feedErrChan)
		Require(t, err)
		Require(t, l2stackA.Start())
		l2clientA := ClientForStack(t, l2stackA)

		l1NodeConfigB.BlockValidator.Enable = false
		l1NodeConfigB.DataAvailability.Enable = true
		l1NodeConfigB.DataAvailability.RestfulClientAggregatorConfig = das.DefaultRestfulClientAggregatorConfig
		l1NodeConfigB.DataAvailability.RestfulClientAggregatorConfig.Enable = true
		l1NodeConfigB.DataAvailability.RestfulClientAggregatorConfig.Urls = []string{restServerUrlA}

		l1NodeConfigB.DataAvailability.L1NodeURL = "none"

		l2clientB, _, l2stackB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)
		checkBatchPosting(t, ctx, l1client, l2clientA, l1info, l2info, big.NewInt(1e12), l2clientB)
		requireClose(t, l2stackA)
		requireClose(t, l2stackB)
	}

	err := dasRpcServerA.Shutdown(ctx)
	Require(t, err)
	dasRpcServerB, pubkeyB, backendConfigB, _, _ := startLocalDASServer(t, ctx, dasDataDir, l1client, addresses.SequencerInbox)
	defer func() {
		err = dasRpcServerB.Shutdown(ctx)
		Require(t, err)
	}()
	authorizeDASKeyset(t, ctx, pubkeyB, l1info, l1client)

	// Restart the node on the new keyset against the new DAS server running on the same disk as the first with new keys

	l2stackA, err := arbnode.CreateDefaultStackForTest(nodeDir)
	Require(t, err)

	l2chainDb, err := l2stackA.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)

	l2arbDb, err := l2stackA.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	l2blockchain, err := arbnode.GetBlockChain(l2chainDb, nil, chainConfig, arbnode.ConfigDefaultL2Test())
	Require(t, err)
	l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, backendConfigB)
	nodeA, err := arbnode.CreateNode(ctx, l2stackA, l2chainDb, l2arbDb, l1NodeConfigA, l2blockchain, l1client, addresses, sequencerTxOptsPtr, nil, feedErrChan)
	Require(t, err)
	Require(t, l2stackA.Start())
	l2clientA := ClientForStack(t, l2stackA)

	l2clientB, _, l2stackB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)
	checkBatchPosting(t, ctx, l1client, l2clientA, l1info, l2info, big.NewInt(2e12), l2clientB)

	Require(t, l2stackA.Close())
	Require(t, l2stackB.Close())
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
		_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*5)
		Require(t, err)

		l2balance, err := client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
		Require(t, err)

		if l2balance.Cmp(expectedBalance) != 0 {
			Fail(t, "Unexpected balance:", l2balance)
		}

	}
}

func TestDASComplexConfigAndRestMirror(t *testing.T) {
	initTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	chainConfig := params.ArbitrumDevTestDASChainConfig()
	l1info, l1client, _, l1stack := createTestL1BlockChain(t, nil)
	defer requireClose(t, l1stack)
	l1Reader := headerreader.New(l1client, headerreader.TestConfig)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()
	feedErrChan := make(chan error, 10)
	addresses := DeployOnTestL1(t, ctx, l1info, l1client, chainConfig.ChainID)

	keyDir, fileDataDir, dbDataDir := t.TempDir(), t.TempDir(), t.TempDir()
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	Require(t, err)

	serverConfig := das.DataAvailabilityConfig{
		Enable: true,

		LocalCacheConfig: das.TestBigCacheConfig,

		LocalFileStorageConfig: das.LocalFileStorageConfig{
			Enable:  true,
			DataDir: fileDataDir,
		},
		LocalDBStorageConfig: das.LocalDBStorageConfig{
			Enable:  true,
			DataDir: dbDataDir,
		},

		KeyConfig: das.KeyConfig{
			KeyDir: keyDir,
		},

		RequestTimeout: 5 * time.Second,
		// L1NodeURL: normally we would have to set this but we are passing in the already constructed client and addresses to the factory
	}

	dasServerStack, lifecycleManager, err := arbnode.SetUpDataAvailability(ctx, &serverConfig, l1Reader, addresses)
	Require(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	_, err = das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, dasServerStack)
	Require(t, err)
	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restServer, err := das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, dasServerStack)

	pubkeyA := pubkey
	authorizeDASKeyset(t, ctx, pubkeyA, l1info, l1client)

	//
	l1NodeConfigA := arbnode.ConfigDefaultL1Test()
	l1NodeConfigA.DataAvailability = das.DataAvailabilityConfig{
		Enable: true,

		LocalCacheConfig: das.TestBigCacheConfig,

		// AggregatorConfig set up below
		RequestTimeout: 5 * time.Second,
	}
	beConfigA := das.BackendConfig{
		URL:                 "http://" + rpcLis.Addr().String(),
		PubKeyBase64Encoded: blsPubToBase64(pubkey),
		SignerMask:          1,
	}
	l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, beConfigA)
	l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig = das.DefaultRestfulClientAggregatorConfig
	l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Enable = true
	l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Urls = []string{"http://" + restLis.Addr().String()}
	l1NodeConfigA.DataAvailability.L1NodeURL = "none"

	dataSigner := signature.DataSignerFromPrivateKey(l1info.Accounts["Sequencer"].PrivateKey)

	Require(t, err)

	// Setup L2 chain
	l2info, l2stackA, l2chainDb, l2arbDb, l2blockchain := createL2BlockChain(t, nil, "", chainConfig)
	l2info.GenerateAccount("User2")

	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	sequencerTxOptsPtr := &sequencerTxOpts
	nodeA, err := arbnode.CreateNode(ctx, l2stackA, l2chainDb, l2arbDb, l1NodeConfigA, l2blockchain, l1client, addresses, sequencerTxOptsPtr, dataSigner, feedErrChan)
	Require(t, err)
	Require(t, l2stackA.Start())
	l2clientA := ClientForStack(t, l2stackA)

	// Create node to sync from chain
	l1NodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	l1NodeConfigB.DataAvailability = das.DataAvailabilityConfig{
		Enable: true,

		LocalCacheConfig: das.TestBigCacheConfig,

		// AggregatorConfig set up below

		L1NodeURL:      "none",
		RequestTimeout: 5 * time.Second,
	}

	l1NodeConfigB.BlockValidator.Enable = false
	l1NodeConfigB.DataAvailability.Enable = true
	l1NodeConfigB.DataAvailability.RestfulClientAggregatorConfig = das.DefaultRestfulClientAggregatorConfig
	l1NodeConfigB.DataAvailability.RestfulClientAggregatorConfig.Enable = true
	l1NodeConfigB.DataAvailability.RestfulClientAggregatorConfig.Urls = []string{"http://" + restLis.Addr().String()}
	l1NodeConfigB.DataAvailability.L1NodeURL = "none"
	l2clientB, _, l2stackB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)

	checkBatchPosting(t, ctx, l1client, l2clientA, l1info, l2info, big.NewInt(1e12), l2clientB)

	requireClose(t, l2stackA)
	requireClose(t, l2stackB)

	err = restServer.Shutdown()
	Require(t, err)
}

func enableLogging(logLvl int) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(logLvl))
	log.Root().SetHandler(glogger)
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
