package arbio

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

var CHAIN_ID = big.NewInt(0xA4B12)

type ArbMessage struct {
	From    common.Address
	Deposit *big.Int
	Tx      *types.Transaction
}

func Process(db state.Database, blockHashRetriever func(uint64) common.Hash, lastStateRoot common.Hash, blockNumber uint64, blockTime *big.Int, msg ArbMessage) (common.Hash, error) {
	statedb, err := state.New(lastStateRoot, db, nil)
	if err != nil {
		return common.Hash{}, err
	}

	chainConfig := params.AllEthashProtocolChanges
	chainConfig.ChainID = CHAIN_ID
	chainConfig.Ethash = nil

	statedb.AddBalance(msg.From, msg.Deposit)

	if msg.Tx != nil {
		statedb.Prepare(msg.Tx.Hash(), 0)

		var gasLimit uint64 = 100_000_000

		bigBlockNumber := new(big.Int).SetUint64(blockNumber)
		ethMsg, err := msg.Tx.AsMessage(types.MakeSigner(chainConfig, bigBlockNumber), nil)
		if err != nil {
			return common.Hash{}, err
		}
		// TODO use a custom message struct that has a custom from address
		if ethMsg.From() != msg.From {
			return common.Hash{}, errors.New("wrong From address for transaction")
		}

		blockContext := vm.BlockContext{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     blockHashRetriever,

			Coinbase:    common.Address{},
			GasLimit:    gasLimit,
			BlockNumber: bigBlockNumber,
			Time:        blockTime,
			Difficulty:  big.NewInt(0),
			BaseFee:     big.NewInt(0),
		}
		evm := vm.NewEVM(blockContext, vm.TxContext{}, statedb, chainConfig, vm.Config{})
		evm.Reset(core.NewEVMTxContext(ethMsg), statedb)

		var gasPool core.GasPool = core.GasPool(gasLimit)
		_, err = core.ApplyMessage(evm, ethMsg, &gasPool)
		if err != nil {
			return common.Hash{}, err
		}
	}

	return statedb.Commit(true)
}
