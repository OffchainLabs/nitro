// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sol

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

type proxyBackend struct {
	MockContractBackend
	code        []byte
	callErr     error
	lastCallMsg ethereum.CallMsg
}

func (b *proxyBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return b.code, nil
}

func (b *proxyBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	b.lastCallMsg = call
	return nil, b.callErr
}

// Regression test for https://github.com/OffchainLabs/nitro/pull/4617: a WETH
// behind a proxy has only a delegatecall stub as bytecode, so the bytecode scan
// misclassified it as non-WETH and broke auto-deposit.
func Test_stakeTokenSupportsWethDeposit_ProxiedWeth(t *testing.T) {
	ctx := context.Background()
	stakeToken := common.HexToAddress("0x00000000000000000000000000000000deadbeef")

	proxyCode := common.FromHex("0x363d3d373d3d3d363d73bebebebebebebebebebebebebebebebebebebebe5af43d82803e903d91602b57fd5bf3")
	require.False(t, bytes.Contains(proxyCode, wethDepositSelector))

	backend := &proxyBackend{code: proxyCode, callErr: nil}

	got, err := stakeTokenSupportsWethDeposit(ctx, backend, common.Address{}, stakeToken)
	require.NoError(t, err)
	require.True(t, got)

	require.Equal(t, wethDepositSelector, backend.lastCallMsg.Data)
	require.Equal(t, &stakeToken, backend.lastCallMsg.To)
	require.Nil(t, backend.lastCallMsg.Value)
}

func Test_stakeTokenSupportsWethDeposit_PlainERC20(t *testing.T) {
	ctx := context.Background()
	stakeToken := common.HexToAddress("0x000000000000000000000000000000000badc0de")

	backend := &proxyBackend{callErr: errors.New("execution reverted")}

	got, err := stakeTokenSupportsWethDeposit(ctx, backend, common.Address{}, stakeToken)
	require.NoError(t, err)
	require.False(t, got)
}

func Test_stakeTokenSupportsWethDeposit_TransientError(t *testing.T) {
	ctx := context.Background()
	stakeToken := common.HexToAddress("0x00000000000000000000000000000000deadbeef")

	backend := &proxyBackend{callErr: errors.New("connection refused")}

	got, err := stakeTokenSupportsWethDeposit(ctx, backend, common.Address{}, stakeToken)
	require.Error(t, err)
	require.False(t, got)
}
