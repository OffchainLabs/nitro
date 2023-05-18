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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func DeployOneStepProofEntry(t *testing.T, ctx context.Context, auth *bind.TransactOpts, client *ethclient.Client) common.Address {
	osp0, _, _, err := ospgen.DeployOneStepProver0(auth, client)
	if err != nil {
		Fail(t, err)
	}
	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	if err != nil {
		Fail(t, err)
	}
	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	if err != nil {
		Fail(t, err)
	}
	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	if err != nil {
		Fail(t, err)
	}
	ospEntry, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	if err != nil {
		Fail(t, err)
	}
	_, err = EnsureTxSucceeded(ctx, client, tx)
	if err != nil {
		Fail(t, err)
	}
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
			staker.StatusFinished,
			staker.StatusFinished,
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

func makeBatch(t *testing.T, l2Node *arbnode.Node, l2Info *BlockchainTestInfo, backend *ethclient.Client, sequencer *bind.TransactOpts, seqInbox *mocksgen.SequencerInboxStub, seqInboxAddr common.Address, isChallenger bool) {
	ctx := context.Background()

	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < 10; i++ {
		value := i
		if i == 5 && isChallenger {
			value++
		}
		err := writeTxToBatch(batchBuffer, l2Info.PrepareTx("Owner", "Destination", 1000000, big.NewInt(value), []byte{}))
		Require(t, err)
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)
	message := append([]byte{0}, compressed...)

	maxUint256 := new(big.Int).Lsh(common.Big1, 256)
	maxUint256.Sub(maxUint256, common.Big1)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin0(sequencer, maxUint256, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	nodeSeqInbox, err := arbnode.NewSequencerInbox(backend, seqInboxAddr, 0)
	Require(t, err)
	batches, err := nodeSeqInbox.LookupBatchesInRange(ctx, receipt.BlockNumber, receipt.BlockNumber)
	Require(t, err)
	if len(batches) == 0 {
		Fail(t, "batch not found after AddSequencerL2BatchFromOrigin")
	}
	err = l2Node.InboxTracker.AddSequencerBatches(ctx, backend, batches)
	Require(t, err)
	_, err = l2Node.InboxTracker.GetBatchMetadata(0)
	Require(t, err, "failed to get batch metadata after adding batch:")
}

func confirmLatestBlock(ctx context.Context, t *testing.T, l1Info *BlockchainTestInfo, backend arbutil.L1Interface) {
	for i := 0; i < 12; i++ {
		SendWaitTestTransactions(t, ctx, backend, []*types.Transaction{
			l1Info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
}

func setupSequencerInboxStub(ctx context.Context, t *testing.T, l1Info *BlockchainTestInfo, l1Client arbutil.L1Interface, chainConfig *params.ChainConfig) (common.Address, *mocksgen.SequencerInboxStub, common.Address) {
	txOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)
	bridgeAddr, tx, bridge, err := mocksgen.DeployBridgeUnproxied(&txOpts, l1Client)
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

func RunChallengeTest(t *testing.T, asserterIsCorrect bool) {
	t.Parallel()
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

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	configByValidationNode(t, conf, valStack)

	fatalErrChan := make(chan error, 10)
	asserterRollupAddresses := DeployOnTestL1(t, ctx, l1Info, l1Backend, chainConfig)

	deployerTxOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)
	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)
	asserterTxOpts := l1Info.GetDefaultTransactOpts("asserter", ctx)
	challengerTxOpts := l1Info.GetDefaultTransactOpts("challenger", ctx)

	asserterBridgeAddr, asserterSeqInbox, asserterSeqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)
	challengerBridgeAddr, challengerSeqInbox, challengerSeqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)

	asserterL2Info, asserterL2Stack, asserterL2ChainDb, asserterL2ArbDb, asserterL2Blockchain := createL2BlockChain(t, nil, "", chainConfig)
	asserterRollupAddresses.Bridge = asserterBridgeAddr
	asserterRollupAddresses.SequencerInbox = asserterSeqInboxAddr
	asserterL2, err := arbnode.CreateNode(ctx, asserterL2Stack, asserterL2ChainDb, asserterL2ArbDb, NewFetcherFromConfig(conf), asserterL2Blockchain, l1Backend, asserterRollupAddresses, nil, nil, nil, fatalErrChan)
	Require(t, err)
	err = asserterL2.Start(ctx)
	Require(t, err)

	challengerL2Info, challengerL2Stack, challengerL2ChainDb, challengerL2ArbDb, challengerL2Blockchain := createL2BlockChain(t, nil, "", chainConfig)
	challengerRollupAddresses := *asserterRollupAddresses
	challengerRollupAddresses.Bridge = challengerBridgeAddr
	challengerRollupAddresses.SequencerInbox = challengerSeqInboxAddr
	challengerL2, err := arbnode.CreateNode(ctx, challengerL2Stack, challengerL2ChainDb, challengerL2ArbDb, NewFetcherFromConfig(conf), challengerL2Blockchain, l1Backend, &challengerRollupAddresses, nil, nil, nil, fatalErrChan)
	Require(t, err)
	err = challengerL2.Start(ctx)
	Require(t, err)

	asserterL2Info.GenerateAccount("Destination")
	challengerL2Info.SetFullAccountInfo("Destination", asserterL2Info.GetInfoWithPrivKey("Destination"))
	makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, false)
	makeBatch(t, challengerL2, challengerL2Info, l1Backend, &sequencerTxOpts, challengerSeqInbox, challengerSeqInboxAddr, true)

	trueSeqInboxAddr := challengerSeqInboxAddr
	trueDelayedBridge := challengerBridgeAddr
	expectedWinner := l1Info.GetAddress("challenger")
	if asserterIsCorrect {
		trueSeqInboxAddr = asserterSeqInboxAddr
		trueDelayedBridge = asserterBridgeAddr
		expectedWinner = l1Info.GetAddress("asserter")
	}
	ospEntry := DeployOneStepProofEntry(t, ctx, &deployerTxOpts, l1Backend)

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		Fail(t, err)
	}
	wasmModuleRoot := locator.LatestWasmModuleRoot()
	if (wasmModuleRoot == common.Hash{}) {
		Fail(t, "latest machine not found")
	}

	asserterGenesis := asserterL2.Execution.ArbInterface.BlockChain().Genesis()
	challengerGenesis := challengerL2.Execution.ArbInterface.BlockChain().Genesis()
	if asserterGenesis.Hash() != challengerGenesis.Hash() {
		Fail(t, "asserter and challenger have different genesis hashes")
	}
	asserterLatestBlock := asserterL2.Execution.ArbInterface.BlockChain().CurrentBlock()
	challengerLatestBlock := challengerL2.Execution.ArbInterface.BlockChain().CurrentBlock()
	if asserterLatestBlock.Hash() == challengerLatestBlock.Hash() {
		Fail(t, "asserter and challenger have the same end block")
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

	asserterValidator, err := staker.NewStatelessBlockValidator(asserterL2.InboxReader, asserterL2.InboxTracker, asserterL2.TxStreamer, asserterL2Blockchain, asserterL2ChainDb, asserterL2ArbDb, nil, &conf.BlockValidator)
	if err != nil {
		Fail(t, err)
	}
	err = asserterValidator.Start(ctx)
	if err != nil {
		Fail(t, err)
	}
	defer asserterValidator.Stop()
	asserterManager, err := staker.NewChallengeManager(ctx, l1Backend, &asserterTxOpts, asserterTxOpts.From, challengeManagerAddr, 1, asserterL2Blockchain, asserterL2.InboxTracker, asserterValidator, 0, 0)
	if err != nil {
		Fail(t, err)
	}
	challengerValidator, err := staker.NewStatelessBlockValidator(challengerL2.InboxReader, challengerL2.InboxTracker, challengerL2.TxStreamer, challengerL2Blockchain, challengerL2ChainDb, challengerL2ArbDb, nil, &conf.BlockValidator)
	if err != nil {
		Fail(t, err)
	}
	err = challengerValidator.Start(ctx)
	if err != nil {
		Fail(t, err)
	}
	defer challengerValidator.Stop()
	challengerManager, err := staker.NewChallengeManager(ctx, l1Backend, &challengerTxOpts, challengerTxOpts.From, challengeManagerAddr, 1, challengerL2Blockchain, challengerL2.InboxTracker, challengerValidator, 0, 0)
	if err != nil {
		Fail(t, err)
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
			Fail(t, "challenge step", i, "hit error:", err)
		}
		if tx == nil {
			Fail(t, "no move")
		}
		_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
		if err != nil {
			if !currentCorrect && strings.Contains(err.Error(), "BAD_SEQINBOX_MESSAGE") {
				t.Log("challenge complete! Tx failed as expected:", err)
				return
			}
			Fail(t, err)
		}

		confirmLatestBlock(ctx, t, l1Info, l1Backend)

		winner, err := resultReceiver.Winner(&bind.CallOpts{})
		if err != nil {
			Fail(t, err)
		}
		if winner == (common.Address{}) {
			continue
		}
		if winner != expectedWinner {
			Fail(t, "wrong party won challenge")
		}
	}

	Fail(t, "challenge timed out without winner")
}
