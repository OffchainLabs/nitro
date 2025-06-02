package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// this test calls the ArbBlockNumber function on the ArbSys precompile
func TestDimTxOpArbSysBlockNumber(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cleanup()
	defer cancel()

	_, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)

	_, precompileTestContract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPrecompile)

	_, receipt := callOnContract(t, builder, auth, precompileTestContract.TestArbSysArbBlockNumber)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
