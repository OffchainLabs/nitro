package arbtest

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestBlobAndInternalTxsReject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User")
	builder.L2Info.GenerateAccount("User2")
	l2ChainID := builder.L2Info.Signer.ChainID()

	privKey := GetTestKeyForAccountName(t, "User")
	txDataBlob := &types.BlobTx{
		ChainID:   &uint256.Int{l2ChainID.Uint64()},
		Nonce:     0,
		GasFeeCap: &uint256.Int{params.GWei},
		Gas:       500000,
		To:        builder.L2Info.GetAddress("User2"),
		Value:     &uint256.Int{0},
	}
	blobTx, err := types.SignNewTx(privKey, types.NewCancunSigner(l2ChainID), txDataBlob)
	Require(t, err)
	err = builder.L2.Client.SendTransaction(ctx, blobTx)
	if err == nil && !errors.Is(err, types.ErrTxTypeNotSupported) {
		t.Fatalf("did not receive expected error when submitting blob transaction. Want: %v, Got: %v", types.ErrTxTypeNotSupported, err)
	}

	txDataInternal := &types.ArbitrumInternalTx{ChainId: l2ChainID}
	internalTx := types.NewTx(txDataInternal)
	err = builder.L2.Client.SendTransaction(ctx, internalTx)
	if err == nil && !errors.Is(err, types.ErrTxTypeNotSupported) {
		t.Fatalf("did not receive expected error when submitting arbitrum internal transaction. Want: %v, Got: %v", types.ErrTxTypeNotSupported, err)
	}
}
func TestBlobAndInternalTxsAsDelayedMsgReject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	l1Txs := make([]*types.Transaction, 0, 4)
	txAcceptStatus := make(map[common.Hash]bool, 4)
	l2ChainID := builder.L2Info.Signer.ChainID()

	privKey := GetTestKeyForAccountName(t, "Owner")
	txDataBlob := &types.BlobTx{
		ChainID:   &uint256.Int{l2ChainID.Uint64()},
		Nonce:     0,
		GasFeeCap: &uint256.Int{params.GWei},
		Gas:       500000,
		To:        builder.L2Info.GetAddress("User2"),
		Value:     &uint256.Int{0},
	}
	delayedBlobTx, err := types.SignNewTx(privKey, types.NewCancunSigner(l2ChainID), txDataBlob)
	Require(t, err)
	txAcceptStatus[delayedBlobTx.Hash()] = false
	l1TxBlob := WrapL2ForDelayed(t, delayedBlobTx, builder.L1Info, "User", 100000)
	l1Txs = append(l1Txs, l1TxBlob)

	txDataInternal := &types.ArbitrumInternalTx{ChainId: l2ChainID}
	delayedInternalTx := types.NewTx(txDataInternal)
	txAcceptStatus[delayedInternalTx.Hash()] = false
	l1TxInternal := WrapL2ForDelayed(t, delayedInternalTx, builder.L1Info, "User", 100000)
	l1Txs = append(l1Txs, l1TxInternal)

	delayedTx1 := builder.L2Info.PrepareTx("Owner", "User2", 50001, big.NewInt(10000), nil)
	txAcceptStatus[delayedTx1.Hash()] = false
	l1tx := WrapL2ForDelayed(t, delayedTx1, builder.L1Info, "User", 100000)
	l1Txs = append(l1Txs, l1tx)

	delayedTx2 := builder.L2Info.PrepareTx("Owner", "User2", 50001, big.NewInt(10000), nil)
	txAcceptStatus[delayedTx2.Hash()] = false
	l1tx = WrapL2ForDelayed(t, delayedTx2, builder.L1Info, "User", 100000)
	l1Txs = append(l1Txs, l1tx)

	errs := builder.L1.L1Backend.TxPool().Add(l1Txs, true, false)
	for _, err := range errs {
		Require(t, err)
	}

	confirmLatestBlock(ctx, t, builder.L1Info, builder.L1.Client)
	for _, tx := range l1Txs {
		_, err = builder.L1.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	blocknum, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	for i := int64(0); i <= int64(blocknum); i++ {
		block, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(i))
		Require(t, err)
		for _, tx := range block.Transactions() {
			if _, ok := txAcceptStatus[tx.Hash()]; ok {
				txAcceptStatus[tx.Hash()] = true
			}
		}
	}
	if !txAcceptStatus[delayedTx1.Hash()] || !txAcceptStatus[delayedTx2.Hash()] {
		t.Fatalf("transaction of valid transaction type wasn't accepted as a delayed message")
	}
	if txAcceptStatus[delayedBlobTx.Hash()] {
		t.Fatalf("blob transaction was successfully accepted as a delayed message")
	}
	if txAcceptStatus[delayedInternalTx.Hash()] {
		t.Fatalf("arbitrum internal transaction was successfully accepted as a delayed message")
	}
}
