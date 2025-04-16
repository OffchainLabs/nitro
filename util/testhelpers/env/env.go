// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package env

import (
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
)

// There are two CI steps, one to run tests using the path state scheme, and one to run tests using the hash state scheme.
// An environment variable controls that behavior.
func GetTestStateScheme() string {
	stateScheme := rawdb.HashScheme
	if *testflag.StateSchemeFlag == rawdb.PathScheme || *testflag.StateSchemeFlag == rawdb.HashScheme {
		stateScheme = *testflag.StateSchemeFlag
	}
	log.Debug("test state scheme", "testStateScheme", stateScheme)
	return stateScheme
}
