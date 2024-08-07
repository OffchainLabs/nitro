// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package testhelpers

import "github.com/ethereum/go-ethereum/node"

func CreateStackConfigForTest(dataDir string) *node.Config {
	stackConf := node.DefaultConfig
	stackConf.DataDir = dataDir
	stackConf.UseLightweightKDF = true
	stackConf.WSPort = 0
	stackConf.WSModules = append(stackConf.WSModules, "eth", "debug")
	stackConf.HTTPPort = 0
	stackConf.HTTPHost = ""
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth", "debug")
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.NoDial = true
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NAT = nil
	stackConf.DBEngine = "leveldb"
	return &stackConf
}
