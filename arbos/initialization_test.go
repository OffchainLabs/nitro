//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestJsonMarshalUnmarshalSimple(t *testing.T) {
	input := ArbosInitializationInfo{
		[]common.Address{common.BytesToAddress([]byte{3, 4, 5})},
		[]common.Hash{},
		common.Address{},
		[]InitializationDataForRetryable{},
		[]AccountInitializationInfo{},
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
