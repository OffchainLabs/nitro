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
	"github.com/offchainlabs/arbstate/arbos"
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

func BuildBlock(statedb *state.StateDB, blockData *arbos.BlockData, chainContext core.ChainContext) (*types.Block, error) {
	gasPool := core.GasPool(blockData.Header.GasLimit)
	receipts := make(types.Receipts, 0, len(blockData.Txes))
	for _, tx := range blockData.Txes {
		receipt, err := core.ApplyTransaction(
			chainConfig,
			chainContext,
			&blockData.Header.Coinbase,
			&gasPool,
			statedb,
			blockData.Header,
			tx,
			&blockData.Header.GasUsed,
			vm.Config{},
			)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	blockData.Header.Root = statedb.IntermediateRoot(true)

	block := types.NewBlock(blockData.Header, blockData.Txes, nil, receipts, trie.NewStackTrie(nil))
	return block, nil
}
