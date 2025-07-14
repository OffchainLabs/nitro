package arbtest

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/core/types"

	gethexec "github.com/offchainlabs/nitro/execution/gethexec/protos"
)

// Acknowledgement flag that timeboost will wait for to know sequencer processed
// Inclusion list succesfully
const ACK_FLAG = 0xc0

func createL1AndL2NodeForTimeboost(
	ctx context.Context,
	t *testing.T,
	delayedSequencer bool,
	blobsEnabled bool,
) (*NodeBuilder, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l1StackConfig.HTTPPort = 8545
	builder.l1StackConfig.WSPort = 8546
	builder.l1StackConfig.HTTPHost = "0.0.0.0"
	builder.l1StackConfig.HTTPVirtualHosts = []string{"*"}
	builder.l1StackConfig.WSHost = "0.0.0.0"
	builder.l1StackConfig.DataDir = t.TempDir()
	builder.l1StackConfig.WSModules = append(builder.l1StackConfig.WSModules, "eth")
	builder.l2StackConfig.HTTPPort = 8945
	builder.l2StackConfig.HTTPHost = "0.0.0.0"
	builder.l2StackConfig.IPCPath = tmpPath(t, "test.ipc")
	builder.useL1StackConfig = true

	// poster config
	builder.nodeConfig.BatchPoster.Enable = false

	// validator config
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.ValidationPoll = 2 * time.Second
	builder.nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", arbValidationPort)
	builder.nodeConfig.DelayedSequencer.Enable = delayedSequencer
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1

	// sequencer config
	builder.nodeConfig.Sequencer = false
	builder.nodeConfig.ParentChainReader.Enable = true // This flag is necessary to enable sequencing transactions with espresso behavior
	builder.nodeConfig.ParentChainReader.UseFinalityData = true
	builder.nodeConfig.Dangerous.NoSequencerCoordinator = true
	builder.execConfig.Sequencer.Enable = false
	builder.execConfig.Caching.StateScheme = "hash"
	builder.execConfig.Caching.Archive = true

	// Enable timeboost sequencer
	builder.nodeConfig.TimeboostSequencer.Enable = true
	builder.nodeConfig.TimeboostSequencer.BlockRetryDuration = time.Second
	builder.nodeConfig.TimeboostSequencer.MaxTxDataSize = 8000
	builder.nodeConfig.TimeboostSequencer.NonceCacheSize = 1024
	builder.nodeConfig.TimeboostSequencer.MaxRevertGasReject = 0
	builder.nodeConfig.TimeboostSequencer.ParentChainFinalizationTime = 20 * time.Minute
	builder.nodeConfig.TimeboostSequencer.MaxAcceptableTimestampDelta = time.Hour
	builder.nodeConfig.TimeboostSequencer.EnableProfiling = false
	builder.nodeConfig.TimeboostSequencer.TimeboostBridgeConfig.InternalTimeboostGrpcUrl = "localhost:5000"

	cleanup := builder.Build(t)

	mnemonic := "indoor dish desk flag debris potato excuse depart ticket judge file exit"
	err := builder.L1Info.GenerateAccountWithMnemonic("CommitmentTask", mnemonic, 5)
	Require(t, err)
	builder.L1.TransferBalance(t, "Faucet", "CommitmentTask", new(big.Int).Mul(big.NewInt(9e18), big.NewInt(1000)), builder.L1Info)

	return builder, cleanup
}

func GenerateInclusionLists(t *testing.T, users []string, builder *NodeBuilder, numIncls int) []*gethexec.InclusionList {
	var incls []*gethexec.InclusionList
	// Create given number of inclusion lists
	for i := range numIncls {
		var txns []*gethexec.Transaction
		// Every user generates a transaction and put into inclusion list
		for _, userName := range users {
			tx := builder.L2Info.PrepareTx("Owner", userName, builder.L2Info.TransferGas, big.NewInt(2), nil)
			txBytes, err := tx.MarshalBinary()
			Require(t, err)

			time := tx.Time().Unix()
			if time < 0 {
				t.Fatalf("Invalid timestamp %d", time)
			}
			protoTx := gethexec.Transaction{
				EncodedTxn: txBytes,
				Address:    []byte{0x00},
				Timestamp:  uint64(time),
			}
			txns = append(txns, &protoTx)
		}
		if i < 0 {
			t.Fatalf("Invalid index %d", i)
		}
		incl := &gethexec.InclusionList{
			Round:               uint64(i),
			ConsensusTimestamp:  uint64(i),
			EncodedTxns:         txns,
			DelayedMessagesRead: 0,
		}
		incls = append(incls, incl)
	}
	return incls
}

func SendInclusionLists(t *testing.T, incls []*gethexec.InclusionList) {
	// Connect to the default port of listener
	grpcConn, err := grpc.NewClient("localhost:55000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	Require(t, err)
	grpcClient := gethexec.NewForwardApiClient(grpcConn)
	defer grpcConn.Close()

	// Iterate over each inclusion list
	for _, incl := range incls {
		// Send via grpc
		_, err := grpcClient.SubmitInclusionList(context.Background(), incl)
		Require(t, err)
	}
}

func TestEspressoTimeboostSequencer(t *testing.T) {

	t.Run("Run simple test to see if it builds the block", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		valNodeCleanup := createValidationNode(ctx, t, true)
		defer valNodeCleanup()
		// In future, we also need to create a version of
		// delayed sequencer for timeboost
		builder, cleanup := createL1AndL2NodeForTimeboost(ctx, t, true, false)
		defer cleanup()

		err := waitForL1Node(ctx)
		Require(t, err)

		var users []string

		const numUsers = 10
		const numIncls = 1

		for num := 0; num < numUsers; num++ {
			userName := fmt.Sprintf("My_User_%d", num)
			builder.L2Info.GenerateAccount(userName)
			users = append(users, userName)
		}

		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)

		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls)
		SendInclusionLists(t, inclusionLists)

		// Wait for sometime for the block to be produced
		time.Sleep(time.Second * 10)

		// Check that the database now has updated block
		blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)

		// msgCntAfter should be 1 greater than msgCntBefore
		if blockNumberAfter-blockNumberBefore <= 0 {
			t.Fatalf("expected difference between blockNumberAfter and blockNumberBefore to be greater than 0, got: %d", blockNumberAfter-blockNumberBefore)
		}

		// Check that if that block contains all the tx hashes
		if blockNumberAfter > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}

		// Get all the transactions from all the blocks after the blockNumberBefore
		var transactions []*types.Transaction
		for i := blockNumberBefore + 1; i <= blockNumberAfter; i++ {
			if i > math.MaxInt64 {
				t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
			}
			block, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(int64(i)))
			Require(t, err)
			blockTransactions := block.Transactions()
			transactionsWithoutStartBlock := blockTransactions[1:]
			transactions = append(transactions, transactionsWithoutStartBlock...)
		}

		count := 0
		// Iterate over each inclusion list the order they were sent
		for _, incl := range inclusionLists {
			// And compare each transaction was sent and processed in order
			for _, protoTxn := range incl.EncodedTxns {
				tx := transactions[count]
				var expected types.Transaction
				err = expected.UnmarshalBinary(protoTxn.EncodedTxn)
				Require(t, err)
				if tx.Hash() != expected.Hash() {
					t.Fatalf("txHash doesn't match, got %s, want %s.", tx.Hash().Hex(), expected.Hash().Hex())
				}
				count++
			}
		}
		if count != numUsers*numIncls || len(transactions) != count {
			t.Fatalf("expected inclusion and transaction to match. got %d inclusion txns, got %d processed transactions", count, len(transactions))
		}
	})
}
