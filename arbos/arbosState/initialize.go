//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"errors"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/statetransfer"
)

func InitializeArbosInDatabase(db ethdb.Database, initData statetransfer.InitDataReader) (common.Hash, error) {
	stateDatabase := state.NewDatabase(db)
	statedb, err := state.New(common.Hash{}, stateDatabase, nil)
	if err != nil {
		log.Fatal("failed to init empty statedb", err)
	}

	burner := burn.NewSystemBurner(false)
	arbosState, err := InitializeArbosState(statedb, burner)
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
	if listname, err := initData.NextList(); err != nil || listname != "AddressTableContents" {
		return common.Hash{}, fmt.Errorf("expected: AddressTableContents, found: %v err: %w", listname, err)
	}
	for i := 0; initData.More(); i++ {
		addr, err := initData.GetNextAddress()
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
	if err := initData.CloseList(); err != nil {
		return common.Hash{}, errors.New("close AddressTableContents failed")
	}

	if listname, err := initData.NextList(); err != nil || listname != "RetryableData" {
		return common.Hash{}, fmt.Errorf("expected: RetryableData, found: %v err: %w", listname, err)
	}
	err = initializeRetryables(arbosState.RetryableState(), initData, 0)
	if err != nil {
		return common.Hash{}, err
	}
	if err := initData.CloseList(); err != nil {
		return common.Hash{}, errors.New("close RetryableData failed")
	}

	if listname, err := initData.NextList(); err != nil || listname != "Accounts" {
		return common.Hash{}, fmt.Errorf("expected: Accounts, found: %v err: %w", listname, err)
	}
	for initData.More() {
		account, err := initData.GetNextAccountInit()
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
	if err := initData.CloseList(); err != nil {
		return common.Hash{}, errors.New("close Accounts failed")
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

func initializeRetryables(rs *retryables.RetryableState, initData statetransfer.InitDataReader, currentTimestampToUse uint64) error {
	for initData.More() {
		r, err := initData.GetNextRetryableData()
		if err != nil {
			return err
		}
		var to *common.Address
		if r.To != (common.Address{}) {
			to = &r.To
		}
		_, err = rs.CreateRetryable(currentTimestampToUse, r.Id, r.Timeout, r.From, to, r.Callvalue, r.Beneficiary, r.Calldata)
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
		err := l1pState.SetUserSpecifiedAggregator(account.Addr, account.AggregatorToPay)
		if err != nil {
			return err
		}
	}

	return nil
}
