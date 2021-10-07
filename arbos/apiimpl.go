package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type ArbosAPIImpl struct {
	state        *ArbosState
	currentBlock *blockInProgress
	currentTx    *txInProgress
	coinbaseAddr common.Address
	precompiles  map[common.Address]ArbosPrecompile
}

func NewArbosAPIImpl(backingStorage BackingEvmStorage) *ArbosAPIImpl {
	return &ArbosAPIImpl{
		OpenArbosState(backingStorage),
		nil,
		nil,
		common.BytesToAddress(crypto.Keccak256Hash([]byte("Arbitrum coinbase address")).Bytes()[:20]),
		make(map[common.Address]ArbosPrecompile),
	}
}

func (impl *ArbosAPIImpl) SplitInboxMessage(inputBytes []byte) ([]MessageSegment, error) {
	return ParseIncomingL1Message(bytes.NewReader(inputBytes), impl)
}

func (impl *ArbosAPIImpl) FinalizeBlock(header *types.Header, state *state.StateDB, txs types.Transactions) {
	//TODO: transfer funds from coinbase addr to aggregators and network fee recipient
}

func (impl *ArbosAPIImpl) StartTxHook(msg core.Message, state vm.StateDB) (uint64, error) {  // uint64 return is extra gas to charge
	impl.currentTx = newTxInProgress()
	extraGasChargeWei, aggregator := impl.currentTx.getExtraGasChargeWei()
	gasPrice := msg.GasPrice()
	extraGas := new(big.Int).Div(extraGasChargeWei, gasPrice)
	var extraGasI64 int64
	if extraGas.IsInt64() {
		extraGasI64 = extraGas.Int64()
	} else {
		extraGasI64 = math.MaxInt64
	}
	if aggregator != nil {
		impl.currentBlock.creditAggregator(*aggregator, new(big.Int).Mul(gasPrice, big.NewInt(extraGasI64)))
	}
	return uint64(extraGasI64), nil
}

func (impl *ArbosAPIImpl) EndTxHook(
	msg core.Message,
	totalGasUsed uint64,
	extraGasCharged uint64,
	state vm.StateDB,
) error {
	return nil
}

func (impl *ArbosAPIImpl) Precompiles() map[common.Address]ArbosPrecompile {
	return impl.precompiles
}

type unsignedTxSegment struct {
	api         *ArbosAPIImpl
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
	//TODO: generate Transaction from seg
	seg.api.currentBlock = newBlockInProgress(seg)
	return []*types.Transaction{}, seg.api.state.LastTimestampSeen().Big(), seg.api.coinbaseAddr, nil
}

type blockInProgress struct {
	segmentsRemaining    []MessageSegment
	weiOwedToAggregators map[common.Address]*big.Int
}

func newBlockInProgress(seg MessageSegment) *blockInProgress {
	return &blockInProgress{
		[]MessageSegment{ seg },
		make(map[common.Address]*big.Int),
	}
}

func (bip *blockInProgress) creditAggregator(agg common.Address, wei *big.Int) {
	old, exists := bip.weiOwedToAggregators[agg]
	if !exists {
		old = big.NewInt(0)
	}
	bip.weiOwedToAggregators[agg] = new(big.Int).Add(old, wei)
}

type txInProgress struct {
}

func newTxInProgress() *txInProgress {
	return &txInProgress{}
}

func (tx *txInProgress) getExtraGasChargeWei() (*big.Int, *common.Address) {  // returns wei to charge, address to give it to
	//TODO
	return big.NewInt(0), nil
}

