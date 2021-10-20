package arbstate

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
)


type TestChainContext struct {
}

func (r *TestChainContext) Engine() consensus.Engine {
	return arbos.Engine{}
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

	header := arbos.L1IncomingMessageHeader{
		arbos.L1MessageType_EthDeposit,
		addr,
		common.BigToHash(big.NewInt(864513)),
		common.BigToHash(big.NewInt(8794561564)),
		common.BigToHash(big.NewInt(3)),
		common.BigToHash(big.NewInt(10000000000000)),
	}
	msgBuf := bytes.Buffer{}
	if err := arbos.HashToWriter(balance, &msgBuf); err != nil {
		t.Error(err)
	}
	msg := arbos.L1IncomingMessage{
		&header,
		msgBuf.Bytes(),
	}

	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}

	header.RequestId = common.BigToHash(big.NewInt(4))
	msgBuf2 := bytes.Buffer{}
	if err := arbos.HashToWriter(balance2, &msgBuf2); err != nil {
		t.Error(err)
	}
	msg2 := arbos.L1IncomingMessage{
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


func RunMessagesThroughAPI(t *testing.T, msgs [][]byte, statedb *state.StateDB) {
	chainId := big.NewInt(6456554)
	for _, data := range msgs {
		msg, err := arbos.ParseIncomingL1Message(bytes.NewReader(data))
		if err != nil {
			t.Error(err)
		}
		segment, err := arbos.IncomingMessageToSegment(msg, chainId)
		if err != nil {
			t.Error(err)
		}
		chainContext := &TestChainContext{}
		header := &types.Header{
			Number:     big.NewInt(1000),
			Difficulty: big.NewInt(1000),
		}
		gasPool := core.GasPool(100000)
		for _, tx := range segment.Txes {
			_, err := core.ApplyTransaction(testChainConfig, chainContext, nil, &gasPool, statedb, header, tx, &header.GasUsed, vm.Config{})
			if err != nil {
				t.Fatal(err)
			}
		}

		arbos.FinalizeBlock(nil, nil, nil, statedb)
	}
}

