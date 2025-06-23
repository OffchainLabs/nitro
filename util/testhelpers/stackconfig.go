// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package testhelpers

import (
	"time"

	"github.com/ethereum/go-ethereum/node"
)

func CreateStackConfigForTest(dataDir string) *node.Config {
	stackConf := node.DefaultConfig
	// stackConf.Name is used when creating data path used by the node
	// if stackConf not set, program binary name is used instead,
	// hardcode it so as it's possible to run the tests that need to know the path also from different test binary name,
	// eg. when debugging with dlv test the debug binary name differs from normal test build
	stackConf.Name = "test-stack-name"
	stackConf.DataDir = dataDir
	stackConf.UseLightweightKDF = true
	stackConf.WSPort = 0
	stackConf.WSModules = append(stackConf.WSModules, "eth", "debug")
	stackConf.HTTPPort = 0
	stackConf.HTTPHost = ""
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth", "debug")
	// FIXME remove
	stackConf.HTTPTimeouts.ReadTimeout = time.Hour
	stackConf.AuthPort = 0
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.NoDial = true
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NAT = nil
	stackConf.DBEngine = "leveldb"
	return &stackConf
}
