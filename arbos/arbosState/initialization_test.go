//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/statetransfer"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	tryMarshalUnmarshal(
		&statetransfer.ArbosInitializationInfo{
			AddressTableContents: []common.Address{pseudorandomAddressForTesting(nil, 0)},
			SendPartials:         []common.Hash{pseudorandomHashForTesting(nil, 1), pseudorandomHashForTesting(nil, 2)},
			DefaultAggregator:    pseudorandomAddressForTesting(nil, 3),
			RetryableData:        []statetransfer.InitializationDataForRetryable{pseudorandomRetryableInitForTesting(nil, 4)},
			Accounts:             []statetransfer.AccountInitializationInfo{pseudorandomAccountInitInfoForTesting(nil, 5)},
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

	genesisAlloc, err := GetGenesisAllocFromJSON(marshaled)
	Require(t, err)
	genesis := core.Genesis{Alloc: genesisAlloc}

	raw := rawdb.NewMemoryDatabase()
	Require(t, err)

	block, err := genesis.Commit(raw)
	Require(t, err)
	stateDb, err := state.New(block.Header().Root, state.NewDatabase(raw), nil)
	Require(t, err)

	arbState, err := OpenArbosState(stateDb, &burn.SystemBurner{})
	Require(t, err)
	checkAddressTable(arbState, input.AddressTableContents, t)
	checkSendAccum(arbState, input.SendPartials, t)
	checkDefaultAgg(arbState, input.DefaultAggregator, t)
	checkRetryables(arbState, input.RetryableData, t)
	checkAccounts(stateDb, arbState, input.Accounts, t)
}

func pseudorandomHashForTesting(salt *common.Hash, x uint64) common.Hash {
	if salt == nil {
		return crypto.Keccak256Hash(common.Hash{}.Bytes(), util.IntToHash(int64(x)).Bytes())
	} else {
		return crypto.Keccak256Hash(salt.Bytes(), util.IntToHash(int64(x)).Bytes())
	}
}

func pseudorandomAddressForTesting(salt *common.Hash, x uint64) common.Address {
	return common.BytesToAddress(pseudorandomHashForTesting(salt, x).Bytes()[:20])
}

func pseudorandomUint64ForTesting(salt *common.Hash, x uint64) uint64 {
	return binary.BigEndian.Uint64(pseudorandomHashForTesting(salt, x).Bytes()[:8])
}

func pseudorandomRetryableInitForTesting(salt *common.Hash, x uint64) statetransfer.InitializationDataForRetryable {
	newSalt := pseudorandomHashForTesting(salt, x)
	salt = &newSalt
	return statetransfer.InitializationDataForRetryable{
		Id:          pseudorandomHashForTesting(salt, 0),
		Timeout:     pseudorandomUint64ForTesting(salt, 1),
		From:        pseudorandomAddressForTesting(salt, 2),
		To:          pseudorandomAddressForTesting(salt, 3),
		Callvalue:   pseudorandomHashForTesting(salt, 4).Big(),
		Beneficiary: pseudorandomAddressForTesting(salt, 5),
		Calldata:    pseudorandomDataForTesting(salt, 6, 256),
	}
}

func pseudorandomAccountInitInfoForTesting(salt *common.Hash, x uint64) statetransfer.AccountInitializationInfo {
	newSalt := pseudorandomHashForTesting(salt, x)
	salt = &newSalt
	aggToPay := pseudorandomAddressForTesting(salt, 7)
	return statetransfer.AccountInitializationInfo{
		Addr:       pseudorandomAddressForTesting(salt, 0),
		Nonce:      pseudorandomUint64ForTesting(salt, 1),
		EthBalance: pseudorandomHashForTesting(salt, 2).Big(),
		ContractInfo: &statetransfer.AccountInitContractInfo{
			Code:            pseudorandomDataForTesting(salt, 3, 256),
			ContractStorage: pseudorandomHashHashMapForTesting(salt, 4, 16),
		},
		AggregatorInfo: &statetransfer.AccountInitAggregatorInfo{
			FeeCollector: pseudorandomAddressForTesting(salt, 5),
			BaseFeeL1Gas: pseudorandomHashForTesting(salt, 6).Big(),
		},
		AggregatorToPay: &aggToPay,
	}
}

func pseudorandomDataForTesting(salt *common.Hash, x uint64, maxSize uint64) []byte {
	newSalt := pseudorandomHashForTesting(salt, x)
	salt = &newSalt
	size := pseudorandomUint64ForTesting(salt, 1) % maxSize
	ret := []byte{}
	for uint64(len(ret)) < size {
		ret = append(ret, pseudorandomHashForTesting(salt, uint64(len(ret))).Bytes()...)
	}
	return ret[:size]
}

func pseudorandomHashHashMapForTesting(salt *common.Hash, x uint64, maxItems uint64) map[common.Hash]common.Hash {
	size := int(pseudorandomUint64ForTesting(salt, 0) % maxItems)
	ret := make(map[common.Hash]common.Hash)
	for i := 0; i < size; i++ {
		ret[pseudorandomHashForTesting(salt, 2*x+1)] = pseudorandomHashForTesting(salt, 2*x+2)
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

func checkSendAccum(arbState *ArbosState, expected []common.Hash, t *testing.T) {
	sa := arbState.SendMerkleAccumulator()
	partials, err := sa.GetPartials()
	Require(t, err)
	if len(partials) != len(expected) {
		t.Fatal()
	}
	pexp := make([]*common.Hash, len(expected))
	for i, partial := range partials {
		if *partial != expected[i] {
			t.Fatal()
		}
		pexp[i] = &expected[i]
	}
	acc2, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(pexp)
	Require(t, err)
	a2Root, err := acc2.Root()
	Require(t, err)
	saRoot, err := sa.Root()
	Require(t, err)
	if a2Root != saRoot {
		t.Fatal()
	}
}

func checkDefaultAgg(arbState *ArbosState, expected common.Address, t *testing.T) {
	da, err := arbState.L1PricingState().DefaultAggregator()
	Require(t, err)
	if da != expected {
		Fail(t)
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
	for _, acct := range accts {
		addr := acct.Addr
		if db.GetNonce(addr) != acct.Nonce {
			t.Fatal()
		}
		if db.GetBalance(addr).Cmp(acct.EthBalance) != 0 {
			t.Fatal()
		}
		if acct.ContractInfo != nil {
			if !bytes.Equal(acct.ContractInfo.Code, db.GetCode(addr)) {
				t.Fatal()
			}
			err := db.ForEachStorage(addr, func(key common.Hash, value common.Hash) bool {
				val2, exists := acct.ContractInfo.ContractStorage[key]
				if !exists {
					t.Fatal()
				}
				if value != val2 {
					t.Fatal()
				}
				return false
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if acct.AggregatorInfo != nil {
			fc, err := l1p.AggregatorFeeCollector(addr)
			Require(t, err)
			if fc != acct.AggregatorInfo.FeeCollector {
				t.Fatal()
			}
			charge, err := l1p.FixedChargeForAggregatorL1Gas(addr)
			Require(t, err)
			if charge.Cmp(acct.AggregatorInfo.BaseFeeL1Gas) != 0 {
				Fail(t)
			}
		}
		if acct.AggregatorToPay != nil {
			prefAgg, _, err := l1p.PreferredAggregator(addr)
			Require(t, err)
			if prefAgg != *acct.AggregatorToPay {
				Fail(t)
			}
		}
	}
	_ = l1p
}
