package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestGenerateLightBlock(t *testing.T) {
	ctx, b, cleanup := setupEnv(t)
	defer cleanup()

	b.L2Info.GenerateAccount("User")
	tx := b.L2Info.PrepareTx("Owner", "User", b.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := b.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	r, err := EnsureTxSucceeded(ctx, b.L2.Client, tx)
	Require(t, err)

	t.Logf("tx gas used for L2: %d", r.GasUsedForL2())
	recordBlock(t, r.BlockNumber.Uint64(), b, rawdb.LocalTarget(), rawdb.TargetWavm)
}

func TestGenerateHeavyBlock(t *testing.T) {
	ctx, b, cleanup := setupEnv(t)
	defer cleanup()

	txGasLimit, _ := getLimits(t, b)
	txOpts := getTxOpts(ctx, b, txGasLimit)

	stylus, bigMap := deployContracts(ctx, t, b, txOpts)

	stylusTx := b.L2Info.PrepareTxTo("Faucet", &stylus, b.L2Info.TransferGas, nil, argsForStorageWrite(testhelpers.RandomHash(), testhelpers.RandomHash()))

	lighterMapTx := getSucceededMapTx(ctx, t, b, bigMap, &txOpts, 1400)
	heavyMapTx := getSucceededMapTx(ctx, t, b, bigMap, &txOpts, 1420)

	txes := types.Transactions{
		stylusTx,
		b.L2Info.PrepareTxTo("Owner", lighterMapTx.To(), txGasLimit*2, big.NewInt(0), lighterMapTx.Data()),
		b.L2Info.PrepareTxTo("Owner", heavyMapTx.To(), txGasLimit*2, big.NewInt(0), heavyMapTx.Data()),
	}

	currentHeight, err := b.L2.Client.BlockNumber(ctx)
	Require(t, err)

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: currentHeight + 1,
		Timestamp:   arbmath.SaturatingUCast[uint64](time.Now().Unix()),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	hooks := gethexec.MakeZeroTxSizeSequencingHooksForTesting(txes, nil, nil, nil)
	_, err = b.L2.ExecNode.ExecEngine.SequenceTransactions(header, hooks, nil)
	Require(t, err)

	receipt0, err := EnsureTxSucceeded(ctx, b.L2.Client, txes[0])
	Require(t, err)
	receipt1, err := EnsureTxSucceeded(ctx, b.L2.Client, txes[1])
	Require(t, err)
	receipt2, err := EnsureTxSucceeded(ctx, b.L2.Client, txes[2])
	Require(t, err)

	if receipt0.BlockHash != receipt1.BlockHash || receipt0.BlockHash != receipt2.BlockHash {
		t.Fatalf("expected txes to be in the same block, got %s, %s, %s", receipt0.BlockHash, receipt1.BlockHash, receipt2.BlockHash)
	}

	t.Logf("tx 0 gas used for L2: %d", receipt0.GasUsedForL2())
	t.Logf("tx 1 gas used for L2: %d", receipt1.GasUsedForL2())
	t.Logf("tx 2 gas used for L2: %d", receipt2.GasUsedForL2())

	recordBlock(t, receipt0.BlockNumber.Uint64(), b, rawdb.LocalTarget(), rawdb.TargetWavm)
}

func setupEnv(t *testing.T) (context.Context, *NodeBuilder, func()) {
	ctx := t.Context()

	b := NewNodeBuilder(ctx).DefaultConfig(t, true)
	b.takeOwnership = true

	b.RequireScheme(t, rawdb.HashScheme)
	b.nodeConfig.BlockValidator.Enable = false
	b.nodeConfig.Staker.Enable = true
	b.nodeConfig.BatchPoster.Enable = true
	b.nodeConfig.ParentChainReader.Enable = true
	b.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(b.nodeConfig, valStack)

	cleanup := b.Build(t)
	return ctx, b, cleanup
}

func getTxOpts(ctx context.Context, b *NodeBuilder, txGasLimit uint64) bind.TransactOpts {
	opts := b.L2Info.GetDefaultTransactOpts("Owner", ctx)
	opts.GasLimit = 2 * txGasLimit
	return opts
}

func getLimits(t *testing.T, b *NodeBuilder) (uint64, uint64) {
	statedb, err := b.L2.ExecNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err)
	burner := burn.NewSystemBurner(nil, false)
	arbosSt, err := arbosState.OpenArbosState(statedb, burner)
	Require(t, err)

	txGasLimit, err := arbosSt.L2PricingState().PerTxGasLimit()
	Require(t, err)
	blockGasLimit, err := arbosSt.L2PricingState().PerBlockGasLimit()
	Require(t, err)

	t.Logf("tx gas limit: %d", txGasLimit)
	t.Logf("block gas limit: %d", blockGasLimit)
	t.Logf("actual limit: %d", txGasLimit+blockGasLimit)

	return txGasLimit, blockGasLimit
}

func deployContracts(ctx context.Context, t *testing.T, b *NodeBuilder, txOpts bind.TransactOpts) (common.Address, *localgen.BigMap) {
	_, bigMap := b.L2.DeployBigMap(t, txOpts)
	programAddress := deployWasm(t, ctx, txOpts, b.L2.Client, rustFile("storage"))
	return programAddress, bigMap
}

func getSucceededMapTx(ctx context.Context, t *testing.T, b *NodeBuilder, bigMap *localgen.BigMap, txOpts *bind.TransactOpts, toAdd int64) *types.Transaction {
	succesfullTx, err := bigMap.ClearAndAddValues(txOpts, big.NewInt(0), big.NewInt(toAdd))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, b.L2.Client, succesfullTx)
	Require(t, err)
	return succesfullTx
}
