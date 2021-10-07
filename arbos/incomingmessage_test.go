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

	if statedb.GetBalance(addr).Cmp(big.NewInt(0)) != 0 {
		t.Fatal()
	}

	header := L1IncomingMessageHeader{
		L1MessageType_EthDeposit,
		common.BigToAddress(big.NewInt(4684)),
		common.BigToHash(big.NewInt(864513)),
		common.BigToHash(big.NewInt(8794561564)),
		common.BigToHash(big.NewInt(3)),
		common.BigToHash(big.NewInt(10000000000000)),
	}
	msgBuf := bytes.Buffer{}
	if err := AddressToWriter(addr, &msgBuf); err != nil {
		t.Error(err)
	}
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

	segments, err := api.SplitInboxMessage(serialized)
	if err != nil {
		t.Error(err)
	}
	if len(segments) != 1 {
		t.Fatal()
	}

	txs, _, _, err := segments[0].CreateBlockContents(statedb)
	if err != nil {
		t.Error(err)
	}
	if len(txs) != 0 {
		t.Fatal()
	}

	api.FinalizeBlock(nil, statedb, []*types.Transaction{})

	balanceAfter := statedb.GetBalance(addr)
	if balanceAfter.Cmp(balance.Big()) != 0 {
		t.Fatal()
	}
}