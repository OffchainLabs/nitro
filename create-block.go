package arbstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/arbstate/arbos2"
)

var chainConfig *params.ChainConfig = &params.ChainConfig{
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

func CreateBlock(statedb *state.StateDB, lastBlockHeader *types.Header, chainContext core.ChainContext, segment arbos2.MessageSegment) (*types.Block, error) {
	block, err := arbos2.CreateBlockTemplate(statedb.Copy(), lastBlockHeader, segment)
	if err != nil {
		return nil, err
	}

	header := block.Header()
	txs := block.Transactions()
	var gasPool core.GasPool = core.GasPool(header.GasLimit)
	var receipts types.Receipts
	for _, tx := range txs {
		receipt, err := core.ApplyTransaction(chainConfig, chainContext, &header.Coinbase, &gasPool, statedb, header, tx, &header.GasUsed, vm.Config{})
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}

	arbos2.Finalize(header, statedb, txs, receipts)

	header.Root = statedb.IntermediateRoot(true)

	block = types.NewBlock(header, block.Transactions(), nil, receipts, trie.NewStackTrie(nil))
	return block, nil
}
