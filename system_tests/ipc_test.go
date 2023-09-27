// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/cmd/genericconf"
)

func TestIpcRpc(t *testing.T) {
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")

	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath

	stackConf := stackConfigForTest(t)
	ipcConfig.Apply(stackConf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetL2StackConfig(stackConf).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNode.L1Stack)
	defer testNode.L2Node.StopAndWait()

	_, err := ethclient.Dial(ipcPath)
	Require(t, err)
}
