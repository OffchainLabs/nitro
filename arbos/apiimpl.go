package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
)

type ArbosAPIImpl struct {
	state *ArbosState
}

func NewArbosAPIImpl() *ArbosAPIImpl {
	return &ArbosAPIImpl{}
}

func (impl *ArbosAPIImpl) SplitInboxMessage(inputBytes []byte) ([]MessageSegment, error) {
	return ParseIncomingL1Message(bytes.NewReader(inputBytes), impl.state)
}

func (impl *ArbosAPIImpl) FinalizeBlock(header *types.Header, state *state.StateDB, txs types.Transactions) {
	//TODO
}

func (impl *ArbosAPIImpl) StartTxHook(msg core.Message, state vm.StateDB) (uint64, error) {  // uint64 return is extra gas to charge
	//TODO
	return 0, nil
}

func (impl *ArbosAPIImpl) EndTxHook(
	msg core.Message,
	totalGasUsed uint64,
	extraGasCharged uint64,
	state vm.StateDB,
) error {
	//TODO
	return nil
}

func (impl *ArbosAPIImpl) Precompiles() map[common.Address]ArbosPrecompile {
	//TODO
	return make(map[common.Address]ArbosPrecompile)
}

type unsignedTxSegment struct {
	arbosState  *ArbosState
	gasLimit    *big.Int
	gasPrice    *big.Int
	nonce       *big.Int
	destination common.Address
	callvalue   *big.Int
	calldata    []byte
}

func (seg *unsignedTxSegment) CreateBlockContents(
	beforeState *state.StateDB,
) (
	[]*types.Transaction, // transactions to (try to) put in the block
	*big.Int,             // timestamp
	common.Address,       // coinbase address
	error,
) {
	//TODO
	return []*types.Transaction{}, big.NewInt(0), common.Address{}, nil
}

