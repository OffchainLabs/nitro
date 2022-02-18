//go:build fullchallengetest
// +build fullchallengetest

//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/offchainlabs/arbstate/solgen/go/mocksgen"
	"github.com/offchainlabs/arbstate/solgen/go/ospgen"
	"github.com/offchainlabs/arbstate/validator"
)

func DeployOneStepProofEntry(t *testing.T, auth *bind.TransactOpts, client *ethclient.Client) common.Address {
	osp0, _, _, err := ospgen.DeployOneStepProver0(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospEntry, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = arbutil.EnsureTxSucceeded(context.Background(), client, tx)
	if err != nil {
		t.Fatal(err)
	}
	return ospEntry
}

func CreateChallenge(
	t *testing.T,
	auth *bind.TransactOpts,
	client *ethclient.Client,
	ospEntry common.Address,
	sequencerInbox common.Address,
	delayedBridge common.Address,
	wasmModuleRoot common.Hash,
	startGlobalState validator.GoGlobalState,
	endGlobalState validator.GoGlobalState,
	numBlocks uint64,
	asserter common.Address,
	challenger common.Address,
) (*mocksgen.MockResultReceiver, common.Address) {
	resultReceiverAddr, _, resultReceiver, err := mocksgen.DeployMockResultReceiver(auth, client)
	if err != nil {
		t.Fatal(err)
	}

	executionChallengeFactoryAddr, tx, _, err := challengegen.DeployExecutionChallengeFactory(auth, client, ospEntry)
	if err != nil {
		t.Fatal(err)
	}
	_, err = arbutil.EnsureTxSucceeded(context.Background(), client, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, tx, blockChallengeFactory, err := challengegen.DeployBlockChallengeFactory(auth, client, executionChallengeFactoryAddr)
	if err != nil {
		t.Fatal(err)
	}
	_, err = arbutil.EnsureTxSucceeded(context.Background(), client, tx)
	if err != nil {
		t.Fatal(err)
	}

	tx, err = blockChallengeFactory.CreateChallenge(
		auth,
		challengegen.IBlockChallengeFactoryChallengeContracts{
			ResultReceiver: resultReceiverAddr,
			SequencerInbox: sequencerInbox,
			DelayedBridge:  delayedBridge,
		},
		wasmModuleRoot,
		[2]uint8{
			validator.STATUS_FINISHED,
			validator.STATUS_FINISHED,
		},
		[2]challengegen.GlobalState{
			startGlobalState.AsSolidityStruct(),
			endGlobalState.AsSolidityStruct(),
		},
		numBlocks,
		asserter,
		challenger,
		big.NewInt(100000),
		big.NewInt(100000),
	)
	receipt, err := arbutil.EnsureTxSucceeded(context.Background(), client, tx)
	if err != nil {
		t.Fatal(err)
	}

	challengeCreatedEvent, err := blockChallengeFactory.ParseChallengeCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		t.Fatal(err)
	}

	return resultReceiver, challengeCreatedEvent.Challenge
}

func writeTxToBatch(writer io.Writer, tx *types.Transaction) error {
	txData, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	var segment []byte
	segment = append(segment, arbstate.BatchSegmentKindL2Message)
	segment = append(segment, arbos.L2MessageKind_SignedTx)
	segment = append(segment, txData...)
	err = rlp.Encode(writer, segment)
	return err
}

func makeBatch(t *testing.T, l2Node *arbnode.Node, l2Info *BlockchainTestInfo, backend *ethclient.Client, sequencer *bind.TransactOpts, seqInbox *mocksgen.SequencerInboxStub, seqInboxAddr common.Address, isChallenger bool) {
	ctx := context.Background()

	batchBuffer := bytes.NewBuffer([]byte{0})
	batchWriter := brotli.NewWriter(batchBuffer)
	for i := int64(0); i < 10; i++ {
		value := i
		if i == 5 && isChallenger {
			value++
		}
		err := writeTxToBatch(batchWriter, l2Info.PrepareTx("Owner", "Destination", 1000000, big.NewInt(value), []byte{}))
		if err != nil {
			t.Fatal(err)
		}
	}
	err := batchWriter.Flush()
	if err != nil {
		t.Fatal(err)
	}

	tx, err := seqInbox.AddSequencerL2BatchFromOrigin(sequencer, big.NewInt(1), batchBuffer.Bytes(), big.NewInt(0), common.Address{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := arbutil.EnsureTxSucceeded(ctx, backend, tx)
	if err != nil {
		t.Fatal(err)
	}

	nodeSeqInbox, err := arbnode.NewSequencerInbox(backend, seqInboxAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	batches, err := nodeSeqInbox.LookupBatchesInRange(ctx, receipt.BlockNumber, receipt.BlockNumber)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) == 0 {
		t.Fatal("batch not found after AddSequencerL2BatchFromOrigin")
	}
	err = l2Node.InboxTracker.AddSequencerBatches(ctx, backend, batches)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l2Node.InboxTracker.GetBatchMetadata(0)
	if err != nil {
		t.Fatal("failed to get batch metadata after adding batch:", err)
	}
}

func confirmLatestBlock(ctx context.Context, t *testing.T, l1Info *BlockchainTestInfo, backend arbutil.L1Interface) {
	for i := 0; i < 12; i++ {
		SendWaitTestTransactions(t, ctx, backend, []*types.Transaction{
			l1Info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
}

func runChallengeTest(t *testing.T, asserterIsCorrect bool) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info := NewL1TestInfo(t)
	l1Info.GenerateGenesysAccount("deployer", initialBalance)
	l1Info.GenerateGenesysAccount("asserter", initialBalance)
	l1Info.GenerateGenesysAccount("challenger", initialBalance)
	l1Info.GenerateGenesysAccount("sequencer", initialBalance)

	l1Info, l1Backend, _, _ := CreateTestL1BlockChain(t, l1Info)
	conf := arbnode.NodeConfigL1Test
	conf.BlockValidator = false
	conf.BatchPoster = false
	conf.InboxReaderConfig.CheckDelay = time.Second
	rollupAddresses := DeployOnTestL1(t, ctx, l1Info, l1Backend)

	deployerTxOpts := l1Info.GetDefaultTransactOpts("deployer")
	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer")
	asserterTxOpts := l1Info.GetDefaultTransactOpts("asserter")
	challengerTxOpts := l1Info.GetDefaultTransactOpts("challenger")
	delayedBridge, _, _, err := mocksgen.DeployBridgeStub(&deployerTxOpts, l1Backend)
	if err != nil {
		t.Fatal(err)
	}

	asserterSeqInboxAddr, _, asserterSeqInbox, err := mocksgen.DeploySequencerInboxStub(&deployerTxOpts, l1Backend, delayedBridge, l1Info.GetAddress("sequencer"))
	if err != nil {
		t.Fatal(err)
	}
	challengerSeqInboxAddr, _, challengerSeqInbox, err := mocksgen.DeploySequencerInboxStub(&deployerTxOpts, l1Backend, delayedBridge, l1Info.GetAddress("sequencer"))
	if err != nil {
		t.Fatal(err)
	}

	asserterL2Info, asserterL2Stack, asserterL2ChainDb, asserterL2Blockchain := createL2BlockChain(t, nil)
	rollupAddresses.SequencerInbox = asserterSeqInboxAddr
	asserterL2, err := arbnode.CreateNode(asserterL2Stack, asserterL2ChainDb, &conf, asserterL2Blockchain, l1Backend, rollupAddresses, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = asserterL2.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	challengerL2Info, challengerL2Stack, challengerL2ChainDb, challengerL2Blockchain := createL2BlockChain(t, nil)
	rollupAddresses.SequencerInbox = challengerSeqInboxAddr
	challengerL2, err := arbnode.CreateNode(challengerL2Stack, challengerL2ChainDb, &conf, challengerL2Blockchain, l1Backend, rollupAddresses, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = challengerL2.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	asserterL2Info.GenerateAccount("Destination")
	challengerL2Info.SetFullAccountInfo("Destination", asserterL2Info.GetInfoWithPrivKey("Destination"))
	makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, false)
	makeBatch(t, challengerL2, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, true)

	trueSeqInboxAddr := challengerSeqInboxAddr
	expectedWinner := l1Info.GetAddress("challenger")
	if asserterIsCorrect {
		trueSeqInboxAddr = asserterSeqInboxAddr
		expectedWinner = l1Info.GetAddress("asserter")
	}
	ospEntry := DeployOneStepProofEntry(t, &deployerTxOpts, l1Backend)

	wasmModuleRoot, err := validator.GetInitialModuleRoot(ctx)
	if err != nil {
		t.Fatal(err)
	}

	asserterGenesis := asserterL2.ArbInterface.BlockChain().Genesis()
	challengerGenesis := challengerL2.ArbInterface.BlockChain().Genesis()
	if asserterGenesis.Hash() != challengerGenesis.Hash() {
		t.Fatal("asserter and challenger have different genesis hashes")
	}
	asserterLatestBlock := asserterL2.ArbInterface.BlockChain().CurrentBlock()
	challengerLatestBlock := challengerL2.ArbInterface.BlockChain().CurrentBlock()
	if asserterLatestBlock.Hash() == challengerLatestBlock.Hash() {
		t.Fatal("asserter and challenger have the same end block")
	}

	asserterStartGlobalState := validator.GoGlobalState{
		BlockHash:  asserterGenesis.Hash(),
		Batch:      1,
		PosInBatch: 0,
	}
	asserterEndGlobalState := validator.GoGlobalState{
		BlockHash:  asserterLatestBlock.Hash(),
		Batch:      2,
		PosInBatch: 0,
	}
	numBlocks := asserterLatestBlock.NumberU64() - asserterGenesis.NumberU64()

	resultReceiver, challenge := CreateChallenge(
		t,
		&deployerTxOpts,
		l1Backend,
		ospEntry,
		trueSeqInboxAddr,
		delayedBridge,
		wasmModuleRoot,
		asserterStartGlobalState,
		asserterEndGlobalState,
		numBlocks,
		l1Info.GetAddress("asserter"),
		l1Info.GetAddress("challenger"),
	)

	confirmLatestBlock(ctx, t, l1Info, l1Backend)
	asserterManager, err := validator.NewChallengeManager(ctx, l1Backend, &asserterTxOpts, asserterTxOpts.From, challenge, asserterL2Blockchain, asserterL2.InboxReader, asserterL2.InboxTracker, asserterL2.TxStreamer, 0, 4, 12)
	if err != nil {
		t.Fatal(err)
	}

	challengerManager, err := validator.NewChallengeManager(ctx, l1Backend, &challengerTxOpts, challengerTxOpts.From, challenge, challengerL2Blockchain, challengerL2.InboxReader, challengerL2.InboxTracker, challengerL2.TxStreamer, 0, 4, 12)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		var tx *types.Transaction
		var currentCorrect bool
		// Gas cost is slightly reduced if done in the same timestamp or block as previous call.
		// This might make gas estimation undersestimate next move.
		// Invoke a new L1 block, with a new timestamp, before estimating.
		time.Sleep(time.Second)
		SendWaitTestTransactions(t, ctx, l1Backend, []*types.Transaction{
			l1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})

		if i%2 == 0 {
			currentCorrect = !asserterIsCorrect
			tx, err = challengerManager.Act(ctx)
		} else {
			currentCorrect = asserterIsCorrect
			tx, err = asserterManager.Act(ctx)
		}
		if err != nil {
			if !currentCorrect && (strings.Contains(err.Error(), "lost challenge") ||
				strings.Contains(err.Error(), "SAME_OSP_END") ||
				strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE")) {
				t.Log("challenge completed! asserter hit expected error:", err)
				return
			}
			t.Fatal("challenge step", i, "hit error:", err)
		}
		if tx == nil {
			t.Fatal("no move")
		}
		_, err = arbutil.EnsureTxSucceeded(ctx, l1Backend, tx)
		if err != nil {
			if !currentCorrect && strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE") {
				t.Log("challenge complete! Tx failed as expected:", err)
				return
			}
			t.Fatal(err)
		}

		confirmLatestBlock(ctx, t, l1Info, l1Backend)

		winner, err := resultReceiver.Winner(&bind.CallOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if winner == (common.Address{}) {
			continue
		}
		if winner != expectedWinner {
			t.Fatal("wrong party won challenge")
		}
	}

	t.Fatal("challenge timed out without winner")
}

func TestFullChallengeAsserterIncorrect(t *testing.T) {
	runChallengeTest(t, false)
}

func TestFullChallengeAsserterCorrect(t *testing.T) {
	runChallengeTest(t, true)
}
