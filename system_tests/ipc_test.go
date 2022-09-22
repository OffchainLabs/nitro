// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestIpcRpc(t *testing.T) {
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")

	stackConf := getTestStackConfig(t)
	stackConf.IPCPath = ipcPath

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, _, l2stack, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nil, nil, &stackConf)
	defer requireClose(t, l1stack)
	defer requireClose(t, l2stack)

	_, err := ethclient.Dial(ipcPath)
	Require(t, err)
}
