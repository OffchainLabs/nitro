// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package env

import (
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

// There are two CI steps, one to run tests using the path state scheme, and one to run tests using the hash state scheme.
// An environment variable controls that behavior.
func GetTestStateScheme() string {
	testhelpers.ParseFlag()
	stateScheme := rawdb.HashScheme
	if *testhelpers.StateSchemeFlag == rawdb.PathScheme || *testhelpers.StateSchemeFlag == rawdb.HashScheme {
		stateScheme = *testhelpers.StateSchemeFlag
	}
	log.Debug("test state scheme", "testStateScheme", stateScheme)
	return stateScheme
}
