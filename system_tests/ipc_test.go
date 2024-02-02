// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestIpcRpc(t *testing.T) {
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.IPCPath = ipcPath
	cleanup := builder.Build(t)
	defer cleanup()

	_, err := ethclient.Dial(ipcPath)
	Require(t, err)
}

func TestPendingBlockTimeAndNumberAdvance(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)

	_, _, testTimeAndNr, err := mocksgen.DeployPendingBlkTimeAndNrAdvanceCheck(&auth, builder.L2.Client)
	Require(t, err)

	time.Sleep(1 * time.Second)

	_, err = testTimeAndNr.IsAdvancing(&auth)
	Require(t, err)
}
