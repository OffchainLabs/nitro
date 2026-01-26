package server_common

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/malicious"
)

func ResolveModuleRoot(locator *MachineLocator, moduleRoot common.Hash) common.Hash {
	if !malicious.OverrideWasmModuleRoot() {
		return moduleRoot
	}
	if locator == nil {
		return moduleRoot
	}
	latest := locator.LatestWasmModuleRoot()
	if latest == (common.Hash{}) {
		return moduleRoot
	}
	return latest
}
