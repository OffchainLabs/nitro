// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package jsonapi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func TestPreimagesMapJson(t *testing.T) {
	t.Parallel()
	for _, preimages := range []PreimagesMapJson{
		{},
		{make(map[common.Hash][]byte)},
		{map[common.Hash][]byte{
			{}: {},
		}},
		{map[common.Hash][]byte{
			{1}: {1},
			{2}: {1, 2},
			{3}: {1, 2, 3},
		}},
	} {
		t.Run(fmt.Sprintf("%v preimages", len(preimages.Map)), func(t *testing.T) {
			// These test cases are fast enough that t.Parallel() probably isn't worth it
			serialized, err := preimages.MarshalJSON()
			Require(t, err, "Failed to marshal preimagesj")

			// Make sure that `serialized` is a valid JSON map
			stringMap := make(map[string]string)
			err = json.Unmarshal(serialized, &stringMap)
			Require(t, err, "Failed to unmarshal preimages as string map")
			if len(stringMap) != len(preimages.Map) {
				t.Errorf("Got %v entries in string map but only had %v preimages", len(stringMap), len(preimages.Map))
			}

			var deserialized PreimagesMapJson
			err = deserialized.UnmarshalJSON(serialized)
			Require(t, err)

			if (len(preimages.Map) > 0 || len(deserialized.Map) > 0) && !reflect.DeepEqual(preimages, deserialized) {
				t.Errorf("Preimages map %v serialized to %v but then deserialized to different map %v", preimages, string(serialized), deserialized)
			}
		})
	}
}
