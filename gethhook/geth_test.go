// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethhook

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
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
	ArbitrumChainParams: params.ArbitrumDevTestParams(),
}

func TestEthDepositMessage(t *testing.T) {

	_, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	addr := common.HexToAddress("0x32abcdeffffff")
	balance := common.BigToHash(big.NewInt(789789897789798))
	balance2 := common.BigToHash(big.NewInt(98))

	if statedb.GetBalance(addr).Sign() != 0 {
		Fail(t)
	}

	firstRequestId := common.BigToHash(big.NewInt(3))
	header := arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_EthDeposit,
		Poster:      addr,
		BlockNumber: 864513,
		Timestamp:   8794561564,
		RequestId:   &firstRequestId,
		L1BaseFee:   big.NewInt(10000000000000),
	}
	msgBuf := bytes.Buffer{}
	if err := util.AddressToWriter(addr, &msgBuf); err != nil {
		t.Error(err)
	}
	if err := util.HashToWriter(balance, &msgBuf); err != nil {
		t.Error(err)
	}
	msg := arbostypes.L1IncomingMessage{
		Header: &header,
		L2msg:  msgBuf.Bytes(),
	}

	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}

	secondRequestId := common.BigToHash(big.NewInt(4))
	header.RequestId = &secondRequestId
	header.Poster = util.RemapL1Address(addr)
	msgBuf2 := bytes.Buffer{}
	if err := util.AddressToWriter(addr, &msgBuf2); err != nil {
		t.Error(err)
	}
	if err := util.HashToWriter(balance2, &msgBuf2); err != nil {
		t.Error(err)
	}
	msg2 := arbostypes.L1IncomingMessage{
		Header: &header,
		L2msg:  msgBuf2.Bytes(),
	}
	serialized2, err := msg2.Serialize()
	if err != nil {
		t.Error(err)
	}

	RunMessagesThroughAPI(t, [][]byte{serialized, serialized2}, statedb)

	balanceAfter := statedb.GetBalance(addr)
	if balanceAfter.Cmp(new(big.Int).Add(balance.Big(), balance2.Big())) != 0 {
		Fail(t)
	}
}

func RunMessagesThroughAPI(t *testing.T, msgs [][]byte, statedb *state.StateDB) {
	chainId := big.NewInt(6456554)
	for _, data := range msgs {
		msg, err := arbostypes.ParseIncomingL1Message(bytes.NewReader(data), nil)
		if err != nil {
			t.Error(err)
		}
		txes, err := arbos.ParseL2Transactions(msg, chainId, nil)
		if err != nil {
			t.Error(err)
		}
		chainContext := &TestChainContext{}
		header := &types.Header{
			Number:     big.NewInt(1000),
			Difficulty: big.NewInt(1000),
		}
		gasPool := core.GasPool{}
		gasPool.AddGas(100000)
		for _, tx := range txes {
			_, _, err := core.ApplyTransaction(testChainConfig, chainContext, nil, &gasPool, statedb, header, header.ExcessDataGas, tx, &header.GasUsed, vm.Config{}, nil)
			if err != nil {
				Fail(t, err)
			}
		}

		arbos.FinalizeBlock(nil, nil, statedb, testChainConfig)
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
