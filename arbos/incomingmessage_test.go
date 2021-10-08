package arbos

import (
	"bytes"
	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"io"
	"math/big"
	"testing"
)

func TestSerializeAndParseL1Message(t *testing.T) {
	header := L1IncomingMessageHeader{
		L1MessageType_EndOfBlock,
		common.BigToAddress(big.NewInt(4684)),
		common.BigToHash(big.NewInt(864513)),
		common.BigToHash(big.NewInt(8794561564)),
		common.BigToHash(big.NewInt(3)),
		common.BigToHash(big.NewInt(10000000000000)),
	}
	msg := L1IncomingMessage{
		&header,
		[]byte{3, 2, 1},
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}
	segments, err := ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err != nil {
		t.Error(err)
	}
	if len(segments) != 0 {
		t.Fail()
	}
}

func TestBrotli(t *testing.T) {
	orig := []byte("This is a long and repetitive string. Yadda yadda yadda yadda yadda. The quick brown fox jumped over the lazy dog.")
	outBuf := bytes.Buffer{}
	bwr := brotli.NewWriter(&outBuf)
	_, err := bwr.Write(orig)
	if err != nil {
		t.Error(err)
	}
	bwr.Flush()
	compressed := outBuf.Bytes()
	if len(compressed) >= len(orig) {
		t.Fatal("compression didn't make it smaller")
	}
	decompressor := brotli.NewReader(bytes.NewReader(compressed))
	result, err := io.ReadAll(decompressor)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(orig, result) != 0 {
		t.Fatal("decompressed data doesn't match original")
	}
}

func TestEthDepositMessage(t *testing.T) {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		panic("failed to init empty statedb")
	}
	api := NewArbosAPIImpl(statedb)


	addr := common.BigToAddress(big.NewInt(51395080))
	balance := common.BigToHash(big.NewInt(789789897789798))
	balance2 := common.BigToHash(big.NewInt(98))

	if statedb.GetBalance(addr).Cmp(big.NewInt(0)) != 0 {
		t.Fatal()
	}

	header := L1IncomingMessageHeader{
		L1MessageType_EthDeposit,
		addr,
		common.BigToHash(big.NewInt(864513)),
		common.BigToHash(big.NewInt(8794561564)),
		common.BigToHash(big.NewInt(3)),
		common.BigToHash(big.NewInt(10000000000000)),
	}
	msgBuf := bytes.Buffer{}
	if err := HashToWriter(balance, &msgBuf); err != nil {
		t.Error(err)
	}
	msg := L1IncomingMessage{
		&header,
		msgBuf.Bytes(),
	}

	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}

	header.requestId = common.BigToHash(big.NewInt(4))
	msgBuf2 := bytes.Buffer{}
	if err := HashToWriter(balance2, &msgBuf2); err != nil {
		t.Error(err)
	}
	msg2 := L1IncomingMessage{
		&header,
		msgBuf2.Bytes(),
	}
	serialized2, err := msg2.Serialize()
	if err != nil {
		t.Error(err)
	}

	RunMessagesThroughAPI([][]byte{ serialized, serialized2 }, api, statedb, t)

	balanceAfter := statedb.GetBalance(addr)
	if balanceAfter.Cmp(new(big.Int).Add(balance.Big(), balance2.Big())) != 0 {
		t.Fatal()
	}
}

func RunMessagesThroughAPI(msgs [][]byte, api ArbosAPI, statedb *state.StateDB, t *testing.T) {
	for _, msg := range msgs {
		segments, err := api.SplitInboxMessage(msg)
		if err != nil {
			t.Error(err)
		}
		for _, segment := range segments {
			txs, _, _, err := segment.CreateBlockContents(statedb)
			if err != nil {
				t.Error(err)
			}
			for _, tx := range txs {
				_ = tx
				msg, err := tx.AsMessage(nil, big.NewInt(1000000000))
				if err != nil {
					t.Error(err)
				}
				extraGas, err := api.StartTxHook(msg, statedb)
				if err != nil {
					t.Error(err)
				}
				err = api.EndTxHook(msg, extraGas, extraGas, statedb)
				if err != nil {
					t.Error(err)
				}
			}

			api.FinalizeBlock(nil, statedb, []*types.Transaction{})
		}
	}
}
