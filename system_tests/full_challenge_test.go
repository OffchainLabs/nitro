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

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/offchainlabs/arbstate/solgen/go/mocksgen"
	"github.com/offchainlabs/arbstate/solgen/go/ospgen"
)

func DeployOneStepProofEntry(t *testing.T, auth *bind.TransactOpts, client bind.ContractBackend, delayedBridge common.Address, seqInbox common.Address) common.Address {
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
	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client, seqInbox, delayedBridge)
	if err != nil {
		t.Fatal(err)
	}
	ospEntry, _, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	if err != nil {
		t.Fatal(err)
	}
	return ospEntry
}

func CreateChallenge(
	t *testing.T,
	auth *bind.TransactOpts,
	client bind.ContractBackend,
	ospEntry common.Address,
	wasmModuleRoot common.Hash,
	startGlobalState arbnode.GoGlobalState,
	endGlobalState arbnode.GoGlobalState,
	numBlocks uint64,
	asserter common.Address,
	challenger common.Address,
) (*mocksgen.MockResultReceiver, common.Address) {
	resultReceiverAddr, _, resultReceiver, err := mocksgen.DeployMockResultReceiver(auth, client)
	if err != nil {
		t.Fatal(err)
	}

	challenge, _, _, err := challengegen.DeployBlockChallenge(
		auth,
		client,
		ospEntry,
		resultReceiverAddr,
		wasmModuleRoot,
		[2]challengegen.GlobalState{
			startGlobalState.AsSolidityStruct(),
			endGlobalState.AsSolidityStruct(),
		},
		numBlocks,
		asserter,
		challenger,
		big.NewInt(100),
		big.NewInt(100),
	)
	if err != nil {
		t.Fatal(err)
	}

	return resultReceiver, challenge
}

func createTransactOpts(t *testing.T) *bind.TransactOpts {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		t.Fatal(err)
	}
	return opts
}

func createGenesisAlloc(accts ...*bind.TransactOpts) core.GenesisAlloc {
	alloc := make(core.GenesisAlloc)
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(20), nil)
	for _, opts := range accts {
		alloc[opts.From] = core.GenesisAccount{
			Balance: new(big.Int).Set(amount),
		}
	}
	return alloc
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

func makeBatch(t *testing.T, l2Node *arbnode.Node, l2Info *BlockchainTestInfo, backend *backends.SimulatedBackend, sequencer *bind.TransactOpts, seqInbox *mocksgen.SequencerInboxStub, seqInboxAddr common.Address, isChallenger bool) {
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

	tx, err := seqInbox.AddSequencerL2BatchFromOrigin(sequencer, big.NewInt(0), batchBuffer.Bytes(), big.NewInt(0), big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
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
	err = l2Node.InboxTracker.AddSequencerBatches(ctx, backend, batches)
	if err != nil {
		t.Fatal(err)
	}
}

func runChallengeTest(t *testing.T, asserterIsCorrect bool) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)

	ctx := context.Background()
	deployer := createTransactOpts(t)
	asserter := createTransactOpts(t)
	challenger := createTransactOpts(t)
	sequencer := createTransactOpts(t)
	alloc := createGenesisAlloc(deployer, asserter, challenger, sequencer)
	backend := backends.NewSimulatedBackend(alloc, 1_000_000_000)
	backend.Commit()

	delayedBridge, _, _, err := mocksgen.DeployBridgeStub(deployer, backend)
	if err != nil {
		t.Fatal(err)
	}

	asserterSeqInboxAddr, _, asserterSeqInbox, err := mocksgen.DeploySequencerInboxStub(deployer, backend, delayedBridge, sequencer.From)
	if err != nil {
		t.Fatal(err)
	}
	challengerSeqInboxAddr, _, challengerSeqInbox, err := mocksgen.DeploySequencerInboxStub(deployer, backend, delayedBridge, sequencer.From)
	if err != nil {
		t.Fatal(err)
	}

	backend.Commit()

	asserterL2Info, asserterL2 := CreateTestL2(t, ctx)
	challengerL2Info, challengerL2 := CreateTestL2(t, ctx)

	asserterL2Info.GenerateAccount("Destination")
	challengerL2Info.SetFullAccountInfo("Destination", asserterL2Info.GetInfoWithPrivKey("Destination"))
	makeBatch(t, asserterL2, asserterL2Info, backend, sequencer, asserterSeqInbox, asserterSeqInboxAddr, false)
	makeBatch(t, asserterL2, asserterL2Info, backend, sequencer, challengerSeqInbox, challengerSeqInboxAddr, false)

	trueSeqInboxAddr := challengerSeqInboxAddr
	expectedWinner := challenger.From
	if asserterIsCorrect {
		trueSeqInboxAddr = asserterSeqInboxAddr
		expectedWinner = asserter.From
	}
	ospEntry := DeployOneStepProofEntry(t, deployer, backend, delayedBridge, trueSeqInboxAddr)
	backend.Commit()

	wasmModuleRoot := asserterL2.BlockValidator.GetInitialModuleRoot()

	asserterGenesis := asserterL2.ArbInterface.BlockChain().Genesis()
	asserterLatestBlock := asserterL2.ArbInterface.BlockChain().CurrentBlock()

	asserterStartGlobalState := arbnode.GoGlobalState{
		BlockHash:  asserterGenesis.Hash(),
		Batch:      0,
		PosInBatch: 0,
	}
	asserterEndGlobalState := arbnode.GoGlobalState{
		BlockHash:  asserterLatestBlock.Hash(),
		Batch:      1,
		PosInBatch: 0,
	}
	numBlocks := asserterLatestBlock.NumberU64() - asserterGenesis.NumberU64()

	resultReceiver, challenge := CreateChallenge(
		t,
		deployer,
		backend,
		ospEntry,
		wasmModuleRoot,
		asserterStartGlobalState,
		asserterEndGlobalState,
		numBlocks,
		asserter.From,
		challenger.From,
	)

	backend.Commit()

	asserterManager, err := arbnode.NewFullChallengeManager(ctx, asserterL2, backend, asserter, challenge, 0, 4)
	if err != nil {
		t.Fatal(err)
	}

	challengerManager, err := arbnode.NewFullChallengeManager(ctx, challengerL2, backend, challenger, challenge, 0, 4)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			_, err = challengerManager.Act(ctx)
			if err != nil {
				if asserterIsCorrect && strings.Contains(err.Error(), "SAME_OSP_END") {
					t.Log("challenge completed! challenger hit expected error:", err)
					return
				}
				t.Fatal(err)
			}
		} else {
			_, err = asserterManager.Act(ctx)
			if err != nil {
				if !asserterIsCorrect && strings.Contains(err.Error(), "lost challenge") {
					t.Log("challenge completed! asserter hit expected error:", err)
					return
				}
				t.Fatal(err)
			}
		}
		backend.Commit()

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
