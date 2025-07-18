package arbtest

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	protos "github.com/EspressoSystems/timeboost-proto/go-generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/core/types"
)

// Acknowledgement flag that timeboost will wait for to know sequencer processed
// Inclusion list succesfully
const ACK_FLAG = 0xc0

func createL1AndL2NodeForTimeboost(
	ctx context.Context,
	t *testing.T,
	delayedSequencer bool,
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

func ConvertTxsToGethexecTxs(t *testing.T, txs []*types.Transaction) []*protos.Transaction {
	var txns []*protos.Transaction
	for _, tx := range txs {
		txBytes, err := tx.MarshalBinary()
		Require(t, err)

		time := tx.Time().Unix()
		if time < 0 {
			t.Fatalf("Invalid timestamp %d", time)
		}
		protoTx := protos.Transaction{
			EncodedTxn: txBytes,
			Address:    []byte{0x00},
			Timestamp:  uint64(time),
		}
		txns = append(txns, &protoTx)
	}
	return txns
}

func GenerateInclusionLists(t *testing.T, users []string, builder *NodeBuilder, numIncls int, transactionsList [][]*types.Transaction) []*protos.InclusionList {
	var incls []*protos.InclusionList
	// Create given number of inclusion lists

	for i := 0; i < numIncls; i++ {
		// Every user generates a transaction and put into inclusion list
		txns := ConvertTxsToGethexecTxs(t, transactionsList[i])
		if i < 0 {
			t.Fatalf("Invalid index %d", i)
		}
		incl := &protos.InclusionList{
			Round:               uint64(i),
			ConsensusTimestamp:  uint64(i),
			EncodedTxns:         txns,
			DelayedMessagesRead: 0,
		}
		incls = append(incls, incl)
	}
	return incls
}

func SendInclusionLists(t *testing.T, incls []*protos.InclusionList) {
	// Connect to the default port of listener
	grpcConn, err := grpc.NewClient("localhost:55000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	Require(t, err)
	grpcClient := protos.NewForwardApiClient(grpcConn)
	defer grpcConn.Close()

	// Iterate over each inclusion list
	for _, incl := range incls {
		// Send via grpc
		_, err := grpcClient.SubmitInclusionList(context.Background(), incl)
		Require(t, err)
	}
}

func TestEspressoTimeboostSequencer(t *testing.T) {
	const numUsers = 10

	// Start the same blockchain for all tests
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	numIncls := 1

	valNodeCleanup := createValidationNode(ctx, t, true)
	defer valNodeCleanup()
	// In future, we also need to create a version of
	// delayed sequencer for timeboost
	builder, cleanup := createL1AndL2NodeForTimeboost(ctx, t, true)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	var users []string

	for num := 0; num < numUsers; num++ {
		userName := fmt.Sprintf("My_User_%d", num)
		builder.L2Info.GenerateAccount(userName)
		users = append(users, userName)
	}

	/**
		################################################################
			Test Scenarios: These tests are used to check if the timeboost
			by reading the inclusion list from the Timeboost listener
			which is sent using sailfish
		################################################################
	**/
	t.Run("Run simple test to see if it builds the block", func(t *testing.T) {

		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)

		var txns []*types.Transaction
		// Every user generates a transaction and put into inclusion list
		for _, userName := range users {
			tx := builder.L2Info.PrepareTx("Owner", userName, builder.L2Info.TransferGas, big.NewInt(2000000000000000000), nil)
			txns = append(txns, tx)
		}

		txnsList := make([][]*types.Transaction, 0)
		txnsList = append(txnsList, txns)
		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls, txnsList)
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
		if len(transactions) != count {
			t.Fatalf("expected inclusion and transaction to match. got %d inclusion txns, got %d processed transactions", count, len(transactions))
		}

	})

	/**
		################################################################
		Test Scenarios: These tests are used to check if the timeboost
		sequencer follows the rules of rounds which include:
			1. Only same rounds are included in the block
			2. A block is full, the next block should be created using
			the transactions left from the previous round
		################################################################
	**/

	t.Run("only same rounds are included in the block", func(t *testing.T) {
		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberBefore > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberBefore)
		}
		numIncls := 3
		// Create three transactions list where each user is the sender to another user
		txnsList := make([][]*types.Transaction, 0)

		// Create three transaction lists, each transaction list will form a different inclusion list with a different round id
		for i := 0; i < numIncls; i++ {
			txns := make([]*types.Transaction, 0)
			txn := builder.L2Info.PrepareTx(users[i], users[1+1], builder.L2Info.TransferGas, big.NewInt(1), nil)
			txns = append(txns, txn)
			txnsList = append(txnsList, txns)
		}

		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls, txnsList)

		SendInclusionLists(t, inclusionLists)

		// Wait for sometime for the block to be produced
		time.Sleep(time.Second * 20)

		blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberAfter > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}

		// Check that if that block contains all the tx hashes
		if blockNumberAfter-blockNumberBefore <= 0 {
			t.Fatalf("expected difference between blockNumberAfter and blockNumberBefore to be greater than 0, got: %d", blockNumberAfter-blockNumberBefore)
		}

		// Check that three blocks should have been produced each with a different round id
		if numIncls < 0 {
			t.Fatalf("numIncls cannt be converted to uint64")
		}
		if blockNumberAfter-blockNumberBefore != uint64(numIncls) {
			t.Fatalf("expected blockNumberAfter to be %d, got: %d", numIncls, blockNumberAfter-blockNumberBefore)
		}

		var blocksTxns [][]*types.Transaction
		round := 0
		// Iterate over each block and check that the round id is correct
		for i := blockNumberBefore + 1; i <= blockNumberAfter; i++ {
			if i > math.MaxInt64 {
				t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
			}
			block, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(int64(i)))
			Require(t, err)
			txns := block.Transactions()
			blocksTxns = append(blocksTxns, txns[1:])
			if block.Time() != inclusionLists[round].Round {
				t.Fatalf("expected round to be %d, got: %d", inclusionLists[round].Round, block.Time())
			}
			round++
		}

		for i := 0; i < numIncls; i++ {
			// Check that the block contains the correct number of transactions
			if len(blocksTxns[i]) != len(txnsList[i]) {
				t.Fatalf("expected number of transactions to be %d, got: %d", len(txnsList[i]), len(blocksTxns[i]))
			}
			// Check that the transactions are correct
			for j := 0; j < len(blocksTxns[i]); j++ {
				tx := blocksTxns[i][j]
				var expected types.Transaction
				encodedTransaction, err := txnsList[i][j].MarshalBinary()
				Require(t, err)
				err = expected.UnmarshalBinary(encodedTransaction)
				Require(t, err)
				if tx.Hash() != expected.Hash() {
					t.Fatalf("txHash doesn't match, got %s, want %s.", tx.Hash().Hex(), expected.Hash().Hex())
				}
			}
		}

	})

	t.Run("if a block is full, the next block should be created using the transactions left from the previous round", func(t *testing.T) {

		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberBefore > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberBefore)
		}
		numIncls := 3
		// Create three transactions list where each user is the sender to another user
		txnsList := make([][]*types.Transaction, 0)

		// Create three transaction lists, each transaction list will form a different inclusion list with a different round id
		for i := 0; i < numIncls; i++ {
			txns := make([]*types.Transaction, 0)
			// For inclusion list with round id 0, add multiple transactions so that it forms multiple nitro blocks
			if i == 0 {
				for j := 0; j < 30; j++ {
					txn := builder.L2Info.PrepareTx(users[i], users[i+1], builder.L2Info.TransferGas, big.NewInt(1), nil)
					txns = append(txns, txn)
				}
			} else {
				txn := builder.L2Info.PrepareTx(users[i], users[1+1], builder.L2Info.TransferGas, big.NewInt(1), nil)
				txns = append(txns, txn)
			}
			txnsList = append(txnsList, txns)
		}
		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls, txnsList)
		SendInclusionLists(t, inclusionLists)

		// Wait for sometime for the block to be produced
		time.Sleep(time.Second * 20)

		blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberAfter > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}

		if numIncls < 0 {
			t.Fatalf("expected numIncls to be greater than 0, got: %d", numIncls)
		}
		// This check ensures that more blocks were created than the number of inclusion lists
		if blockNumberAfter-blockNumberBefore <= uint64(numIncls) {
			t.Fatalf("expected difference between blockNumberAfter and blockNumberBefore should be greater than 0, got: %d", blockNumberAfter-blockNumberBefore)
		}

		// Initially the round number should be 0 and roundTransactions should contain the transactions from the first inclusion list
		roundNumber := 0
		roundsTransactions := make([]*types.Transaction, 0)
		roundsTransactions = append(roundsTransactions, txnsList[0]...)

		// Initially we will fill the roundsTransactions with the transactions from the first inclusion list
		// Iterate over each block and check that the round id is correct
		for i := blockNumberBefore + 1; i <= blockNumberAfter; i++ {
			if i > math.MaxInt64 {
				t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
			}
			block, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(int64(i)))
			Require(t, err)
			txns := block.Transactions()
			// Remove the first transaction because that is the transaction which just marks the start of the block
			txns = txns[1:]

			for _, txn := range txns {
				var expected types.Transaction
				encodedTransaction, err := txn.MarshalBinary()
				Require(t, err)
				err = expected.UnmarshalBinary(encodedTransaction)
				Require(t, err)
				if expected.Hash() != roundsTransactions[0].Hash() {
					t.Fatalf("txHash doesn't match, got %s, want %s.", expected.Hash().Hex(), roundsTransactions[0].Hash().Hex())
				}
				if len(roundsTransactions) == 1 {
					// Remove the roundsTransactions[0] from the roundsTransactions
					roundNumber++
					if roundNumber == numIncls {
						// This condition means that everything was processing successfully
						break
					}
					roundsTransactions = txnsList[roundNumber]
				} else {
					roundsTransactions = roundsTransactions[1:]
				}
			}
		}
	})

	/**
		################################################################
		Test Scenarios: These tests are used to check that transactions
		follow some rules:
			1. Given transaction should not be greater than the max tx data size
			2. Higher nonce transactions should not be included in the block
			3. Lower nonce transactions should not be included in the block
			4. Transactions with gas fee lower than the base fee should
			not be included in the block

		################################################################
	**/

	t.Run("timeboost sequencer removes transactions with invalid nonce", func(t *testing.T) {

		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberBefore > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberBefore)
		}
		// Create three transactions list where each user is the sender to another user
		txnsList := make([][]*types.Transaction, 0)

		// Create three transaction lists, each transaction list will form a different inclusion list with a different round id
		txns := make([]*types.Transaction, 0)
		// Transactions with higher nonce should not be included in the block
		txnWithHigherNonce := builder.L2Info.PrepareTxWithInvalidNonce(users[1], users[2], builder.L2Info.TransferGas, big.NewInt(1), nil, true)
		txns = append(txns, txnWithHigherNonce)
		// Transaction with lower nonce should not be included in the block
		txnWithLowerNonce := builder.L2Info.PrepareTxWithInvalidNonce(users[2], users[1], builder.L2Info.TransferGas, big.NewInt(1), nil, false)
		txns = append(txns, txnWithLowerNonce)
		txnsList = append(txnsList, txns)

		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls, txnsList)
		SendInclusionLists(t, inclusionLists)

		// Wait for sometime for the block to be produced
		time.Sleep(time.Second * 20)

		blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberAfter > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}

		// No blocks should be created
		if blockNumberAfter-blockNumberBefore > 0 {
			t.Fatalf("expected no blocks to be created, got: %d", blockNumberAfter-blockNumberBefore)
		}
	})

	t.Run("timeboost sequencer removes transactions with invalid gas fee", func(t *testing.T) {

		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberBefore > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberBefore)
		}
		// Create three transactions list where each user is the sender to another user
		txnsList := make([][]*types.Transaction, 0)

		// Create three transaction lists, each transaction list will form a different inclusion list with a different round id
		txns := make([]*types.Transaction, 0)
		// Transactions with invalid gas fee
		txnWithHigherGasFee := builder.L2Info.PrepareTxWithInvalidGasFee(users[6], users[7], builder.L2Info.TransferGas, big.NewInt(1), nil)
		txns = append(txns, txnWithHigherGasFee)
		txnsList = append(txnsList, txns)

		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls, txnsList)
		SendInclusionLists(t, inclusionLists)

		// Wait for sometime for the block to be produced
		time.Sleep(time.Second * 20)

		blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberAfter > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}

		// No blocks should be created
		if blockNumberAfter-blockNumberBefore > 0 {
			t.Fatalf("expected no blocks to be created, got: %d", blockNumberAfter-blockNumberBefore)
		}
	})

	t.Run("timeboost sequencer removes transactions with invalid size", func(t *testing.T) {

		blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberBefore > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberBefore)
		}
		// Create three transactions list where each user is the sender to another user
		txnsList := make([][]*types.Transaction, 0)

		// Create three transaction lists, each transaction list will form a different inclusion list with a different round id
		txns := make([]*types.Transaction, 0)
		// Add data bytes which are greater than 95000
		data := make([]byte, 195000)
		txnWithInvalidSize := builder.L2Info.PrepareTx(users[4], users[5], builder.L2Info.TransferGas, big.NewInt(1), data)
		txns = append(txns, txnWithInvalidSize)
		txnsList = append(txnsList, txns)

		// Generate and send inclusion lists
		inclusionLists := GenerateInclusionLists(t, users, builder, numIncls, txnsList)
		SendInclusionLists(t, inclusionLists)

		// Wait for sometime for the block to be produced
		time.Sleep(time.Second * 20)

		blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if blockNumberAfter > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}

		// No blocks should be created
		if blockNumberAfter-blockNumberBefore > 0 {
			t.Fatalf("expected no blocks to be created, got: %d", blockNumberAfter-blockNumberBefore)
		}
	})

}
