// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func makeEncodableTestTxes(count int) types.Transactions {
	txes := make(types.Transactions, 0, count)
	for i := range count {
		txes = append(txes, types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i),
			Gas:      21000,
			GasPrice: big.NewInt(1),
			Value:    big.NewInt(int64(i)),
		}))
	}
	return txes
}

// parseL2Message round-trips an L2 message body through the arbos parser.
func parseL2Message(t *testing.T, l2Message []byte) types.Transactions {
	t.Helper()
	msg := &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{Kind: arbostypes.L1MessageType_L2Message},
		L2msg:  l2Message,
	}
	chainId := big.NewInt(412346)
	const arbosVersion = 0
	parsed, err := arbos.ParseL2Transactions(msg, chainId, arbosVersion)
	if err != nil {
		t.Fatalf("failed to parse L2 message: %v", err)
	}
	return parsed
}

func checkTxHashes(t *testing.T, got, expected types.Transactions) {
	t.Helper()
	if len(got) != len(expected) {
		t.Fatalf("expected %d txs, got %d", len(expected), len(got))
	}
	for i, tx := range expected {
		if got[i].Hash() != tx.Hash() {
			t.Errorf("tx %d: expected hash %v, got %v", i, tx.Hash(), got[i].Hash())
		}
	}
}

func TestL2MessageBatchDataFromTxes(t *testing.T) {
	txes := makeEncodableTestTxes(3)
	data, err := L2MessageBatchDataFromTxes(txes)
	if err != nil {
		t.Fatalf("failed to encode batch: %v", err)
	}
	if data[0] != arbos.L2MessageKind_Batch {
		t.Errorf("expected batch kind byte %d, got %d", arbos.L2MessageKind_Batch, data[0])
	}
	checkTxHashes(t, parseL2Message(t, data), txes)

	emptyData, err := L2MessageBatchDataFromTxes(nil)
	if err != nil {
		t.Fatalf("failed to encode empty batch: %v", err)
	}
	if !bytes.Equal(emptyData, []byte{arbos.L2MessageKind_Batch}) {
		t.Errorf("expected empty batch to be just the kind byte, got %v", emptyData)
	}
}

func TestMessageFromTxesSingleTx(t *testing.T) {
	txes := makeEncodableTestTxes(1)
	header := &arbostypes.L1IncomingMessageHeader{Kind: arbostypes.L1MessageType_L2Message}
	msg, err := MessageFromTxes(header, []TxResult{{Tx: txes[0]}})
	if err != nil {
		t.Fatalf("failed to build message: %v", err)
	}
	if msg.L2msg[0] != arbos.L2MessageKind_SignedTx {
		t.Errorf("expected single-tx kind byte %d, got %d", arbos.L2MessageKind_SignedTx, msg.L2msg[0])
	}
	checkTxHashes(t, parseL2Message(t, msg.L2msg), txes)
}
