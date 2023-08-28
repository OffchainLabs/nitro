// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build benchmarks
// +build benchmarks

package arbtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestBenchmarkGas(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, l2node, l2client, _, _, _, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		return EnsureTxFailed(t, ctx, l2client, tx)
	}

	auth := l2info.GetDefaultTransactOpts("Faucet", ctx)
	auth.GasLimit = 32000000

	var programTest *mocksgen.Benchmarks
	timed(t, "deploy", func() {
		_, _, contract, err := mocksgen.DeployBenchmarks(&auth, l2client)
		Require(t, err)
		programTest = contract
	})
	bench := func(name string, lambda func() *types.Receipt) {
		now := time.Now()
		receipt := lambda()
		passed := time.Since(now)
		ratio := float64(passed.Nanoseconds()) / float64(receipt.GasUsedForL2())
		fmt.Printf("Bench %-10v %v %.2f\n", name, formatTime(passed), ratio)
	}
	bench("ecrecover", func() *types.Receipt {
		return ensure(programTest.FillBlockRecover(&auth))
	})
	bench("keccak", func() *types.Receipt {
		return ensure(programTest.FillBlockHash(&auth))
	})
	bench("quick step", func() *types.Receipt {
		return ensure(programTest.FillBlockQuickStep(&auth))
	})
}
