package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/big"

	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var (
	sequencerPrivKey     = flag.String("sequencer-private-key", "cb5790da63720727af975f42c79f69918580209889225fa7128c92402a6d3a65", "Sequencer private key hex (no 0x prefix)")
	evilSequencerPrivKey = flag.String("evil-sequencer-private-key", "b0c3d5fa3891e7029918fdf0ed5448e0d6b7642c4ee2c8fa921bc703b4bc7c9f", "Evil sequencer private key hex (no 0x prefix)")
	endpoint             = flag.String("l1-endpoint", "http://localhost:8545", "Ethereum L1 JSON-RPC endpoint")
	honestSeqInboxAddr   = flag.String("honest-sequencer-inbox-addr", "0x191f7df213d19be0567eb6383bbc6193a5ee6b07", "Address of the honest sequencer inbox")
	evilSeqInboxAddr     = flag.String("evil-sequencer-inbox-addr", "0x948160aba0f99a9d3041e511c22cc4adc5c221d2", "Address of the evil sequencer inbox")
	honestInboxAddr      = flag.String("honest-inbox-addr", "0x04449bd67f67f52c8de81982225b9aee6ced0f3e", "Address of the honest inbox")
	evilInboxAddr        = flag.String("evil-inbox-addr", "0xa9136ffaebd6939a7a9c08d1ecaba59bfbdb9197", "Address of the evil inbox")
	deploymentBlock      = flag.Int64("deployment-block", 0, "Block number of the Arbitrum contracts deployment")
)

// TODO: Need to give the evil validators seed ERC20 tokens. The evil validator needs to approve the rollup
// and challenge manager contracts to spend its allowance of mock WETH.
// TODO: Do the funds briding here into my evil sequencer inbox, then advance both inboxes with whatever is needed.

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	noErr := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	privKey, err := crypto.HexToECDSA(*sequencerPrivKey)
	noErr(err)
	rpcClient, err := rpc.Dial(*endpoint)
	noErr(err)
	client := ethclient.NewClient(rpcClient)
	chainId, err := client.ChainID(ctx)
	noErr(err)
	sequencerTxOpts, err := bind.NewKeyedTransactorWithChainID(privKey, chainId)
	noErr(err)

	addr := common.HexToAddress(*honestSeqInboxAddr)
	seqInbox, err := arbnode.NewSequencerInbox(client, addr, *deploymentBlock)
	noErr(err)
	evilAddr := common.HexToAddress(*evilSeqInboxAddr)
	evilSeqInbox, err := arbnode.NewSequencerInbox(client, evilAddr, *deploymentBlock)
	noErr(err)
	seqInboxBindings, err := bridgegen.NewSequencerInbox(addr, client)
	noErr(err)
	evilSeqInboxBindings, err := bridgegen.NewSequencerInbox(evilAddr, client)
	noErr(err)

	bridgeAddr, err := seqInboxBindings.Bridge(&bind.CallOpts{Context: ctx})
	noErr(err)
	deployedAt := uint64(*deploymentBlock)
	bridge, err := arbnode.NewDelayedBridge(client, bridgeAddr, deployedAt)
	noErr(err)
	deployedAtBig := arbmath.UintToBig(deployedAt)
	messages, err := bridge.LookupMessagesInRange(ctx, deployedAtBig, nil, nil)
	noErr(err)
	if len(messages) == 0 {
		panic("no messages")
	}
	initMessage, err := messages[0].Message.ParseInitMessage()
	noErr(err)

	fmt.Printf("Honest init mesage: %+v\n", initMessage)

	evilBridgeAddr, err := evilSeqInboxBindings.Bridge(&bind.CallOpts{Context: ctx})
	noErr(err)
	deployedAt = uint64(*deploymentBlock)
	bridge, err = arbnode.NewDelayedBridge(client, evilBridgeAddr, deployedAt)
	noErr(err)
	deployedAtBig = arbmath.UintToBig(deployedAt)
	messages, err = bridge.LookupMessagesInRange(ctx, deployedAtBig, nil, nil)
	noErr(err)
	if len(messages) == 0 {
		panic("no messages")
	}
	evilInitMsg, err := messages[0].Message.ParseInitMessage()
	noErr(err)

	if string(evilInitMsg.SerializedChainConfig) != string(initMessage.SerializedChainConfig) {
		panic("Not equal serialized chain config")
	}
	if evilInitMsg.InitialL1BaseFee.Cmp(initMessage.InitialL1BaseFee) != 0 {
		panic("Not equal initial L1 base fee")
	}

	fmt.Println("")
	fmt.Printf("Evil init mesage: %+v\n", evilInitMsg)
	fmt.Println("")

	ensureTxSucceeds := func(tx *types.Transaction) {
		if waitErr := challenge_testing.WaitForTx(ctx, client, tx); waitErr != nil {
			panic(err)
		}
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			panic(err)
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			panic("receipt was not successful")
		}
	}

	fromBlock := big.NewInt(*deploymentBlock)
	batches, err := seqInbox.LookupBatchesInRange(ctx, fromBlock, nil)
	noErr(err)
	fmt.Println("got batches from honest", len(batches))
	evilBatches, err := evilSeqInbox.LookupBatchesInRange(ctx, fromBlock, nil)
	noErr(err)
	fmt.Println("got batches from evil", len(evilBatches))

	// fmt.Printf("Honest first %+v\n", batches[0])
	// fmt.Println("")
	// fmt.Printf("Evil first %+v\n", evilBatches[0])

	// tx, err := evilSeqInboxBindings.SetIsBatchPoster(sequencerTxOpts, sequencerTxOpts.From, true)
	// if err != nil {
	// 	panic(err)
	// }
	// ensureTxSucceeds(tx)
	// tx, err = evilSeqInboxBindings.SetIsSequencer(sequencerTxOpts, sequencerTxOpts.From, true)
	// if err != nil {
	// 	panic(err)
	// }
	// ensureTxSucceeds(tx)
	_ = ensureTxSucceeds

	_ = sequencerTxOpts
	gasPrice := big.NewInt(params.GWei * 100)
	data := hexutil.MustDecode("0x0f4d14e9000000000000000000000000000000000000000000000000000082f79cd90000")
	gotInboxAddr := common.HexToAddress(*honestInboxAddr)
	gotEvilInboxAddr := common.HexToAddress(*evilInboxAddr)
	nonce, err := client.NonceAt(ctx, sequencerTxOpts.From, nil)
	noErr(err)
	txDynamic := types.DynamicFeeTx{
		To:        &gotInboxAddr,
		Value:     big.NewInt(params.Ether),
		Gas:       1000000,
		GasFeeCap: gasPrice,
		Data:      data,
		Nonce:     nonce,
	}
	tx, err := sequencerTxOpts.Signer(sequencerTxOpts.From, types.NewTx(&txDynamic))
	noErr(err)
	err = client.SendTransaction(ctx, tx)
	noErr(err)
	ensureTxSucceeds(tx)

	// Same tx but to the malicious inbox.
	nonce, err = client.NonceAt(ctx, sequencerTxOpts.From, nil)
	noErr(err)
	txDynamic = types.DynamicFeeTx{
		To:        &gotEvilInboxAddr,
		Value:     big.NewInt(params.Ether),
		Data:      data,
		Gas:       1000000,
		GasFeeCap: gasPrice,
		Nonce:     nonce,
	}
	tx, err = sequencerTxOpts.Signer(sequencerTxOpts.From, types.NewTx(&txDynamic))
	noErr(err)
	err = client.SendTransaction(ctx, tx)
	noErr(err)
	ensureTxSucceeds(tx)

	// Wait until the balance is fully bridged over.

	// for _, batch := range batches {
	// 	if batch.SequenceNumber == 0 {
	// 		continue
	// 	}
	// 	rawBatch, err := batch.Serialize(ctx, client)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Println("Batch sequence number", batch.SequenceNumber)
	// 	fmt.Printf("%#x\n", rawBatch[40:])
	// 	tx, err := evilSeqInboxBindings.AddSequencerL2BatchFromOrigin0(
	// 		sequencerTxOpts,
	// 		new(big.Int).SetUint64(batch.SequenceNumber),
	// 		rawBatch,
	// 		new(big.Int).SetUint64(batch.AfterDelayedCount),
	// 		common.Address{},
	// 		big.NewInt(0),
	// 		big.NewInt(0),
	// 	)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	ensureTxSucceeds(tx)
	// 	fmt.Println("Tx with hash", tx.Hash().Hex())
	// }

	// funnelPrivKey, err := crypto.HexToECDSA("b6b15c8cb491557369f3c7d2c287b053eb229daa9c22138887752191c9520659")
	// if err != nil {
	// 	panic(err)
	// }
	// funnelOpts, err := bind.NewKeyedTransactorWithChainID(funnelPrivKey, chainId)
	// if err != nil {
	// 	panic(err)
	// }
	//submitBoldBatch(ctx, sequencerTxOpts, funnelOpts, evilSeqInboxBindings, evilAddr, 1)
	// for _, batch := range batches {
	// }
	// TODO: Replay batches from some source sequencer inbox, and then diverge at desired points.
	// Long running process.
}

func submitBoldBatch(
	ctx context.Context,
	sequencerTxOpts *bind.TransactOpts,
	txOpts *bind.TransactOpts,
	seqInbox *bridgegen.SequencerInbox,
	seqInboxAddr common.Address,
	messagesPerBatch int64,
) {
	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < messagesPerBatch; i++ {
		to := common.Address{}
		value := big.NewInt(i)
		tx := prepareTx(txOpts, &to, value, []byte{})
		if err := writeTxToBatch(batchBuffer, tx); err != nil {
			panic(err)
		}
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	if err != nil {
		panic(err)
	}
	message := append([]byte{0}, compressed...)

	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin0(
		sequencerTxOpts,
		seqNum,
		message,
		big.NewInt(1),
		common.Address{},
		big.NewInt(0),
		big.NewInt(0),
	)
	if err != nil {
		panic(err)
	}
	_ = tx
}

func prepareTx(txOpts *bind.TransactOpts, to *common.Address, value *big.Int, data []byte) *types.Transaction {
	txData := &types.DynamicFeeTx{
		To:    to,
		Value: value,
		Data:  data,
	}
	tx := types.NewTx(txData)
	signed, err := txOpts.Signer(txOpts.From, tx)
	if err != nil {
		panic(err)
	}
	return signed

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
