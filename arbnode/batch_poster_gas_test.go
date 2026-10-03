// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util"
)

// fakeRPCClient answers eth_estimateGas and eth_call from canned results.
type fakeRPCClient struct {
	estimate    hexutil.Uint64
	estimateErr error
	callErr     error
	calls       []string
}

func (c *fakeRPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	c.calls = append(c.calls, method)
	switch method {
	case "eth_estimateGas":
		if c.estimateErr != nil {
			return c.estimateErr
		}
		gas, ok := result.(*hexutil.Uint64)
		if !ok {
			return errors.New("unexpected result type for eth_estimateGas")
		}
		*gas = c.estimate
		return nil
	case "eth_call":
		return c.callErr
	default:
		return errors.New("unexpected method " + method)
	}
}

func (c *fakeRPCClient) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeRPCClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return errors.New("not implemented")
}

func (c *fakeRPCClient) Close() {}

func TestEstimateGas(t *testing.T) {
	revertErr := errors.New("execution reverted: some reason")
	otherErr := errors.New("connection reset")

	for _, tc := range []struct {
		name          string
		client        *fakeRPCClient
		expectedGas   uint64
		expectedErrs  []error
		expectedCalls []string
	}{
		{
			name:          "success",
			client:        &fakeRPCClient{estimate: 30000},
			expectedGas:   30000,
			expectedCalls: []string{"eth_estimateGas"},
		},
		{
			name:          "revert then successful eth_call still errors",
			client:        &fakeRPCClient{estimateErr: revertErr},
			expectedErrs:  []error{revertErr},
			expectedCalls: []string{"eth_estimateGas", "eth_call"},
		},
		{
			name:          "revert reports both the estimation and eth_call errors",
			client:        &fakeRPCClient{estimateErr: revertErr, callErr: otherErr},
			expectedErrs:  []error{revertErr, otherErr},
			expectedCalls: []string{"eth_estimateGas", "eth_call"},
		},
		{
			name:          "non-revert error is returned as is",
			client:        &fakeRPCClient{estimateErr: otherErr},
			expectedErrs:  []error{otherErr},
			expectedCalls: []string{"eth_estimateGas"},
		},
		{
			name:          "zero estimate is an error",
			client:        &fakeRPCClient{estimate: 0},
			expectedErrs:  []error{util.ErrZeroGasEstimate},
			expectedCalls: []string{"eth_estimateGas"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gas, err := estimateGas(tc.client, context.Background(), estimateGasParams{}, "latest")
			if len(tc.expectedErrs) == 0 && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			for _, expectedErr := range tc.expectedErrs {
				if !errors.Is(err, expectedErr) {
					t.Fatalf("expected error %v, got %v", expectedErr, err)
				}
			}
			if gas != tc.expectedGas {
				t.Errorf("expected gas %d, got %d", tc.expectedGas, gas)
			}
			if len(tc.client.calls) != len(tc.expectedCalls) {
				t.Fatalf("expected calls %v, got %v", tc.expectedCalls, tc.client.calls)
			}
			for i, method := range tc.expectedCalls {
				if tc.client.calls[i] != method {
					t.Errorf("call %d: expected %s, got %s", i, method, tc.client.calls[i])
				}
			}
		})
	}
}
