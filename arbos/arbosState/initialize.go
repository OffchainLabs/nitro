// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbosState

import (
	"errors"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/statetransfer"
)

func MakeGenesisBlock(parentHash common.Hash, blockNumber uint64, timestamp uint64, stateRoot common.Hash, chainConfig *params.ChainConfig) *types.Block {
	head := &types.Header{
		Number:     new(big.Int).SetUint64(blockNumber),
		Nonce:      types.EncodeNonce(1), // the genesis block reads the init message
		Time:       timestamp,
		ParentHash: parentHash,
		Extra:      nil,
		GasLimit:   l2pricing.GethBlockGasLimit,
		GasUsed:    0,
		BaseFee:    big.NewInt(l2pricing.InitialBaseFeeWei),
		Difficulty: big.NewInt(1),
		MixDigest:  common.Hash{},
		Coinbase:   common.Address{},
		Root:       stateRoot,
	}

	genesisHeaderInfo := types.HeaderInfo{
		SendRoot:           common.Hash{},
		SendCount:          0,
		L1BlockNumber:      0,
		ArbOSFormatVersion: chainConfig.ArbitrumChainParams.InitialArbOSVersion,
	}
	genesisHeaderInfo.UpdateHeaderWithInfo(head)

	return types.NewBlock(head, nil, nil, nil, trie.NewStackTrie(nil))
}

func InitializeArbosInDatabase(db ethdb.Database, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig) (common.Hash, error) {
	stateDatabase := state.NewDatabase(db)
	statedb, err := state.New(common.Hash{}, stateDatabase, nil)
	if err != nil {
		log.Fatal("failed to init empty statedb", err)
	}

	burner := burn.NewSystemBurner(nil, false)
	arbosState, err := InitializeArbosState(statedb, burner, chainConfig)
	if err != nil {
		log.Fatal("failed to open the ArbOS state", err)
	}

	addrTable := arbosState.AddressTable()
	addrTableSize, err := addrTable.Size()
	if err != nil {
		return common.Hash{}, err
	}
	if addrTableSize != 0 {
		return common.Hash{}, errors.New("address table must be empty")
	}
	addressReader, err := initData.GetAddressTableReader()
	if err != nil {
		return common.Hash{}, err
	}
	for i := 0; addressReader.More(); i++ {
		addr, err := addressReader.GetNext()
		if err != nil {
			return common.Hash{}, err
		}
		slot, err := addrTable.Register(*addr)
		if err != nil {
			return common.Hash{}, err
		}
		if uint64(i) != slot {
			return common.Hash{}, errors.New("address table slot mismatch")
		}
	}
	if err := addressReader.Close(); err != nil {
		return common.Hash{}, err
	}

	retriableReader, err := initData.GetRetriableDataReader()
	if err != nil {
		return common.Hash{}, err
	}
	err = initializeRetryables(arbosState.RetryableState(), retriableReader, 0)
	if err != nil {
		return common.Hash{}, err
	}

	accountDataReader, err := initData.GetAccountDataReader()
	if err != nil {
		return common.Hash{}, err
	}
	for accountDataReader.More() {
		account, err := accountDataReader.GetNext()
		if err != nil {
			return common.Hash{}, err
		}
		err = initializeArbosAccount(statedb, arbosState, *account)
		if err != nil {
			return common.Hash{}, err
		}
		statedb.SetBalance(account.Addr, account.EthBalance)
		statedb.SetNonce(account.Addr, account.Nonce)
		if account.ContractInfo != nil {
			statedb.SetCode(account.Addr, account.ContractInfo.Code)
			for k, v := range account.ContractInfo.ContractStorage {
				statedb.SetState(account.Addr, k, v)
			}
		}
	}
	if err := accountDataReader.Close(); err != nil {
		return common.Hash{}, err
	}
	root, err := statedb.Commit(true)
	if err != nil {
		return common.Hash{}, err
	}
	err = stateDatabase.TrieDB().Commit(root, true, nil)
	if err != nil {
		return common.Hash{}, err
	}
	return root, nil
}

func initializeRetryables(rs *retryables.RetryableState, initData statetransfer.RetriableDataReader, currentTimestampToUse uint64) error {
	for initData.More() {
		r, err := initData.GetNext()
		if err != nil {
			return err
		}
		var to *common.Address
		if r.To != (common.Address{}) {
			to = &r.To
		}
		_, err = rs.CreateRetryable(r.Id, r.Timeout, r.From, to, r.Callvalue, r.Beneficiary, r.Calldata)
		if err != nil {
			return err
		}
	}
	return initData.Close()
}

func initializeArbosAccount(statedb *state.StateDB, arbosState *ArbosState, account statetransfer.AccountInitializationInfo) error {
	l1pState := arbosState.L1PricingState()
	if account.AggregatorInfo != nil {
		err := l1pState.SetAggregatorFeeCollector(account.Addr, account.AggregatorInfo.FeeCollector)
		if err != nil {
			return err
		}
	}
	if account.AggregatorToPay != nil {
		err := l1pState.SetUserSpecifiedAggregator(account.Addr, account.AggregatorToPay)
		if err != nil {
			return err
		}
	}

	return nil
}
