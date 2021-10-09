package arbos

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

type ArbosAPIImpl struct {
	stateDB vm.StateDB
	state        *ArbosState
	currentBlock *blockInProgress
	currentTx    *txInProgress
	coinbaseAddr common.Address
}

func NewArbosAPIImpl(stateDB vm.StateDB) *ArbosAPIImpl {
	return &ArbosAPIImpl{
		stateDB:      stateDB,
		state:        OpenArbosState(stateDB),
		coinbaseAddr: common.BytesToAddress(crypto.Keccak256Hash([]byte("Arbitrum coinbase address")).Bytes()[:20]),
	}
}

func (impl *ArbosAPIImpl) SplitInboxMessage(inputBytes []byte) ([]MessageSegment, error) {
	return ParseIncomingL1Message(bytes.NewReader(inputBytes))
}

func (impl *ArbosAPIImpl) FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts) {
	// process deposit, if there is one
	deposit := impl.currentBlock.depositSegmentRemaining
	if deposit != nil {
		impl.stateDB.AddBalance(deposit.addr, deposit.balance.Big())
	}

	// reimburse aggregators from the coinbase address
	origCoinbaseWei := impl.stateDB.GetBalance(impl.coinbaseAddr)
	newCoinbaseWei := new(big.Int).Set(origCoinbaseWei)
	for agg, amount := range impl.currentBlock.weiOwedToAggregators {
		if amount.Cmp(newCoinbaseWei) <= 0 {
			newCoinbaseWei = new(big.Int).Sub(newCoinbaseWei, amount)
			impl.stateDB.AddBalance(agg, amount)
		}
	}
	impl.stateDB.SubBalance(impl.coinbaseAddr, new(big.Int).Sub(origCoinbaseWei, newCoinbaseWei))
}

func (impl *ArbosAPIImpl) StartTxHook(msg core.Message) (uint64, error) { // uint64 return is extra gas to charge
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
	extraGasChargeWei = new(big.Int).Mul(gasPrice, big.NewInt(extraGasI64))
	if aggregator != nil {
		impl.currentBlock.creditAggregator(*aggregator, extraGasChargeWei)
	}
	return uint64(extraGasI64), nil
}

func (impl *ArbosAPIImpl) EndTxHook(
	msg core.Message,
	totalGasUsed uint64,
	extraGasCharged uint64,
) error {
	return nil
}

func (impl *ArbosAPIImpl) GetExtraSegmentToBeNextBlock() *MessageSegment {
	return nil
}

func Precompiles() map[common.Address]ArbosPrecompile {
	return nil
}

type ethDeposit struct {
	addr    common.Address
	balance common.Hash
}

func (deposit *ethDeposit) CreateBlockContents(
	_ *state.StateDB,
	api     *ArbosAPIImpl,
) (
	[]*types.Transaction, // transactions to (try to) put in the block
	*big.Int, // timestamp
	common.Address, // coinbase address
	uint64, // gas limit
	error,
) {
	api.currentBlock = newBlockInProgress(nil, deposit)
	var gasLimit uint64 = 1e10 // TODO
	return []*types.Transaction{}, api.state.LastTimestampSeen(), api.coinbaseAddr, gasLimit, nil
}

type txSegment struct {
	tx  *types.Transaction
}

func (seg *txSegment) CreateBlockContents(
	_ *state.StateDB,
	api     *ArbosAPIImpl,
) (
	[]*types.Transaction, // transactions to (try to) put in the block
	*big.Int, // timestamp
	common.Address, // coinbase address
	uint64, // gas limit
	error,
) {
	api.currentBlock = newBlockInProgress(seg, nil)
	var gasLimit uint64 = 1e10 // TODO
	return []*types.Transaction{seg.tx}, api.state.LastTimestampSeen(), api.coinbaseAddr, gasLimit, nil
}

type blockInProgress struct {
	txSegmentRemaining      MessageSegment
	depositSegmentRemaining *ethDeposit
	weiOwedToAggregators    map[common.Address]*big.Int
}

func newBlockInProgress(seg MessageSegment, deposit *ethDeposit) *blockInProgress {
	return &blockInProgress{
		seg,
		deposit,
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

func (tx *txInProgress) getExtraGasChargeWei() (*big.Int, *common.Address) { // returns wei to charge, address to give it to
	//TODO
	return big.NewInt(0), nil
}

// Implementation of Transaction for txSegment

func (seg *txSegment) txType() byte                          { return seg.tx.Type() }
func (seg *txSegment) chainID() *big.Int                     { return seg.tx.ChainId() }
func (seg *txSegment) accessList() types.AccessList          { return seg.tx.AccessList() }
func (seg *txSegment) data() []byte                          { return seg.tx.Data() }
func (seg *txSegment) gas() uint64                           { return seg.tx.Gas() }
func (seg *txSegment) gasPrice() *big.Int                    { return seg.tx.GasPrice() }
func (seg *txSegment) gasTipCap() *big.Int                   { return seg.tx.GasTipCap() }
func (seg *txSegment) gasFeeCap() *big.Int                   { return seg.tx.GasFeeCap() }
func (seg *txSegment) value() *big.Int                       { return seg.tx.Value() }
func (seg *txSegment) nonce() uint64                         { return seg.tx.Nonce() }
func (seg *txSegment) to() *common.Address                   { return seg.tx.To() }
func (seg txSegment) rawSignatureValues() (v, r, s *big.Int) { return seg.tx.RawSignatureValues() }
