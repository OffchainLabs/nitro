package arbos

import (
	"bytes"
	"io"
	"math/big"
	"testing"

	"github.com/andybalholm/brotli"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestSerializeAndParseL1Message(t *testing.T) {
	chainId := big.NewInt(6345634)
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
	newMsg, err := ParseIncomingL1Message(bytes.NewReader(serialized))
	if err != nil {
		t.Error(err)
	}
	segments, err := ExtractL1MessageSegments(newMsg, chainId)
	if err != nil {
		t.Error(err)
	}
	if len(segments) != 1 {
		t.Fatal("unexpected segment count")
	}
	segment := segments[0]
	if len(segment.txes) != 0 {
		t.Fatal("unexpected tx count")
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

	header.RequestId = common.BigToHash(big.NewInt(4))
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

	RunMessagesThroughAPI(t, [][]byte{serialized, serialized2}, statedb)

	balanceAfter := statedb.GetBalance(addr)
	if balanceAfter.Cmp(new(big.Int).Add(balance.Big(), balance2.Big())) != 0 {
		t.Fatal()
	}
}

type TestChainContext struct {
}

func (r *TestChainContext) Engine() consensus.Engine {
	return Engine{}
}

func (r *TestChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	return &types.Header{}
}

var testChainConfig = &params.ChainConfig{
	ChainID:             big.NewInt(0),
	HomesteadBlock:      big.NewInt(0),
	DAOForkBlock:        nil,
	DAOForkSupport:      true,
	EIP150Block:         big.NewInt(0),
	EIP150Hash:          common.Hash{},
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	MuirGlacierBlock:    big.NewInt(0),
	BerlinBlock:         big.NewInt(0),
	LondonBlock:         big.NewInt(0),
}

func RunMessagesThroughAPI(t *testing.T, msgs [][]byte, statedb *state.StateDB) {
	chainId := big.NewInt(6456554)
	for _, data := range msgs {
		msg, err := ParseIncomingL1Message(bytes.NewReader(data))
		if err != nil {
			t.Error(err)
		}
		segments, err := ExtractL1MessageSegments(msg, chainId)
		if err != nil {
			t.Error(err)
		}
		for _, segment := range segments {
			chainContext := &TestChainContext{}
			header := &types.Header{
				Number:     big.NewInt(1000),
				Difficulty: big.NewInt(1000),
			}
			gasPool := core.GasPool(100000)
			for _, tx := range segment.txes {
				_, err := core.ApplyTransaction(testChainConfig, chainContext, nil, &gasPool, statedb, header, tx, &header.GasUsed, vm.Config{})
				if err != nil {
					t.Fatal(err)
				}
			}

			FinalizeBlock(nil, nil, nil)
		}
	}
}
