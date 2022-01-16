//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/statetransfer"
)

func GetGenesisAllocFromJSON(encoded []byte) (map[common.Address]core.GenesisAccount, error) {
	initData := statetransfer.ArbosInitializationInfo{}
	err := json.Unmarshal(encoded, &initData)
	if err != nil {
		return nil, err
	}
	return GetGenesisAllocFromArbos(&initData)
}

func GetGenesisAllocFromArbos(initData *statetransfer.ArbosInitializationInfo) (map[common.Address]core.GenesisAccount, error) {
	stateDBForArbos, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		return nil, err
	}

	arbosState, err := OpenArbosState(stateDBForArbos, &burn.SystemBurner{})
	if err != nil {
		return nil, err
	}

	addrTable := arbosState.AddressTable()
	addrTableSize, err := addrTable.Size()
	if err != nil {
		return nil, err
	}
	if addrTableSize != 0 {
		panic("address table must be empty")
	}
	for i, addr := range initData.AddressTableContents {
		slot, err := addrTable.Register(addr)
		if err != nil {
			return nil, err
		}
		if uint64(i) != slot {
			panic("address table slot mismatch")
		}
	}

	err = merkleAccumulator.InitializeMerkleAccumulatorFromPartials(arbosState.backingStorage.OpenSubStorage(sendMerkleSubspace), initData.SendPartials)
	if err != nil {
		return nil, err
	}
	err = arbosState.L1PricingState().SetDefaultAggregator(initData.DefaultAggregator)
	if err != nil {
		return nil, err
	}
	err = initializeRetryables(arbosState.RetryableState(), initData.RetryableData, 0)
	if err != nil {
		return nil, err
	}

	genesysAlloc := make(map[common.Address]core.GenesisAccount)

	for _, account := range initData.Accounts {
		err = initializeArbosAccount(stateDBForArbos, arbosState, account)
		if err != nil {
			return nil, err
		}
		accountData := core.GenesisAccount{
			Nonce:   account.Nonce,
			Balance: account.EthBalance,
		}
		if account.ContractInfo != nil {
			accountData.Code = account.ContractInfo.Code
			accountData.Storage = account.ContractInfo.ContractStorage
		}
		genesysAlloc[account.Addr] = accountData
	}

	arbosAccount := arbosState.backingStorage.Account()
	arbosStorage := make(map[common.Hash]common.Hash)
	_, err = stateDBForArbos.Commit(false)
	if err != nil {
		return nil, err
	}
	err = stateDBForArbos.ForEachStorage(arbosAccount, func(key common.Hash, value common.Hash) bool { arbosStorage[key] = value; return true })
	if err != nil {
		return nil, err
	}
	genesysAlloc[arbosAccount] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Storage: arbosStorage,
	}
	return genesysAlloc, nil
}

func initializeRetryables(rs *retryables.RetryableState, data []statetransfer.InitializationDataForRetryable, currentTimestampToUse uint64) error {
	for _, r := range data {
		var to *common.Address
		if r.To != (common.Address{}) {
			to = &r.To
		}
		_, err := rs.CreateRetryable(currentTimestampToUse, r.Id, r.Timeout, r.From, to, r.Callvalue, r.Beneficiary, r.Calldata)
		if err != nil {
			return err
		}
	}
	return nil
}

func initializeArbosAccount(statedb *state.StateDB, arbosState *ArbosState, account statetransfer.AccountInitializationInfo) error {
	l1pState := arbosState.L1PricingState()
	if account.AggregatorInfo != nil {
		err := l1pState.SetAggregatorFeeCollector(account.Addr, account.AggregatorInfo.FeeCollector)
		if err != nil {
			return err
		}
		err = l1pState.SetFixedChargeForAggregatorL1Gas(account.Addr, account.AggregatorInfo.BaseFeeL1Gas)
		if err != nil {
			return err
		}
	}
	if account.AggregatorToPay != nil {
		err := l1pState.SetPreferredAggregator(account.Addr, *account.AggregatorToPay)
		if err != nil {
			return err
		}
	}

	return nil
}
