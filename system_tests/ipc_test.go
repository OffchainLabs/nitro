// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
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
