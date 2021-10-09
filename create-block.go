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

func CreateBlock(statedb *state.StateDB, lastBlockHeader *types.Header, chainContext core.ChainContext, segment arbos.MessageSegment) (*types.Block, error) {
	api := arbos.NewArbosAPIImpl(statedb)
	txs, timestamp, coinbase, gasLimit, err := segment.CreateBlockContents(statedb, api)
	if err != nil {
		return nil, err
	}

	var lastBlockHash common.Hash
	blockNumber := new(big.Int)
	if lastBlockHeader != nil {
		lastBlockHash = lastBlockHeader.Hash()
		blockNumber.Add(lastBlockHeader.Number, big.NewInt(1))
	}

	header := &types.Header{
		ParentHash:  lastBlockHash,
		UncleHash:   [32]byte{},
		Coinbase:    coinbase,
		Root:        [32]byte{},  // Filled in later
		TxHash:      [32]byte{},  // Filled in later
		ReceiptHash: [32]byte{},  // Filled in later
		Bloom:       [256]byte{}, // Filled in later
		Difficulty:  big.NewInt(1),
		Number:      blockNumber,
		GasLimit:    gasLimit,
		GasUsed:     0, // Filled in later
		Time:        timestamp.Uint64(),
		Extra:       []byte{},   // Unused
		MixDigest:   [32]byte{}, // Unused
		Nonce:       [8]byte{},  // Unused
		BaseFee:     new(big.Int),
	}

	gasPool := core.GasPool(header.GasLimit)
	receipts := make(types.Receipts, 0, len(txs))
	for _, tx := range txs {
		receipt, err := core.ApplyTransaction(chainConfig, chainContext, &header.Coinbase, &gasPool, statedb, header, tx, &header.GasUsed, vm.Config{})
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}

	api.FinalizeBlock(header, statedb, txs, receipts)

	header.Root = statedb.IntermediateRoot(true)

	block := types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil))
	return block, nil
}
