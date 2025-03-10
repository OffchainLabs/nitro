// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/solgen/go/yulgen"
	"github.com/offchainlabs/nitro/staker"
	legacystaker "github.com/offchainlabs/nitro/staker/legacy"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func DeployOneStepProofEntry(t *testing.T, ctx context.Context, auth *bind.TransactOpts, client *ethclient.Client) common.Address {
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

	ospHostIo, tx, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	ospEntry, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	return ospEntry
}

func CreateChallenge(
	t *testing.T,
	ctx context.Context,
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
	challengeManagerLogic, tx, _, err := challengegen.DeployChallengeManager(auth, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	challengeManagerAddr, tx, _, err := mocksgen.DeploySimpleProxy(auth, client, challengeManagerLogic)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	challengeManager, err := challengegen.NewChallengeManager(challengeManagerAddr, client)
	Require(t, err)

	resultReceiverAddr, _, resultReceiver, err := mocksgen.DeployMockResultReceiver(auth, client, challengeManagerAddr)
	Require(t, err)
	tx, err = challengeManager.Initialize(auth, resultReceiverAddr, sequencerInbox, delayedBridge, ospEntry)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	tx, err = resultReceiver.CreateChallenge(
		auth,
		wasmModuleRoot,
		[2]uint8{
			legacystaker.StatusFinished,
			legacystaker.StatusFinished,
		},
		[2]mocksgen.GlobalState{
			{
				Bytes32Vals: [2][32]byte{startGlobalState.BlockHash, startGlobalState.SendRoot},
				U64Vals:     [2]uint64{startGlobalState.Batch, startGlobalState.PosInBatch},
			},
			{
				Bytes32Vals: [2][32]byte{endGlobalState.BlockHash, endGlobalState.SendRoot},
				U64Vals:     [2]uint64{endGlobalState.Batch, endGlobalState.PosInBatch},
			},
		},
		numBlocks,
		asserter,
		challenger,
		big.NewInt(100000),
		big.NewInt(100000),
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	return resultReceiver, challengeManagerAddr
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

const makeBatch_MsgsPerBatch = int64(5)

func makeBatch(t *testing.T, l2Node *arbnode.Node, l2Info *BlockchainTestInfo, backend *ethclient.Client, sequencer *bind.TransactOpts, seqInbox *mocksgen.SequencerInboxStub, seqInboxAddr common.Address, modStep int64) {
	ctx := context.Background()

	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < makeBatch_MsgsPerBatch; i++ {
		value := i
		if i == modStep {
			value++
		}
		err := writeTxToBatch(batchBuffer, l2Info.PrepareTx("Owner", "Destination", 1000000, big.NewInt(value), []byte{}))
		Require(t, err)
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)
	message := append([]byte{0}, compressed...)

	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(sequencer, seqNum, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
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

func confirmLatestBlock(ctx context.Context, t *testing.T, l1Info *BlockchainTestInfo, backend *ethclient.Client) {
	t.Helper()
	// With SimulatedBeacon running in on-demand block production mode, the
	// finalized block is considered to be be the nearest multiple of 32 less
	// than or equal to the block number.
	for i := 0; i < 32; i++ {
		SendWaitTestTransactions(t, ctx, backend, []*types.Transaction{
			l1Info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
}

func setupSequencerInboxStub(ctx context.Context, t *testing.T, l1Info *BlockchainTestInfo, l1Client *ethclient.Client, chainConfig *params.ChainConfig) (common.Address, *mocksgen.SequencerInboxStub, common.Address) {
	txOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)
	bridgeAddr, tx, bridge, err := mocksgen.DeployBridgeUnproxied(&txOpts, l1Client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)
	reader4844, tx, _, err := yulgen.DeployReader4844(&txOpts, l1Client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)
	timeBounds := mocksgen.ISequencerInboxMaxTimeVariation{
		DelayBlocks:   big.NewInt(10000),
		FutureBlocks:  big.NewInt(10000),
		DelaySeconds:  big.NewInt(10000),
		FutureSeconds: big.NewInt(10000),
	}
	seqInboxAddr, tx, seqInbox, err := mocksgen.DeploySequencerInboxStub(
		&txOpts,
		l1Client,
		bridgeAddr,
		l1Info.GetAddress("sequencer"),
		timeBounds,
		big.NewInt(117964),
		reader4844,
		false,
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)
	tx, err = bridge.SetSequencerInbox(&txOpts, seqInboxAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)
	tx, err = bridge.SetDelayedInbox(&txOpts, seqInboxAddr, true)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)
	tx, err = seqInbox.AddInitMessage(&txOpts, chainConfig.ChainID)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)
	return bridgeAddr, seqInbox, seqInboxAddr
}

func RunChallengeTest(t *testing.T, asserterIsCorrect bool, useStubs bool, challengeMsgIdx int64, wasmRootDir string) {
	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LvlInfo)
	log.SetDefault(log.NewLogger(glogger))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info := builder.L1Info
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("challenger", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)

	chainConfig := builder.chainConfig
	conf := builder.nodeConfig
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false
	conf.InboxReader.CheckDelay = time.Second

	var valStack *node.Node
	var mockSpawn *mockSpawner
	builder.valnodeConfig.Wasm.RootPath = wasmRootDir
	if useStubs {
		mockSpawn, valStack = createMockValidationNode(t, ctx, &builder.valnodeConfig.Arbitrator)
	} else {
		// For now validation only works with HashScheme set
		builder.execConfig.Caching.StateScheme = rawdb.HashScheme
		_, valStack = createTestValidationNode(t, ctx, builder.valnodeConfig)
	}
	configByValidationNode(conf, valStack)

	builder.BuildL1(t)
	l1Backend := builder.L1.Client

	deployerTxOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)
	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)
	asserterTxOpts := l1Info.GetDefaultTransactOpts("asserter", ctx)
	challengerTxOpts := l1Info.GetDefaultTransactOpts("challenger", ctx)

	asserterBridgeAddr, asserterSeqInbox, asserterSeqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)
	challengerBridgeAddr, challengerSeqInbox, challengerSeqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)

	asserterRollupAddresses := builder.addresses
	asserterRollupAddresses.Bridge = asserterBridgeAddr
	asserterRollupAddresses.SequencerInbox = asserterSeqInboxAddr

	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()
	asserterL2 := builder.L2.ConsensusNode
	asserterL2Info := builder.L2Info
	asserterExec := builder.L2.ExecNode

	challengerRollupAddresses := *builder.addresses
	challengerRollupAddresses.Bridge = challengerBridgeAddr
	challengerRollupAddresses.SequencerInbox = challengerSeqInboxAddr
	challengerL2Info := NewArbTestInfo(t, chainConfig.ChainID)
	challengerParams := SecondNodeParams{
		addresses: &challengerRollupAddresses,
		initData:  &challengerL2Info.ArbInitData,
	}
	challenger, challengerCleanup := builder.Build2ndNode(t, &challengerParams)
	defer challengerCleanup()
	challengerL2 := challenger.ConsensusNode
	challengerExec := challenger.ExecNode

	asserterL2Info.GenerateAccount("Destination")
	challengerL2Info.SetFullAccountInfo("Destination", asserterL2Info.GetInfoWithPrivKey("Destination"))

	if challengeMsgIdx < 1 || challengeMsgIdx > 3*makeBatch_MsgsPerBatch {
		Fatal(t, "challengeMsgIdx illegal")
	}

	// seqNum := common.Big2
	makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)
	makeBatch(t, challengerL2, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, challengeMsgIdx-1)

	// seqNum.Add(seqNum, common.Big1)
	makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)
	makeBatch(t, challengerL2, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, challengeMsgIdx-makeBatch_MsgsPerBatch-1)

	// seqNum.Add(seqNum, common.Big1)
	makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)
	makeBatch(t, challengerL2, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, challengeMsgIdx-makeBatch_MsgsPerBatch*2-1)

	trueSeqInboxAddr := challengerSeqInboxAddr
	trueDelayedBridge := challengerBridgeAddr
	expectedWinner := l1Info.GetAddress("challenger")
	if asserterIsCorrect {
		trueSeqInboxAddr = asserterSeqInboxAddr
		trueDelayedBridge = asserterBridgeAddr
		expectedWinner = l1Info.GetAddress("asserter")
	}
	ospEntry := DeployOneStepProofEntry(t, ctx, &deployerTxOpts, l1Backend)

	var wasmModuleRoot common.Hash
	if useStubs {
		wasmModuleRoot = mockWasmModuleRoots[0]
	} else {
		locator, err := server_common.NewMachineLocator(wasmRootDir)
		Require(t, err)
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

	asserterValidator, err := staker.NewStatelessBlockValidator(asserterL2.InboxReader, asserterL2.InboxTracker, asserterL2.TxStreamer, asserterExec.Recorder, asserterL2.ArbDB, nil, StaticFetcherFrom(t, &conf.BlockValidator), valStack)
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
	asserterManager, err := legacystaker.NewChallengeManager(ctx, l1Backend, &asserterTxOpts, asserterTxOpts.From, challengeManagerAddr, 1, asserterValidator, 0, 0)
	if err != nil {
		Fatal(t, err)
	}
	challengerValidator, err := staker.NewStatelessBlockValidator(challengerL2.InboxReader, challengerL2.InboxTracker, challengerL2.TxStreamer, challengerExec.Recorder, challengerL2.ArbDB, nil, StaticFetcherFrom(t, &conf.BlockValidator), valStack)
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
	challengerManager, err := legacystaker.NewChallengeManager(ctx, l1Backend, &challengerTxOpts, challengerTxOpts.From, challengeManagerAddr, 1, challengerValidator, 0, 0)
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
				strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE")) {
				t.Log("challenge completed! asserter hit expected error:", err)
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
