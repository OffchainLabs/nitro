// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/programs"
)

func TestProgramStorageCacheLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.StylusTarget.MaxStorageCacheSlots = 1
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	l2client := builder.L2.Client
	l2info := builder.L2Info
	multicall := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	revertMessage := []byte(programs.ErrStorageCacheLimitExceeded.Error())
	reverter := deployContract(t, ctx, auth, l2client, deployContractInitCode(revertMessage, true))
	revertArgs := multicallAppend(multicallEmptyArgs(), vm.CALL, reverter, nil)

	_, err := l2client.CallContract(ctx, ethereum.CallMsg{To: &multicall, Gas: 1e9, Data: revertArgs}, nil)
	require.ErrorContains(t, err, "execution reverted")

	revertTx := l2info.PrepareTxTo("Owner", &multicall, 1e9, nil, revertArgs)
	require.NoError(t, l2client.SendTransaction(ctx, revertTx))
	EnsureTxFailed(t, ctx, l2client, revertTx)

	storeArgs := multicallEmptyArgs()
	storeArgs = multicallAppendStore(storeArgs, common.HexToHash("0x01"), common.HexToHash("0x11"), false, true)
	storeArgs = multicallAppendStore(storeArgs, common.HexToHash("0x02"), common.HexToHash("0x22"), false, true)

	_, err = l2client.CallContract(ctx, ethereum.CallMsg{To: &multicall, Gas: 1e9, Data: storeArgs}, nil)
	require.ErrorContains(t, err, programs.ErrStorageCacheLimitExceeded.Error())

	limitTx := l2info.PrepareTxTo("Owner", &multicall, 1e9, nil, storeArgs)
	err = l2client.SendTransaction(ctx, limitTx)
	if err == nil || !strings.Contains(err.Error(), state.ErrSeqFilter.Error()) {
		t.Fatalf("transaction exceeding the storage cache limit should have been rejected with %q, got %v", state.ErrSeqFilter, err)
	}
}
