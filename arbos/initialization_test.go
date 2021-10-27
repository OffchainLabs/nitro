//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/util"
	"testing"
)

func TestJsonMarshalUnmarshalSimple(t *testing.T) {
	input := ArbosInitializationInfo{
		[]common.Address{pseudorandomAddressForTesting(nil, 0)},
		[]common.Hash{pseudorandomHashForTesting(nil, 1), pseudorandomHashForTesting(nil, 2)},
		pseudorandomAddressForTesting(nil, 3),
		[]InitializationDataForRetryable{pseudorandomRetryableInitForTesting(nil, 4)},
		[]AccountInitializationInfo{pseudorandomAccountInitInfoForTesting(nil, 5)},
	}
	if len(input.AddressTableContents) != 1 {
		t.Fatal(input)
	}
	marshaled, err := json.Marshal(&input)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(marshaled) {
		t.Fatal()
	}
	if len(marshaled) == 0 {
		t.Fatal()
	}

	output := ArbosInitializationInfo{}
	err = json.Unmarshal(marshaled, &output)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.AddressTableContents) != 1 {
		t.Fatal(output)
	}
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

func pseudorandomRetryableInitForTesting(salt *common.Hash, x uint64) InitializationDataForRetryable {
	newSalt := pseudorandomHashForTesting(salt, x)
	salt = &newSalt
	return InitializationDataForRetryable{
		pseudorandomHashForTesting(salt, 0),
		pseudorandomUint64ForTesting(salt, 1),
		pseudorandomAddressForTesting(salt, 2),
		pseudorandomAddressForTesting(salt, 3),
		pseudorandomHashForTesting(salt, 4).Big(),
		pseudorandomDataForTesting(salt, 5, 256),
	}
}

func pseudorandomAccountInitInfoForTesting(salt *common.Hash, x uint64) AccountInitializationInfo {
	newSalt := pseudorandomHashForTesting(salt, x)
	salt = &newSalt
	aggToPay := pseudorandomAddressForTesting(salt, 7)
	return AccountInitializationInfo{
		pseudorandomAddressForTesting(salt, 0),
		pseudorandomUint64ForTesting(salt, 1),
		pseudorandomHashForTesting(salt, 2).Big(),
		&AccountInitContractInfo{
			pseudorandomDataForTesting(salt, 3, 256),
			pseudorandomHashHashMapForTesting(salt, 4, 16),
		},
		&AccountInitAggregatorInfo{
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
