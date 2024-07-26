// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbosState

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	prand := testhelpers.NewPseudoRandomDataSource(t, 1)
	tryMarshalUnmarshal(
		&statetransfer.ArbosInitializationInfo{
			AddressTableContents: []common.Address{prand.GetAddress()},
			RetryableData:        []statetransfer.InitializationDataForRetryable{pseudorandomRetryableInitForTesting(prand)},
			Accounts:             []statetransfer.AccountInitializationInfo{pseudorandomAccountInitInfoForTesting(prand)},
		},
		t,
	)
}

func tryMarshalUnmarshal(input *statetransfer.ArbosInitializationInfo, t *testing.T) {
	marshaled, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(marshaled) {
		t.Fatal()
	}
	if len(marshaled) == 0 {
		t.Fatal()
	}

	output := statetransfer.ArbosInitializationInfo{}
	err = json.Unmarshal(marshaled, &output)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.AddressTableContents) != 1 {
		t.Fatal(output)
	}

	var initData statetransfer.ArbosInitializationInfo
	err = json.Unmarshal(marshaled, &initData)
	Require(t, err)

	raw := rawdb.NewMemoryDatabase()

	initReader := statetransfer.NewMemoryInitDataReader(&initData)
	chainConfig := params.ArbitrumDevTestChainConfig()

	cacheConfig := core.DefaultCacheConfigWithScheme(env.GetTestStateScheme())
	stateroot, err := InitializeArbosInDatabase(raw, cacheConfig, initReader, chainConfig, arbostypes.TestInitMessage, 0, 0)
	Require(t, err)

	triedbConfig := cacheConfig.TriedbConfig()
	stateDb, err := state.New(stateroot, state.NewDatabaseWithConfig(raw, triedbConfig), nil)
	Require(t, err)

	arbState, err := OpenArbosState(stateDb, &burn.SystemBurner{})
	Require(t, err)
	checkAddressTable(arbState, input.AddressTableContents, t)
	checkRetryables(arbState, input.RetryableData, t)
	checkAccounts(stateDb, arbState, input.Accounts, t)
}

func pseudorandomRetryableInitForTesting(prand *testhelpers.PseudoRandomDataSource) statetransfer.InitializationDataForRetryable {
	return statetransfer.InitializationDataForRetryable{
		Id:          prand.GetHash(),
		Timeout:     prand.GetUint64(),
		From:        prand.GetAddress(),
		To:          prand.GetAddress(),
		Callvalue:   new(big.Int).SetBytes(prand.GetHash().Bytes()[1:]),
		Beneficiary: prand.GetAddress(),
		Calldata:    prand.GetData(256),
	}
}

func pseudorandomAccountInitInfoForTesting(prand *testhelpers.PseudoRandomDataSource) statetransfer.AccountInitializationInfo {
	aggToPay := prand.GetAddress()
	return statetransfer.AccountInitializationInfo{
		Addr:       prand.GetAddress(),
		Nonce:      prand.GetUint64(),
		EthBalance: prand.GetHash().Big(),
		ContractInfo: &statetransfer.AccountInitContractInfo{
			Code:            prand.GetData(256),
			ContractStorage: pseudorandomHashHashMapForTesting(prand, 16),
		},
		AggregatorInfo: &statetransfer.AccountInitAggregatorInfo{
			FeeCollector: prand.GetAddress(),
			BaseFeeL1Gas: prand.GetHash().Big(),
		},
		AggregatorToPay: &aggToPay,
	}
}

func pseudorandomHashHashMapForTesting(prand *testhelpers.PseudoRandomDataSource, maxItems uint64) map[common.Hash]common.Hash {
	size := int(prand.GetUint64() % maxItems)
	ret := make(map[common.Hash]common.Hash)
	for i := 0; i < size; i++ {
		ret[prand.GetHash()] = prand.GetHash()
	}
	return ret
}

func checkAddressTable(arbState *ArbosState, addrTable []common.Address, t *testing.T) {
	atab := arbState.AddressTable()
	atabSize, err := atab.Size()
	Require(t, err)
	if atabSize != uint64(len(addrTable)) {
		Fail(t)
	}
	for i, addr := range addrTable {
		res, exists, err := atab.LookupIndex(uint64(i))
		Require(t, err)
		if !exists {
			Fail(t)
		}
		if res != addr {
			Fail(t)
		}
	}
}

func checkRetryables(arbState *ArbosState, expected []statetransfer.InitializationDataForRetryable, t *testing.T) {
	ret := arbState.RetryableState()
	for _, exp := range expected {
		found, err := ret.OpenRetryable(exp.Id, 0)
		Require(t, err)
		if found == nil {
			Fail(t)
		}
		// TODO: detailed comparison
	}
}

func checkAccounts(db *state.StateDB, arbState *ArbosState, accts []statetransfer.AccountInitializationInfo, t *testing.T) {
	l1p := arbState.L1PricingState()
	posterTable := l1p.BatchPosterTable()
	for _, acct := range accts {
		addr := acct.Addr
		if db.GetNonce(addr) != acct.Nonce {
			t.Fatal()
		}
		if db.GetBalance(addr).ToBig().Cmp(acct.EthBalance) != 0 {
			t.Fatal()
		}
		if acct.ContractInfo != nil {
			if !bytes.Equal(acct.ContractInfo.Code, db.GetCode(addr)) {
				t.Fatal()
			}
			err := state.ForEachStorage(db, addr, func(key common.Hash, value common.Hash) bool {
				if key == (common.Hash{}) {
					// Unfortunately, geth doesn't seem capable of giving us storage keys any more.
					// Even with the triedb Preimages set to true, it doesn't record the necessary
					// hashed storage key -> raw storage key mapping. This means that geth will always
					// give us an empty storage key when iterating, which we can't validate.
					return true
				}
				val2, exists := acct.ContractInfo.ContractStorage[key]
				if !exists {
					t.Fatal("address", addr, "key", key, "found in storage as", value, "but not in initialization data")
				}
				if value != val2 {
					t.Fatal("address", addr, "key", key, "value", val2, "isn't what was specified in initialization data", value)
				}
				return true
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		isPoster, err := posterTable.ContainsPoster(addr)
		Require(t, err)
		if acct.AggregatorInfo != nil && isPoster {
			posterInfo, err := posterTable.OpenPoster(addr, false)
			Require(t, err)
			fc, err := posterInfo.PayTo()
			Require(t, err)
			if fc != acct.AggregatorInfo.FeeCollector {
				t.Fatal()
			}
		}
	}
	_ = l1p
}
