package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestBitro(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)

	// retryableSetup is being called by tests that validate blocks.
	// For now validation only works with HashScheme set.
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	builder.nodeConfig.BatchPoster.Enable = false
	builder.nodeConfig.BlockValidator.Enable = false
	builder.nodeConfig.Staker.Enable = true
	builder.nodeConfig.ParentChainReader.Enable = true
	builder.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	valConf.UseJit = true
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(builder.nodeConfig, valStack)

	builder.execConfig.Sequencer.MaxRevertGasReject = 0

	builder.Build(t)

	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("Beneficiary")
	builder.L2Info.GenerateAccount("Burn")

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)

	// // burn some gas so that the faucet's Callvalue + Balance never exceeds a uint256
	// discard := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	// builder.L2.TransferBalance(t, "Faucet", "Burn", discard, builder.L2Info)

	lookupL2Tx := func(l1Receipt *types.Receipt) *types.Transaction {
		messages, err := delayedBridge.LookupMessagesInRange(ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber, nil)
		Require(t, err)
		if len(messages) == 0 {
			Fatal(t, "didn't find message for submission")
		}
		var submissionTxs []*types.Transaction
		msgTypes := map[uint8]bool{
			arbostypes.L1MessageType_SubmitRetryable: true,
			arbostypes.L1MessageType_EthDeposit:      true,
			arbostypes.L1MessageType_L2Message:       true,
		}
		txTypes := map[uint8]bool{
			types.ArbitrumSubmitRetryableTxType: true,
			types.ArbitrumDepositTxType:         true,
			types.ArbitrumContractTxType:        true,
		}
		for _, message := range messages {
			if !msgTypes[message.Message.Header.Kind] {
				continue
			}
			txs, err := arbos.ParseL2Transactions(message.Message, chaininfo.ArbitrumDevTestChainConfig().ChainID)
			Require(t, err)
			for _, tx := range txs {
				if txTypes[tx.Type()] {
					submissionTxs = append(submissionTxs, tx)
				}
			}
		}
		if len(submissionTxs) != 1 {
			Fatal(t, "expected 1 tx from submission, found", len(submissionTxs))
		}
		return submissionTxs[0]
	}

	teardown := func() {

		// check the integrity of the RPC
		blockNum, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err, "failed to get L2 block number")
		for number := uint64(0); number < blockNum; number++ {
			block, err := builder.L2.Client.BlockByNumber(ctx, arbmath.UintToBig(number))
			Require(t, err, "failed to get L2 block", number, "of", blockNum)
			if block.Number().Uint64() != number {
				Fatal(t, "block number mismatch", number, block.Number().Uint64())
			}
		}

		cancel()

		builder.L2.ConsensusNode.StopAndWait()
		requireClose(t, builder.L1.Stack)
	}
	defer teardown()

	user2Address := builder.L2Info.GetAddress("User2")
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto redeem the retryable
	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL2.NoSend = true
	usertxoptsL2.GasMargin = 0
	tx, err := nodeInterface.EstimateRetryableTicket(
		&usertxoptsL2,
		usertxoptsL2.From,
		deposit,
		user2Address,
		callValue,
		beneficiaryAddress,
		beneficiaryAddress,
		[]byte{0x32, 0x42, 0x32, 0x88}, // increase the cost to beyond that of params.TxGas
	)
	Require(t, err, "failed to estimate retryable submission")
	estimate := tx.Gas()
	expectedEstimate := params.TxGas + params.TxDataNonZeroGasEIP2028*4
	if float64(estimate) > float64(expectedEstimate)*(1+gasestimator.EstimateGasErrorRatio) {
		t.Errorf("estimated retryable ticket at %v gas but expected %v, with error margin of %v",
			estimate,
			expectedEstimate,
			gasestimator.EstimateGasErrorRatio,
		)
	}

	// submit & auto redeem the retryable using the gas estimate
	usertxoptsL1 := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		user2Address,
		callValue,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		arbmath.UintToBig(estimate),
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		[]byte{0x32, 0x42, 0x32, 0x88},
	)
	Require(t, err)

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t)
	}

	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if !arbmath.BigEquals(l2balance, callValue) {
		Fatal(t, "Unexpected balance:", l2balance)
	}
	testFlatCallTracer(t, ctx, builder.L2.Client.Client())
}
