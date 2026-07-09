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
	callErr     error
	lastCallMsg ethereum.CallMsg
}

func (b *proxyBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	b.lastCallMsg = call
	return nil, b.callErr
}

type rpcRevertError struct{}

func (rpcRevertError) Error() string  { return "some opaque rpc message" }
func (rpcRevertError) ErrorCode() int { return 3 }

func Test_stakeTokenSupportsWethDeposit_ProxiedWeth(t *testing.T) {
	ctx := context.Background()
	stakeToken := common.HexToAddress("0x00000000000000000000000000000000deadbeef")

	proxyCode := common.FromHex("0x363d3d373d3d3d363d73bebebebebebebebebebebebebebebebebebebebe5af43d82803e903d91602b57fd5bf3")
	require.False(t, bytes.Contains(proxyCode, wethDepositSelector))

	backend := &proxyBackend{callErr: nil}

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

	for _, tc := range []struct {
		name    string
		callErr error
	}{
		{"bare", errors.New("execution reverted")},
		{"with reason suffix", errors.New("execution reverted: SafeERC20: low-level call failed")},
		{"rpc code 3", rpcRevertError{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := &proxyBackend{callErr: tc.callErr}
			got, err := stakeTokenSupportsWethDeposit(ctx, backend, common.Address{}, stakeToken)
			require.NoError(t, err)
			require.False(t, got)
		})
	}
}

func Test_stakeTokenSupportsWethDeposit_TransientError(t *testing.T) {
	ctx := context.Background()
	stakeToken := common.HexToAddress("0x00000000000000000000000000000000deadbeef")

	backend := &proxyBackend{callErr: errors.New("connection refused")}

	got, err := stakeTokenSupportsWethDeposit(ctx, backend, common.Address{}, stakeToken)
	require.ErrorContains(t, err, "connection refused")
	require.False(t, got)
}
