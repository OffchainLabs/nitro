package arbtest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/blsSignatures"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"

	"github.com/offchainlabs/nitro/das/dasrpc"

	"github.com/offchainlabs/nitro/das"
)

func startLocalDASServer(
	t *testing.T,
	ctx context.Context,
	dataDir string,
	l1client arbutil.L1Interface,
	seqInboxAddress common.Address,
) (*http.Server, *blsSignatures.PublicKey, dasrpc.BackendConfig) {
	lis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	keyDir := t.TempDir()
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	Require(t, err)
	dasConfig := das.LocalDiskDASConfig{
		KeyDir:  keyDir,
		DataDir: dataDir,
	}
	localDas, err := das.NewLocalDiskDASWithL1Info(dasConfig, l1client, seqInboxAddress)
	Require(t, err)
	dasServer, err := dasrpc.StartDASRPCServerOnListener(ctx, lis, localDas)
	Require(t, err)
	config := dasrpc.BackendConfig{
		URL:                 lis.Addr().String(),
		PubKeyBase64Encoded: blsPubToBase64(pubkey),
		SignerMask:          1,
	}
	return dasServer, pubkey, config
}

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

func aggConfigForBackend(t *testing.T, backendConfig dasrpc.BackendConfig) das.AggregatorConfig {
	backendsJsonByte, err := json.Marshal([]dasrpc.BackendConfig{backendConfig})
	Require(t, err)
	return das.AggregatorConfig{
		AssumedHonest: 1,
		Backends:      string(backendsJsonByte),
	}
}

func TestDASRekey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	chainConfig := params.ArbitrumDevTestDASChainConfig()
	l1info, l1client, _, l1stack := CreateTestL1BlockChain(t, nil)
	defer l1stack.Close()
	addresses := DeployOnTestL1(t, ctx, l1info, l1client, chainConfig.ChainID)

	// Setup DAS servers
	dasDataDir := t.TempDir()
	dasServerA, pubkeyA, backendConfigA := startLocalDASServer(t, ctx, dasDataDir, l1client, addresses.SequencerInbox)
	authorizeDASKeyset(t, ctx, pubkeyA, l1info, l1client)

	// Setup L2 chain
	l2info, l2stack, l2chainDb, l2blockchain := createL2BlockChain(t, nil, chainConfig)
	l2info.GenerateAccount("User2")

	// Setup DAS config
	l1NodeConfigA := arbnode.ConfigDefaultL1Test()
	l1NodeConfigA.DataAvailability.ModeImpl = "aggregator"
	l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, backendConfigA)

	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	sequencerTxOptsPtr := &sequencerTxOpts
	nodeA, err := arbnode.CreateNode(l2stack, l2chainDb, l1NodeConfigA, l2blockchain, l1client, addresses, sequencerTxOptsPtr, nil)
	Require(t, err)
	Require(t, nodeA.Start(ctx))
	l2clientA := ClientForArbBackend(t, nodeA.Backend)

	l1NodeConfigB := arbnode.ConfigDefaultL1Test()
	l1NodeConfigB.BatchPoster.Enable = false
	l1NodeConfigB.BlockValidator.Enable = false
	l1NodeConfigB.DataAvailability.ModeImpl = "aggregator"
	l1NodeConfigB.DataAvailability.AggregatorConfig = aggConfigForBackend(t, backendConfigA)
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)
	checkBatchPosting(t, ctx, l1client, l2clientA, l2clientB, l1info, l2info, big.NewInt(1e12))
	nodeA.StopAndWait()
	nodeB.StopAndWait()

	err = dasServerA.Shutdown(ctx)
	Require(t, err)
	dasServerB, pubkeyB, backendConfigB := startLocalDASServer(t, ctx, dasDataDir, l1client, addresses.SequencerInbox)
	defer func() {
		err = dasServerB.Shutdown(ctx)
		Require(t, err)
	}()
	authorizeDASKeyset(t, ctx, pubkeyB, l1info, l1client)

	// Restart the node on the new keyset against the new DAS server running on the same disk as the first with new keys

	l2stack, err = arbnode.CreateDefaultStack()
	Require(t, err)
	l2blockchain, err = arbnode.GetBlockChain(l2chainDb, nil, chainConfig)
	Require(t, err)
	l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, backendConfigB)
	nodeA, err = arbnode.CreateNode(l2stack, l2chainDb, l1NodeConfigA, l2blockchain, l1client, addresses, sequencerTxOptsPtr, nil)
	Require(t, err)
	Require(t, nodeA.Start(ctx))
	l2clientA = ClientForArbBackend(t, nodeA.Backend)

	l1NodeConfigB.DataAvailability.AggregatorConfig = aggConfigForBackend(t, backendConfigB)
	l2clientB, nodeB = Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)
	checkBatchPosting(t, ctx, l1client, l2clientA, l2clientB, l1info, l2info, big.NewInt(2e12))

	nodeA.StopAndWait()
	nodeB.StopAndWait()
}

func checkBatchPosting(t *testing.T, ctx context.Context, l1client, l2clientA, l2clientB *ethclient.Client, l1info, l2info info, expectedBalance *big.Int) {
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

	_, err = WaitForTx(ctx, l2clientB, tx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(expectedBalance) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}
}
