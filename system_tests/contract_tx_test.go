// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func testContractTxDeploy(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create builder but don't build yet
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false

	// Fund the test account at GENESIS - before building
	from := common.HexToAddress("0x123412341234")

	// Add to genesis allocation in L2Info
	// This ensures BOTH geth and Nethermind have the account from block 0
	if builder.L2Info.ArbInitData.Accounts == nil {
		builder.L2Info.ArbInitData.Accounts = []statetransfer.AccountInitializationInfo{}
	}

	builder.L2Info.ArbInitData.Accounts = append(builder.L2Info.ArbInitData.Accounts,
		statetransfer.AccountInitializationInfo{
			Addr:         from,
			EthBalance:   big.NewInt(1e18),
			Nonce:        0,
			ContractInfo: nil,
		})

	// Also add to the L2Info.Accounts map for test helpers to use
	// Use SetFullAccountInfo which handles the atomic.Uint64 correctly
	builder.L2Info.SetFullAccountInfo("TestAccount", &AccountInfo{
		Address:    from,
		PrivateKey: nil, // No private key needed for ArbitrumContractTx
	})

	// NOW set execution mode and build
	builder = builder.WithExecutionClientMode(executionClientMode)

	cleanup := builder.Build(t)
	defer cleanup()

	// Wait for initialization to complete
	if executionClientMode == ExecutionClientModeComparison || executionClientMode == ExecutionClientModeExternal {
		time.Sleep(time.Second * 2)
	}

	// Verify account was funded at genesis in both clients
	balance, err := builder.L2.Client.BalanceAt(ctx, from, nil)
	Require(t, err)
	if balance.Cmp(big.NewInt(1e18)) != 0 {
		Fatal(t, "Test account not funded at genesis, got balance:", balance)
	}
	t.Log("Verified account funded at genesis with balance:", balance)

	// NO TransferBalance call - account is already funded!

	for stateNonce := uint64(0); stateNonce < 2; stateNonce++ {
		msgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)

		var delayedMessagesRead uint64
		if msgCount > 0 {
			lastMessage, err := builder.L2.ConsensusNode.TxStreamer.GetMessage(msgCount - 1)
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
		// #nosec G115
		requestId[0] = uint8(stateNonce)

		contractTx := &types.ArbitrumContractTx{
			ChainId:   chaininfo.ArbitrumDevTestChainConfig().ChainID,
			RequestId: requestId,
			From:      from,
			GasFeeCap: big.NewInt(1e9),
			Gas:       1e6,
			To:        nil,
			Value:     big.NewInt(0),
			Data:      deployCode,
		}

		l2Msg := []byte{arbos.L2MessageKind_ContractTx}
		l2Msg = append(l2Msg, arbmath.Uint64ToU256Bytes(contractTx.Gas)...)
		l2Msg = append(l2Msg, arbmath.U256Bytes(contractTx.GasFeeCap)...)
		l2Msg = append(l2Msg, common.Hash{}.Bytes()...) // to is zero, translated into nil
		l2Msg = append(l2Msg, arbmath.U256Bytes(contractTx.Value)...)
		l2Msg = append(l2Msg, contractTx.Data...)

		err = builder.L2.ConsensusNode.TxStreamer.AddMessages(msgCount, true, []arbostypes.MessageWithMetadata{
			{
				Message: &arbostypes.L1IncomingMessage{
					Header: &arbostypes.L1IncomingMessageHeader{
						Kind:        arbostypes.L1MessageType_L2Message,
						Poster:      from,
						BlockNumber: 0,
						Timestamp:   0,
						RequestId:   &contractTx.RequestId,
						L1BaseFee:   &big.Int{},
					},
					L2msg:              l2Msg,
					LegacyBatchGasCost: nil,
					BatchDataStats:     nil,
				},
				DelayedMessagesRead: delayedMessagesRead,
			},
		}, nil)
		Require(t, err)

		txHash := types.NewTx(contractTx).Hash()
		t.Log("made contract tx", contractTx, "with hash", txHash)

		receipt, err := WaitForTx(ctx, builder.L2.Client, txHash, time.Second*10)
		Require(t, err)
		if receipt.Status != types.ReceiptStatusSuccessful {
			Fatal(t, "Receipt has non-successful status", receipt.Status)
		}

		expectedAddr := crypto.CreateAddress(from, stateNonce)
		if receipt.ContractAddress != expectedAddr {
			Fatal(t, "expected address", from, "nonce", stateNonce, "to deploy to", expectedAddr, "but got", receipt.ContractAddress)
		}
		t.Log("deployed contract", receipt.ContractAddress, "from address", from, "with nonce", stateNonce)

		code, err := builder.L2.Client.CodeAt(ctx, receipt.ContractAddress, nil)
		Require(t, err)
		if !bytes.Equal(code, []byte{0xFE}) {
			Fatal(t, "expected contract", receipt.ContractAddress, "code of 0xFE but got", hex.EncodeToString(code))
		}

		stateNonce++
	}
}

func TestContractTxDeployInternal(t *testing.T) {
	testContractTxDeploy(t, ExecutionClientModeInternal)
}

func TestContractTxDeployExternal(t *testing.T) {
	testContractTxDeploy(t, ExecutionClientModeExternal)
}

func TestContractTxDeployComparison(t *testing.T) {
	testContractTxDeploy(t, ExecutionClientModeComparison)
}
