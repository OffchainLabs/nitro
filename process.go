package arbstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
)

//var CHAIN_ID = big.NewInt(0xA4B12)
var CHAIN_ID = big.NewInt(4) // Rinkeby

const GAS_LIMIT uint64 = 100_000_000

type ArbMessage struct {
	From      common.Address
	Deposit   *big.Int
	Timestamp uint64
	Tx        *types.Transaction `rlp:"optional"`
}

type BlockRetriever interface {
	GetBlockHash(uint64) common.Hash
}

func Process(statedb *state.StateDB, lastBlockHeader *types.Header, retriever BlockRetriever, msg ArbMessage) (*types.Header, error) {
	chainConfig := params.AllEthashProtocolChanges
	chainConfig.ChainID = CHAIN_ID
	chainConfig.Ethash = nil

	var blockNumber uint64
	var lastBlockHash common.Hash
	if lastBlockHeader != nil {
		blockNumber = lastBlockHeader.Number.Uint64() + 1
	} else {
		blockNumber = 1
		lastBlockHash = lastBlockHeader.Hash()
	}

	if lastBlockHeader != nil && msg.Timestamp < lastBlockHeader.Time {
		msg.Timestamp = lastBlockHeader.Time
	}

	if msg.Deposit != nil {
		statedb.AddBalance(msg.From, msg.Deposit)
	}

	var gasPool core.GasPool = core.GasPool(GAS_LIMIT)

	if msg.Tx != nil {
		statedb.Prepare(msg.Tx.Hash(), 0)

		bigBlockNumber := new(big.Int).SetUint64(blockNumber)
		ethMsg, err := msg.Tx.AsMessage(types.MakeSigner(chainConfig, bigBlockNumber), nil)
		if err != nil {
			return nil, err
		}
		// TODO use a custom message struct that has a custom from address
		if ethMsg.From() != msg.From {
			return nil, errors.New("wrong From address for transaction")
		}

		blockContext := vm.BlockContext{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     retriever.GetBlockHash,

			Coinbase:    common.Address{},
			GasLimit:    GAS_LIMIT,
			BlockNumber: bigBlockNumber,
			Time:        new(big.Int).SetUint64(msg.Timestamp),
			Difficulty:  big.NewInt(0),
			BaseFee:     big.NewInt(0),
		}
		txContext := core.NewEVMTxContext(ethMsg)
		evm := vm.NewEVM(blockContext, txContext, statedb, chainConfig, vm.Config{})

		_, err = core.ApplyMessage(evm, ethMsg, &gasPool)
		if err != nil {
			return nil, err
		}
	}

	newHeader := &types.Header{
		ParentHash:  lastBlockHash,
		UncleHash:   [32]byte{},
		Coinbase:    [20]byte{},
		Root:        statedb.IntermediateRoot(true),
		TxHash:      [32]byte{},  // TODO
		ReceiptHash: [32]byte{},  // TODO
		Bloom:       [256]byte{}, // TODO
		Difficulty:  big.NewInt(1),
		Number:      new(big.Int).SetUint64(blockNumber),
		GasLimit:    GAS_LIMIT,
		GasUsed:     GAS_LIMIT - gasPool.Gas(),
		Time:        msg.Timestamp,
		Extra:       []byte{},
		MixDigest:   [32]byte{},
		Nonce:       [8]byte{},
		BaseFee:     big.NewInt(1), // TODO
	}
	return newHeader, nil
}
