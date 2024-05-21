// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/celestia"
	celestiaTypes "github.com/offchainlabs/nitro/das/celestia/types"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"

	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func init() {
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

// TODO:
// Find a way to trigger the other two cases
// add Fee stuff
// Cleanup code
// make release, preimage oracle, write up, send to Ottersec

func DeployOneStepProofEntryCelestia(t *testing.T, ctx context.Context, auth *bind.TransactOpts, client *ethclient.Client) common.Address {
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	ospMem, tx, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	ospMath, tx, _, err := ospgen.DeployOneStepProverMath(auth, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	ospHostIo, tx, _, err := mocksgen.DeployOneStepProverHostIoCelestiaMock(auth, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	ospEntry, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	return ospEntry
}

func writeTxToCelestiaBatch(writer io.Writer, tx *types.Transaction) error {
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

func makeCelestiaBatch(t *testing.T, l2Node *arbnode.Node, celestiaDA *celestia.CelestiaDA, undecided bool, counterfactual bool, mockStream *mocksgen.Mockstream, deployer *bind.TransactOpts, l2Info *BlockchainTestInfo, backend *ethclient.Client, sequencer *bind.TransactOpts, seqInbox *mocksgen.SequencerInboxStub, seqInboxAddr common.Address, modStep int64) {
	ctx := context.Background()

	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < makeBatch_MsgsPerBatch; i++ {
		value := i
		if i == modStep {
			value++
		}
		err := writeTxToCelestiaBatch(batchBuffer, l2Info.PrepareTx("Owner", "Destination", 1000000, big.NewInt(value), []byte{}))
		Require(t, err)
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)
	message := append([]byte{0}, compressed...)
	message, err = celestiaDA.Store(ctx, message)
	Require(t, err)

	buf := bytes.NewBuffer(message)

	header, err := buf.ReadByte()
	Require(t, err)
	if !celestia.IsCelestiaMessageHeaderByte(header) {
		err := errors.New("tried to deserialize a message that doesn't have the Celestia header")
		Require(t, err)
	}

	blobPointer := celestiaTypes.BlobPointer{}
	blobBytes := buf.Bytes()
	err = blobPointer.UnmarshalBinary(blobBytes)
	Require(t, err)

	dataCommitment, err := celestiaDA.Prover.Trpc.DataCommitment(ctx, blobPointer.BlockHeight-1, blobPointer.BlockHeight+1)
	if err != nil {
		t.Log("Error when fetching data commitment:", err)
	}
	Require(t, err)
	mockStream.SubmitDataCommitment(deployer, [32]byte(dataCommitment.DataCommitment), blobPointer.BlockHeight-1, blobPointer.BlockHeight+1)
	if counterfactual {
		mockStream.UpdateGenesisState(deployer, (blobPointer.BlockHeight - 1100))
	} else if undecided {
		t.Log("Block Height before change: ", blobPointer.BlockHeight)
		mockStream.UpdateGenesisState(deployer, (blobPointer.BlockHeight - 100))
	}
	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin0(sequencer, seqNum, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	nodeSeqInbox, err := arbnode.NewSequencerInbox(backend, seqInboxAddr, 0)
	Require(t, err)
	batches, err := nodeSeqInbox.LookupBatchesInRange(ctx, receipt.BlockNumber, receipt.BlockNumber)
	Require(t, err)
	if len(batches) == 0 {
		Fatal(t, "batch not found after AddSequencerL2BatchFromOrigin")
	}
	err = l2Node.InboxTracker.AddSequencerBatches(ctx, backend, batches)
	Require(t, err)
	_, err = l2Node.InboxTracker.GetBatchMetadata(0)
	Require(t, err, "failed to get batch metadata after adding batch:")
}

func RunCelestiaChallengeTest(t *testing.T, asserterIsCorrect bool, useStubs bool, challengeMsgIdx int64, undecided bool, counterFactual bool) {

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info := NewL1TestInfo(t)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("challenger", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)

	chainConfig := params.ArbitrumDevTestChainConfig()
	l1Info, l1Backend, _, _ := createTestL1BlockChain(t, l1Info)
	conf := arbnode.ConfigDefaultL1Test()
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false
	conf.InboxReader.CheckDelay = time.Second
	chainConfig.ArbitrumChainParams.CelestiaDA = true

	deployerTxOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)
	blobstream, tx, mockStreamWrapper, err := mocksgen.DeployMockstream(&deployerTxOpts, l1Backend)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)

	conf.Celestia = celestia.DAConfig{
		Enable:      true,
		GasPrice:    0.1,
		Rpc:         "http://localhost:26658",
		NamespaceId: "000008e5f679bf7116cb",
		AuthToken:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJwdWJsaWMiLCJyZWFkIiwid3JpdGUiLCJhZG1pbiJdfQ.8iCpZJaiui7QPTCj4m5f2M7JyHkJtr6Xha0bmE5Vv7Y",
		ValidatorConfig: &celestia.ValidatorConfig{
			TendermintRPC:  "http://localhost:26657",
			BlobstreamAddr: blobstream.Hex(),
		},
	}

	t.Log("Blobstream Address: ", blobstream.Hex())

	celestiaDa, err := celestia.NewCelestiaDA(&conf.Celestia, l1Backend)
	Require(t, err)
	// Initialize Mockstream before the tests
	header, err := celestiaDa.Client.Header.NetworkHead(ctx)
	Require(t, err)
	mockStreamWrapper.Initialize(&deployerTxOpts, header.Height())

	var valStack *node.Node
	var mockSpawn *mockSpawner
	if useStubs {
		mockSpawn, valStack = createMockValidationNode(t, ctx, &valnode.TestValidationConfig.Arbitrator)
	} else {
		_, valStack = createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	}
	configByValidationNode(t, conf, valStack)

	fatalErrChan := make(chan error, 10)
	asserterRollupAddresses, initMessage := DeployOnTestL1(t, ctx, l1Info, l1Backend, chainConfig)

	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)
	asserterTxOpts := l1Info.GetDefaultTransactOpts("asserter", ctx)
	challengerTxOpts := l1Info.GetDefaultTransactOpts("challenger", ctx)

	asserterBridgeAddr, asserterSeqInbox, asserterSeqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)
	challengerBridgeAddr, challengerSeqInbox, challengerSeqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)

	asserterL2Info, asserterL2Stack, asserterL2ChainDb, asserterL2ArbDb, asserterL2Blockchain := createL2BlockChainWithStackConfig(t, nil, "", chainConfig, initMessage, nil, nil)
	asserterRollupAddresses.Bridge = asserterBridgeAddr
	asserterRollupAddresses.SequencerInbox = asserterSeqInboxAddr
	asserterExec, err := gethexec.CreateExecutionNode(ctx, asserterL2Stack, asserterL2ChainDb, asserterL2Blockchain, l1Backend, gethexec.ConfigDefaultTest)
	Require(t, err)
	parentChainID := big.NewInt(1337)
	asserterL2, err := arbnode.CreateNode(ctx, asserterL2Stack, asserterExec, asserterL2ArbDb, NewFetcherFromConfig(conf), chainConfig, l1Backend, asserterRollupAddresses, nil, nil, nil, fatalErrChan, parentChainID, nil)
	Require(t, err)
	err = asserterL2.Start(ctx)
	Require(t, err)

	challengerL2Info, challengerL2Stack, challengerL2ChainDb, challengerL2ArbDb, challengerL2Blockchain := createL2BlockChainWithStackConfig(t, nil, "", chainConfig, initMessage, nil, nil)
	challengerRollupAddresses := *asserterRollupAddresses
	challengerRollupAddresses.Bridge = challengerBridgeAddr
	challengerRollupAddresses.SequencerInbox = challengerSeqInboxAddr
	challengerExec, err := gethexec.CreateExecutionNode(ctx, challengerL2Stack, challengerL2ChainDb, challengerL2Blockchain, l1Backend, gethexec.ConfigDefaultTest)
	Require(t, err)
	challengerL2, err := arbnode.CreateNode(ctx, challengerL2Stack, challengerExec, challengerL2ArbDb, NewFetcherFromConfig(conf), chainConfig, l1Backend, &challengerRollupAddresses, nil, nil, nil, fatalErrChan, parentChainID, nil)
	Require(t, err)
	err = challengerL2.Start(ctx)
	Require(t, err)

	asserterL2Info.GenerateAccount("Destination")
	challengerL2Info.SetFullAccountInfo("Destination", asserterL2Info.GetInfoWithPrivKey("Destination"))

	if challengeMsgIdx < 1 || challengeMsgIdx > 3*makeBatch_MsgsPerBatch {
		Fatal(t, "challengeMsgIdx illegal")
	}

	// seqNum := common.Big2
	makeCelestiaBatch(t, asserterL2, celestiaDa, undecided, counterFactual, mockStreamWrapper, &deployerTxOpts, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)
	makeCelestiaBatch(t, challengerL2, celestiaDa, undecided, counterFactual, mockStreamWrapper, &deployerTxOpts, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, challengeMsgIdx-1)

	// seqNum.Add(seqNum, common.Big1)
	makeCelestiaBatch(t, asserterL2, celestiaDa, undecided, counterFactual, mockStreamWrapper, &deployerTxOpts, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)
	makeCelestiaBatch(t, challengerL2, celestiaDa, undecided, counterFactual, mockStreamWrapper, &deployerTxOpts, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, challengeMsgIdx-makeBatch_MsgsPerBatch-1)

	// seqNum.Add(seqNum, common.Big1)
	makeCelestiaBatch(t, asserterL2, celestiaDa, undecided, counterFactual, mockStreamWrapper, &deployerTxOpts, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)
	makeCelestiaBatch(t, challengerL2, celestiaDa, undecided, counterFactual, mockStreamWrapper, &deployerTxOpts, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, challengeMsgIdx-makeBatch_MsgsPerBatch*2-1)

	trueSeqInboxAddr := challengerSeqInboxAddr
	trueDelayedBridge := challengerBridgeAddr
	expectedWinner := l1Info.GetAddress("challenger")
	if asserterIsCorrect {
		trueSeqInboxAddr = asserterSeqInboxAddr
		trueDelayedBridge = asserterBridgeAddr
		expectedWinner = l1Info.GetAddress("asserter")
	}
	ospEntry := DeployOneStepProofEntryCelestia(t, ctx, &deployerTxOpts, l1Backend)

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		Fatal(t, err)
	}
	var wasmModuleRoot common.Hash
	if useStubs {
		wasmModuleRoot = mockWasmModuleRoot
	} else {
		wasmModuleRoot = locator.LatestWasmModuleRoot()
		if (wasmModuleRoot == common.Hash{}) {
			Fatal(t, "latest machine not found")
		}
	}

	asserterGenesis := asserterExec.ArbInterface.BlockChain().Genesis()
	challengerGenesis := challengerExec.ArbInterface.BlockChain().Genesis()
	if asserterGenesis.Hash() != challengerGenesis.Hash() {
		Fatal(t, "asserter and challenger have different genesis hashes")
	}
	asserterLatestBlock := asserterExec.ArbInterface.BlockChain().CurrentBlock()
	challengerLatestBlock := challengerExec.ArbInterface.BlockChain().CurrentBlock()
	if asserterLatestBlock.Hash() == challengerLatestBlock.Hash() {
		Fatal(t, "asserter and challenger have the same end block")
	}

	asserterStartGlobalState := validator.GoGlobalState{
		BlockHash:  asserterGenesis.Hash(),
		Batch:      1,
		PosInBatch: 0,
	}
	asserterEndGlobalState := validator.GoGlobalState{
		BlockHash:  asserterLatestBlock.Hash(),
		Batch:      4,
		PosInBatch: 0,
	}
	numBlocks := asserterLatestBlock.Number.Uint64() - asserterGenesis.NumberU64()

	resultReceiver, challengeManagerAddr := CreateChallenge(
		t,
		ctx,
		&deployerTxOpts,
		l1Backend,
		ospEntry,
		trueSeqInboxAddr,
		trueDelayedBridge,
		wasmModuleRoot,
		asserterStartGlobalState,
		asserterEndGlobalState,
		numBlocks,
		l1Info.GetAddress("asserter"),
		l1Info.GetAddress("challenger"),
	)

	confirmLatestBlock(ctx, t, l1Info, l1Backend)

	// Add the L1 backend to Celestia DA
	celestiaDa.Prover.EthClient = l1Backend

	asserterValidator, err := staker.NewStatelessBlockValidator(asserterL2.InboxReader, asserterL2.InboxTracker, asserterL2.TxStreamer, asserterExec.Recorder, asserterL2ArbDb, nil, nil, celestiaDa, StaticFetcherFrom(t, &conf.BlockValidator), valStack)
	if err != nil {
		Fatal(t, err)
	}
	if useStubs {
		asserterRecorder := newMockRecorder(asserterValidator, asserterL2.TxStreamer)
		asserterValidator.OverrideRecorder(t, asserterRecorder)
	}
	err = asserterValidator.Start(ctx)
	if err != nil {
		Fatal(t, err)
	}
	defer asserterValidator.Stop()
	asserterManager, err := staker.NewChallengeManager(ctx, l1Backend, &asserterTxOpts, asserterTxOpts.From, challengeManagerAddr, 1, asserterValidator, 0, 0)
	if err != nil {
		Fatal(t, err)
	}
	challengerValidator, err := staker.NewStatelessBlockValidator(challengerL2.InboxReader, challengerL2.InboxTracker, challengerL2.TxStreamer, challengerExec.Recorder, challengerL2ArbDb, nil, nil, celestiaDa, StaticFetcherFrom(t, &conf.BlockValidator), valStack)
	if err != nil {
		Fatal(t, err)
	}
	if useStubs {
		challengerRecorder := newMockRecorder(challengerValidator, challengerL2.TxStreamer)
		challengerValidator.OverrideRecorder(t, challengerRecorder)
	}
	err = challengerValidator.Start(ctx)
	if err != nil {
		Fatal(t, err)
	}
	defer challengerValidator.Stop()
	challengerManager, err := staker.NewChallengeManager(ctx, l1Backend, &challengerTxOpts, challengerTxOpts.From, challengeManagerAddr, 1, challengerValidator, 0, 0)
	if err != nil {
		Fatal(t, err)
	}

	confirmLatestBlock(ctx, t, l1Info, l1Backend)

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
				strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE")) ||
				strings.Contains(err.Error(), "BLOBSTREAM_UNDECIDED") {
				t.Log("challenge completed! asserter hit expected error:", err)
				return
			} else if (currentCorrect && counterFactual) && strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE") {
				t.Log("counterfactual challenge challenge completed! asserter hit expected error:", err)
				return
			}
			Fatal(t, "challenge step", i, "hit error:", err)
		}
		if tx == nil {
			Fatal(t, "no move")
		}

		if useStubs {
			if len(mockSpawn.ExecSpawned) != 0 {
				if len(mockSpawn.ExecSpawned) != 1 {
					Fatal(t, "bad number of spawned execRuns: ", len(mockSpawn.ExecSpawned))
				}
				if mockSpawn.ExecSpawned[0] != uint64(challengeMsgIdx) {
					Fatal(t, "wrong spawned execRuns: ", mockSpawn.ExecSpawned[0], " expected: ", challengeMsgIdx)
				}
				return
			}
		}

		_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
		if err != nil {
			if !currentCorrect && strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE") {
				t.Log("challenge complete! Tx failed as expected:", err)
				return
			}
			Fatal(t, err)
		}

		confirmLatestBlock(ctx, t, l1Info, l1Backend)

		winner, err := resultReceiver.Winner(&bind.CallOpts{})
		if err != nil {
			Fatal(t, err)
		}
		if winner == (common.Address{}) {
			continue
		}
		if winner != expectedWinner {
			Fatal(t, "wrong party won challenge")
		}
	}

	Fatal(t, "challenge timed out without winner")
}

func TestCelestiaChallengeManagerFullAsserterIncorrect(t *testing.T) {
	t.Parallel()
	RunCelestiaChallengeTest(t, false, false, makeBatch_MsgsPerBatch+1, false, false)
}

func TestCelestiaChallengeManagerFullAsserterCorrect(t *testing.T) {
	t.Parallel()
	RunCelestiaChallengeTest(t, true, false, makeBatch_MsgsPerBatch+2, false, false)
}

func TestCelestiaChallengeManagerFullAsserterIncorrectUndecided(t *testing.T) {
	t.Parallel()
	RunCelestiaChallengeTest(t, false, false, makeBatch_MsgsPerBatch+1, true, false)
}

func TestCelestiaChallengeManagerFullAsserterCorrectUndecided(t *testing.T) {
	t.Parallel()
	RunCelestiaChallengeTest(t, true, false, makeBatch_MsgsPerBatch+2, true, false)
}

func TestCelestiaChallengeManagerFullAsserterIncorrectCounterfactual(t *testing.T) {
	t.Parallel()
	RunCelestiaChallengeTest(t, false, false, makeBatch_MsgsPerBatch+1, false, true)
}

func TestCelestiaChallengeManagerFullAsserterCorrectCounterfactual(t *testing.T) {
	t.Parallel()
	RunCelestiaChallengeTest(t, true, false, makeBatch_MsgsPerBatch+2, false, true)
}
