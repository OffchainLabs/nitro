// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDeploy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	_, simple := deploySimple(t, ctx, auth, client)

	tx, err := simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fatal(t, "Unexpected counter value", counter)
	}
}

func TestEstimate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	gasPrice := big.NewInt(params.GWei / 10)

	// set the gas price
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), client)
	Require(t, err, "could not deploy ArbOwner contract")
	tx, err := arbOwner.SetMinimumL2BaseFee(&auth, gasPrice)
	Require(t, err, "could not set L2 gas price")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	// connect to arbGasInfo precompile
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), client)
	Require(t, err, "could not deploy contract")

	// wait for price to come to equilibrium
	equilibrated := false
	numTriesLeft := 20
	for !equilibrated && numTriesLeft > 0 {
		// make an empty block to let the gas price update
		l2info.GasPrice = new(big.Int).Mul(l2info.GasPrice, big.NewInt(2))
		TransferBalance(t, "Owner", "Owner", common.Big0, l2info, client, ctx)

		// check if the price has equilibrated
		_, _, _, _, _, setPrice, err := arbGasInfo.GetPricesInWei(&bind.CallOpts{})
		Require(t, err, "could not get L2 gas price")
		if gasPrice.Cmp(setPrice) == 0 {
			equilibrated = true
		}
		numTriesLeft--
	}
	if !equilibrated {
		Fatal(t, "L2 gas price did not converge", gasPrice)
	}

	initialBalance, err := client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")

	// deploy a test contract
	_, tx, simple, err := mocksgen.DeploySimple(&auth, client)
	Require(t, err, "could not deploy contract")
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	header, err := client.HeaderByNumber(ctx, receipt.BlockNumber)
	Require(t, err, "could not get header")
	if header.BaseFee.Cmp(gasPrice) != 0 {
		Fatal(t, "Header has wrong basefee", header.BaseFee, gasPrice)
	}

	balance, err := client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")
	expectedCost := receipt.GasUsed * gasPrice.Uint64()
	observedCost := initialBalance.Uint64() - balance.Uint64()
	if expectedCost != observedCost {
		Fatal(t, "Expected deployment to cost", expectedCost, "instead of", observedCost)
	}

	tx, err = simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fatal(t, "Unexpected counter value", counter)
	}
}

func TestComponentEstimate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	l1BaseFee := big.NewInt(l1pricing.InitialPricePerUnitWei)
	l2BaseFee := GetBaseFee(t, client, ctx)

	colors.PrintGrey("l1 basefee ", l1BaseFee)
	colors.PrintGrey("l2 basefee ", l2BaseFee)

	userBalance := big.NewInt(1e16)
	maxPriorityFeePerGas := big.NewInt(0)
	maxFeePerGas := arbmath.BigMulByUfrac(l2BaseFee, 3, 2)

	l2info.GenerateAccount("User")
	TransferBalance(t, "Owner", "User", userBalance, l2info, client, ctx)

	from := l2info.GetAddress("User")
	to := testhelpers.RandomAddress()
	gas := uint64(100000000)
	calldata := []byte{0x00, 0x12}
	value := big.NewInt(4096)

	nodeAbi, err := node_interfacegen.NodeInterfaceMetaData.GetAbi()
	Require(t, err)

	nodeMethod := nodeAbi.Methods["gasEstimateComponents"]
	estimateCalldata := append([]byte{}, nodeMethod.ID...)
	packed, err := nodeMethod.Inputs.Pack(to, false, calldata)
	Require(t, err)
	estimateCalldata = append(estimateCalldata, packed...)

	msg := ethereum.CallMsg{
		From:      from,
		To:        &types.NodeInterfaceAddress,
		Gas:       gas,
		GasFeeCap: maxFeePerGas,
		GasTipCap: maxPriorityFeePerGas,
		Value:     value,
		Data:      estimateCalldata,
	}
	returnData, err := client.CallContract(ctx, msg, nil)
	Require(t, err)

	outputs, err := nodeMethod.Outputs.Unpack(returnData)
	Require(t, err)
	if len(outputs) != 4 {
		Fatal(t, "expected 4 outputs from gasEstimateComponents, got", len(outputs))
	}

	gasEstimate, _ := outputs[0].(uint64)
	gasEstimateForL1, _ := outputs[1].(uint64)
	baseFee, _ := outputs[2].(*big.Int)
	l1BaseFeeEstimate, _ := outputs[3].(*big.Int)

	execNode := getExecNode(t, node)
	tx := l2info.SignTxAs("User", &types.DynamicFeeTx{
		ChainID:   execNode.ArbInterface.BlockChain().Config().ChainID,
		Nonce:     0,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasEstimate,
		To:        &to,
		Value:     value,
		Data:      calldata,
	})

	l2Estimate := gasEstimate - gasEstimateForL1

	colors.PrintBlue("Est. ", gasEstimate, " - ", gasEstimateForL1, " = ", l2Estimate)

	if !arbmath.BigEquals(l1BaseFeeEstimate, l1BaseFee) {
		Fatal(t, l1BaseFeeEstimate, l1BaseFee)
	}
	if !arbmath.BigEquals(baseFee, l2BaseFee) {
		Fatal(t, baseFee, l2BaseFee.Uint64())
	}

	Require(t, client.SendTransaction(ctx, tx))
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	l2Used := receipt.GasUsed - receipt.GasUsedForL1
	colors.PrintMint("True ", receipt.GasUsed, " - ", receipt.GasUsedForL1, " = ", l2Used)

	if l2Estimate != l2Used {
		Fatal(t, l2Estimate, l2Used)
	}
}

func callFindBatchContainig(t *testing.T, ctx context.Context, client *ethclient.Client, nodeAbi *abi.ABI, blockNum uint64) uint64 {
	findBatch := nodeAbi.Methods["findBatchContainingBlock"]
	callData := append([]byte{}, findBatch.ID...)
	packed, err := findBatch.Inputs.Pack(blockNum)
	Require(t, err)
	callData = append(callData, packed...)
	msg := ethereum.CallMsg{
		To:   &types.NodeInterfaceAddress,
		Data: callData,
	}
	returnData, err := client.CallContract(ctx, msg, nil)
	Require(t, err)
	outputs, err := findBatch.Outputs.Unpack(returnData)
	Require(t, err)
	if len(outputs) != 1 {
		Fatal(t, "expected 1 output from findBatchContainingBlock, got", len(outputs))
	}
	gotBatchNum, ok := outputs[0].(uint64)
	if !ok {
		Fatal(t, "bad output from findBatchContainingBlock")
	}
	return gotBatchNum
}

func callGetL1Confirmations(t *testing.T, ctx context.Context, client *ethclient.Client, nodeAbi *abi.ABI, blockHash common.Hash) uint64 {
	getConfirmations := nodeAbi.Methods["getL1Confirmations"]
	callData := append([]byte{}, getConfirmations.ID...)
	packed, err := getConfirmations.Inputs.Pack(blockHash)
	Require(t, err)
	callData = append(callData, packed...)
	msg := ethereum.CallMsg{
		To:   &types.NodeInterfaceAddress,
		Data: callData,
	}
	returnData, err := client.CallContract(ctx, msg, nil)
	Require(t, err)
	outputs, err := getConfirmations.Outputs.Unpack(returnData)
	Require(t, err)
	if len(outputs) != 1 {
		Fatal(t, "expected 1 output from findBatchContainingBlock, got", len(outputs))
	}
	confirmations, ok := outputs[0].(uint64)
	if !ok {
		Fatal(t, "bad output from findBatchContainingBlock")
	}
	return confirmations
}

func TestFindBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l1Info := NewL1TestInfo(t)
	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("challenger", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)

	l1Info, l1Backend, _, _ := createTestL1BlockChain(t, l1Info)
	conf := arbnode.ConfigDefaultL1Test()
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false

	chainConfig := params.ArbitrumDevTestChainConfig()
	fatalErrChan := make(chan error, 10)
	rollupAddresses := DeployOnTestL1(t, ctx, l1Info, l1Backend, chainConfig)

	bridgeAddr, seqInbox, seqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)

	rollupAddresses.Bridge = bridgeAddr
	rollupAddresses.SequencerInbox = seqInboxAddr
	l2Info := NewArbTestInfo(t, chainConfig.ChainID)
	consensus, _ := createL2Nodes(t, ctx, conf, chainConfig, l1Backend, l2Info, rollupAddresses, nil, nil, fatalErrChan)
	err := consensus.Start(ctx)
	Require(t, err)

	l2Client := ClientForStack(t, consensus.Stack)
	nodeAbi, err := node_interfacegen.NodeInterfaceMetaData.GetAbi()
	Require(t, err)
	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)

	l2Info.GenerateAccount("Destination")
	makeBatch(t, consensus, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)
	makeBatch(t, consensus, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)
	makeBatch(t, consensus, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)

	for blockNum := uint64(0); blockNum < uint64(MsgPerBatch)*3; blockNum++ {
		gotBatchNum := callFindBatchContainig(t, ctx, l2Client, nodeAbi, blockNum)
		expBatchNum := uint64(0)
		if blockNum > 0 {
			expBatchNum = 1 + (blockNum-1)/uint64(MsgPerBatch)
		}
		if expBatchNum != gotBatchNum {
			Fatal(t, "wrong result from findBatchContainingBlock. blocknum ", blockNum, " expected ", expBatchNum, " got ", gotBatchNum)
		}
		batchL1Block, err := consensus.InboxTracker.GetBatchL1Block(gotBatchNum).Await(ctx)
		Require(t, err)
		blockHeader, err := l2Client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		blockHash := blockHeader.Hash()

		minCurrentL1Block, err := l1Backend.BlockNumber(ctx)
		Require(t, err)
		gotConfirmations := callGetL1Confirmations(t, ctx, l2Client, nodeAbi, blockHash)
		maxCurrentL1Block, err := l1Backend.BlockNumber(ctx)
		Require(t, err)

		if gotConfirmations > (maxCurrentL1Block-batchL1Block) || gotConfirmations < (minCurrentL1Block-batchL1Block) {
			Fatal(t, "wrong number of confirmations. got ", gotConfirmations)
		}
	}
}
