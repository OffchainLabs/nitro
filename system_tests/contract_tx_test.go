// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestContractTxDeploy(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeconfig := arbnode.ConfigDefaultL2Test()
	l2info, node, client := CreateTestL2WithConfig(t, ctx, nil, nodeconfig, false)
	defer node.StopAndWait()

	from := common.HexToAddress("0x123412341234")
	TransferBalanceTo(t, "Faucet", from, big.NewInt(1e18), l2info, client, ctx)

	for stateNonce := uint64(0); stateNonce < 2; stateNonce++ {
		pos, err := node.TxStreamer.GetMessageCount()
		Require(t, err)
		var delayedMessagesRead uint64
		if pos > 0 {
			lastMessage, err := node.TxStreamer.GetMessage(pos - 1)
			Require(t, err)
			delayedMessagesRead = lastMessage.DelayedMessagesRead
		}
		// Deploys a single 0xFE (INVALID) byte as a smart contract
		deployCode := []byte{
			0x60, 0xFE, // PUSH1 0xFE
			0x60, 0x00, // PUSH1 0
			0x53,       // MSTORE8
			0x60, 0x01, // PUSH1 1
			0x60, 0x00, // PUSH1 0
			0xF3, // RETURN
		}
		var requestId common.Hash
		requestId[0] = uint8(stateNonce)
		contractTx := &types.ArbitrumContractTx{
			ChainId:   params.ArbitrumDevTestChainConfig().ChainID,
			RequestId: requestId,
			From:      from,
			GasFeeCap: big.NewInt(1e9),
			Gas:       1e6,
			To:        nil,
			Value:     big.NewInt(0),
			Data:      deployCode,
		}
		l2Msg := []byte{arbos.L2MessageKind_ContractTx}
		l2Msg = append(l2Msg, math.U256Bytes(arbmath.UintToBig(contractTx.Gas))...)
		l2Msg = append(l2Msg, math.U256Bytes(contractTx.GasFeeCap)...)
		l2Msg = append(l2Msg, common.Hash{}.Bytes()...) // to is zero, translated into nil
		l2Msg = append(l2Msg, math.U256Bytes(contractTx.Value)...)
		l2Msg = append(l2Msg, contractTx.Data...)

		err = node.TxStreamer.AddMessages(pos, true, []arbstate.MessageWithMetadata{
			{
				Message: &arbos.L1IncomingMessage{
					Header: &arbos.L1IncomingMessageHeader{
						Kind:        arbos.L1MessageType_L2Message,
						Poster:      from,
						BlockNumber: 0,
						Timestamp:   0,
						RequestId:   &contractTx.RequestId,
						L1BaseFee:   &big.Int{},
					},
					L2msg:        l2Msg,
					BatchGasCost: new(uint64),
				},
				DelayedMessagesRead: delayedMessagesRead,
			},
		})
		Require(t, err)

		txHash := types.NewTx(contractTx).Hash()
		t.Log("made contract tx", contractTx, "with hash", txHash)
		receipt, err := WaitForTx(ctx, client, txHash, time.Second*10)
		Require(t, err)
		if receipt.Status != types.ReceiptStatusSuccessful {
			Fail(t, "Receipt has non-successful status", receipt.Status)
		}

		expectedAddr := crypto.CreateAddress(from, stateNonce)
		if receipt.ContractAddress != expectedAddr {
			Fail(t, "expected address", from, "nonce", stateNonce, "to deploy to", expectedAddr, "but got", receipt.ContractAddress)
		}
		t.Log("deployed contract", receipt.ContractAddress, "from address", from, "with nonce", stateNonce)
		stateNonce++

		code, err := client.CodeAt(ctx, receipt.ContractAddress, nil)
		Require(t, err)
		if !bytes.Equal(code, []byte{0xFE}) {
			Fail(t, "expected contract", receipt.ContractAddress, "code of 0xFE but got", hex.EncodeToString(code))
		}
	}
}
