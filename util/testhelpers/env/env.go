// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package env

import (
	"flag"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	stateSchemeFlag = flag.String("TEST_STATE_SCHEME", "", "State scheme to use for tests")
)

// There are two CI steps, one to run tests using the path state scheme, and one to run tests using the hash state scheme.
// An environment variable controls that behavior.
func GetTestStateScheme() string {
	stateScheme := rawdb.HashScheme
	if *stateSchemeFlag == rawdb.PathScheme || *stateSchemeFlag == rawdb.HashScheme {
		stateScheme = *stateSchemeFlag
	}
	log.Debug("test state scheme", "testStateScheme", stateScheme)
	return stateScheme
}
