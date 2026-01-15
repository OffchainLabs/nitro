// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package testhelpers

import (
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

func CreateStackConfigForTest(dataDir string) *node.Config {
	stackConf := node.DefaultConfig
	// stackConf.Name is used when creating data path used by the node
	// if stackConf is not set, program binary name is used instead
	// We hardcode it to enable running the tests that need to know the path also when test binary name is different than default,
	// eg. when debugging with dlv test the debug binary name differs from normal test build
	stackConf.Name = "test-stack-name"
	stackConf.DataDir = dataDir
	stackConf.UseLightweightKDF = true
	stackConf.WSPort = 0
	stackConf.WSModules = append(stackConf.WSModules, "eth", "debug")
	stackConf.HTTPPort = 0
	stackConf.HTTPHost = ""
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth", "debug")
	stackConf.AuthPort = 0
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.NoDial = true
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NAT = nil
	stackConf.DBEngine = env.GetTestDatabaseEngine()
	return &stackConf
}
