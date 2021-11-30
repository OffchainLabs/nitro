//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/statetransfer"
	"testing"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	tryMarshalUnmarshal(
		&statetransfer.ArbosInitializationInfo{
			[]common.Address{pseudorandomAddressForTesting(nil, 0)},
			[]common.Hash{pseudorandomHashForTesting(nil, 1), pseudorandomHashForTesting(nil, 2)},
			pseudorandomAddressForTesting(nil, 3),
			[]statetransfer.InitializationDataForRetryable{pseudorandomRetryableInitForTesting(nil, 4)},
			[]statetransfer.AccountInitializationInfo{pseudorandomAccountInitInfoForTesting(nil, 5)},
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

	db, err := OpenStateDBForTest()
	if err != nil {
		t.Fatal(err)
	}
	if err := InitializeArbosFromJSON(db, marshaled); err != nil {
		t.Fatal(err)
	}
	arbState := OpenArbosState(db)
	checkAddressTable(arbState, input.AddressTableContents, t)
	checkSendAccum(arbState, input.SendPartials, t)
	checkDefaultAgg(arbState, input.DefaultAggregator, t)
	checkRetryables(arbState, input.RetryableData, t)
	checkAccounts(db, arbState, input.Accounts, t)
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
		pseudorandomHashForTesting(salt, 0),
		pseudorandomUint64ForTesting(salt, 1),
		pseudorandomAddressForTesting(salt, 2),
		pseudorandomAddressForTesting(salt, 3),
		pseudorandomHashForTesting(salt, 4).Big(),
		pseudorandomDataForTesting(salt, 5, 256),
	}
}

func pseudorandomAccountInitInfoForTesting(salt *common.Hash, x uint64) statetransfer.AccountInitializationInfo {
	newSalt := pseudorandomHashForTesting(salt, x)
	salt = &newSalt
	aggToPay := pseudorandomAddressForTesting(salt, 7)
	return statetransfer.AccountInitializationInfo{
		pseudorandomAddressForTesting(salt, 0),
		pseudorandomUint64ForTesting(salt, 1),
		pseudorandomHashForTesting(salt, 2).Big(),
		&statetransfer.AccountInitContractInfo{
			pseudorandomDataForTesting(salt, 3, 256),
			pseudorandomHashHashMapForTesting(salt, 4, 16),
		},
		&statetransfer.AccountInitAggregatorInfo{
			pseudorandomAddressForTesting(salt, 5),
			pseudorandomHashForTesting(salt, 6).Big(),
		},
		&aggToPay,
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
	if atab.Size() != uint64(len(addrTable)) {
		t.Fatal()
	}
	for i, addr := range addrTable {
		res, exists := atab.LookupIndex(uint64(i))
		if !exists {
			t.Fatal()
		}
		if res != addr {
			t.Fatal()
		}
	}
}

func checkSendAccum(arbState *ArbosState, expected []common.Hash, t *testing.T) {
	sa := arbState.SendMerkleAccumulator()
	partials := sa.GetPartials()
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
	acc2 := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(pexp)
	if acc2.Root() != sa.Root() {
		t.Fatal()
	}
}

func checkDefaultAgg(arbState *ArbosState, expected common.Address, t *testing.T) {
	if arbState.L1PricingState().DefaultAggregator() != expected {
		t.Fatal()
	}
}

func checkRetryables(arbState *ArbosState, expected []statetransfer.InitializationDataForRetryable, t *testing.T) {
	ret := arbState.RetryableState()
	for _, exp := range expected {
		found := ret.OpenRetryable(exp.Id, 0)
		if found == nil {
			t.Fatal()
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
			if l1p.AggregatorFeeCollector(addr) != acct.AggregatorInfo.FeeCollector {
				t.Fatal()
			}
			if l1p.FixedChargeForAggregatorL1Gas(addr).Cmp(acct.AggregatorInfo.BaseFeeL1Gas) != 0 {
				t.Fatal()
			}
		}
		if acct.AggregatorToPay != nil {
			prefAgg, _ := l1p.PreferredAggregator(addr)
			if prefAgg != *acct.AggregatorToPay {
				t.Fatal()
			}
		}
	}
	_ = l1p
}
