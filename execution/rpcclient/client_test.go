// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	utilrpc "github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// mockExecutionService implements a minimal execution RPC service for testing.
type mockExecutionService struct {
	err error
}

func (s *mockExecutionService) HeadMessageIndex(_ context.Context) (arbutil.MessageIndex, error) {
	return 0, s.err
}

func createMockExecutionNode(t *testing.T, errToReturn error) *node.Node {
	t.Helper()
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{execution.RPCNamespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	testhelpers.RequireImpl(t, err)

	stack.RegisterAPIs([]rpc.API{{
		Namespace: execution.RPCNamespace,
		Service:   &mockExecutionService{err: errToReturn},
		Public:    true,
	}})

	testhelpers.RequireImpl(t, stack.Start())
	t.Cleanup(func() { _ = stack.Close() })
	return stack
}

func TestClientErrorHandling(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	tests := []struct {
		name        string
		serverErr   error
		expectedErr error
	}{
		{
			name:        "ResultNotFound mapped to sentinel",
			serverErr:   execution.ErrResultNotFound,
			expectedErr: execution.ErrResultNotFound,
		},
		{
			name:        "ResultNotFound wrapped in longer message mapped to sentinel",
			serverErr:   fmt.Errorf("execution context: %w", execution.ErrResultNotFound),
			expectedErr: execution.ErrResultNotFound,
		},
		{
			name:        "ErrRetrySequencer mapped to sentinel",
			serverErr:   execution.ErrRetrySequencer,
			expectedErr: execution.ErrRetrySequencer,
		},
		{
			name:        "ErrRetrySequencer wrapped in longer message mapped to sentinel",
			serverErr:   fmt.Errorf("rpc context: %w", execution.ErrRetrySequencer),
			expectedErr: execution.ErrRetrySequencer,
		},
		{
			name:        "generic error message is preserved",
			serverErr:   errors.New("unexpected failure"),
			expectedErr: errors.New("unexpected failure"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stack := createMockExecutionNode(t, tc.serverErr)

			config := &utilrpc.ClientConfig{
				URL:     "self",
				Timeout: 5 * time.Second,
			}
			testhelpers.RequireImpl(t, config.Validate())

			client := NewClient(func() *utilrpc.ClientConfig { return config }, stack)
			testhelpers.RequireImpl(t, client.Start(ctx))
			defer client.StopAndWait()

			_, err := client.HeadMessageIndex().Await(ctx)

			if err == nil {
				t.Fatal("expected an error from server, got nil")
			}
			switch {
			case errors.Is(tc.expectedErr, execution.ErrResultNotFound), errors.Is(tc.expectedErr, execution.ErrRetrySequencer):
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected sentinel error %v, got %v", tc.expectedErr, err)
				}
			default:
				if err.Error() != tc.expectedErr.Error() {
					t.Errorf("expected error message %q, got %q", tc.expectedErr.Error(), err.Error())
				}
			}
		})
	}
}
