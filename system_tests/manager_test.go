package arbtest

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/node"
	"math/big"
	"reflect"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/valnode"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
)

func TestManager(t *testing.T) {
	managerTestImpl(t, true, false)
}

func managerTestImpl(t *testing.T, faultyStaker bool, honestStakerInactive bool) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	_, l2nodeA, l2clientA, _, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfigImpl(t, ctx, true, nil, nil, l2chainConfig, nil, l2info)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()
	execNodeA := getExecNode(t, l2nodeA)

	if faultyStaker {
		l2info.GenerateGenesisAccount("FaultyAddr", common.Big1)
	}
	l2clientB, l2nodeB := Create2ndNodeWithConfig(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, arbnode.ConfigDefaultL1Test(), gethexec.ConfigDefaultTest(), nil)
	defer l2nodeB.StopAndWait()
	execNodeB := getExecNode(t, l2nodeB)

	nodeAGenesis := execNodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := execNodeB.Backend.APIBackend().CurrentHeader().Hash()
	if faultyStaker {
		if nodeAGenesis == nodeBGenesis {
			Fail(t, "node A L2 genesis hash", nodeAGenesis, "== node B L2 genesis hash", nodeBGenesis)
		}
	} else {
		if nodeAGenesis != nodeBGenesis {
			Fail(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
		}
	}

	BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2clientA, ctx)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("ValidatorA")
	TransferBalance(t, "Faucet", "ValidatorA", balance, l1info, l1client, ctx)
	l1authA := l1info.GetDefaultTransactOpts("ValidatorA", ctx)

	l1info.GenerateAccount("ValidatorB")
	TransferBalance(t, "Faucet", "ValidatorB", balance, l1info, l1client, ctx)
	l1authB := l1info.GetDefaultTransactOpts("ValidatorB", ctx)

	valWalletAddrAPtr, err := staker.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := staker.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	if valWalletAddrA == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	rollup, err := rollupgen.NewRollupAdminLogic(l2nodeA.DeployInfo.Rollup, l1client)
	Require(t, err)
	tx, err := rollup.SetValidator(&deployAuth, []common.Address{valWalletAddrA, l1authB.From}, []bool{true, true})
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	valConfig := staker.L1ValidatorConfig{}

	valWalletA, err := staker.NewContractValidatorWallet(nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.DeployInfo.Rollup, l2nodeA.L1Reader, &l1authA, 0, func(common.Address) {})
	Require(t, err)
	if honestStakerInactive {
		valConfig.Strategy = "Defensive"
	} else {
		valConfig.Strategy = "MakeNodes"
	}

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		execNodeA,
		l2nodeA.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	managerA, err := staker.NewManager(ctx, valWalletA.RollupAddress(), &l1authA, bind.CallOpts{}, l2nodeA.L1Reader.Client(), statelessA)
	Require(t, err)
	managerA.Start(ctx)

	valWalletB, err := staker.NewEoaValidatorWallet(l2nodeB.DeployInfo.Rollup, l2nodeB.L1Reader.Client(), &l1authB)
	Require(t, err)
	valConfig.Strategy = "MakeNodes"
	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		execNodeB,
		l2nodeB.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessB.Start(ctx)
	Require(t, err)
	managerB, err := staker.NewManager(ctx, valWalletB.RollupAddress(), &l1authB, bind.CallOpts{}, l2nodeB.L1Reader.Client(), statelessB)
	Require(t, err)
	managerB.Start(ctx)

	l2info.GenerateAccount("BackgroundUser")
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)
	if faultyStaker {
		err = l2clientB.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2clientB, tx)
		Require(t, err)
	}

	// Continually make L2 transactions in a background thread
	backgroundTxsCtx, cancelBackgroundTxs := context.WithCancel(ctx)
	backgroundTxsShutdownChan := make(chan struct{})
	defer (func() {
		cancelBackgroundTxs()
		<-backgroundTxsShutdownChan
	})()
	go (func() {
		defer close(backgroundTxsShutdownChan)
		err := makeBackgroundTxs(backgroundTxsCtx, l2info, l2clientA, l2clientB, faultyStaker)
		if !errors.Is(err, context.Canceled) {
			log.Warn("error making background txs", "err", err)
		}
	})()
	<-ctx.Done()
}

func TestExecutionStateMsgCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	res, err := l2node.TxStreamer.ResultAtCount(1)
	Require(t, err)
	msgCount, err := manager.ExecutionStateMsgCount(ctx, &protocol.ExecutionState{GlobalState: protocol.GoGlobalState{Batch: 1, BlockHash: res.BlockHash}})
	Require(t, err)
	if msgCount != 1 {
		Fail(t, "Unexpected msg batch", msgCount, "(expected 1)")
	}
}

func TestExecutionStateAtMessageNumber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	res, err := l2node.TxStreamer.ResultAtCount(1)
	Require(t, err)
	expectedState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			Batch:     1,
			BlockHash: res.BlockHash,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	executionState, err := manager.ExecutionStateAtMessageNumber(ctx, 1)
	Require(t, err)
	if !reflect.DeepEqual(executionState, expectedState) {
		Fail(t, "Unexpected executionState", executionState, "(expected ", expectedState, ")")
	}
	Require(t, err)
}

func TestHistoryCommitmentUpTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	res0, err := l2node.TxStreamer.ResultAtCount(0)
	Require(t, err)
	res1, err := l2node.TxStreamer.ResultAtCount(1)
	Require(t, err)
	expectedHistoryCommitment, err := commitments.New(
		[]common.Hash{
			validator.GoGlobalState{
				BlockHash:  res0.BlockHash,
				SendRoot:   res0.SendRoot,
				Batch:      0,
				PosInBatch: 0,
			}.Hash(),
			validator.GoGlobalState{
				BlockHash:  res1.BlockHash,
				SendRoot:   res1.SendRoot,
				Batch:      1,
				PosInBatch: 0,
			}.Hash(),
		},
	)
	Require(t, err)
	historyCommitment, err := manager.HistoryCommitmentUpTo(ctx, 1)
	Require(t, err)
	if !reflect.DeepEqual(historyCommitment, expectedHistoryCommitment) {
		Fail(t, "Unexpected HistoryCommitment", historyCommitment, "(expected ", expectedHistoryCommitment, ")")
	}
}

func TestBigStepCommitmentUpTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	commitment, err := manager.BigStepCommitmentUpTo(ctx, common.Hash{}, 1, 3)
	Require(t, err)
	if commitment.Height != 3 {
		Fail(t, "Unexpected commitment height", commitment.Height, "(expected ", 3, ")")
	}
}

func TestSmallStepCommitmentUpTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	commitment, err := manager.SmallStepCommitmentUpTo(ctx, common.Hash{}, 1, 3, 2)
	Require(t, err)
	if commitment.Height != 2 {
		Fail(t, "Unexpected commitment height", commitment.Height, "(expected ", 2, ")")
	}
}

func setupManger(t *testing.T, ctx context.Context) (*arbnode.Node, *node.Node, *staker.StateManager) {
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	_, l2node, l2client, _, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfigImpl(t, ctx, true, nil, nil, l2chainConfig, nil, l2info)
	execNode := getExecNode(t, l2node)
	BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2client, ctx)
	l2info.GenerateAccount("BackgroundUser")
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	tx := l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	for i := uint64(0); i < 10; i++ {
		l2info.Accounts["BackgroundUser"].Nonce = i
		tx = l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big0, nil)
		err = l2client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig
	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		execNode,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)
	manager, err := staker.NewStateManager(stateless, nil, 4, 16)
	Require(t, err)
	return l2node, l1stack, manager
}
