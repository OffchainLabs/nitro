// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
)

func TestIpcRpc(t *testing.T) {
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")

	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.WSPort = 0
	stackConf.UseLightweightKDF = true
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.NAT = nil
	stackConf.DataDir = t.TempDir()
	stackConf.IPCPath = ipcPath

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, _, l2stack, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nil, nil, &stackConf)
	defer requireClose(t, l1stack)
	defer requireClose(t, l2stack)

	_, err := ethclient.Dial(ipcPath)
	Require(t, err)
}
